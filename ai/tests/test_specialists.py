import copy
import json
from pathlib import Path
import tempfile
import unittest
from concurrent.futures import ThreadPoolExecutor
import torch
from smallworld_ai.engine import Engine
from smallworld_ai.entity import EntityPolicy
from smallworld_ai.specialists import SpecialistPolicy, RACES, TRAITS
from smallworld_ai.model import batch_decisions, merge_decision_batches, policy_from_checkpoint, export_policy
from smallworld_ai.league import policy_payload
from smallworld_ai.cpu_evaluator import load_export
from smallworld_ai.curriculum import SpecialistCurriculum
from smallworld_ai.inference import BatchedInference
from smallworld_ai.entity import PLAYER_WIDTH, R, T
from smallworld_ai import STATE_FEATURES


class SpecialistTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls): torch.set_num_threads(1)

    def decision(self, players=4, focus=True, seat=0, seed=301, race=None, trait=None):
        with Engine(rules='surface') as e:
            r=e.request(op='reset',players=players,map='generated',seed=seed,focus=focus,focus_player=seat,
                        **(dict(focus_race=race or RACES[0], focus_trait=trait or TRAITS[0]) if focus else {}))
            while r['decision']['player'] != seat:
                r=e.request(op='step',policy='greedy')
            return r['decision']

    def test_all_races_and_traits_are_assigned_across_every_seat(self):
        for i in range(max(len(RACES),len(TRAITS))):
            d=self.decision(players=6,seat=i%6,race=RACES[i%len(RACES)],trait=TRAITS[i],seed=600+i)
            races={c['expert_race'] for c in d['candidates']}
            self.assertEqual(races,{RACES[i%len(RACES)]})
            self.assertTrue(all(TRAITS[i] in c['expert_traits'] for c in d['candidates']))

    def test_warm_start_preserves_actor_value_and_freezes_backbone(self):
        torch.manual_seed(33)
        old=EntityPolicy(32);new=SpecialistPolicy(32);new.warm_start(old)
        for d in [self.decision(),self.decision(focus=False)]:
            before=old(*batch_decisions([d],'cpu'))
            after=new(*batch_decisions([d],'cpu',new))
            torch.testing.assert_close(before[0].probs,after[0].probs,atol=1e-7,rtol=1e-6)
            torch.testing.assert_close(before[1],after[1])
        self.assertTrue(all(p.requires_grad==name.startswith('specialist_') for name,p in new.named_parameters()))

    def test_only_participating_experts_update_and_market_does_not_train_experts(self):
        p=SpecialistPolicy(32)
        with torch.no_grad():
            p.specialist_coordinator[-1].weight.normal_(0,.1)
            p.specialist_selector[-1].weight.normal_(0,.1)
        batch=batch_decisions([self.decision()],'cpu',p)
        opt=torch.optim.Adam(p.parameters())
        saved={name:x.detach().clone() for name,x in p.named_parameters()}
        out=p(*batch);(-out[0].log_prob(torch.tensor([0])).mean()+out[1].square().mean()).backward();opt.step()
        self.assertGreater(sum(float((x-saved[name]).detach().abs().sum()) for name,x in p.named_parameters() if name.startswith('specialist_races.0.')),0)
        self.assertGreater(sum(float((x-saved[name]).detach().abs().sum()) for name,x in p.named_parameters() if name.startswith('specialist_general.')),0)
        for name,x in p.named_parameters():
            if not name.startswith('specialist_') or name.startswith('specialist_races.1.'):
                torch.testing.assert_close(x,saved[name],atol=0,rtol=0)
        opt.zero_grad(set_to_none=True)
        p(*batch_decisions([self.decision(focus=False)],'cpu',p))[0].log_prob(torch.tensor([0])).mean().backward()
        self.assertTrue(all(x.grad is None for module in (*p.specialist_races,*p.specialist_traits) for x in module.parameters()))

    def test_dense_packed_order_padding_routing_and_checkpoint(self):
        p=SpecialistPolicy(32)
        with torch.no_grad():
            for h in (p.specialist_coordinator,p.specialist_defense,p.specialist_selector):h[-1].weight.normal_(0,.1)
        d=[self.decision(2),self.decision(6,seed=302,trait=TRAITS[3])]
        batch=batch_decisions(d,'cpu',p)
        packed=p(*batch);p.pack_candidates=False;dense=p(*batch)
        torch.testing.assert_close(packed[0].probs,dense[0].probs,atol=2e-6,rtol=2e-5)
        merged=merge_decision_batches([batch_decisions([v],'cpu',p) for v in d])
        torch.testing.assert_close(p(*merged)[0].probs,dense[0].probs)
        perm=torch.randperm(batch[3].shape[1]);inverse=perm.argsort();other=list(batch)
        other[3]=batch[3][:,perm];other[4]=batch[4][:,perm][:,:,perm];other[5]=batch[5][:,perm]
        other[6]=torch.where(batch[6]>=0,inverse[batch[6].clamp_min(0)],-1)
        torch.testing.assert_close(p(*other)[0].probs,dense[0].probs,atol=2e-6,rtol=2e-5)
        restored=policy_from_checkpoint(policy_payload(p))
        torch.testing.assert_close(restored(*batch)[0].probs,dense[0].probs)
        with tempfile.TemporaryDirectory() as tmp:
            path=Path(tmp)/'model.json';export_policy(p,path,'specialist-test','Specialist test')
            _,exported,_=load_export(path)
            torch.testing.assert_close(exported(*batch)[0].probs,dense[0].probs)

    def test_redeployment_head_and_shared_threat_learn(self):
        p=SpecialistPolicy(32)
        with Engine(rules='surface') as e:
            r=e.request(op='reset',players=2,map='generated',seed=51,focus=True,focus_player=0,focus_race=RACES[0],focus_trait=TRAITS[0])
            for _ in range(120):
                d=r['decision']; kinds=[c['action']['kind'] for c in d['candidates']]
                if 'redeploy' in kinds: break
                r=e.request(op='step',policy='greedy')
            else:self.fail('no redeployment opportunity')
        batch=batch_decisions([d],'cpu',p);action=torch.tensor([kinds.index('redeploy')])
        out=p.training_forward(batch,action)
        loss=-out[0].log_prob(action).mean()+torch.nn.functional.binary_cross_entropy_with_logits(out[3],torch.zeros_like(out[3]))
        loss.backward()
        self.assertGreater(float(p.specialist_defense[-1].weight.grad.abs().sum()),0)
        self.assertGreater(float(p.specialist_survival[-1].weight.grad.abs().sum()),0)
        self.assertTrue(all(x.grad is None or torch.isfinite(x.grad).all() for x in p.parameters()))

    def test_threat_sees_disconnected_tiles_every_player_and_visible_market(self):
        torch.manual_seed(119)
        p=SpecialistPolicy(32)
        with torch.no_grad():p.specialist_survival[-1].weight.normal_(0,.1)
        batch=list(batch_decisions([self.decision(6)],'cpu',p))
        batch[0]=batch[0].detach().requires_grad_(True)
        batch[3]=batch[3].detach().requires_grad_(True)
        batch[4]=torch.zeros_like(batch[4])  # no tile is adjacent to any other
        threat=p.training_forward(tuple(batch),torch.tensor([0]))[3][0,0]
        threat.backward()
        self.assertGreater(float(batch[3].grad[0,1:].abs().sum()),0)
        for seat in range(6):
            start=STATE_FEATURES+seat*PLAYER_WIDTH
            self.assertGreater(float(batch[0].grad[0,start:start+PLAYER_WIDTH].abs().sum()),0)
        market_start=STATE_FEATURES+6*PLAYER_WIDTH
        self.assertGreater(float(batch[0].grad[0,market_start:market_start+5*(2+R+T)].abs().sum()),0)

    def test_curriculum_balances_resumes_and_keeps_natural_selection(self):
        c=SpecialistCurriculum(94,[2,3,4,5,6],.75)
        rows=[c.next() for _ in range(800)]
        self.assertEqual(sum(r['focus'] for r in rows),600)
        saved=c.state_dict()
        for key in ('race','trait','players'):
            values=saved['requests'][key].values();self.assertLessEqual(max(values)-min(values),1)
        restored=SpecialistCurriculum(0,[2,3,4,5,6],.75,saved)
        self.assertEqual([c.next() for _ in range(100)],[restored.next() for _ in range(100)])

    def test_specialists_work_as_batched_frozen_opponents(self):
        p=SpecialistPolicy(32).eval()
        decisions=[self.decision(players=players,seed=77+players) for players in (2,3,4,5,6)]
        expected=[p.choose(d) for d in decisions]
        inference=BatchedInference('cpu',wait_seconds=.02)
        try:
            with ThreadPoolExecutor(max_workers=5) as pool:
                actual=list(pool.map(lambda d:inference.choose(p,d),decisions))
            self.assertEqual(actual,expected)
        finally:inference.close()

if __name__=='__main__':unittest.main()
