import contextlib
import io
from pathlib import Path
import tempfile
import unittest
from unittest.mock import patch

import torch

from smallworld_ai import ACTION_FEATURES, SCHEMA, STATE_FEATURES
from smallworld_ai.device import select_device
from smallworld_ai.engine import DEFAULT_ENGINE, Engine, LearnerEnv
from smallworld_ai.model import Policy, batch_decisions, export_policy, load_checkpoint
from smallworld_ai.train import advantages, parser, run


class TrainingTests(unittest.TestCase):
    def test_gae_stops_at_terminal(self):
        advantage, returns = advantages(torch.tensor([[1.], [2.]]), torch.tensor([[0.], [1.]]),
                                        torch.zeros(2, 1), torch.tensor([100.]), .9, 1.)
        torch.testing.assert_close(returns, torch.tensor([[2.8], [2.]]))
        torch.testing.assert_close(returns, advantage)

    def test_padding_cannot_be_selected(self):
        def decision(count):
            return {"schema": SCHEMA, "state": [0.]*STATE_FEATURES,
                    "candidates": [{"features": [0.]*ACTION_FEATURES} for _ in range(count)]}
        policy = Policy(16)
        distribution, _ = policy(*batch_decisions([decision(1), decision(5)], "cpu"))
        self.assertTrue((distribution.sample((100,))[:, 0] == 0).all())
        self.assertTrue((distribution.probs[0, 1:] == 0).all())

    @patch("smallworld_ai.device.nvidia_report", return_value="Test driver")
    @patch("torch.cuda.is_available", return_value=False)
    def test_auto_cpu_and_required_cuda(self, *_):
        device, report = select_device("auto")
        self.assertEqual(device.type, "cpu")
        self.assertIn("notice", report)
        with self.assertRaisesRegex(RuntimeError, "CUDA compute test failed"):
            select_device("cuda")

    @unittest.skipUnless(DEFAULT_ENGINE.is_file(), "build ai/bin/botenv first")
    def test_train_resume_export_and_go_parity(self):
        torch.set_num_threads(1)
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            arguments = ["--device", "cpu", "--envs", "2", "--rollout-steps", "32", "--total-steps", "128",
                         "--epochs", "2", "--batch-size", "32", "--hidden", "16", "--output", str(root), "--snapshot-every", "1"]
            with contextlib.redirect_stdout(io.StringIO()):
                run(parser().parse_args(arguments))
            policy, checkpoint = load_checkpoint(root / "latest.pt")
            self.assertEqual(checkpoint["step"], 128)
            original_weights = {k:v.clone() for k,v in policy.state_dict().items()}
            arguments[arguments.index("--total-steps")+1] = "256"
            arguments.extend(["--resume", str(root / "latest.pt")])
            with contextlib.redirect_stdout(io.StringIO()):
                run(parser().parse_args(arguments))
            policy, checkpoint = load_checkpoint(root / "latest.pt")
            self.assertEqual(checkpoint["step"], 256)
            self.assertTrue(any(not torch.equal(original_weights[k], v) for k,v in policy.state_dict().items()))
            self.assertTrue(all(torch.isfinite(p).all() for p in policy.parameters()))
            export_policy(policy, root / "models" / "test.json", "test-v1", "Test policy")
            with Engine(model_dir=root / "models") as engine:
                response = engine.request(op="reset", players=2, map="map2players", seed=123456)
                for _ in range(30):
                    d = response["decision"]
                    tensors = batch_decisions([d], "cpu")
                    states, candidates, _ = tensors
                    with torch.inference_mode():
                        expected = policy.actor(torch.cat((states.unsqueeze(1).expand(-1,candidates.shape[1],-1), candidates), -1)).flatten()
                    actual = engine.request(op="scores", policy="test-v1")["policy_scores"]
                    torch.testing.assert_close(torch.tensor(actual), expected, atol=1e-5, rtol=1e-5)
                    reordered = dict(d, candidates=list(reversed(d["candidates"])))
                    with torch.inference_mode():
                        distribution, _ = policy(*batch_decisions([d, reordered], "cpu"))
                    torch.testing.assert_close(distribution.probs[0], distribution.probs[1].flip(0))
                    response = engine.request(op="step", policy="test-v1")
                    if response["done"]:
                        break

    @unittest.skipUnless(DEFAULT_ENGINE.is_file(), "build ai/bin/botenv first")
    def test_learner_transitions_and_limit(self):
        env = LearnerEnv(DEFAULT_ENGINE, 10, 3, ["map3players"], "greedy")
        try:
            obs = env.reset()
            for _ in range(30):
                self.assertEqual(obs["player"], env.seat)
                obs, reward, done, truncated, _ = env.step(0)
                self.assertFalse(truncated)
                if done:
                    break
        finally:
            env.close()
        with Engine(max_steps=1) as engine:
            engine.request(op="reset", players=2, map="map2players", seed=1)
            response = engine.request(op="step", policy="random")
            self.assertTrue(response["truncated"])
            self.assertFalse(response["done"])
            with self.assertRaisesRegex(RuntimeError, "episode ended"):
                engine.request(op="step", policy="random")


if __name__ == "__main__":
    unittest.main()
