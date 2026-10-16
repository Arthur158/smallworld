import contextlib
import io
import json
from pathlib import Path
import tempfile
import unittest

import torch

from smallworld_ai.feedback import rejection_loss
from smallworld_ai.model import load_checkpoint
from smallworld_ai.train import parser, run


class RejectionLearningTests(unittest.TestCase):
    def setUp(self):
        torch.set_num_threads(1)

    def test_rejection_lowers_only_the_attempted_choice_without_rule_features(self):
        logits = torch.tensor([[3., 1., 0., -100.]], requires_grad=True)
        mask = torch.tensor([[True, True, True, False]])
        loss = rejection_loss(logits, torch.tensor([0]), torch.tensor([0.]), mask)
        expected = -torch.log1p(-logits[:, :3].softmax(-1)[0, 0])
        torch.testing.assert_close(loss, expected)
        loss.backward()
        self.assertGreater(logits.grad[0, 0].item(), 0)
        self.assertTrue((logits.grad[0, 1:3] < 0).all())
        self.assertEqual(logits.grad[0, 3].item(), 0)
        updated = logits.detach() - .1*logits.grad
        self.assertLess(updated.softmax(-1)[0, 0], logits.softmax(-1)[0, 0])
        # Reordering action IDs cannot change the learning rule.
        order = torch.tensor([2, 3, 0, 1])
        permuted = rejection_loss(logits.detach()[:, order], torch.tensor([2]), torch.tensor([0.]), mask[:, order])
        torch.testing.assert_close(permuted, loss)

    def test_success_does_not_teach_tactical_value_and_singleton_is_safe(self):
        for accepted, mask in ((1., [True, True]), (0., [True, False])):
            logits = torch.tensor([[1000., -1000.]], requires_grad=True)
            loss = rejection_loss(logits, torch.tensor([0]), torch.tensor([accepted]), torch.tensor([mask]))
            loss.backward()
            self.assertEqual(loss.item(), 0)
            self.assertTrue((logits.grad == 0).all())

    def test_confident_rejections_have_finite_useful_gradients(self):
        logits = torch.tensor([[1000., -1000.]], requires_grad=True)
        loss = rejection_loss(logits, torch.tensor([0]), torch.tensor([0.]), torch.ones(1, 2, dtype=torch.bool))
        loss.backward()
        self.assertTrue(torch.isfinite(loss))
        torch.testing.assert_close(logits.grad, torch.tensor([[1., -1.]]))

    def test_real_training_without_acceptance_head_resumes_and_accounts_for_modes(self):
        with tempfile.TemporaryDirectory() as directory:
            args = ['--rules', 'surface', '--architecture', 'joint', '--hidden', '32',
                    '--device', 'cpu', '--envs', '2', '--rollout-steps', '32', '--total-steps', '128',
                    '--epochs', '1', '--batch-size', '32', '--opponent', 'random',
                    '--maps', 'map4players', '--players', '4', '--rejection-loss', '.05',
                    '--exploration-fraction', '.5', '--async-rollouts', '--output', directory]
            with contextlib.redirect_stdout(io.StringIO()):
                run(parser().parse_args(args))
            _, checkpoint = load_checkpoint(Path(directory)/'latest.pt')
            self.assertIsNone(checkpoint['acceptance'])
            args[args.index('--total-steps')+1] = '192'
            args += ['--resume', str(Path(directory)/'latest.pt')]
            with contextlib.redirect_stdout(io.StringIO()):
                run(parser().parse_args(args))
            model, resumed = load_checkpoint(Path(directory)/'latest.pt')
            self.assertEqual(resumed['step'], 192)
            self.assertEqual(model.format_version, 4)
            metrics = [json.loads(line) for line in (Path(directory)/'metrics.jsonl').read_text().splitlines()]
            self.assertTrue(any(m['rejection_loss'] > 0 for m in metrics))
            for row in metrics:
                count = sum(row[m+'_decisions'] for m in ('policy', 'exploration', 'retry'))
                rejected = sum(row[m+'_decisions']*(row[m+'_rejection_rate'] or 0) for m in ('policy', 'exploration', 'retry'))
                self.assertEqual(count, 64)
                self.assertAlmostEqual(rejected/count, row['rollout_rejection_rate'])


if __name__ == '__main__':
    unittest.main()
