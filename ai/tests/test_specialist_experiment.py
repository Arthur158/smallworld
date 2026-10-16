from argparse import Namespace
from pathlib import Path
import unittest

from specialist_experiment import commands
from smallworld_ai.specialist_evaluate import fixtures, opponent_lineup
from smallworld_ai.specialists import RACES, TRAITS
from collections import Counter


class ExperimentTests(unittest.TestCase):
    def test_every_player_count_faces_the_full_opponent_pool(self):
        opponents=['random','greedy','parent','training5','training6']
        jobs=fixtures(100,1,[2,3,4,5,6])
        for players in range(2,7):
            faced=Counter()
            for job in jobs:
                if job['players']!=players:continue
                lineup=opponent_lineup(job,opponents)
                self.assertEqual(lineup,opponent_lineup(job,opponents))
                faced.update(rival for seat,rival in enumerate(lineup) if seat!=job['seat'])
            self.assertEqual(set(faced),set(opponents))
        # A saved fixture predating this change retains its original lineup.
        self.assertEqual(opponent_lineup(dict(index=3,players=2),opponents),['training5','training6'])

    def test_evaluation_covers_roster_players_and_seats_reproducibly(self):
        jobs=fixtures(100,2,[2,3,4,5,6])
        self.assertEqual(jobs,fixtures(100,2,[2,3,4,5,6]))
        self.assertNotEqual(jobs,fixtures(101,2,[2,3,4,5,6]))
        self.assertEqual(len({j['seed'] for j in jobs}),len(jobs))
        for players in range(2,7):
            rows=[j for j in jobs if j['players']==players]
            self.assertEqual({j['race'] for j in rows},set(RACES))
            self.assertEqual({j['trait'] for j in rows},set(TRAITS))
            self.assertEqual({j['seat'] for j in rows},set(range(players)))
        self.assertEqual({j['players'] for j in jobs[:5]},set(range(2,7)))

    def test_arms_share_parent_budget_and_frozen_opponents(self):
        args=Namespace(envs=4,rollout_steps=32,steps=256,arms=['baseline','curriculum','specialists'],
                       checkpoint=Path('/tmp/parent.pt'),output=Path('/tmp/arms'),engine=Path('/tmp/botenv'),
                       device='auto',opponent_device='cpu',batch_size=64,epochs=2,seed=55,
                       fraction=.75,freeze_backbone=False)
        parent=dict(format_version=6,iteration=10,step=1000,config={})
        plans=commands(args,parent)
        for arm,command in plans.items():
            self.assertEqual(command[command.index('--resume')+1],'/tmp/parent.pt')
            self.assertEqual(command[command.index('--total-steps')+1],'1256')
            self.assertEqual(command[command.index('--snapshot-every')+1],'13')
            self.assertEqual(command[command.index('--league-fraction')+1],'0')
            self.assertEqual('--specialist-curriculum' in command,arm!='baseline')
            self.assertEqual('--upgrade-specialists' in command,arm=='specialists')
        self.assertIn('--no-freeze-backbone',plans['specialists'])
        args.steps=257
        with self.assertRaises(ValueError):commands(args,parent)
