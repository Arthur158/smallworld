"""One isolated Go process per environment; no web server or training accounts."""
import json
from pathlib import Path
import random
import selectors
import subprocess

ROOT = Path(__file__).resolve().parents[1]
DEFAULT_ENGINE = ROOT / "bin" / "botenv"


class Engine:
    def __init__(self, binary=DEFAULT_ENGINE, model_dir=None, max_steps=4096):
        binary = Path(binary).resolve()
        if not binary.is_file():
            raise FileNotFoundError(f"Build the engine first: (cd backend2 && go build -o ../ai/bin/botenv ./cmd/botenv). Missing: {binary}")
        self.generator = None
        self.map_info = None
        command = [str(binary), "-max-steps", str(max_steps)]
        if model_dir:
            command.extend(["-model-dir", str(Path(model_dir).resolve())])
        self.process = subprocess.Popen(command, stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                                        text=True, bufsize=1)

    def request(self, **request):
        if request.get("op") == "reset":
            split = request.pop("map_split", "train")
            distribution = request.pop("map_distribution", "standard")
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


class LearnerEnv:
    """A learner step ends at that same player's next decision, not an opponent's.

    Opponent snapshots are frozen for an entire episode. Rewards contain own
    coin change / 100 plus +1/-1/0 for win/loss/tie at the true game end.
    """
    def __init__(self, binary, seed, players, maps, opponent, max_steps=4096):
        self.engine = Engine(binary, max_steps=max_steps)
        self.rng = random.Random(seed)
        self.players, self.maps, self.opponent = players, maps, opponent
        self.snapshots = []
        self.seat = 0
        self.response = None
        self.opponent_policy = None
        self.episode_return = 0.0

    def reset(self):
        self.seat = self.rng.randrange(self.players)
        self.episode_return = 0.0
        policy = self.opponent
        if policy == "mixed":
            policy = self.rng.choice(["random", "greedy", "selfplay"] if self.snapshots else ["random", "greedy"])
        self.opponent_policy = self.rng.choice(self.snapshots) if policy == "selfplay" and self.snapshots else policy
        if self.opponent_policy == "selfplay":
            self.opponent_policy = "random"
        self.response = self.engine.request(op="reset", seed=self.rng.randrange(1, 2**53),
                                            players=self.players, map=self.rng.choice(self.maps))
        self._advance_opponents()
        return self.response["decision"]

    def _advance_opponents(self):
        while not (self.response["done"] or self.response["truncated"]) and self.response["decision"]["player"] != self.seat:
            if isinstance(self.opponent_policy, str):
                self.response = self.engine.request(op="step", policy=self.opponent_policy)
            else:
                self.response = self.engine.request(op="step", action=self.opponent_policy.choose(self.response["decision"]))

    def step(self, action):
        before = self.response["scores"][self.seat]
        self.response = self.engine.request(op="step", action=int(action))
        self._advance_opponents()
        reward = (self.response["scores"][self.seat] - before) / 100.0
        if self.response["done"]:
            reward += outcome(self.response["scores"], self.seat)
        self.episode_return += reward
        info = {"scores": self.response["scores"], "seat": self.seat,
                "engine_steps": self.response["steps"], "map": self.engine.map_info, "episode_return": self.episode_return}
        return self.response["decision"], reward, self.response["done"], self.response["truncated"], info

    def close(self):
        self.engine.close()
