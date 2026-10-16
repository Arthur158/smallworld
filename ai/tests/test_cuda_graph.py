import copy
from concurrent.futures import ThreadPoolExecutor
import unittest
import torch
from smallworld_ai.cuda_graph import CapturedInference,GPU_LOCK,padded_batch
from smallworld_ai.entity import EntityPolicy
from smallworld_ai.relational import RelationalPolicy
from smallworld_ai.engine import Engine
from smallworld_ai.model import batch_decisions
from smallworld_ai.inference import BatchedInference


class CaptureTests(unittest.TestCase):
    @classmethod
    def setUpClass(cls):torch.set_num_threads(1)
    def decision(self,seed=1):
        with Engine(rules='surface') as e:return e.request(op='reset',players=4,map='map4players',seed=seed)['decision']

    def test_cpu_fallback_and_padding(self):
        model=EntityPolicy(32).eval();batch=batch_decisions([self.decision()],'cpu')
        capture=CapturedInference(enabled=True)
        with torch.inference_mode():
            expected,value=model(*batch);actual,v=capture(model,batch)
            torch.testing.assert_close(expected.probs,actual.probs);torch.testing.assert_close(value,v)
            padded=padded_batch(batch,4,96,256);d,w=model(*padded)
            torch.testing.assert_close(d.probs[0,:batch[2].shape[1]],expected.probs[0],rtol=2e-5,atol=2e-6)
        self.assertEqual(capture.captures,0)

    @unittest.skipUnless(torch.cuda.is_available(),'CUDA required')
    def test_replay_retains_outputs_tracks_weight_updates_and_evicts(self):
        model=EntityPolicy(32).cuda().eval();batch=batch_decisions([self.decision()],'cuda')
        cache=CapturedInference(enabled=True,max_entries=1)
        with torch.inference_mode():
            for _ in range(5):saved,values=cache(model,batch)
            frozen=saved.probs.clone();values_frozen=values.clone()
            with torch.no_grad():model.entity.actor[-1].weight.normal_(0,.05);model.entity.critic[-1].bias.add_(.3)
            expected,ev=model(*batch);actual,av=cache(model,batch)
            torch.testing.assert_close(expected.probs,actual.probs,rtol=5e-5,atol=5e-6);torch.testing.assert_close(ev,av)
            torch.testing.assert_close(saved.probs,frozen);torch.testing.assert_close(values,values_frozen)
            for _ in range(5):cache(model,batch,value=False)
        self.assertGreater(cache.hits,0);self.assertGreater(cache.evictions,0);self.assertEqual(cache.failures,0)
        self.assertLessEqual(cache.bytes,cache.max_bytes);cache.clear()

    @unittest.skipUnless(torch.cuda.is_available(),'CUDA required')
    def test_alternating_shapes_do_not_recapture_on_every_eviction(self):
        model=RelationalPolicy(32).cuda().eval()
        batch=batch_decisions([self.decision()],'cuda')
        cache=CapturedInference(enabled=True,max_entries=1)
        with torch.inference_mode():
            expected,_=model(*batch)
            for _ in range(60):
                for value in (True,False):
                    actual,_=cache(model,batch,value=value)
                    torch.testing.assert_close(expected.probs,actual.probs,rtol=5e-5,atol=5e-6)
        self.assertLessEqual(cache.captures,3)
        self.assertGreater(cache.hits,20)
        self.assertGreater(cache.evictions,0)
        self.assertEqual(cache.failures,0)
        cache.clear()

    @unittest.skipUnless(torch.cuda.is_available(),'CUDA required')
    def test_capture_is_safe_with_opponent_thread_and_learner(self):
        host=RelationalPolicy(32).eval();learner=EntityPolicy(32).cuda().eval()
        decision=self.decision();cache=CapturedInference(enabled=True)
        service=BatchedInference('cuda',captured=cache)
        expected=host.choose(decision)
        try:
            with ThreadPoolExecutor(max_workers=4) as pool:
                futures=[pool.submit(lambda:[service.choose(host,decision) for _ in range(16)]) for _ in range(4)]
                for _ in range(25):
                    with GPU_LOCK,torch.inference_mode():
                        batch=batch_decisions([decision],'cuda');actual,_=cache(learner,batch)
                        actual.probs.argmax(-1).cpu()
                for future in futures:self.assertEqual(future.result(),[expected]*16)
            self.assertGreater(cache.captures,0);self.assertEqual(cache.failures,0)
        finally:service.close();cache.clear()

if __name__=='__main__':unittest.main()
