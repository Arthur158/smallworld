import unittest
import torch
from smallworld_ai.engine import Engine
from smallworld_ai.model import batch_decisions

class PackedTests(unittest.TestCase):
    def test_packed_observations_preserve_full_decisions_and_game_outcomes(self):
        for rules in ('legacy','surface'):
            with self.subTest(rules=rules),Engine(rules=rules) as text,Engine(rules=rules,packed=True) as packed:
                a=text.request(op='reset',seed=51,players=2,map='map2players')
                b=packed.request(op='reset',seed=51,players=2,map='map2players')
                for _ in range(120):
                    self.assertEqual(a['scores'],b['scores'])
                    self.assertEqual(a['done'],b['done'])
                    self.assertEqual([c['action'] for c in a['decision']['candidates']],[c['action'] for c in b['decision']['candidates']])
                    if a['done']:break
                    for x,y in zip(batch_decisions([a['decision']],'cpu'),batch_decisions([b['decision']],'cpu')):
                        torch.testing.assert_close(x,y,atol=0,rtol=0)
                    a=text.request(op='step',policy='greedy');b=packed.request(op='step',policy='greedy')

if __name__=='__main__':unittest.main()
