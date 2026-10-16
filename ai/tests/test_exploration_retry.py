import unittest
from concurrent.futures import ThreadPoolExecutor
from unittest.mock import patch
import torch
from torch.distributions import Categorical
from smallworld_ai import ACTION_FEATURES
from smallworld_ai.graph import A,S,N
from smallworld_ai.hierarchical import HierarchicalPolicy,KINDS
from smallworld_ai.exploration import training_distribution,sample_training_actions
from smallworld_ai.rollout import collect_async

class RetryTests(unittest.TestCase):
    def setUp(self):
        torch.set_num_threads(1)
        self.model=HierarchicalPolicy(32,joint=True)
        features=torch.zeros(1,4,A)
        features[:,:,ACTION_FEATURES+1]=1
        features[:,:2,ACTION_FEATURES+KINDS]=1
        features[:,2:,ACTION_FEATURES+KINDS+1]=1
        self.batch=(torch.zeros(1,S),features,torch.ones(1,4,dtype=torch.bool))
        self.base=Categorical(logits=torch.tensor([[20.,19.,-5.,-6.]]))
        self.group=int(self.model.group_ids(self.model.group_features(features))[0,2])

    def test_retry_stays_with_piece_and_uses_another_remaining_destination(self):
        context=torch.tensor([self.group])
        for _ in range(20):
            action,logprob,retry,mode=sample_training_actions(self.model,self.base,self.batch,.05,1.5,context)
            self.assertIn(action.item(),[2,3])
            self.assertEqual(retry.item(),self.group)
            self.assertEqual(mode.item(),self.group)
            update=training_distribution(self.model,self.base,self.batch,.05,1.5,mode)
            torch.testing.assert_close(logprob,update.log_prob(action))
        mask=self.batch[2].clone();mask[0,2]=False
        batch=(*self.batch[:2],mask)
        base=Categorical(logits=torch.tensor([[20.,19.,-1e9,-6.]]))
        action,_,retry,_=sample_training_actions(self.model,base,batch,.05,1.5,context)
        self.assertEqual(action.item(),3)
        self.assertEqual(retry.item(),self.group)

    def test_exhausted_group_releases_lock_without_invalid_probabilities(self):
        mask=torch.tensor([[True,True,False,False]])
        batch=(*self.batch[:2],mask)
        base=Categorical(logits=torch.tensor([[20.,19.,-1e9,-1e9]]))
        for _ in range(20):
            action,logprob,retry,mode=sample_training_actions(self.model,base,batch,.05,1.5,torch.tensor([self.group]))
            self.assertIn(action.item(),[0,1])
            self.assertIn(mode.item(),[-1,-2])
            self.assertTrue(torch.isfinite(logprob).all())
            self.assertNotEqual(retry.item(),self.group)

    def test_exploration_coin_and_ppo_condition_on_same_mode(self):
        for fraction in (0.,.05,1.):
            for _ in range(20):
                action,old,_,mode=sample_training_actions(self.model,self.base,self.batch,fraction)
                new=training_distribution(self.model,self.base,self.batch,fraction,retry_groups=mode).log_prob(action)
                torch.testing.assert_close(old,new)
                if fraction==0:self.assertEqual(mode.item(),-1)
                if fraction==1:self.assertEqual(mode.item(),-2)

    def test_async_retry_state_is_cleared_only_after_success(self):
        class Env:
            exploration_retry_group=-1
            count=0
            def observation(self):
                return dict(schema='smallworld-bot-v2',state=[0.]*S,graph=dict(nodes=[[0.]*N],edges=[]),candidates=[dict(features=[0.]*A,target=-1,source=-1)])
            def step(self):
                self.count+=1
                return self.observation(),0.,False,False,dict(rejected=self.count==1)
        env=Env(); contexts=[]
        def sample(policy,dist,batch,fraction,temp,retry_groups,*options):
            contexts.append(int(retry_groups[0]))
            return torch.tensor([0]),torch.tensor([0.]),torch.tensor([55]),torch.tensor([-2 if retry_groups[0]<0 else 55])
        with patch('smallworld_ai.rollout.sample_training_actions',sample),ThreadPoolExecutor(max_workers=1) as pool:
            collect_async(self.model,[env],[env.observation()],pool,3,'cpu',lambda pair:pair[0].step(),lambda *args:None,.05)
        self.assertEqual(contexts,[-1,55,-1])
        self.assertEqual(env.exploration_retry_group,-1)

if __name__=='__main__':unittest.main()
