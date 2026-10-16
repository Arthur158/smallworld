"""Bounded CUDA inference replay with ordinary execution as a fallback.

All collection-side CUDA work (including tensor copies) shares GPU_LOCK. CUDA
capture must never overlap other CUDA work in the same training process.
Returned tensors own their storage; a later replay cannot overwrite a rollout.
"""
from collections import OrderedDict
from threading import RLock
import torch
from torch.distributions import Categorical

GPU_LOCK=RLock()


def padded_batch(batch,b,n,c):
    f=torch.nn.functional
    size,choices=batch[2].shape;regions=batch[5].shape[1]
    if size>b or choices>c or regions>n:raise ValueError('capture shape too small')
    # Duplicate one real observation for unused batch rows. Each padded row must
    # contain a real candidate/tile so softmax and pooling never see empty sets.
    rows=torch.arange(b,device=batch[0].device).clamp_max(size-1)
    x=[t.index_select(0,rows) for t in batch]
    return (x[0],f.pad(x[1],(0,0,0,c-choices)),f.pad(x[2],(0,c-choices)),
            f.pad(x[3],(0,0,0,n-regions)),f.pad(x[4],(0,n-regions,0,n-regions)),
            f.pad(x[5],(0,n-regions)),f.pad(x[6],(0,0,0,c-choices),value=-1))


class CapturedInference:
    def __init__(self,enabled=False,max_entries=32,max_bytes=384<<20):
        self.enabled,self.max_entries,self.max_bytes=enabled,max_entries,max_bytes
        self.entries=OrderedDict();self.seen=OrderedDict();self.bytes=0
        self.hits=self.misses=self.captures=self.evictions=self.failures=0

    def evict(self):
        key,entry=self.entries.popitem(last=False)
        self.bytes-=entry['cost'];self.evictions+=1
        # Mixed leagues alternate many model/shape combinations. Re-capturing
        # an evicted key immediately can cost more than ordinary inference.
        # Require sustained new demand before this key may allocate again.
        self.seen[key]=-128

    def ordinary(self,model,batch,value):
        return model(*batch) if value or not hasattr(model,'action_distribution') else (model.action_distribution(*batch),None)

    def __call__(self,model,batch,value=True):
        with GPU_LOCK:
            if len(batch) != 7 or not self.enabled or batch[0].device.type!='cuda' or getattr(model,'schema',None)!='smallworld-bot-v2':
                return self.ordinary(model,batch,value)
            if model.training or torch.is_grad_enabled():raise ValueError('captured inference requires eval and no gradients')
            b,c=batch[2].shape;n=batch[5].shape[1]
            if b>8 or c>512 or n>160:
                self.misses+=1;return self.ordinary(model,batch,value)
            shape=(1<<(b-1).bit_length(),64 if n<=64 else 96 if n<=96 else 160,256 if c<=256 else 512)
            key=(id(model),shape,value)
            entry=self.entries.get(key)
            if entry is None:
                self.misses+=1
                seen=self.seen.pop(key,0)+1;self.seen[key]=seen
                while len(self.seen)>1024:self.seen.popitem(last=False)
                if seen<4 or seen<0:return self.ordinary(model,batch,value)
                # Leave headroom and evict BEFORE capture, not after allocating.
                while self.entries and (len(self.entries)>=self.max_entries or self.bytes>self.max_bytes-(48<<20)):
                    self.evict()
                free,_=torch.cuda.mem_get_info(batch[0].device)
                if free<(512<<20):return self.ordinary(model,batch,value)
                before=torch.cuda.memory_reserved(batch[0].device)
                static=padded_batch(batch,*shape)
                stream=torch.cuda.Stream(device=batch[0].device)
                stream.wait_stream(torch.cuda.current_stream(batch[0].device))
                with torch.cuda.stream(stream):
                    for _ in range(3):self.ordinary(model,static,value)
                torch.cuda.current_stream(batch[0].device).wait_stream(stream)
                torch.cuda.synchronize(batch[0].device)
                graph=torch.cuda.CUDAGraph()
                try:
                    with torch.cuda.graph(graph):
                        distribution,values=self.ordinary(model,static,value)
                        logits=distribution.logits
                    graph.replay()
                    cost=max(0,torch.cuda.memory_reserved(batch[0].device)-before)+sum(p.numel()*p.element_size() for p in model.parameters())
                    if cost>self.max_bytes:
                        self.seen[key]=-1000000
                        return self.ordinary(model,batch,value)
                    while self.entries and self.bytes+cost>self.max_bytes:
                        self.evict()
                    entry=dict(model=model,static=static,graph=graph,logits=logits,values=values,cost=cost)
                    self.entries[key]=entry;self.bytes+=cost;self.captures+=1
                except (RuntimeError,torch.OutOfMemoryError):
                    self.failures+=1;self.seen[key]=-1000000
                    # Warmup succeeded: graph capture alone is optional. A later
                    # ordinary model failure is propagated normally to the caller.
                    return self.ordinary(model,batch,value)
            else:
                self.hits+=1;self.entries.move_to_end(key)
                for dst,src in zip(entry['static'],padded_batch(batch,*shape)):dst.copy_(src)
                entry['graph'].replay()
            logits=entry['logits'][:b,:c].clone()
            values=entry['values'][:b].clone() if value else None
            return Categorical(logits=logits,validate_args=False),values

    def stats(self):
        return {'cuda_graph_hits':self.hits,'cuda_graph_misses':self.misses,
                'cuda_graph_captures':self.captures,'cuda_graph_evictions':self.evictions,
                'cuda_graph_failures':self.failures,'cuda_graph_cache_mb':round(self.bytes/1e6,1)}

    def clear(self):
        with GPU_LOCK:self.entries.clear();self.seen.clear();self.bytes=0
