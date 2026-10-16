import contextlib
import copy
import io
import json
from pathlib import Path
import tempfile
import unittest

import torch
from smallworld_ai.engine import Engine
from smallworld_ai.feedback import survival_labels, SELF_OWNER
from smallworld_ai.graph import S, A, N
from smallworld_ai.hierarchical import HierarchicalPolicy
from smallworld_ai.model import batch_decisions, export_policy, load_checkpoint
from smallworld_ai.relational import RelationalPolicy, restore_optimizer
from smallworld_ai.train import advantages, parser, run


class RelationalTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        torch.set_num_threads(1)

    def test_preserves_old_policy_values_and_optimizer_then_learns_new_branch(self):
        torch.manual_seed(19)
        old=HierarchicalPolicy(32,joint=True)
        opt=torch.optim.Adam(old.parameters())
        with Engine(rules='surface') as engine:
            d=engine.request(op='reset',players=4,map='generated',seed=905)['decision']
        batch=batch_decisions([d],'cpu')
        dist,value=old(*batch);(-dist.logits[0,0]+value.square().mean()).backward();opt.step()
        model=RelationalPolicy(32);model.warm_start(old)
        before,values=old(*batch);after,new_values=model(*batch)
        torch.testing.assert_close(before.probs,after.probs,rtol=1e-6,atol=1e-7)
        torch.testing.assert_close(values,new_values)
        new_opt=torch.optim.Adam(model.parameters())
        restore_optimizer(new_opt,{'optimizer':opt.state_dict()},model,old)
        for name,p in old.named_parameters():
            for key,v in opt.state[p].items():
                torch.testing.assert_close(v,new_opt.state[dict(model.named_parameters())[name]][key])
        dist,value,hidden,survival,_=model.training_forward(batch,torch.tensor([0]))
        (-dist.logits[0,0]+value.square().mean()+survival.square().mean()).backward()
        self.assertGreater(float(model.relation_actor[-1].weight.grad.abs().sum()),0)
        self.assertGreater(float(model.relation_node.weight.grad.abs().sum()),0)
        self.assertTrue(all(p.grad is None or torch.isfinite(p.grad).all() for p in model.parameters()))

    def test_go_export_matches_nonzero_attention_branch_across_game(self):
        torch.manual_seed(72)
        model=RelationalPolicy(32).eval()
        with torch.no_grad():
            model.relation_actor[-1].weight.normal_(0,.025)
        with tempfile.TemporaryDirectory() as directory:
            export_policy(model,Path(directory)/'attention.json','attention-test','Attention test')
            with Engine(rules='surface',model_dir=directory) as engine:
                r=engine.request(op='reset',players=4,map='generated',seed=77,focus=True,curriculum=14)
                for _ in range(32):
                    d=r['decision'];distribution,_=model(*batch_decisions([d],'cpu'))
                    scores=engine.request(op='scores',policy='attention-test')['policy_scores']
                    torch.testing.assert_close(torch.tensor(scores).log_softmax(0),distribution.logits[0],atol=5e-5,rtol=5e-5)
                    r=engine.request(op='step',policy='greedy')
                    if r['done']:break

    def test_padding_permutation_and_packed_gradients_do_not_change_policy(self):
        torch.manual_seed(20)
        batch=(torch.randn(2,S),torch.randn(2,5,A),torch.tensor([[1,1,0,0,0],[1,1,1,1,1]],dtype=torch.bool),
               torch.randn(2,6,N),torch.eye(6).repeat(2,1,1),torch.ones(2,6),torch.randint(-1,6,(2,5,2)))
        model=RelationalPolicy(32)
        with torch.no_grad():model.relation_actor[-1].weight.normal_(0,.03)
        dense=copy.deepcopy(model);dense.pack_candidates=False
        results=[]
        for p in (model,dense):
            d,v,hidden,surv,_=p.training_forward(batch,torch.tensor([1,3]))
            loss=-d.log_prob(torch.tensor([1,3])).mean()+v.square().mean()+surv.square().mean()
            loss.backward();results.append((d.probs,v,surv))
        for a,b in zip(*results):torch.testing.assert_close(a,b,atol=2e-6,rtol=2e-5)
        for a,b in zip(model.parameters(),dense.parameters()):
            if a.grad is not None:torch.testing.assert_close(a.grad,b.grad,atol=4e-6,rtol=2e-4)
        model.eval()
        padded=list(batch);padded[3]=torch.cat((batch[3],torch.randn(2,3,N)),1)
        padded[4]=torch.nn.functional.pad(batch[4],(0,3,0,3));padded[5]=torch.nn.functional.pad(batch[5],(0,3))
        torch.testing.assert_close(model(*batch)[0].probs,model(*padded)[0].probs,atol=1e-6,rtol=1e-5)
        perm=torch.tensor([3,0,5,2,1,4]);inverse=perm.argsort()
        other=list(batch);other[3]=batch[3][:,perm];other[4]=batch[4][:,perm][:,:,perm]
        other[6]=torch.where(batch[6]>=0,inverse[batch[6].clamp_min(0)],-1)
        torch.testing.assert_close(model(*batch)[0].probs,model(*other)[0].probs,atol=1e-6,rtol=1e-5)

    def test_turn_credit_is_independent_of_number_of_redeployment_clicks(self):
        for clicks in (1,30,200):
            n=clicks+1;rewards=torch.zeros(n,1);rewards[-1]=1
            dones=torch.zeros(n,1);dones[-1]=1
            turns=torch.zeros(n,1);turns[clicks-1]=1;turns[-1]=1
            adv,_=advantages(rewards,dones,torch.zeros(n,1),torch.zeros(1),1,.95,turns)
            self.assertAlmostEqual(float(adv[0]),.95,places=6)
        before={'schema':'smallworld-bot-v2','player':0,'graph':{'nodes':[[0.]*N for _ in range(3)]}}
        before['graph']['nodes'][0][SELF_OWNER]=before['graph']['nodes'][1][SELF_OWNER]=1
        after=copy.deepcopy(before);after['graph']['nodes'][1][SELF_OWNER]=0
        self.assertEqual(survival_labels(before,after),[1.,0.,-1.])
        after['player']=1
        with self.assertRaises(ValueError):survival_labels(before,after)

    def test_real_training_upgrade_and_resume_preserves_history_and_has_defense_targets(self):
        with tempfile.TemporaryDirectory() as directory:
            base=['--rules','surface','--hidden','32','--device','cpu','--envs','2','--rollout-steps','32',
                  '--total-steps','64','--batch-size','16','--epochs','1','--opponent','random','--players','4',
                  '--maps','map4players','--async-rollouts','--acceptance-loss','.05','--output',directory]
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(base+['--architecture','joint']))
            _,old=load_checkpoint(Path(directory)/'latest.pt')
            upgraded=base+['--resume',str(Path(directory)/'latest.pt'),'--upgrade-relational','--survival-loss','.05','--gae-clock','turn']
            upgraded[upgraded.index('--total-steps')+1]='192'
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(upgraded))
            model,new=load_checkpoint(Path(directory)/'latest.pt')
            self.assertEqual(model.format_version,5);self.assertEqual(new['step'],192)
            self.assertGreater(sum(new['action_counts']['chosen'].values()),sum(old['action_counts']['chosen'].values()))
            self.assertTrue(any(x['format_version']==4 for x in new['opponent_pool']))
            logs=[json.loads(line) for line in (Path(directory)/'metrics.jsonl').read_text().splitlines()]
            self.assertGreater(sum(x['survival_tiles'] for x in logs),0)
            upgraded[upgraded.index('--total-steps')+1]='256'
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(upgraded))
            model,last=load_checkpoint(Path(directory)/'latest.pt')
            self.assertEqual(last['step'],256);self.assertEqual(last['config']['architecture'],'relational')
            self.assertTrue(all(torch.isfinite(p).all() for p in model.parameters()))

if __name__=='__main__':unittest.main()
