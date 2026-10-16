"""A function-preserving expansion of the joint actor with relational reasoning.

Tiles attend to adjacent tiles twice and the whole board once. Player/tribe
context is supplied before attention, and residual connections retain tile
identity. No race, terrain or piece-specific tactical rules are embedded here.
"""
import math
import torch
from torch import nn
from torch.nn import functional as F
from torch.distributions import Categorical

from .graph import S, A, N
from .hierarchical import HierarchicalPolicy

WIDTH = 128
HEADS = 4


class RelationBlock(nn.Module):
    def __init__(self):
        super().__init__()
        self.norm1 = nn.LayerNorm(WIDTH)
        self.qkv = nn.Linear(WIDTH, 3*WIDTH)
        self.projection = nn.Linear(WIDTH, WIDTH)
        self.norm2 = nn.LayerNorm(WIDTH)
        self.ff1 = nn.Linear(WIDTH, 2*WIDTH)
        self.ff2 = nn.Linear(2*WIDTH, WIDTH)

    def forward(self, h, allowed, node_mask):
        b, n, _ = h.shape
        q, k, v = self.qkv(self.norm1(h)).chunk(3, -1)
        def heads(x):
            return x.reshape(b, n, HEADS, WIDTH//HEADS).transpose(1, 2)
        scores = heads(q) @ heads(k).transpose(-1, -2) / math.sqrt(WIDTH//HEADS)
        weights = scores.masked_fill(~allowed.unsqueeze(1), -1e9).softmax(-1)
        attended = (weights @ heads(v)).transpose(1, 2).reshape(b, n, WIDTH)
        h = h+self.projection(attended)
        h = h+self.ff2(F.relu(self.ff1(self.norm2(h))))
        return h*node_mask.unsqueeze(-1)


class RelationalPolicy(HierarchicalPolicy):
    def __init__(self, hidden=320):
        super().__init__(hidden, joint=True)
        self.format_version = 5
        self.architecture = 'relational-v1'
        self.relation_state = nn.Sequential(nn.Linear(S, 256), nn.ReLU(), nn.Linear(256, WIDTH), nn.ReLU())
        self.relation_node = nn.Linear(N, WIDTH)
        self.relations = nn.ModuleList([RelationBlock() for _ in range(3)])
        self.relation_actor = nn.Sequential(nn.Linear(A+5*WIDTH, 512), nn.ReLU(),
                                           nn.Linear(512, 256), nn.ReLU(), nn.Linear(256, 1))
        self.relation_critic = nn.Sequential(nn.Linear(3*WIDTH, 512), nn.ReLU(),
                                            nn.Linear(512, 256), nn.ReLU(), nn.Linear(256, 1))
        # Training-only prediction of real territory retention across opponents'
        # turns. Its target comes from observed ownership, never game rule tables.
        self.survival_head = nn.Sequential(nn.Linear(WIDTH, 64), nn.ReLU(), nn.Linear(64, 1))
        for head in (self.relation_actor[-1], self.relation_critic[-1]):
            nn.init.zeros_(head.weight)
            nn.init.zeros_(head.bias)

    def relational_board(self, states, nodes, adjacency, node_mask):
        context = self.relation_state(states)
        h = F.relu(self.relation_node(nodes)+context.unsqueeze(1))*node_mask.unsqueeze(-1)
        n = nodes.shape[1]
        valid = node_mask.bool()
        local = ((adjacency > 0) | torch.eye(n, dtype=torch.bool, device=nodes.device).unsqueeze(0)) & valid.unsqueeze(1)
        global_mask = valid.unsqueeze(1).expand(-1, n, -1)
        for i, block in enumerate(self.relations):
            h = block(h, local if i < 2 else global_mask, node_mask)
        mean = h.sum(1)/node_mask.sum(1, keepdim=True).clamp_min(1)
        maximum = h.masked_fill(~valid.unsqueeze(-1), -1e9).amax(1)
        return h, torch.cat((context, mean, maximum), -1)

    def relational_forward(self, states, candidates, mask, nodes, adjacency, node_mask, targets,
                           value=True, actions=None):
        original = super().joint_forward(states, candidates, mask, nodes, adjacency, node_mask, targets,
                                         value=value, auxiliary_actions=actions)
        distribution, values = original[:2]
        h, context = self.relational_board(states, nodes, adjacency, node_mask)
        survival = self.survival_head(h).squeeze(-1) if actions is not None else None
        board_embeddings = h
        h = torch.cat((torch.zeros_like(h[:, :1]), h), 1)
        first = self.relation_actor[0]
        shared = F.linear(context, first.weight[:, A+2*WIDTH:], first.bias)
        packed = self.training and torch.is_grad_enabled() and getattr(self, 'pack_candidates', True)
        if packed:
            rows, columns = mask.nonzero(as_tuple=True)
            indices = targets[rows, columns]+1
            local = torch.cat((candidates[rows, columns], h[rows, indices[:, 0]], h[rows, indices[:, 1]]), -1)
            shared = shared[rows]
        else:
            def gather(index):
                return h.gather(1, (index+1).unsqueeze(-1).expand(-1, -1, WIDTH))
            local = torch.cat((candidates, gather(targets[..., 0]), gather(targets[..., 1])), -1)
            shared = shared.unsqueeze(1)
        x = F.relu(shared+F.linear(local, first.weight[:, :A+2*WIDTH]))
        scores = self.relation_actor[4](F.relu(self.relation_actor[2](x))).squeeze(-1)
        if packed:
            residual = torch.zeros_like(distribution.logits)
            residual[rows, columns] = scores
        else:
            residual = scores
        result = (Categorical(logits=(distribution.logits+residual).masked_fill(~mask, -1e9), validate_args=False),
                  values+self.relation_critic(context).squeeze(-1) if value else None)
        return (*result, original[2], survival, board_embeddings) if actions is not None else result

    def forward(self, *batch):
        return self.relational_forward(*batch)

    def action_distribution(self, *batch):
        return self.relational_forward(*batch, value=False)[0]

    def training_forward(self, batch, actions):
        return self.relational_forward(*batch, actions=actions)

    def greedy_actions(self, distribution, batch):
        return distribution.logits.argmax(-1)

    def warm_start(self, old):
        if getattr(old, 'format_version', 0) != 4 or old.hidden != self.hidden:
            raise ValueError('relational upgrade requires a joint policy of the same hidden size')
        missing, unexpected = self.load_state_dict(old.state_dict(), strict=False)
        if unexpected or any(not name.startswith(('relation', 'survival_head')) for name in missing):
            raise ValueError('incompatible relational upgrade')


def restore_optimizer(optimizer, checkpoint, policy, old_policy=None):
    """Retain Adam moments by parameter identity when adding the new branch."""
    if old_policy is None:
        optimizer.load_state_dict(checkpoint['optimizer'])
        return
    old_state = checkpoint['optimizer']
    if len(old_state['param_groups']) != 1:
        raise ValueError('upgrade supports the trainer single Adam group')
    old_names = list(dict(old_policy.named_parameters()))
    old_ids = old_state['param_groups'][0]['params']
    if len(old_names) != len(old_ids):
        raise ValueError('optimizer parameter mismatch')
    moments = dict(zip(old_names, old_ids))
    state = optimizer.state_dict()
    for (name, parameter), new_id in zip(policy.named_parameters(), state['param_groups'][0]['params']):
        if name in moments and moments[name] in old_state['state']:
            saved = old_state['state'][moments[name]]
            if saved['exp_avg'].shape != parameter.shape:
                raise ValueError('optimizer moment shape changed')
            state['state'][new_id] = saved
    optimizer.load_state_dict(state)
