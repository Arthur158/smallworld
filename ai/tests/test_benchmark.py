from argparse import Namespace
import hashlib
import importlib.util
import json
from pathlib import Path
import tempfile
import unittest
import torch

from smallworld_ai.benchmark import report_pairs, run
from smallworld_ai.engine import DEFAULT_ENGINE
from smallworld_ai.hierarchical import HierarchicalPolicy
from smallworld_ai.league import policy_payload
from smallworld_ai.model import atomic_torch_save

spec = importlib.util.spec_from_file_location('league_pull_model', Path(__file__).parents[1]/'pull_model.py')
pull = importlib.util.module_from_spec(spec)
spec.loader.exec_module(pull)


def rows(winner):
    result=[]
    for seed in range(12):
        for seat in range(4):
            for contender in ('challenger','incumbent'):
                scores = [50]*4
                scores[seat] = 60 if contender == winner else 40
                policies=['rival-0','rival-1','rival-2']
                policies.insert(seat, 'eval-'+contender)
                result.append(dict(seed=seed,seat=seat,contender=contender,scores=scores,policies=policies,
                                   actions=[10]*4,rejections=[0]*4,successful_actions=[{} for _ in range(4)]))
    return result


class BenchmarkTests(unittest.TestCase):
    def test_promotion_requires_paired_advantage_and_preserves_losses(self):
        self.assertTrue(report_pairs(rows('challenger'))['promote'])
        self.assertFalse(report_pairs(rows('incumbent'))['promote'])
        self.assertFalse(report_pairs(rows('neither'))['promote'])
        with self.assertRaisesRegex(ValueError, 'incomplete'):
            report_pairs(rows('challenger')[:-1])
        sample=rows('challenger')
        with self.assertRaisesRegex(ValueError, 'duplicate'):
            report_pairs(sample+[sample[0]])

    def test_pull_uses_verified_champion_and_requires_opt_in_for_latest(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            latest=root/'latest.pt';latest.write_bytes(b'latest')
            self.assertEqual(pull.select_checkpoint(root)[0],latest)
            (root/'league.json').write_text('{}')
            with self.assertRaisesRegex(ValueError, 'No evaluated champion'):
                pull.select_checkpoint(root)
            self.assertEqual(pull.select_checkpoint(root,'latest')[0],latest)
            (root/'champions').mkdir()
            champion=root/'champions'/'older.pt';champion.write_bytes(b'older')
            (root/'champion.json').write_text(json.dumps(dict(checkpoint=str(champion),status='evaluated',sha256=hashlib.sha256(b'older').hexdigest())))
            self.assertEqual(pull.select_checkpoint(root)[0],champion)
            champion.write_bytes(b'corrupted')
            with self.assertRaisesRegex(ValueError,'checksum'):
                pull.select_checkpoint(root)

    def test_actual_four_player_games_replay_and_produce_a_verified_export_source(self):
        torch.set_num_threads(1)
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            members=[]
            for i in range(3):
                path=root/f'rival-{i}.pt'
                atomic_torch_save(policy_payload(HierarchicalPolicy(32,joint=True))|{'step':i,'iteration':i},path)
                members.append(dict(id=f'rival-{i}',name=f'Rival {i}',checkpoint=str(path),sha256=hashlib.sha256(path.read_bytes()).hexdigest()))
            league=root/'league.json';league.write_text(json.dumps(dict(version=1,members=members)))
            (root/'latest.pt').write_bytes((root/'rival-0.pt').read_bytes())
            args=Namespace(run=root,league=league,baseline=root/'rival-0.pt',engine=DEFAULT_ENGINE,
                           maps=3,workers=2,seed=2_100_000_000,interval=60,regression_tolerance=.05,once=True)
            run(args)
            gate=json.loads((root/'champion.json').read_text())
            report=json.loads(Path(gate['report']).read_text())
            self.assertTrue(report['replay_verified'])
            self.assertEqual(report['paired_games'],12)
            self.assertEqual(report['paired_win_credit_delta'],0)
            self.assertFalse(report['promote'])
            self.assertEqual(pull.select_checkpoint(root)[0],Path(gate['checkpoint']))


if __name__=='__main__':unittest.main()
