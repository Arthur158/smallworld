"""Shared, resumable practice quotas; no named-race tactics or reward changes."""
from collections import Counter
import copy
import random
from threading import Lock

from .graph import SPEC


class SpecialistCurriculum:
    def __init__(self, seed, players, fraction=.75, state=None):
        self.races = list(SPEC['training_races'])
        self.traits = list(SPEC['training_traits'])
        self.players = sorted(set(players))
        if not self.players or any(p not in range(2, 7) for p in self.players) or not 0 < fraction < 1:
            raise ValueError('dedicated practice requires players 2..6 and a fraction between 0 and 1')
        self.fraction, self.lock, self.rng = fraction, Lock(), random.Random(seed)
        self.requests = {key: Counter() for key in ('race', 'trait', 'pair', 'players')}
        self.statistics = {key: {} for key in ('race', 'trait')}
        self.episodes = 0
        if state:
            if state['races'] != self.races or state['traits'] != self.traits or state['players'] != self.players or state['fraction'] != fraction:
                raise ValueError('cannot silently change a resumed specialist curriculum')
            self.requests = {key: Counter(value) for key, value in state['requests'].items()}
            self.statistics = copy.deepcopy(state['statistics'])
            self.episodes = state['episodes']
            self.rng.setstate(state['rng'])

    def _least(self, choices, key):
        values = [(key(choice), self.rng.random(), choice) for choice in choices]
        return min(values)[2]

    def next(self):
        with self.lock:
            players = self._least(self.players, lambda p: self.requests['players'][str(p)])
            self.requests['players'][str(players)] += 1
            # Fixed proportions avoid an unlucky run starving ordinary choices.
            focused = int((self.episodes+1)*self.fraction) > int(self.episodes*self.fraction)
            self.episodes += 1
            if not focused:
                return dict(players=players, focus=False)
            race = self._least(self.races, lambda r: self.requests['race'][r])
            trait = self._least(self.traits, lambda t: (self.requests['trait'][t], self.requests['pair'][race+'|'+t]))
            self.requests['race'][race] += 1
            self.requests['trait'][trait] += 1
            self.requests['pair'][race+'|'+trait] += 1
            return dict(players=players, focus=True, focus_race=race, focus_trait=trait)

    def record(self, info, finished):
        with self.lock:
            if info['action']['kind'] != 'pick':
                for kind, names in (('race', [info.get('expert_race', '')]), ('trait', info.get('expert_traits', []))):
                    for name in names:
                        if name in (self.races if kind == 'race' else self.traits):
                            row = self.statistics[kind].setdefault(name, dict(decisions=0, rejected=0, games=0, win_credit=0.))
                            row['decisions'] += 1
                            row['rejected'] += int(info.get('rejected', False))
            if finished:
                scores, seat = info['scores'], info['seat']
                credit = 1/scores.count(scores[seat]) if scores[seat] == max(scores) else 0.
                for kind in ('race', 'trait'):
                    for name in info.get('played_'+kind, []):
                        if name in (self.races if kind == 'race' else self.traits):
                            row = self.statistics[kind].setdefault(name, dict(decisions=0, rejected=0, games=0, win_credit=0.))
                            row['games'] += 1
                            row['win_credit'] += credit

    def state_dict(self):
        with self.lock:
            return dict(races=self.races, traits=self.traits, players=self.players, fraction=self.fraction,
                        requests={key: dict(value) for key, value in self.requests.items()},
                        statistics=copy.deepcopy(self.statistics), episodes=self.episodes, rng=self.rng.getstate())

    def report(self):
        saved = self.state_dict()
        return {kind: {name: dict(assigned_games=saved['requests'][kind].get(name, 0),
                                 **saved['statistics'][kind].get(name, dict(decisions=0, rejected=0, games=0, win_credit=0.)))
                       for name in names} for kind, names in (('race', self.races), ('trait', self.traits))}
