import contextlib
import copy
import io
import json
from pathlib import Path
import tempfile
import unittest
import torch
from smallworld_ai.entity import EntityPolicy
from smallworld_ai.relational import RelationalPolicy,restore_optimizer
from smallworld_ai.model import batch_decisions,export_policy,load_checkpoint,policy_from_checkpoint
from smallworld_ai.league import policy_payload
from smallworld_ai.engine import Engine
from smallworld_ai.train import parser,run


class EntityTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):torch.set_num_threads(1)

    def decision(self,players=4,seed=1):
        with Engine(rules='surface') as e:
            return e.request(op='reset',players=players,map='generated',seed=seed)['decision']

    def test_preserves_policy_auxiliary_outputs_and_adam_moments(self):
        torch.manual_seed(81)
        old=RelationalPolicy(32);opt=torch.optim.Adam(old.parameters())
        batch=batch_decisions([self.decision()],'cpu');actions=torch.tensor([0])
        out=old.training_forward(batch,actions)
        loss=-out[0].log_prob(actions).mean()+out[1].square().mean()+out[3].square().mean()
        loss.backward();opt.step()
        new=EntityPolicy(32);new.warm_start(old)
        optimizer=torch.optim.Adam(new.parameters());restore_optimizer(optimizer,{'optimizer':opt.state_dict()},new,old)
        before=old.training_forward(batch,actions);after=new.training_forward(batch,actions)
        torch.testing.assert_close(before[0].probs,after[0].probs,atol=1e-7,rtol=1e-6)
        for a,b in zip(before[1:],after[1:]):torch.testing.assert_close(a,b,atol=0,rtol=0)
        for name,p in old.named_parameters():
            for key,value in opt.state[p].items():torch.testing.assert_close(value,optimizer.state[dict(new.named_parameters())[name]][key])
        with torch.no_grad():new.entity.actor[-1].weight.normal_(0,.01)
        out=new.training_forward(batch,actions);(-out[0].log_prob(actions).mean()+out[3].square().mean()).backward()
        self.assertGreater(float(new.entity.blocks[0].qkv.weight.grad.abs().sum()),0)
        self.assertTrue(all(p.grad is None or torch.isfinite(p.grad).all() for p in new.parameters()))

    def test_padding_permutation_dense_packed_and_checkpoint_roundtrip(self):
        torch.manual_seed(44)
        batch=batch_decisions([self.decision(2,22),self.decision(6,31)],'cpu')
        model=EntityPolicy(32)
        with torch.no_grad():model.entity.actor[-1].weight.normal_(0,.03)
        dense=copy.deepcopy(model);dense.pack_candidates=False
        actions=batch[2].sum(-1)-1
        for p in (model,dense):
            dist,value=p(*batch);(-dist.log_prob(actions).mean()+value.square().mean()).backward()
        torch.testing.assert_close(model(*batch)[0].probs,dense(*batch)[0].probs,atol=2e-6,rtol=2e-5)
        for a,b in zip(model.parameters(),dense.parameters()):
            if a.grad is not None:torch.testing.assert_close(a.grad,b.grad,atol=5e-6,rtol=5e-4)
        model.eval();padded=list(batch)
        padded[3]=torch.nn.functional.pad(batch[3],(0,0,0,3));padded[4]=torch.nn.functional.pad(batch[4],(0,3,0,3));padded[5]=torch.nn.functional.pad(batch[5],(0,3))
        perm=torch.randperm(batch[3].shape[1]);inverse=perm.argsort();other=list(batch)
        other[3]=batch[3][:,perm];other[4]=batch[4][:,perm][:,:,perm];other[5]=batch[5][:,perm]
        other[6]=torch.where(batch[6]>=0,inverse[batch[6].clamp_min(0)],-1)
        with torch.inference_mode():
            torch.testing.assert_close(model(*batch)[0].probs,model(*padded)[0].probs,atol=2e-6,rtol=2e-5)
            torch.testing.assert_close(model(*batch)[0].probs,model(*other)[0].probs,atol=2e-6,rtol=2e-5)
            restored=policy_from_checkpoint(policy_payload(model)).eval()
            torch.testing.assert_close(model(*batch)[0].probs,restored(*batch)[0].probs)

    def test_nonzero_go_export_matches_all_player_counts(self):
        torch.manual_seed(63);model=EntityPolicy(32).eval()
        with torch.no_grad():
            model.entity.actor[-1].weight.normal_(0,.025)
            for b in model.entity.blocks:b.edge_bias.normal_(0,.2)
        with tempfile.TemporaryDirectory() as directory:
            export_policy(model,Path(directory)/'entity.json','entity-test','Entity test')
            with Engine(rules='surface',model_dir=directory) as e:
                for players in range(2,7):
                    r=e.request(op='reset',players=players,map='generated',seed=91+players,focus=True,focus_player=0,curriculum=players*17)
                    for _ in range(4):
                        d=r['decision']
                        with torch.inference_mode():expected=model.action_distribution(*batch_decisions([d],'cpu')).logits[0]
                        scores=e.request(op='scores',policy='entity-test')['policy_scores']
                        torch.testing.assert_close(torch.tensor(scores).log_softmax(0),expected,atol=8e-5,rtol=8e-5)
                        r=e.request(op='step',policy='greedy')
                        if r['done']:break
            data=json.loads((Path(directory)/'entity.json').read_text());data['entity']['blocks'][0]['edge_bias'][0]=float('nan')
            (Path(directory)/'entity.json').write_text(json.dumps(data))
            with self.assertRaises(RuntimeError):
                with Engine(rules='surface',model_dir=directory) as e:e.request(op='spec')

    def test_training_expansion_and_resume_preserve_league_and_feedback(self):
        with tempfile.TemporaryDirectory() as d:
            args=['--rules','surface','--architecture','relational','--hidden','32','--device','cpu','--envs','2','--rollout-steps','16',
                  '--total-steps','32','--batch-size','8','--epochs','1','--players','4','--maps','map4players','--opponent','random',
                  '--async-rollouts','--objective','win','--rejection-penalty','0','--acceptance-loss','.05','--survival-loss','.05','--response-loss','.05',
                  '--rejection-loss','.05','--output',d]
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            _,old=load_checkpoint(Path(d)/'latest.pt')
            args+=['--resume',str(Path(d)/'latest.pt'),'--upgrade-entity','--cuda-graphs']
            args[args.index('--total-steps')+1]='64'
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            model,new=load_checkpoint(Path(d)/'latest.pt');self.assertEqual(model.format_version,6)
            self.assertEqual(new['step'],64);self.assertGreaterEqual(new['episodes_completed'],old['episodes_completed'])
            self.assertTrue(new['acceptance']['optimizer']['state']);self.assertTrue(new['opponent_response']['optimizer']['state'])
            self.assertTrue(any(p['format_version']==5 for p in new['opponent_pool']))
            args[args.index('--total-steps')+1]='96'
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            _,resumed=load_checkpoint(Path(d)/'latest.pt');self.assertEqual(resumed['step'],96)

if __name__=='__main__':unittest.main()
