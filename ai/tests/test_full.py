import copy
import json
from pathlib import Path
import tempfile
import unittest

import torch

from smallworld_ai.engine import Engine
from smallworld_ai.graph import GraphPolicy, S, SPEC
from smallworld_ai.model import Policy, batch_decisions, export_policy, merge_decision_batches


class FullPolicyTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        torch.set_num_threads(1)

    def decisions(self):
        with Engine(rules='surface') as e:
            return [e.request(op='reset', seed=51+i, map='generated', players=2+i, curriculum=i, focus=True)['decision'] for i in range(2)]

    def test_warm_start_retains_legacy_logits_and_values(self):
        old=Policy(16)
        new=GraphPolicy(32)
        new.warm_start(old)
        for d in self.decisions():
            legacy=copy.deepcopy(d)
            legacy['schema']='smallworld-bot-v1';legacy['state']=legacy['state'][:171]
            for a in legacy['candidates']:a['features']=a['features'][:58]
            actual=new(*batch_decisions([d],'cpu'))
            expected=old(*batch_decisions([legacy],'cpu'))
            torch.testing.assert_close(actual[0].logits,expected[0].logits)
            torch.testing.assert_close(actual[1],expected[1])

    def test_graph_permutation_and_padding_invariance(self):
        decisions=self.decisions();model=GraphPolicy(32)
        packed=merge_decision_batches([batch_decisions([d],'cpu') for d in decisions])
        expected=model(*batch_decisions(decisions,'cpu'))
        actual=model(*packed)
        torch.testing.assert_close(actual[0].logits,expected[0].logits)
        torch.testing.assert_close(actual[1],expected[1])
        for d in decisions:
            reordered=copy.deepcopy(d)
            n=len(d['graph']['nodes']);remap={i:n-1-i for i in range(n)}
            reordered['graph']['nodes'].reverse()
            reordered['graph']['edges']=[[remap[i],remap[j]] for i,j in d['graph']['edges']]
            for a in reordered['candidates']:
                for key in ['target','source']:
                    if a[key]>=0:a[key]=remap[a[key]]
            a=model(*batch_decisions([d],'cpu'));b=model(*batch_decisions([reordered],'cpu'))
            torch.testing.assert_close(a[0].logits,b[0].logits,atol=1e-6,rtol=1e-5)
            torch.testing.assert_close(a[1],b[1],atol=1e-6,rtol=1e-5)
        # Graph parameters actually receive gradients (capacity is trainable).
        (-actual[0].entropy().mean()+actual[1].square().mean()).backward()
        self.assertTrue(all(p.grad is not None and torch.isfinite(p.grad).all() for p in model.parameters()))
        self.assertGreater(sum(float(p.grad.abs().sum()) for p in model.graph.parameters()),0)

    def test_export_go_parity(self):
        model=GraphPolicy(32)
        with tempfile.TemporaryDirectory() as path:
            export_policy(model,Path(path)/'full.json','full-test','Full test')
            with Engine(model_dir=path,rules='surface') as e:
                r=e.request(op='reset',seed=73,map='generated',players=2,curriculum=9,focus=True)
                for _ in range(8):
                    d=r['decision'];dist,_=model(*batch_decisions([d],'cpu'))
                    scores=e.request(op='scores',policy='full-test')['policy_scores']
                    torch.testing.assert_close(torch.log_softmax(torch.tensor(scores),0),dist.logits[0],atol=1e-5,rtol=1e-5)
                    r=e.request(op='step',policy='full-test')

if __name__=='__main__':unittest.main()
