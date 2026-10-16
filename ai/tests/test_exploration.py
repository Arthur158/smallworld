import contextlib
import io
from pathlib import Path
import tempfile
import unittest
from smallworld_ai.exploration import PieceExploration
from smallworld_ai.model import load_checkpoint
from smallworld_ai.train import parser, run

class ExplorationTests(unittest.TestCase):
    def test_all_piece_names_equal_and_repeat_or_rejected_moves_earn_nothing(self):
        tracker=PieceExploration(.02,.04)
        seen=set()
        self.assertEqual(tracker.reward(dict(kind='conquer',stack='Dragon'),True,seen,0),0)
        self.assertEqual(tracker.reward(dict(kind='finish'),False,seen,0),0)
        first=tracker.reward(dict(kind='conquer',stack='arbitrary-unseen-piece'),False,seen,0)
        self.assertGreater(first,0)
        for kind in ('abandon','redeploy','redeploy_out','conquer'):
            self.assertEqual(tracker.reward(dict(kind=kind,stack='arbitrary-unseen-piece'),False,seen,first),0)
        for name in ('Dragon','Humans','never-seen-before'):
            fresh=PieceExploration(.02,.04)
            self.assertEqual(fresh.reward(dict(kind='conquer',stack=name),False,set(),0),.02)

    def test_budget_and_familiarity_remove_the_piece_use_incentive(self):
        tracker=PieceExploration(.02,.025,successful={'conquer:normal':100000})
        rare=tracker.reward(dict(stack='rare'),False,set(),0)
        common=tracker.reward(dict(stack='normal'),False,set(),0)
        self.assertGreater(rare,common*20)
        seen=set();earned=0
        for i in range(10):earned+=tracker.reward(dict(stack=str(i)),False,seen,earned)
        self.assertAlmostEqual(earned,.025)
        familiar=PieceExploration(.02,.04,successful={'conquer:known':10**12})
        self.assertLess(familiar.reward(dict(stack='known'),False,set(),0),1e-6)

    def test_resume_does_not_reset_counts(self):
        tracker=PieceExploration(.02,.04,step=300)
        tracker.reward(dict(stack='x'),False,set(),0)
        state=tracker.state_dict()
        resumed=PieceExploration(.02,.04,step=state['step'],state=state)
        self.assertEqual(tracker.reward(dict(stack='x'),False,set(),0),resumed.reward(dict(stack='x'),False,set(),0))
        self.assertEqual(tracker.state_dict(),resumed.state_dict())

    def test_real_training_saves_curiosity_and_resume_retains_experience(self):
        with tempfile.TemporaryDirectory() as tmp:
            root=Path(tmp)
            args=['--rules','surface','--architecture','joint','--device','cpu','--envs','2','--rollout-steps','32','--total-steps','256','--epochs','1','--batch-size','32','--hidden','32','--exploration-bonus','.02','--exploration-fraction','.05','--async-rollouts','--output',str(root)]
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            _,before=load_checkpoint(root/'latest.pt')
            self.assertGreater(before['exploration']['rewarded_actions'],0)
            self.assertEqual(before['exploration']['step'],before['step'])
            args[args.index('--total-steps')+1]='320';args+=['--resume',str(root/'latest.pt')]
            with contextlib.redirect_stdout(io.StringIO()):run(parser().parse_args(args))
            _,after=load_checkpoint(root/'latest.pt')
            for stack,count in before['exploration']['counts'].items():
                self.assertGreaterEqual(after['exploration']['counts'][stack],count)
            self.assertGreaterEqual(after['exploration']['total_reward'],before['exploration']['total_reward'])
            self.assertEqual(after['exploration']['step'],after['step'])

class PersistentMixtureTests(unittest.TestCase):
    def test_exploration_survives_confident_policy_and_keeps_learned_tile_preferences(self):
        import torch
        from torch.distributions import Categorical
        from smallworld_ai.exploration import training_distribution
        from smallworld_ai.hierarchical import HierarchicalPolicy, KINDS
        from smallworld_ai.graph import A
        from smallworld_ai import ACTION_FEATURES
        model=HierarchicalPolicy(32,joint=True)
        candidates=torch.zeros(1,5,A)
        candidates[:,:,ACTION_FEATURES]=1
        candidates[:,0:2,ACTION_FEATURES+KINDS]=1
        candidates[:,2:4,ACTION_FEATURES+KINDS+1]=1
        mask=torch.tensor([[True,True,True,True,False]])
        logits=torch.tensor([[40.,39.,-20.,-22.,-1e9]],requires_grad=True)
        original=Categorical(logits=logits)
        mixed=training_distribution(model,original,(None,candidates,mask),.05,1.5)
        self.assertAlmostEqual(float(mixed.probs[0,2:4].sum().detach()),.00625,places=6)
        self.assertGreater(float(mixed.probs[0,2].detach()),float(mixed.probs[0,3].detach()))
        self.assertEqual(float(mixed.probs[0,4].detach()),0)
        self.assertIs(training_distribution(model,original,(None,candidates,mask),0),original)
        # PPO collection and updates must use precisely the same behavior law.
        update=training_distribution(model,Categorical(logits=logits),(None,candidates,mask),.05,1.5)
        torch.testing.assert_close(mixed.logits,update.logits)
        (-update.log_prob(torch.tensor([2]))).backward()
        self.assertTrue(torch.isfinite(logits.grad).all())
        self.assertGreater(float(logits.grad[0,2].abs()),0)
        self.assertEqual(int(original.logits.argmax(-1)),0)

    def test_exploratory_success_teaches_stack_selection_and_can_be_unlearned(self):
        import torch
        from torch.distributions import Categorical
        from smallworld_ai.exploration import training_distribution
        from smallworld_ai.hierarchical import HierarchicalPolicy, KINDS
        from smallworld_ai.graph import A
        from smallworld_ai import ACTION_FEATURES
        model = HierarchicalPolicy(32, joint=True)
        candidates = torch.zeros(1, 2, A)
        candidates[:, :, ACTION_FEATURES] = 1
        candidates[:, 0, ACTION_FEATURES+KINDS] = 1
        candidates[:, 1, ACTION_FEATURES+KINDS+1] = 1
        batch = (None, candidates, torch.ones(1, 2, dtype=torch.bool))
        # One destination per piece isolates learning which piece to choose.
        # The previous uniform group sampler had exactly zero gradient here.
        logits = torch.zeros(1, 2, requires_grad=True)
        q = training_distribution(model, Categorical(logits=logits), batch, .15,
                                  retry_groups=torch.tensor([-2]))
        (-q.log_prob(torch.tensor([1]))).backward()
        self.assertLess(float(logits.grad[0, 1]), -.1)
        self.assertGreater(float(logits.grad[0, 0]), .1)
        improved = logits.detach() - .1*logits.grad
        self.assertGreater(float(Categorical(logits=improved).probs[0, 1]), .5)
        harmful = logits.detach() + .1*logits.grad
        self.assertLess(float(Categorical(logits=harmful).probs[0, 1]), .5)

    def test_generic_stack_prior_and_destination_count_do_not_encode_piece_rules(self):
        import torch
        from torch.distributions import Categorical
        from smallworld_ai.exploration import training_distribution
        from smallworld_ai.hierarchical import HierarchicalPolicy, KINDS
        from smallworld_ai.graph import A
        from smallworld_ai import ACTION_FEATURES
        model = HierarchicalPolicy(32, joint=True)
        candidates = torch.zeros(1, 4, A)
        candidates[:, :, ACTION_FEATURES] = 1
        candidates[:, :3, ACTION_FEATURES+KINDS+7] = 1
        batch = (None, candidates, torch.ones(1, 4, dtype=torch.bool))
        q = training_distribution(model, Categorical(logits=torch.zeros(1,4)), batch, 1., piece_prior=3.)
        self.assertAlmostEqual(float(q.probs[0,:3].sum()), .75, places=6)
        candidates[:, :3, ACTION_FEATURES+KINDS+7] = 0
        candidates[:, :3, ACTION_FEATURES+KINDS+13] = 1
        other = training_distribution(model, Categorical(logits=torch.zeros(1,4)), batch, 1., piece_prior=3.)
        torch.testing.assert_close(q.probs, other.probs)

if __name__=='__main__':unittest.main()
