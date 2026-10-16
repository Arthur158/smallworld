import contextlib
import hashlib
import io
import json
from pathlib import Path
import random
import tempfile
import unittest
import torch

from smallworld_ai.hierarchical import HierarchicalPolicy
from smallworld_ai.league import OpponentLeague, add_snapshot, policy_payload
from smallworld_ai.model import atomic_torch_save, load_checkpoint
from smallworld_ai.train import parser, run


class LeagueTests(unittest.TestCase):
    def setUp(self):
        torch.set_num_threads(1)
        self.temp = tempfile.TemporaryDirectory()
        self.root = Path(self.temp.name)
        members = []
        for i in range(3):
            path = self.root / f'rival-{i}.pt'
            atomic_torch_save(policy_payload(HierarchicalPolicy(32, joint=True)) | {'step': i}, path)
            members.append({'id': f'rival-{i}', 'name': f'Rival {i}', 'checkpoint': path.name,
                            'sha256': hashlib.sha256(path.read_bytes()).hexdigest()})
        self.manifest = self.root / 'league.json'
        self.manifest.write_text(json.dumps({'version': 1, 'members': members}))

    def tearDown(self):
        self.temp.cleanup()

    def test_frozen_rivals_survive_changing_pool_and_resume_without_source_files(self):
        league = OpponentLeague(self.manifest)
        pool, seen = [0], 0
        for i in range(100):
            seen = add_snapshot(pool, i, seen, rng=random.Random(i))
        self.assertEqual(len(league.members), 3)
        self.assertFalse(any(p.requires_grad for m in league.members for p in m['policy'].parameters()))
        league.record([50, 60, 40, 50], 0, {1: 'rival-0', 2: 'rival-1', 3: 'rival-0'})
        state = league.state_dict()
        restored = OpponentLeague(state=state)
        for p in self.root.glob('*.pt'): p.unlink()
        self.assertEqual(restored.metrics(), league.metrics())
        for before, after in zip(league.members, restored.members):
            for x, y in zip(before['policy'].parameters(), after['policy'].parameters()):
                torch.testing.assert_close(x, y)
        self.assertEqual(restored.members[0]['statistics']['games'], 1)
        self.assertEqual(restored.members[0]['statistics']['by_players']['4']['credit'], .25)
        restored.record([20,10],0,{1:'rival-0'})
        self.assertEqual(league.members[0]['statistics']['games'], 1)

    def test_harder_rivals_gain_weight_but_every_member_keeps_a_floor(self):
        league = OpponentLeague(self.manifest)
        for _ in range(100):
            league.record([50,60,40,50],0,{1:'rival-0',2:'rival-1',3:'rival-2'})
        weights = [m['sampling_probability'] for m in league.metrics()]
        self.assertGreater(weights[0], weights[2])
        self.assertGreater(weights[2], weights[1])
        self.assertGreaterEqual(min(weights), .5/3)
        for _ in range(20):
            chosen = league.choose(random.Random(), exclude={'rival-0','rival-1'})
            self.assertEqual(league.identify(chosen), 'rival-2')

    def test_modified_weight_file_is_rejected(self):
        with (self.root/'rival-0.pt').open('ab') as file: file.write(b'changed')
        with self.assertRaisesRegex(ValueError, 'checksum'):
            OpponentLeague(self.manifest)

    def test_real_training_plays_and_resumes_permanent_rivals(self):
        output = self.root/'run'
        args = ['--rules','surface','--architecture','joint','--hidden','32','--device','cpu',
                '--envs','2','--players','4','--rollout-steps','32','--total-steps','128','--epochs','1','--batch-size','32',
                '--async-rollouts','--opponent','selfplay','--league-fraction','1',
                '--league',str(self.manifest),'--exploration-fraction','.15','--output',str(output)]
        with contextlib.redirect_stdout(io.StringIO()): run(parser().parse_args(args))
        _, before = load_checkpoint(output/'latest.pt')
        self.assertEqual(len(before['permanent_league']['members']), 3)
        self.assertTrue(all(m['statistics']['selections'] for m in before['permanent_league']['members']))
        args[args.index('--total-steps')+1] = '192'
        del args[args.index('--league'):args.index('--league')+2]
        args += ['--resume', str(output/'latest.pt')]
        for p in self.root.glob('*.pt'): p.unlink()
        with contextlib.redirect_stdout(io.StringIO()): run(parser().parse_args(args))
        _, after = load_checkpoint(output/'latest.pt')
        for a, b in zip(before['permanent_league']['members'], after['permanent_league']['members']):
            self.assertGreaterEqual(b['statistics']['selections'], a['statistics']['selections'])


if __name__ == '__main__': unittest.main()
