"""Frozen permanent rivals plus a bounded, changing historical opponent pool."""
import copy
import hashlib
import json
from pathlib import Path
import random
from threading import Lock


def policy_payload(policy):
    return {"format_version": getattr(policy, 'format_version', 2 if policy.schema == 'smallworld-bot-v2' else 1),
            "schema": policy.schema, "config": {"hidden": policy.hidden, "freeze_backbone": getattr(policy, "backbone_frozen", False)},
            "model": policy.state_dict()}


class OpponentLeague:
    """Never evict named rivals; gently prefer rivals the learner struggles with.

    Every member retains at least half of its uniform sampling probability.
    Performance is the learner's relative finish against that opponent, not a
    tactical heuristic. Repeated seats count once per opponent per game.
    Frozen weights and statistics are embedded in resumable checkpoints.
    """
    def __init__(self, manifest=None, state=None, schema='smallworld-bot-v2'):
        from .model import load_checkpoint, policy_from_checkpoint
        self.lock = Lock()
        self.members = []
        saved = {m['id']: m for m in (state or {}).get('members', [])}
        if manifest:
            path = Path(manifest).resolve()
            specs = json.loads(path.read_text())
            if specs.get('version') != 1 or not specs.get('members'):
                raise ValueError('league manifest needs version 1 and nonempty members')
            specifications = specs['members']
        else:
            path = None
            specifications = list(saved.values())
        identifiers = set()
        hashes = set()
        for spec in specifications:
            identifier = spec['id']
            if not identifier or identifier in identifiers:
                raise ValueError('duplicate or empty league member ID')
            identifiers.add(identifier)
            previous = saved.get(identifier)
            if previous and previous['sha256'] != spec['sha256']:
                raise ValueError('cannot change the weights of an existing league member')
            if path:
                source = Path(spec['checkpoint']).expanduser()
                if not source.is_absolute():
                    source = path.parent / source
                with source.open('rb') as stream:
                    digest = hashlib.file_digest(stream, 'sha256').hexdigest()
                if digest != spec['sha256']:
                    raise ValueError(f'league checkpoint checksum mismatch: {identifier}')
                policy, checkpoint = load_checkpoint(source)
                step = checkpoint.get('step', 0)
            else:
                digest = spec['sha256']
                policy = policy_from_checkpoint(spec['policy'])
                step = spec.get('step', 0)
            if digest in hashes:
                raise ValueError('duplicate weights in permanent opponent league')
            hashes.add(digest)
            if policy.schema != schema:
                raise ValueError('league opponent schema differs from learner')
            policy.cpu().eval().requires_grad_(False)
            statistics = copy.deepcopy((previous or {}).get('statistics', {}))
            statistics.setdefault('selections', 0)
            statistics.setdefault('games', 0)
            statistics.setdefault('learner_relative_score_ema', .5)
            statistics.setdefault('by_players', {})
            self.members.append(dict(id=identifier, name=spec.get('name', identifier),
                                     sha256=digest, step=step, policy=policy,
                                     statistics=statistics))
        self.identities = {id(m['policy']): m['id'] for m in self.members}

    def __bool__(self):
        return bool(self.members)

    def _weights(self, members):
        hardness = [(1-m['statistics']['learner_relative_score_ema'])**2 + .05 for m in members]
        total = sum(hardness)
        return [.5/len(members) + .5*h/total for h in hardness]

    def choose(self, rng, exclude=()):
        with self.lock:
            choices = [m for m in self.members if m['id'] not in exclude] or self.members
            member = rng.choices(choices, weights=self._weights(choices))[0]
            member['statistics']['selections'] += 1
            return member['policy']

    def identify(self, policy):
        return self.identities.get(id(policy))

    def record(self, scores, learner, opponents):
        per_opponent = {}
        for seat, identifier in opponents.items():
            if identifier is not None:
                credit = float(scores[learner] > scores[seat]) + .5*float(scores[learner] == scores[seat])
                per_opponent.setdefault(identifier, []).append(credit)
        with self.lock:
            for member in self.members:
                values = per_opponent.get(member['id'])
                if not values:
                    continue
                score = sum(values)/len(values)
                stats = member['statistics']
                stats['games'] += 1
                stats['learner_relative_score_ema'] += .03*(score-stats['learner_relative_score_ema'])
                key = str(len(scores))
                cell = stats['by_players'].setdefault(key, {'games': 0, 'credit': 0.})
                cell['games'] += 1
                cell['credit'] += score

    def metrics(self):
        with self.lock:
            return [{k: m[k] for k in ('id', 'name', 'step')} |
                    {'sampling_probability': weight, **copy.deepcopy(m['statistics'])}
                    for m, weight in zip(self.members, self._weights(self.members))] if self.members else []

    def state_dict(self):
        with self.lock:
            return {'version': 1, 'members': [
                {k: copy.deepcopy(m[k]) for k in ('id', 'name', 'sha256', 'step', 'statistics')} |
                {'policy': policy_payload(m['policy'])} for m in self.members]}


def add_snapshot(pool, snapshot, snapshots_seen, capacity=8, rng=random):
    """Keep the anchor, newest policy, and a uniform reservoir of older policies.

    snapshots_seen counts policies added after the anchor, including the current
    latest. The previous latest joins the historical reservoir on each update.
    Active episodes keep references to their frozen policies even if evicted.
    """
    if capacity < 3 or not pool or snapshots_seen < 0:
        raise ValueError('invalid opponent league')
    history = list(pool[1:-1])
    slots = capacity-2
    if len(pool) > 1:
        previous_latest = pool[-1]
        if len(history) < slots:
            history.append(previous_latest)
        else:
            index = rng.randrange(max(snapshots_seen, len(history)+1))
            if index < slots:
                history[index] = previous_latest
    pool[:] = [pool[0], *history, snapshot]
    return snapshots_seen+1
