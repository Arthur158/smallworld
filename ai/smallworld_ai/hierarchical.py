"""Learn joint stack-and-tile scores, with a legacy conditional variant.

Every action kind and piece type uses the same learned networks. There are no
piece bonuses, move values, rule-derived tactical features or scripted choices.
The observation contract is still smallworld-bot-v2; export format 3 describes
the conditional variant; format 4 scores complete joint actions.
"""
import torch
from torch import nn
from torch.nn import functional as F
from torch.distributions import Categorical

from .graph import GraphPolicy, S, A, N, SPEC
from . import ACTION_FEATURES

WIDTH = 96
KINDS = len(SPEC['action_kinds'])
STACKS = len(SPEC['stacks'])
GROUP_FEATURES = KINDS + STACKS
GROUPS = KINDS * (STACKS + 1)


class HierarchicalPolicy(GraphPolicy):
    architecture = 'hierarchical-v1'

    def __init__(self, hidden=320, joint=False):
        nn.Module.__init__(self)
        if not 32 <= hidden <= 384:
            raise ValueError('hierarchical hidden size must be in [32, 384]')
        self.hidden = hidden
        self.format_version = 4 if joint else 3
        self.architecture = "joint-v1" if joint else "hierarchical-v1"
        self.graph = nn.ModuleList([nn.Linear(N, WIDTH)] + [nn.Linear(2*WIDTH, WIDTH) for _ in range(4)])
        self.actor = nn.Sequential(nn.Linear(S+A+4*WIDTH, hidden), nn.ReLU(),
                                   nn.Linear(hidden, hidden), nn.ReLU(), nn.Linear(hidden, 1))
        self.critic = nn.Sequential(nn.Linear(S+2*WIDTH, hidden), nn.ReLU(),
                                    nn.Linear(hidden, hidden), nn.ReLU(), nn.Linear(hidden, 1))
        self.group_actor = nn.Sequential(nn.Linear(S+2*WIDTH+GROUP_FEATURES, 128), nn.ReLU(), nn.Linear(128, 1))
        # Start with equal opportunity for every available action/piece group.
        # Target preferences are transferred from the old policy separately.
        nn.init.zeros_(self.group_actor[-1].weight)
        nn.init.zeros_(self.group_actor[-1].bias)

    def encode_board(self, nodes, adjacency, node_mask):
        h = F.relu(self.graph[0](nodes)) * node_mask.unsqueeze(-1)
        for layer in self.graph[1:]:
            h = F.relu(layer(torch.cat((h, adjacency.bmm(h)), -1))) * node_mask.unsqueeze(-1)
        mean = h.sum(1) / node_mask.sum(1, keepdim=True).clamp_min(1)
        maximum = h.masked_fill(~node_mask.bool().unsqueeze(-1), -1e9).amax(1)
        return h, torch.cat((mean, maximum), -1)

    def inputs(self, states, candidates, mask, nodes, adjacency, node_mask, targets):
        h, pooled = self.encode_board(nodes, adjacency, node_mask)
        h = torch.cat((torch.zeros_like(h[:, :1]), h), 1)
        def gather(index):
            return h.gather(1, (index+1).unsqueeze(-1).expand(-1, -1, WIDTH))
        local = torch.cat((candidates, gather(targets[..., 0]), gather(targets[..., 1]),
                           pooled.unsqueeze(1).expand(-1, candidates.shape[1], -1)), -1)
        return local, pooled

    @staticmethod
    def group_features(candidates):
        return candidates[..., ACTION_FEATURES:ACTION_FEATURES+GROUP_FEATURES]

    @staticmethod
    def group_ids(features):
        kind = features[..., :KINDS].argmax(-1)
        stack = features[..., KINDS:]
        stack_id = (stack.argmax(-1)+1) * (stack.sum(-1)>0).long()
        return kind*(STACKS+1)+stack_id

    def joint_forward(self, states, candidates, mask, nodes, adjacency, node_mask, targets, value=True, auxiliary_actions=None):
        h, pooled = self.encode_board(nodes, adjacency, node_mask)
        h = torch.cat((torch.zeros_like(h[:, :1]), h), 1)
        first = self.actor[0]
        # Board and player context are identical for every candidate in a state.
        # Share their projection, while preserving the existing weight layout.
        shared = (F.linear(states, first.weight[:, :S], first.bias)
                  + F.linear(pooled, first.weight[:, S+A+2*WIDTH:]))
        context = torch.cat((states, pooled), -1)
        group = self.group_actor[0]
        group_shared = F.linear(context, group.weight[:, :S+2*WIDTH], group.bias)
        # PPO batches contain very different candidate counts. Score only real
        # candidates during updates; inference stays dense to avoid a CUDA
        # nonzero synchronization on each small game decision.
        packed = self.training and torch.is_grad_enabled() and getattr(self, 'pack_candidates', True)
        if packed:
            rows, columns = mask.nonzero(as_tuple=True)
            choices = candidates[rows, columns]
            indices = targets[rows, columns]+1
            local = torch.cat((choices, h[rows, indices[:, 0]], h[rows, indices[:, 1]]), -1)
            shared, group_shared = shared[rows], group_shared[rows]
        else:
            choices = candidates
            def gather(index):
                return h.gather(1, (index+1).unsqueeze(-1).expand(-1, -1, WIDTH))
            local = torch.cat((choices, gather(targets[..., 0]), gather(targets[..., 1])), -1)
            shared, group_shared = shared.unsqueeze(1), group_shared.unsqueeze(1)
        x = F.relu(shared+F.linear(local, first.weight[:, S:S+A+2*WIDTH]))
        action_hidden = F.relu(self.actor[2](x))
        target_logits = self.actor[4](action_hidden).squeeze(-1)
        group_hidden = F.relu(group_shared+F.linear(self.group_features(choices), group.weight[:, S+2*WIDTH:]))
        scores = target_logits+self.group_actor[2](group_hidden).squeeze(-1)
        if packed:
            logits = scores.new_full(mask.shape, -1e9)
            logits[rows, columns] = scores
        else:
            logits = scores.masked_fill(~mask, -1e9)
        result = (Categorical(logits=logits, validate_args=False), self.critic(context).squeeze(-1) if value else None)
        if auxiliary_actions is not None:
            rows = torch.arange(mask.shape[0], device=mask.device)
            if packed:
                counts = mask.sum(-1)
                offset = counts.cumsum(0)-counts
                position = mask.long().cumsum(-1)[rows, auxiliary_actions]-1+offset
                selected_hidden = action_hidden[position]
            else:
                selected_hidden = action_hidden[rows, auxiliary_actions]
            return (*result, selected_hidden)
        return result

    def training_forward(self, batch, actions):
        if self.format_version != 4:
            raise ValueError('acceptance learning requires a joint policy')
        return self.joint_forward(*batch, auxiliary_actions=actions)

    def action_distribution(self, *batch):
        if self.format_version == 4:
            return self.joint_forward(*batch, value=False)[0]
        return self(*batch)[0]

    def forward(self, states, candidates, mask, nodes, adjacency, node_mask, targets):
        if self.format_version == 4:
            return self.joint_forward(states, candidates, mask, nodes, adjacency, node_mask, targets)
        local, pooled = self.inputs(states, candidates, mask, nodes, adjacency, node_mask, targets)
        first = self.actor[0]
        x = F.relu(F.linear(states, first.weight[:, :S], first.bias).unsqueeze(1) + F.linear(local, first.weight[:, S:]))
        target_logits = self.actor[4](F.relu(self.actor[2](x))).squeeze(-1)
        features = self.group_features(candidates)
        group_ids = self.group_ids(features)
        context = torch.cat((states, pooled), -1)
        first = self.group_actor[0]
        group_hidden = F.relu(F.linear(context, first.weight[:, :S+2*WIDTH], first.bias).unsqueeze(1) + F.linear(features, first.weight[:, S+2*WIDTH:]))
        group_logits = self.group_actor[2](group_hidden).squeeze(-1)
        # Conditional target probabilities sum to one within each available
        # action/piece group. Many targets cannot swamp a less common piece.
        maxima = target_logits.new_full((states.shape[0], GROUPS), -1e9)
        maxima.scatter_reduce_(1, group_ids, target_logits.detach().masked_fill(~mask, -1e9), reduce='amax', include_self=True)
        shifted = (target_logits-maxima.gather(1, group_ids)).masked_fill(~mask, -1e9)
        totals = target_logits.new_zeros((states.shape[0], GROUPS))
        totals.scatter_add_(1, group_ids, shifted.exp()*mask)
        target_logprobs = shifted-totals.gather(1, group_ids).clamp_min(1e-30).log()
        logits = (target_logprobs+group_logits).masked_fill(~mask, -1e9)
        values = self.critic(context).squeeze(-1)
        return Categorical(logits=logits, validate_args=False), values

    def group_entropy(self, distribution, candidates):
        groups = self.group_ids(self.group_features(candidates))
        mass = distribution.probs.new_zeros((groups.shape[0], GROUPS))
        mass.scatter_add_(1, groups, distribution.probs)
        return -(mass*mass.clamp_min(1e-30).log()).sum(-1)

    def greedy_actions(self, distribution, batch):
        if self.format_version == 4:
            return distribution.logits.argmax(-1)
        # Decode the learned hierarchy in the same order used to define it.
        # A piece with many possible tiles must not lose to a singleton action
        # merely because its probability is spread across those destinations.
        groups = self.group_ids(self.group_features(batch[1]))
        mass = distribution.probs.new_zeros((groups.shape[0], GROUPS))
        mass.scatter_add_(1, groups, distribution.probs)
        selected = mass.argmax(-1, keepdim=True)
        return distribution.logits.masked_fill(groups != selected, -1e9).argmax(-1)

    def warm_start(self, old):
        if old.schema != self.schema or hasattr(old, 'group_actor') or old.hidden > self.hidden:
            raise ValueError('hierarchical transfer needs a v2 graph policy no wider than the new model')
        h, g = old.hidden, old.graph[0].out_features
        with torch.no_grad():
            for i, layer in enumerate(self.graph):
                layer.weight.zero_(); layer.bias.zero_()
                if i < len(old.graph):
                    layer.bias[:g].copy_(old.graph[i].bias)
                    if i == 0:
                        layer.weight[:g].copy_(old.graph[i].weight)
                    else:
                        layer.weight[:g, :g].copy_(old.graph[i].weight[:, :g])
                        layer.weight[:g, WIDTH:WIDTH+g].copy_(old.graph[i].weight[:, g:])
                else:
                    layer.weight[:, :WIDTH].copy_(torch.eye(WIDTH, device=layer.weight.device))
                if i < len(old.graph):
                    nn.init.kaiming_uniform_(layer.weight[g:], a=5**.5)
            for network, teacher in ((self.actor, old.actor), (self.critic, old.critic)):
                for layer in (network[0], network[2], network[4]):
                    layer.weight.zero_(); layer.bias.zero_()
                network[0].bias[:h].copy_(teacher[0].bias)
                prefix = S+A if network is self.actor else S
                network[0].weight[:h, :prefix].copy_(teacher[0].weight[:, :prefix])
                blocks = 3 if network is self.actor else 1
                for i in range(blocks):
                    network[0].weight[:h, prefix+i*WIDTH:prefix+i*WIDTH+g].copy_(teacher[0].weight[:, prefix+i*g:prefix+(i+1)*g])
                network[2].weight[:h, :h].copy_(teacher[2].weight)
                network[2].bias[:h].copy_(teacher[2].bias)
                network[4].weight[:, :h].copy_(teacher[4].weight)
                network[4].bias.copy_(teacher[4].bias)
                if self.hidden > h:
                    nn.init.kaiming_uniform_(network[0].weight[h:], a=5**.5)
                    nn.init.kaiming_uniform_(network[2].weight[h:], a=5**.5)
