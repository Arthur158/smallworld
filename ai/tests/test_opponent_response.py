import contextlib
import copy
import io
import json
from pathlib import Path
import tempfile
import unittest

import torch

from smallworld_ai.engine import score_margin, winning_reward
from smallworld_ai.feedback import ResponseHead, response_labels, SELF_OWNER, STACK_START
from smallworld_ai.graph import S, A, N
from smallworld_ai.model import load_checkpoint
from smallworld_ai.relational import RelationalPolicy, WIDTH
from smallworld_ai.train import parser, run


class WinningRewardTests(unittest.TestCase):
    def test_dense_feedback_counts_own_gains_and_leading_opponent_gains(self):
        initial = [10, 12, 8, 9]
        self.assertAlmostEqual(winning_reward(initial, [13,12,8,9],0,False), .03)
        self.assertAlmostEqual(winning_reward(initial, [10,15,8,9],0,False), -.03)
        self.assertAlmostEqual(winning_reward(initial, [10,12,9,9],0,False), 0.)
        # Harming a trailing player must not look like progress against a leader.
        self.assertAlmostEqual(winning_reward(initial, [10,12,0,9],0,False), 0.)

    def test_totals_depend_only_on_final_win_credit_not_points_or_seat(self):
        for seat in range(4):
            for terminal in ([150,149,148,147], [25,24,23,22], [90,90,90,90], [4,5,8,8]):
                states = [[5,5,5,5], [10,40,20,10], [60,41,60,40], list(terminal)]
                # Also exercise a reset after opponents have already scored.
                for first in (0, 1):
                    start = score_margin(states[first], seat)
                    total = sum(winning_reward(states[i],states[i+1],seat,i==len(states)-2,start)
                                for i in range(first,len(states)-1))
                    expected = 2/terminal.count(terminal[seat])-1 if terminal[seat] == max(terminal) else -1
                    self.assertAlmostEqual(total, expected)

    def test_win_objective_rejects_incompatible_reward_options(self):
        for extra in (['--gamma','.99'], ['--exploration-bonus','.02'], ['--rejection-penalty','.001']):
            args=parser().parse_args(['--objective','win','--gamma','1','--rejection-penalty','0']+extra)
            with self.assertRaisesRegex(ValueError,'win objective requires'):
                run(args)


class ResponseLearningTests(unittest.TestCase):
    def setUp(self):
        torch.set_num_threads(1)

    def test_targets_use_actual_public_outcomes_in_same_player_perspective(self):
        nodes = [[0.]*N for _ in range(3)]
        for i, row in enumerate(nodes):
            row[SELF_OWNER+i] = 1
            row[STACK_START] = .1
        before = dict(schema='smallworld-bot-v2',player=2,graph=dict(nodes=nodes))
        after = copy.deepcopy(before)
        after['graph']['nodes'][0][SELF_OWNER] = 0
        after['graph']['nodes'][0][SELF_OWNER+3] = 1
        after['graph']['nodes'][0][STACK_START] = .05
        after['graph']['nodes'][2][STACK_START+1] = .1
        original = copy.deepcopy(before)
        self.assertEqual(response_labels(before, after),dict(owner=[3,1,2],pieces=[0,1,2],owner_before=[0,1,2]))
        self.assertEqual(before, original)
        after['player'] = 1
        with self.assertRaises(ValueError): response_labels(before, after)

    def test_prediction_trains_shared_board_encoder_without_prescribing_actions(self):
        model = RelationalPolicy(32)
        head = ResponseHead(WIDTH)
        batch = (torch.randn(1,S),torch.randn(1,2,A),torch.ones(1,2,dtype=torch.bool),
                 torch.randn(1,3,N),torch.eye(3)[None],torch.ones(1,3),torch.zeros(1,2,2,dtype=torch.long))
        before = model(*batch)[0].probs.detach().clone()
        embeddings = model.training_forward(batch,torch.tensor([0]))[4]
        labels = torch.tensor([[[1,0,0],[2,2,2],[-1,-1,-1]]])
        loss, counts = head(embeddings, labels)
        loss.backward()
        self.assertGreater(model.relation_node.weight.grad.abs().sum().item(), 0)
        self.assertIsNone(model.relation_actor[-1].weight.grad)
        self.assertEqual(counts[0].item(), 2)
        torch.testing.assert_close(model(*batch)[0].probs, before)
        # Unobserved future (padding, terminal, unfinished turns) has no target.
        loss, counts = head(embeddings.detach(), torch.full_like(labels,-1))
        self.assertEqual(loss.item(), 0)
        self.assertTrue((counts == 0).all())

    def test_real_training_keeps_forecast_out_of_export_and_resumes_auxiliary_optimizer(self):
        with tempfile.TemporaryDirectory() as directory:
            args=['--rules','surface','--architecture','relational','--hidden','32',
                  '--device','cpu','--envs','2','--rollout-steps','64','--total-steps','256',
                  '--epochs','1','--batch-size','32','--opponent','random','--player-counts','2','3','4','5','6',
                  '--maps','map4players','--async-rollouts','--objective','win','--gamma','1',
                  '--rejection-penalty','0','--rejection-loss','.05','--response-loss','.05',
                  '--survival-loss','.05','--acceptance-loss','.05','--output',directory]
            with contextlib.redirect_stdout(io.StringIO()): run(parser().parse_args(args))
            model, before=load_checkpoint(Path(directory)/'latest.pt')
            self.assertEqual(model.format_version,5)
            self.assertTrue(before['opponent_response']['optimizer']['state'])
            self.assertFalse(any('response' in name for name in model.state_dict()))
            args[args.index('--total-steps')+1]='384'
            args += ['--resume',str(Path(directory)/'latest.pt')]
            with contextlib.redirect_stdout(io.StringIO()): run(parser().parse_args(args))
            _, after=load_checkpoint(Path(directory)/'latest.pt')
            self.assertEqual(after['step'],384)
            self.assertTrue(any(not torch.equal(value,after['opponent_response']['model'][key])
                                for key,value in before['opponent_response']['model'].items()))
            logs=[json.loads(line) for line in (Path(directory)/'metrics.jsonl').read_text().splitlines()]
            self.assertGreater(sum(m['response_tiles'] for m in logs),0)
            self.assertTrue(all(m['response_tiles'] >= m['survival_tiles'] for m in logs))
            self.assertTrue(all('response_changed_owner_accuracy' in m for m in logs))
            self.assertGreater(sum(row['games'] for row in after['player_count_stats'].values()), 0)
            for players, row in after['player_count_stats'].items():
                self.assertIn(int(players), (2,3,4,5,6))
                self.assertAlmostEqual(row['return_total'], 2*row['win_credit']-row['games'])
                self.assertGreaterEqual(row['games'], before['player_count_stats'].get(players,{}).get('games',0))


if __name__ == '__main__': unittest.main()
