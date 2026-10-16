import contextlib
import copy
import io
from pathlib import Path
import tempfile
import unittest

import torch

from smallworld_ai.engine import Engine
from smallworld_ai.graph import GraphPolicy
from smallworld_ai.hierarchical import HierarchicalPolicy
from smallworld_ai.model import batch_decisions, export_policy, load_checkpoint
from smallworld_ai.train import advantages, parser, run


class HierarchicalTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        torch.set_num_threads(1)

    def decision(self):
        with Engine(rules='surface', max_steps=8192) as engine:
            return engine.request(op='reset', players=2, map='generated', seed=701, focus=True, curriculum=11)['decision']

    def test_piece_group_mass_is_not_determined_by_target_count(self):
        model = HierarchicalPolicy(32)
        d = self.decision()
        packed = batch_decisions([d], 'cpu')
        groups = model.group_ids(model.group_features(packed[1]))[0]
        with torch.no_grad():
            dist, _ = model(*packed)
        for group in groups.unique():
            self.assertAlmostEqual(float(dist.probs[0, groups==group].sum()), 1/len(groups.unique()), places=5)
        # Adding a duplicate target changes that group's target distribution,
        # but cannot buy extra probability for the piece/action type itself.
        doubled=copy.deepcopy(d);doubled['candidates'] += [copy.deepcopy(d['candidates'][0])]*12
        batch=batch_decisions([doubled], 'cpu')
        other_groups=model.group_ids(model.group_features(batch[1]))[0]
        with torch.no_grad():
            other,_=model(*batch)
        for group in groups.unique():
            torch.testing.assert_close(dist.probs[0,groups==group].sum(),other.probs[0,other_groups==group].sum())

    def test_export_go_parity_and_live_gradients(self):
        old=GraphPolicy(32);model=HierarchicalPolicy(48);model.warm_start(old)
        with torch.no_grad():
            model.group_actor[2].weight.uniform_(-0.02,0.02)
        d=self.decision();batch=batch_decisions([d],'cpu')
        dist,value=model(*batch)
        (-dist.log_prob(dist.sample()).mean()+value.square().mean()).backward()
        self.assertTrue(all(p.grad is not None and torch.isfinite(p.grad).all() for p in model.parameters()))
        self.assertGreater(float(model.group_actor[2].weight.grad.abs().sum()),0)
        with tempfile.TemporaryDirectory() as directory:
            export_policy(model,Path(directory)/'test.json','hierarchical-test','Hierarchical test')
            with Engine(model_dir=directory,rules='surface',max_steps=8192) as engine:
                r=engine.request(op='reset',players=2,map='generated',seed=701,focus=True,curriculum=11)
                for _ in range(12):
                    d=r['decision'];logits=model(*batch_decisions([d],'cpu'))[0].logits[0]
                    scores=engine.request(op='scores',policy='hierarchical-test')['policy_scores']
                    torch.testing.assert_close(torch.log_softmax(torch.tensor(scores,dtype=torch.float32),0),logits,atol=3e-5,rtol=3e-5)
                    r=engine.request(op='step',policy='hierarchical-test')

    def test_train_resume_retains_architecture_and_opponent_history(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            args=['--rules','surface','--architecture','hierarchical','--hidden','32','--device','cpu','--envs','2','--rollout-steps','8','--total-steps','32','--batch-size','16','--epochs','1','--snapshot-every','1','--async-rollouts','--output',str(root)]
            with contextlib.redirect_stdout(io.StringIO()):
                run(parser().parse_args(args))
            model,checkpoint=load_checkpoint(root/'latest.pt')
            self.assertIsInstance(model,HierarchicalPolicy)
            self.assertEqual(len(checkpoint['opponent_pool']),3)
            self.assertGreater(sum(checkpoint['action_counts']['chosen'].values()),0)
            args[args.index('--total-steps')+1]='48'
            args += ['--resume',str(root/'latest.pt')]
            with contextlib.redirect_stdout(io.StringIO()):
                run(parser().parse_args(args))
            _,after=load_checkpoint(root/'latest.pt')
            self.assertEqual(after['step'],48)
            self.assertEqual(len(after['opponent_pool']),4)
            self.assertGreater(sum(after['action_counts']['chosen'].values()),sum(checkpoint['action_counts']['chosen'].values()))

    def test_new_head_learns_from_teacher_experience(self):
        with tempfile.TemporaryDirectory() as directory:
            root=Path(directory)
            teacher=GraphPolicy(32)
            torch.save(dict(format_version=2,schema=teacher.schema,config={'hidden':32},model=teacher.state_dict()),root/'teacher.pt')
            args=['--rules','surface','--architecture','hierarchical','--hidden','32','--device','cpu','--envs','2','--rollout-steps','8','--total-steps','16','--batch-size','16','--epochs','1','--bootstrap-steps','32','--bootstrap-epochs','2','--warm-start',str(root/'teacher.pt'),'--output',str(root/'run')]
            with contextlib.redirect_stdout(io.StringIO()):
                run(parser().parse_args(args))
            model,checkpoint=load_checkpoint(root/'run/latest.pt')
            self.assertTrue((root/'run/calibration.json').is_file())
            self.assertTrue(torch.isfinite(model.group_actor[2].weight).all())
            self.assertGreater(float(model.group_actor[2].weight.abs().sum()),0)
            self.assertEqual(checkpoint['step'],16)
            self.assertEqual(checkpoint['opponent_pool'][0]['format_version'],2)

    def test_joint_transfer_preserves_teacher_and_exports_the_same_actions(self):
        old=GraphPolicy(32).eval();model=HierarchicalPolicy(48,joint=True);model.warm_start(old)
        with tempfile.TemporaryDirectory() as directory:
            export_policy(model,Path(directory)/'joint.json','joint-test','Joint test')
            with Engine(model_dir=directory,rules='surface') as engine:
                r=engine.request(op='reset',players=2,map='map2players',seed=532)
                for _ in range(30):
                    d=r['decision'];batch=batch_decisions([d],'cpu')
                    teacher,value=old(*batch);student,new_value=model(*batch)
                    torch.testing.assert_close(teacher.logits,student.logits,atol=1e-5,rtol=1e-5)
                    torch.testing.assert_close(value,new_value,atol=1e-5,rtol=1e-5)
                    scores=engine.request(op='scores',policy='joint-test')['policy_scores']
                    torch.testing.assert_close(torch.log_softmax(torch.tensor(scores,dtype=torch.float32),0),student.logits[0],atol=3e-5,rtol=3e-5)
                    self.assertEqual(model.choose(d),int(student.logits[0].argmax()))
                    if r['done']:break
                    r=engine.request(op='step',policy='greedy')
            loss=-model.group_entropy(student,batch[1]).mean()
            loss.backward()
            self.assertTrue(torch.isfinite(model.group_actor[2].weight.grad).all())

    def test_undiscounted_returns_do_not_reward_finishing_with_fewer_clicks(self):
        # Lambda 1 exposes the return target directly. More legal preparation
        # actions must not reduce the same final game reward.
        for n in (2,50):
            rewards=torch.zeros(n,1);rewards[-1]=1
            dones=torch.zeros(n,1);dones[-1]=1
            _,returns=advantages(rewards,dones,torch.zeros(n,1),torch.zeros(1),1,1)
            torch.testing.assert_close(returns,torch.ones(n,1))

if __name__=='__main__':
    unittest.main()
