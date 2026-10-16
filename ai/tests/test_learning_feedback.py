import contextlib
import copy
import io
from pathlib import Path
import random
import tempfile
import unittest
from unittest.mock import patch
import torch

from smallworld_ai.engine import LearnerEnv
from smallworld_ai.graph import S, A, N
from smallworld_ai.hierarchical import HierarchicalPolicy
from smallworld_ai.league import add_snapshot
from smallworld_ai.model import load_checkpoint
from smallworld_ai.train import parser, run


class FeedbackTests(unittest.TestCase):
    def test_selected_action_embedding_and_gradient_match_dense_and_packed(self):
        torch.set_num_threads(1)
        torch.manual_seed(937)
        batch = (torch.randn(3,S),torch.randn(3,5,A),torch.tensor([[1,0,0,0,0],[1,1,1,1,1],[1,1,1,0,0]],dtype=torch.bool),
                 torch.randn(3,4,N),torch.eye(4).repeat(3,1,1),torch.ones(3,4),torch.randint(-1,4,(3,5,2)))
        actions = torch.tensor([0,4,1])
        model = HierarchicalPolicy(32,joint=True)
        dense = copy.deepcopy(model)
        dense.pack_candidates = False
        for p in (model,dense):
            dist,value,hidden = p.training_forward(batch,actions)
            if p is model:
                expected = (dist.probs.detach(),value.detach(),hidden.detach())
            else:
                for x,y in zip((dist.probs,value,hidden),expected):
                    torch.testing.assert_close(x,y,atol=1e-6,rtol=1e-5)
            head = torch.ones(32,1)
            loss = torch.nn.functional.binary_cross_entropy_with_logits((hidden@head).flatten(),torch.tensor([1.,0.,1.]))
            loss.backward()
        self.assertGreater(float(model.actor[0].weight.grad.abs().sum()),0)
        self.assertGreater(float(model.graph[0].weight.grad.abs().sum()),0)
        self.assertIsNone(model.actor[4].weight.grad)  # supervision trains representation, not a scripted action score
        for a,b in zip(model.parameters(),dense.parameters()):
            if a.grad is not None:
                torch.testing.assert_close(a.grad,b.grad,atol=2e-6,rtol=2e-4)

    def test_historical_league_preserves_anchor_and_latest_with_diversity(self):
        pool=['anchor']; seen=0; rng=random.Random(82)
        for i in range(100):
            seen=add_snapshot(pool,i,seen,rng=rng)
            self.assertEqual(pool[0],'anchor')
            self.assertEqual(pool[-1],i)
            self.assertLessEqual(len(pool),8)
            self.assertEqual(len(pool),len(set(pool)))
        self.assertEqual(seen,100)
        self.assertTrue(any(i<50 for i in pool[1:-1]))
        self.assertEqual(len(pool),8)

    def test_individual_opponent_seats_remain_frozen(self):
        class Engine:
            def __init__(self,*a,**kw):self.calls=[];self.next_player=1
            def request(self,**kw):
                if kw['op']=='step':
                    self.calls.append(kw['policy']);self.next_player=(self.next_player+1)%4
                return dict(done=False,truncated=False,decision=dict(player=self.next_player))
            def close(self):pass
        with patch('smallworld_ai.engine.Engine',Engine):
            env=LearnerEnv('unused',1,4,['map4players'],'mixed')
            with patch.object(env.rng,'randrange',return_value=0), patch.object(env,'_sample_opponent',side_effect=['random','greedy','random']) as choose:
                env.reset()
                self.assertEqual(choose.call_count,3)
                self.assertEqual(env.engine.calls,['random','greedy','random'])
                self.assertEqual(env.opponent_policies,{1:'random',2:'greedy',3:'random'})

    def test_auxiliary_training_checkpoint_resumes_without_changing_export_format(self):
        torch.set_num_threads(1)
        with tempfile.TemporaryDirectory() as directory:
            args=['--device','cpu','--rules','surface','--architecture','joint','--hidden','32',
                  '--envs','2','--rollout-steps','16','--total-steps','32','--epochs','1','--batch-size','16',
                  '--opponent','random','--maps','map4players','--players','4','--acceptance-loss','.05',
                  '--async-rollouts','--snapshot-every','1','--output',directory]
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            model,checkpoint=load_checkpoint(Path(directory)/'latest.pt')
            self.assertEqual(model.format_version,4)
            self.assertIn('acceptance',checkpoint)
            self.assertEqual(checkpoint['league_snapshots_seen'],1)
            self.assertTrue(checkpoint['acceptance']['optimizer']['state'])
            args[args.index('--total-steps')+1]='64'
            args+=['--resume',str(Path(directory)/'latest.pt')]
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            model,resumed=load_checkpoint(Path(directory)/'latest.pt')
            self.assertEqual(resumed['step'],64)
            self.assertEqual(resumed['league_snapshots_seen'],2)
            self.assertTrue(all(torch.isfinite(p).all() for p in model.parameters()))
            self.assertTrue(any(not torch.equal(checkpoint['acceptance']['model'][k],v) for k,v in resumed['acceptance']['model'].items()))

if __name__=='__main__':unittest.main()
