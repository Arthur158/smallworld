from concurrent.futures import ThreadPoolExecutor
import unittest
import torch
from smallworld_ai.engine import Engine
from smallworld_ai.graph import GraphPolicy
from smallworld_ai.hierarchical import HierarchicalPolicy
from smallworld_ai.inference import BatchedInference


class InferenceTests(unittest.TestCase):
    def test_batched_frozen_opponents_match_individual_actions(self):
        torch.set_num_threads(1)
        models=[GraphPolicy(32).eval(),HierarchicalPolicy(32).eval()]
        decisions=[]
        with Engine(rules='surface') as engine:
            for seed in range(8):
                decisions.append(engine.request(op='reset',players=2,map='map2players',seed=seed+1)['decision'])
        pairs=[(models[i%2],d) for i,d in enumerate(decisions)]
        expected=[p.choose(d) for p,d in pairs]
        service=BatchedInference('cpu',wait_seconds=.01)
        try:
            with ThreadPoolExecutor(max_workers=8) as pool:
                actual=list(pool.map(lambda pair: service.choose(*pair),pairs))
            self.assertEqual(actual,expected)
            self.assertEqual(service.stats()['opponent_requests'],8)
            self.assertLess(service.stats()['opponent_batches'],8)
            # Malformed requests must fail in the caller, never silently hang.
            with self.assertRaises((KeyError,ValueError)):
                service.choose(models[0],{})
            self.assertEqual(service.choose(*pairs[0]),expected[0])
        finally:
            service.close()
        self.assertFalse(service.thread.is_alive())

if __name__=='__main__':unittest.main()
