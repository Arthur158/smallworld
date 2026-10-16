"""Batch frozen opponents from concurrent games on the training device.

A single inference thread owns the device copies. Episode opponents remain
frozen; batching changes execution only, never the selected policy or action.
"""
from concurrent.futures import Future
import copy
from queue import Empty, Queue
from threading import Thread
import time
from weakref import WeakKeyDictionary

import torch

from .model import batch_decisions
from .cuda_graph import GPU_LOCK, CapturedInference


class BatchedInference:
    def __init__(self, device, max_batch=64, wait_seconds=0.002, captured=None):
        self.captured = captured or CapturedInference()
        self.device = torch.device(device)
        self.max_batch, self.wait_seconds = max_batch, wait_seconds
        self.queue = Queue()
        self.cache = WeakKeyDictionary()
        self.requests = self.batches = 0
        self.seconds = 0.0
        self.thread = Thread(target=self._run, name='opponent-inference', daemon=True)
        self.thread.start()

    def choose(self, policy, decision):
        future = Future()
        self.queue.put((policy, decision, future))
        return future.result(timeout=120)

    def _run(self):
        while True:
            first = self.queue.get()
            if first is None:
                return
            requests = [first]
            deadline = time.monotonic()+self.wait_seconds
            while len(requests) < self.max_batch:
                try:
                    request = self.queue.get(timeout=max(0, deadline-time.monotonic()))
                except Empty:
                    break
                if request is None:
                    self.queue.put(None)
                    break
                requests.append(request)
            groups = {}
            for policy, decision, future in requests:
                groups.setdefault(policy, []).append((decision, future))
            for policy, items in groups.items():
                started = time.monotonic()
                try:
                    with GPU_LOCK, torch.inference_mode():
                        if policy not in self.cache:
                            self.cache[policy] = copy.deepcopy(policy).to(self.device).eval()
                        batch = batch_decisions([d for d, _ in items], self.device, policy)
                        model = self.cache[policy]
                        distribution, _ = self.captured(model, batch, value=False)
                        actions = (model.greedy_actions(distribution, batch) if hasattr(model, "greedy_actions")
                                   else distribution.logits.argmax(-1)).cpu().tolist()
                    for (_, future), action in zip(items, actions):
                        future.set_result(action)
                    self.requests += len(items)
                    self.batches += 1
                    self.seconds += time.monotonic()-started
                except Exception as error:
                    for _, future in items:
                        future.set_exception(error)
            # Do not retain the last batch's policies after their episodes end.
            del groups, requests, first, policy, items

    def stats(self):
        return {'opponent_requests': self.requests, 'opponent_batches': self.batches,
                'opponent_inference_seconds': round(self.seconds, 3)}

    def close(self):
        self.queue.put(None)
        self.thread.join(timeout=120)
        if self.thread.is_alive():
            raise RuntimeError('opponent inference failed to shut down')
        self.cache.clear()
