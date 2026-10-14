import copy
import json
import unittest

from smallworld_ai.engine import DEFAULT_ENGINE, Engine
from smallworld_ai.maps import MapGenerator


class GeneratedMapTests(unittest.TestCase):
    def test_reproducibility_diversity_and_separate_evaluation(self):
        generator = MapGenerator()
        try:
            first = generator.generate(42, 2, "train", "standard")
            self.assertEqual(first, generator.generate(42, 2, "train", "standard"))
            hashes, sizes, shapes, evaluation_hashes = set(), set(), set(), set()
            for seed in range(30):
                tiles, info = generator.generate(seed, 2+seed % 5, "train", "standard")
                hashes.add(info["hash"])
                sizes.add(len(tiles))
                shapes.add(info["settings"]["shapeMode"])
                _, other = generator.generate(seed, 2+seed % 5, "eval", "standard")
                evaluation_hashes.add(other["hash"])
            self.assertEqual(len(hashes), 30)
            self.assertGreater(len(sizes), 10)
            self.assertEqual(shapes, {"mainland", "islands"})
            self.assertFalse(hashes & evaluation_hashes)
        finally:
            generator.close()

    @unittest.skipUnless(DEFAULT_ENGINE.is_file(), "build ai/bin/botenv first")
    def test_complete_generated_matches(self):
        with Engine() as engine:
            for seed in range(30):
                with self.subTest(seed=seed):
                    players = 2+seed % 5
                    response = engine.request(op="reset", seed=seed, players=players, map="generated",
                                              map_split="test", map_distribution="stress" if seed % 2 else "standard")
                    while not (response["done"] or response["truncated"]):
                        response = engine.request(op="step", policy="greedy" if seed % 2 else "random")
                    self.assertTrue(response["done"])
                    self.assertFalse(response["truncated"])
                    self.assertEqual(len(response["scores"]), players)

    @unittest.skipUnless(DEFAULT_ENGINE.is_file(), "build ai/bin/botenv first")
    def test_invalid_graph_rejected_without_losing_episode(self):
        generator = MapGenerator()
        try:
            tiles, _ = generator.generate(42, 2, "test", "standard")
        finally:
            generator.close()
        with Engine() as engine:
            before = engine.request(op="reset", seed=42, players=2, map="map2players")
            invalid = copy.deepcopy(tiles)
            invalid[0]["AdjacentIDs"].append("missing-tile")
            # Direct protocol request exercises the Go boundary, bypassing the generator.
            engine.process.stdin.write(json.dumps(dict(op="reset", seed=1, players=2, map="generated", tiles=invalid))+"\n")
            engine.process.stdin.flush()
            self.assertIn("invalid adjacency", json.loads(engine.process.stdout.readline())["error"])
            self.assertEqual(before, engine.request(op="observe"))


if __name__ == "__main__":
    unittest.main()
