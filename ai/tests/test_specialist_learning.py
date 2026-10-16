import copy
from pathlib import Path
from types import SimpleNamespace
import tempfile
import unittest
import torch
from smallworld_ai.specialists import SpecialistPolicy, RACES, TRAITS
from smallworld_ai.specialist_learning import ExpertCredit, optimizer_for, contribution_metrics
from smallworld_ai.model import batch_decisions, export_policy
from smallworld_ai.cpu_evaluator import load_export
from smallworld_ai.engine import Engine
from smallworld_ai.train import parser, run


class SpecialistLearningTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):
        torch.set_num_threads(1)
        with Engine(rules='surface') as engine:
            cls.decision=engine.request(op='reset',players=3,map='generated',seed=4491,focus=True,
                focus_player=0,focus_race=RACES[0],focus_trait=TRAITS[0])['decision']
            cls.market=engine.request(op='reset',players=3,map='generated',seed=4492,focus=False)['decision']

    def options(self):
        return SimpleNamespace(learning_rate=1e-4,specialist_learning_rate=5e-4,specialist_adam_eps=1e-8)

    def test_direct_credit_reaches_experts_even_with_zero_coordinator(self):
        torch.manual_seed(12)
        model=SpecialistPolicy(32)
        batch=batch_decisions([self.decision],'cpu',model)
        action=torch.tensor([0]);head=ExpertCredit();optimizer=optimizer_for(model,self.options())
        before={name:value.clone() for name,value in model.state_dict().items()}
        output=model.training_forward(batch,action,expert_credit=True)
        head(model,output[5],batch,action,torch.tensor([1.])).backward()
        self.assertGreater(model.specialist_races[0][0].weight.grad.norm().item(),0)
        self.assertGreater(model.specialist_traits[0][0].weight.grad.norm().item(),0)
        self.assertTrue(all(p.grad is None for p in model.specialist_races[1].parameters()))
        self.assertTrue(all(p.grad is None for p in head.races[1].parameters()))
        optimizer.step()
        for name,value in model.state_dict().items():
            if not name.startswith(('specialist_races.0.','specialist_traits.0.')):
                torch.testing.assert_close(value,before[name],rtol=0,atol=0)

    def test_credit_does_not_train_unplayed_market_experts(self):
        model=SpecialistPolicy(32);head=ExpertCredit()
        batch=batch_decisions([self.market],'cpu',model);action=torch.tensor([0])
        output=model.training_forward(batch,action,expert_credit=True)
        loss=head(model,output[5],batch,action,torch.ones(1))
        self.assertEqual(float(loss),0)
        self.assertTrue(all(p.grad is None for p in head.parameters()))

    def test_named_optimizer_preserves_legacy_and_grouped_moments(self):
        model=SpecialistPolicy(32);legacy=torch.optim.Adam(model.parameters(),lr=3e-5,eps=1e-5)
        parameter=model.specialist_races[0][0].weight
        parameter.grad=torch.ones_like(parameter);legacy.step();legacy.zero_grad(set_to_none=True)
        checkpoint={'optimizer':legacy.state_dict()}
        optimizer=optimizer_for(model,self.options(),checkpoint)
        torch.testing.assert_close(optimizer.state[parameter]['exp_avg'],legacy.state[parameter]['exp_avg'])
        self.assertEqual(optimizer.param_groups[0]['lr'],1e-4)
        self.assertEqual(next(g for g in optimizer.param_groups if g['name']=='experts')['eps'],1e-8)
        resumed=optimizer_for(model,self.options(),{'optimizer':optimizer.state_dict()})
        torch.testing.assert_close(resumed.state[parameter]['exp_avg'],optimizer.state[parameter]['exp_avg'])
        self.assertEqual(float(resumed.state[parameter]['step']),1.)

    def test_ablations_are_read_only_and_export_is_unchanged(self):
        torch.manual_seed(13);model=SpecialistPolicy(32)
        with torch.no_grad():
            model.specialist_coordinator[-1].weight.normal_(0,.3)
        batch=batch_decisions([self.decision],'cpu',model)
        before=copy.deepcopy(model.state_dict());expected=model.action_distribution(*batch).probs.detach()
        metrics=contribution_metrics(model,batch,1)
        self.assertTrue(model.training)
        self.assertGreater(metrics['specialist_race_kl'],0)
        self.assertGreater(metrics['specialist_trait_kl'],0)
        for name,value in model.state_dict().items():torch.testing.assert_close(value,before[name],rtol=0,atol=0)
        with tempfile.TemporaryDirectory() as directory:
            path=Path(directory)/'model.json';export_policy(model,path,'test-learning','test')
            _,exported,_=load_export(path)
            torch.testing.assert_close(exported.action_distribution(*batch).probs,expected,atol=2e-6,rtol=2e-5)

    def test_real_training_resume_keeps_backbone_fixed_and_saves_credit(self):
        with tempfile.TemporaryDirectory() as directory:
            folder=Path(directory)
            flags=['--rules','surface','--architecture','specialists','--hidden','32','--device','cpu',
                   '--freeze-backbone','--specialist-learning-rate','.0005','--specialist-adam-eps','1e-8',
                   '--specialist-credit-loss','.1','--specialist-probe-size','8','--envs','2','--rollout-steps','16',
                   '--batch-size','16','--epochs','1','--opponent','greedy','--player-counts','2','3',
                   '--specialist-curriculum','--output',str(folder),'--total-steps','32']
            run(parser().parse_args(flags));first=torch.load(folder/'latest.pt',weights_only=True)
            run(parser().parse_args(flags+['--resume',str(folder/'latest.pt'),'--total-steps','64','--async-rollouts']))
            second=torch.load(folder/'latest.pt',weights_only=True)
            self.assertEqual(second['step'],64)
            self.assertEqual(second['format_version'],7)
            self.assertTrue(second['config']['freeze_backbone'])
            self.assertTrue(second['specialist_credit']['optimizer']['state'])
            self.assertTrue(all('param_names' in group for group in second['optimizer']['param_groups']))
            for name,value in first['model'].items():
                if not name.startswith('specialist_'):
                    torch.testing.assert_close(value,second['model'][name],rtol=0,atol=0)

if __name__=='__main__':unittest.main()
