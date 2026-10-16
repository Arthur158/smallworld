import copy
import json
import os
from pathlib import Path
import sys
import tempfile
import unittest
from unittest.mock import patch
import torch

from smallworld_ai.cpu_evaluator import load_export, scores
from smallworld_ai.model import Policy, export_policy
from smallworld_ai.graph import GraphPolicy
from smallworld_ai.hierarchical import HierarchicalPolicy
from smallworld_ai.relational import RelationalPolicy
from smallworld_ai.entity import EntityPolicy
from smallworld_ai.engine import Engine


class CPUEvaluatorTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        torch.set_num_threads(1)

    def test_all_export_formats_and_persistent_go_bridge(self):
        torch.manual_seed(880)
        factories=[lambda:Policy(32),lambda:GraphPolicy(32),lambda:HierarchicalPolicy(32),lambda:HierarchicalPolicy(32,joint=True),lambda:RelationalPolicy(32),lambda:EntityPolicy(32)]
        for version,factory in enumerate(factories,1):
            with self.subTest(version=version),tempfile.TemporaryDirectory() as temp:
                model=factory().eval()
                with torch.no_grad():
                    if version>=3:model.group_actor[-1].weight.normal_(0,.02)
                    if version>=5:model.relation_actor[-1].weight.normal_(0,.02)
                    if version==6:
                        model.entity.actor[-1].weight.normal_(0,.02)
                        for block in model.entity.blocks:block.edge_bias.normal_(0,.2)
                file=Path(temp)/'model.json';export_policy(model,file,'model','Model')
                identifier,loaded,digest=load_export(file)
                self.assertEqual(identifier,'model');self.assertEqual(len(digest),64)
                rule='legacy' if version==1 else 'surface'
                # The Go process owns and reuses the persistent Python child.
                with Engine(model_dir=temp,rules=rule) as go:
                    with patch.dict(os.environ,{'BOT_INFERENCE_PYTHON':sys.executable,'PYTHONPATH':str(Path(__file__).resolve().parents[1])}):
                        fast=Engine(model_dir=temp,rules=rule)
                    with fast:
                        for players in (2,4,6):
                            reset=dict(op='reset',seed=880+players,players=players,map=f'map{players}players'+('2islands' if players==6 else ''))
                            one=go.request(**reset);two=fast.request(**reset)
                            for _ in range(3):
                                decision=one['decision']
                                expected=scores(model,decision)
                                actual=scores(loaded,decision)
                                torch.testing.assert_close(torch.tensor(actual),torch.tensor(expected),rtol=1e-6,atol=1e-6)
                                slow_logits=torch.tensor(go.request(op='scores',policy='model')['policy_scores'],dtype=torch.float64).log_softmax(0)
                                fast_logits=torch.tensor(fast.request(op='scores',policy='model')['policy_scores'],dtype=torch.float64).log_softmax(0)
                                torch.testing.assert_close(fast_logits,slow_logits,rtol=1e-4,atol=1e-4)
                                one=go.request(op='step',policy='model');two=fast.request(op='step',policy='model')
                                self.assertEqual(one.get('action'),two.get('action'))
                                self.assertEqual(one['decision'],two['decision'])
                                if one['done']:break

    def test_invalid_export_does_not_silently_use_random_weights(self):
        with tempfile.TemporaryDirectory() as temp:
            file=Path(temp)/'model.json';export_policy(Policy(32),file,'model','Model')
            original=json.loads(file.read_text())
            for change in ('nonfinite','missing','width','schema'):
                data=copy.deepcopy(original)
                if change=='nonfinite':data['layers'][0]['weights'][0][0]=float('nan')
                elif change=='missing':data['layers'].pop()
                elif change=='width':data['layers'][1]['weights'][0].pop()
                else:data['schema']='unknown'
                file.write_text(json.dumps(data))
                with self.assertRaises(ValueError):load_export(file)


if __name__=='__main__':unittest.main()
