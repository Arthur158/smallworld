"""One isolated Go process per environment; no web server or training accounts."""
import base64
import json
from pathlib import Path
import random
import selectors
import subprocess

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_ENGINE = ROOT / "bin" / "botenv"


class Engine:
    def __init__(self, binary=DEFAULT_ENGINE, model_dir=None, max_steps=4096, rules="legacy", packed=False):
        self.rules = rules
        binary = Path(binary).resolve()
        if not binary.is_file():
            raise FileNotFoundError(f"Build the engine first: (cd backend2 && go build -o ../ai/bin/botenv ./cmd/botenv). Missing: {binary}")
        self.generator = None
        self.map_info = None
        command = [str(binary), "-max-steps", str(max_steps)]
        if packed:
            command.append("-packed-observations")
        if model_dir:
            command.extend(["-model-dir", str(Path(model_dir).resolve())])
        self.process = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        text=True, bufsize=1)

    def request(self, **request):
        if request.get("op") == "reset":
            split = request.pop("map_split", "train")
            request.setdefault("rules", self.rules)
            distribution = request.pop("map_distribution", "surface" if request["rules"] == "surface" else "standard")
            if request.get("map") == "generated":
                from .maps import MapGenerator
                if self.generator is None:
                    self.generator = MapGenerator()
                request["tiles"], self.map_info = self.generator.generate(request["seed"], request["players"], split, distribution)
            else:
                self.map_info = {"builtin": request.get("map")}
        if self.process.poll() is not None:
            raise RuntimeError(f"game engine exited with code {self.process.returncode}")
        self.process.stdin.write(json.dumps(request) + "\n")
        self.process.stdin.flush()
        with selectors.DefaultSelector() as selector:
            selector.register(self.process.stdout, selectors.EVENT_READ)
            if not selector.select(30):
                raise RuntimeError("game engine timed out")
        line = self.process.stdout.readline()
        if not line:
            raise RuntimeError("game engine closed its output; see its stderr above")
        response = json.loads(line)
        if response.get("error"):
            raise RuntimeError(f"game engine rejected {request['op']}: {response['error']}")
        packed = response.pop("packed_decision", None)
        if packed:
            import numpy as np
            d = response["decision"]
            def array(name, shape):
                data = base64.b64decode(packed[name] or "", validate=True)
                return np.frombuffer(data, dtype='<f4').reshape(shape)
            d["state"] = array("state", (packed["state_width"],))
            features = array("candidates", (len(d["candidates"]), packed["action_width"]))
            for candidate, values in zip(d["candidates"], features):
                candidate["features"] = values
            if d.get("graph") is not None:
                d["graph"]["nodes"] = array("nodes", (packed["node_count"], packed["node_width"]))
        return response

    def close(self):
        if self.generator is not None:
            self.generator.close()
            self.generator = None
        if self.process.stdin and not self.process.stdin.closed:
            self.process.stdin.close()
        try:
            self.process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()
        if self.process.stdout:
            self.process.stdout.close()

    def __enter__(self):
        return self

    def __exit__(self, *args):
        self.close()


def outcome(scores, seat):
    opponents = scores[:seat] + scores[seat+1:]
    best_other = max(opponents)
    return 1 if scores[seat] > best_other else -1 if scores[seat] < best_other else 0


def score_margin(scores, seat):
    return (scores[seat]-max(scores[:seat]+scores[seat+1:]))/100.


def winning_reward(before, after, seat, done, initial_margin=0.):
    """Potential shaping of final win credit; intermediate margins cancel.

    With gamma=1 the episode sum is exactly 2*shared_win_credit-1.
    Centering the potential at reset also covers seats whose opponents already
    moved before the learner's first decision. Scores are reward labels only;
    private opponent coins are never added to the policy's observation.
    """
    previous = score_margin(before, seat)-initial_margin
    if done:
        credit = 1/after.count(after[seat]) if after[seat] == max(after) else 0.
        return 2*credit-1-previous
    return score_margin(after, seat)-initial_margin-previous


class LearnerEnv:
    """A learner step ends at that same player's next decision, not an opponent's.

    Opponent snapshots are frozen for an entire episode. Legacy rewards contain
    own coin changes plus terminal outcome. The win objective uses final shared
    win credit with temporary relative-score feedback that cancels at game end.
    """
    def __init__(self, binary, seed, players, maps, opponent, max_steps=4096, rules="legacy", curriculum_start=0, curriculum_stride=1, focus_fraction=0.25, rejection_penalty=0.001, packed=True, objective="score"):
        if objective not in ('score', 'win') or objective == 'win' and rejection_penalty != 0:
            raise ValueError('win objective requires zero reward penalty for rejections')
        self.objective, self.initial_margin = objective, 0.
        self.rules, self.curriculum, self.curriculum_stride = rules, curriculum_start, curriculum_stride
        self.focus_fraction, self.rejection_penalty = focus_fraction, rejection_penalty
        self.episodes = 0
        self.engine = Engine(binary, max_steps=max_steps, rules=rules, packed=packed)
        self.rng = random.Random(seed)
        self.player_counts = players if isinstance(players, list) else [players]
        self.players, self.maps, self.opponent = self.player_counts[0], maps, opponent
        self.snapshots = []
        self.seat = 0
        self.response = None
        self.opponent_policy = None
        self.opponent_policies = {}
        self.opponent_inference = None
        self.league = None
        self.league_fraction = .75
        self.opponent_ids = {}
        self.episode_return = 0.0
        self.exploration = None
        self.specialist_curriculum = None
        self.played_race, self.played_trait = set(), set()
        self.exploration_retry_group = -1
        self.exploration_seen = set()
        self.exploration_earned = 0.0

    def reset(self):
        self.players = self.rng.choice(self.player_counts)
        focus = self.rules == "surface" and self.rng.random() < self.focus_fraction
        assignment = self.specialist_curriculum.next() if self.specialist_curriculum else None
        if assignment:
            self.players, focus = assignment['players'], assignment['focus']
        self.played_race.clear()
        self.played_trait.clear()
        self.seat = self.rng.randrange(self.players)
        self.episodes += 1
        self.episode_return = 0.0
        self.exploration_retry_group = -1
        self.exploration_seen.clear()
        self.exploration_earned = 0.0
        # Each opponent seat gets an independent frozen policy for the whole
        # episode, exposing the learner to mixed strategies at the same table.
        self.opponent_policies, self.opponent_ids = {}, {}
        for seat in range(self.players):
            if seat == self.seat:
                continue
            selected = self._sample_opponent()
            self.opponent_policies[seat] = selected
            self.opponent_ids[seat] = self.league.identify(selected) if self.league else None
        self.opponent_policy = next(iter(self.opponent_policies.values()))
        self.response = self.engine.request(op="reset", seed=self.rng.randrange(1, 2**53),
                                            players=self.players, map=self.rng.choice(self.maps),
                                            curriculum=self.curriculum, focus=focus, focus_player=self.seat,
                                            **({key: value for key, value in assignment.items() if key in ('focus_race', 'focus_trait')} if assignment else {}))
        self.curriculum += self.curriculum_stride
        self._advance_opponents()
        self.initial_margin = score_margin(self.response['scores'], self.seat) if self.objective == 'win' else 0.
        return self.response["decision"]

    def _sample_opponent(self):
        policy = self.opponent
        if policy == "mixed":
            if self.rules == "surface" and (self.snapshots or self.league):
                policy = self.rng.choices(["random", "greedy", "selfplay"], weights=[1, 2, 7])[0]
            else:
                policy = self.rng.choice(["random", "greedy", "selfplay"] if self.snapshots else ["random", "greedy"])
        if policy == "selfplay":
            if self.league and (not self.snapshots or self.rng.random() < self.league_fraction):
                return self.league.choose(self.rng, exclude=self.opponent_ids.values())
            return self.rng.choice(self.snapshots) if self.snapshots else "random"
        return policy

    def _advance_opponents(self):
        while not (self.response["done"] or self.response["truncated"]) and self.response["decision"]["player"] != self.seat:
            policy = self.opponent_policies[self.response["decision"]["player"]]
            if isinstance(policy, str):
                self.response = self.engine.request(op="step", policy=policy)
            else:
                action = (self.opponent_inference.choose(policy, self.response["decision"])
                          if self.opponent_inference else policy.choose(self.response["decision"]))
                self.response = self.engine.request(op="step", action=action)

    def step(self, action):
        before_scores = self.response['scores'][:]
        before = before_scores[self.seat]
        decision = self.response["decision"]
        candidate = decision["candidates"][int(action)]
        selected = candidate["action"]
        if selected['kind'] != 'pick':
            if candidate.get('expert_race'): self.played_race.add(candidate['expert_race'])
            self.played_trait.update(candidate.get('expert_traits') or [])
        available = sorted({a["action"]["kind"]+":"+a["action"].get("stack", "") for a in decision["candidates"]})
        self.response = self.engine.request(op="step", action=int(action))
        rejected = self.response.get("rejected", False)
        turn_ended = self.response['done'] or self.response['decision']['player'] != self.seat
        self._advance_opponents()
        from .feedback import survival_labels, response_labels
        survival = survival_labels(decision, self.response['decision']) if turn_ended and not self.response['done'] and not self.response['truncated'] else None
        response = response_labels(decision, self.response['decision']) if survival is not None else None
        reward = (self.response["scores"][self.seat] - before) / 100.0 - (self.rejection_penalty if rejected else 0)
        if self.objective == 'win':
            reward = winning_reward(before_scores, self.response['scores'], self.seat, self.response['done'], self.initial_margin)
        elif self.response["done"]:
            reward += outcome(self.response["scores"], self.seat)
        if self.response['done']:
            if self.league and not self.response['truncated']:
                self.league.record(self.response['scores'], self.seat, self.opponent_ids)
        self.episode_return += reward
        exploration_reward = (self.exploration.reward(selected, rejected, self.exploration_seen, self.exploration_earned)
                              if self.exploration else 0.0)
        self.exploration_earned += exploration_reward
        info = {"scores": self.response["scores"], "seat": self.seat,
                "engine_steps": self.response["steps"], "rejected": rejected, "action": selected, "available": available, "coverage": self.response.get("coverage", {}), "map": self.engine.map_info, "episode_return": self.episode_return, "exploration_reward": exploration_reward, "opponents": dict(self.opponent_ids), "turn_ended": turn_ended, "survival": survival}
        info.update(expert_race=candidate.get('expert_race', ''), expert_traits=candidate.get('expert_traits', []),
                    played_race=sorted(self.played_race), played_trait=sorted(self.played_trait))
        info['opponent_response'] = response
        return self.response["decision"], reward+exploration_reward, self.response["done"], self.response["truncated"], info

    def close(self):
        self.engine.close()
