"""Permutation-equivariant graph policy for the full rules vocabulary."""
import json
from pathlib import Path

import numpy as np
import torch
from torch import nn
from torch.nn import functional as F
from torch.distributions import Categorical

SCHEMA_V2 = 'smallworld-bot-v2'
SPEC = json.loads((Path(__file__).resolve().parents[1] / 'schema-v2.json').read_text())
S, A, N = (SPEC[k] for k in ('state_features', 'action_features', 'node_features'))
G = 64


class GraphPolicy(nn.Module):
    schema = SCHEMA_V2
    state_features, action_features = S, A
    def __init__(self, hidden=192):
        super().__init__()
        if not 8 <= hidden <= 256:
            raise ValueError('hidden size must be in [8, 256]')
        self.hidden = hidden
        self.graph = nn.ModuleList([nn.Linear(N, G), nn.Linear(2*G, G), nn.Linear(2*G, G)])
        self.actor = nn.Sequential(nn.Linear(S+A+3*G, hidden), nn.ReLU(),
                                   nn.Linear(hidden, hidden), nn.ReLU(), nn.Linear(hidden, 1))
        self.critic = nn.Sequential(nn.Linear(S+G, hidden), nn.ReLU(),
                                    nn.Linear(hidden, hidden), nn.ReLU(), nn.Linear(hidden, 1))

    def inputs(self, states, candidates, mask, nodes, adjacency, node_mask, targets):
        h = F.relu(self.graph[0](nodes)) * node_mask.unsqueeze(-1)
        for layer in self.graph[1:]:
            h = F.relu(layer(torch.cat((h, adjacency.bmm(h)), -1))) * node_mask.unsqueeze(-1)
        pooled = h.sum(1) / node_mask.sum(1, keepdim=True).clamp_min(1)
        # The extra zero node represents non-spatial actions, including pick/finish.
        h = torch.cat((torch.zeros_like(h[:, :1]), h), 1)
        def gather(indices):
            return h.gather(1, (indices+1).unsqueeze(-1).expand(-1, -1, G))
        local = torch.cat((candidates, gather(targets[..., 0]), gather(targets[..., 1]),
                           pooled.unsqueeze(1).expand(-1, candidates.shape[1], -1)), -1)
        return local, pooled

    def forward(self, states, candidates, mask, nodes, adjacency, node_mask, targets):
        local, pooled = self.inputs(states, candidates, mask, nodes, adjacency, node_mask, targets)
        # Compute shared state projection once instead of repeating it for every action.
        first = self.actor[0]
        x = F.linear(states, first.weight[:, :S], first.bias).unsqueeze(1)
        x = F.relu(x + F.linear(local, first.weight[:, S:]))
        logits = self.actor[4](F.relu(self.actor[2](x))).squeeze(-1).masked_fill(~mask, -1e9)
        values = self.critic(torch.cat((states, pooled), -1)).squeeze(-1)
        return Categorical(logits=logits, validate_args=False), values

    @torch.inference_mode()
    def choose(self, decision):
        batch = batch_graph([decision], next(self.parameters()).device)
        distribution, _ = self(*batch)
        return int(self.greedy_actions(distribution, batch).item())

    def greedy_actions(self, distribution, batch):
        return distribution.logits.argmax(-1)

    def warm_start(self, old):
        """Retain the v1 function on the shared prefix; learn added information next."""
        from . import STATE_FEATURES, ACTION_FEATURES
        if old.schema != 'smallworld-bot-v1' or old.hidden > self.hidden:
            raise ValueError('warm start needs a v1 policy no wider than the new model')
        h = old.hidden
        with torch.no_grad():
            for network, old_network in ((self.actor, old.actor), (self.critic, old.critic)):
                for layer in (network[0], network[2], network[4]):
                    layer.weight.zero_(); layer.bias.zero_()
                network[0].weight[:h, :STATE_FEATURES].copy_(old_network[0].weight[:, :STATE_FEATURES])
                network[0].bias[:h].copy_(old_network[0].bias)
                network[2].weight[:h, :h].copy_(old_network[2].weight)
                network[2].bias[:h].copy_(old_network[2].bias)
                network[4].weight[:, :h].copy_(old_network[4].weight)
                network[4].bias.copy_(old_network[4].bias)
            self.actor[0].weight[:h, S:S+ACTION_FEATURES].copy_(old.actor[0].weight[:, STATE_FEATURES:])
            # Give extra hidden units live activations with initially zero output.
            # Their downstream weights can then learn instead of remaining dead.
            if self.hidden > h:
                for network in (self.actor, self.critic):
                    nn.init.kaiming_uniform_(network[0].weight[h:], a=5**.5)
                    nn.init.kaiming_uniform_(network[2].weight[h:], a=5**.5)


def batch_graph(decisions, device):
    b = len(decisions)
    c = max(len(d['candidates']) for d in decisions)
    n = max(len(d['graph']['nodes']) for d in decisions)
    if not b or not c or not 1 <= n <= 160:
        raise ValueError('empty/oversized graph decision')
    states = np.zeros((b, S), np.float32)
    candidates = np.zeros((b, c, A), np.float32)
    mask = np.zeros((b, c), np.bool_)
    nodes = np.zeros((b, n, N), np.float32)
    adjacency = np.zeros((b, n, n), np.float32)
    node_mask = np.zeros((b, n), np.float32)
    targets = np.full((b, c, 2), -1, np.int64)
    for i, d in enumerate(decisions):
        if d['schema'] != SCHEMA_V2 or len(d['state']) != S:
            raise ValueError('incompatible graph schema')
        count, regions = len(d['candidates']), len(d['graph']['nodes'])
        if not count:
            raise ValueError('terminal decision cannot be sampled')
        states[i] = d['state']
        candidates[i, :count] = [a['features'] for a in d['candidates']]
        mask[i, :count] = True
        nodes[i, :regions] = d['graph']['nodes']
        node_mask[i, :regions] = 1
        targets[i, :count] = [(a['target'], a['source']) for a in d['candidates']]
        if (targets[i, :count] < -1).any() or (targets[i, :count] >= regions).any():
            raise ValueError('graph target out of range')
        for start, end in d['graph']['edges']:
            if not (0 <= start < regions and 0 <= end < regions):
                raise ValueError('graph edge out of range')
            adjacency[i, start, end] = 1
        adjacency[i] /= np.maximum(adjacency[i].sum(-1, keepdims=True), 1)
    return tuple(torch.from_numpy(v).to(device) for v in (states, candidates, mask, nodes, adjacency, node_mask, targets))


def merge_graph(batches):
    count = sum(b[0].shape[0] for b in batches)
    shapes = [(count,)+tuple(max(b[k].shape[j] for b in batches) for j in range(1, batches[0][k].ndim))
              for k in range(len(batches[0]))]
    output = [batches[0][k].new_full(shape, -1 if k in (6, 7) else 0) for k, shape in enumerate(shapes)]
    offset = 0
    for batch in batches:
        for target, source in zip(output, batch):
            target[(slice(offset, offset+source.shape[0]),)+tuple(slice(0,s) for s in source.shape[1:])] = source
        offset += batch[0].shape[0]
    return tuple(output)
