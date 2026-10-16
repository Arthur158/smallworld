"""Training-only expert credit assignment and measurable contribution diagnostics.

Targets are rollout advantages from real games. No piece/race rules, preferred
actions, or extra inference inputs are supplied. The saved format-7 actor is
unchanged; these credit predictors are not exported or used to choose moves.
"""
import copy
import math
import torch
from torch import nn
from torch.nn import functional as F

from . import ACTION_FEATURES
from .graph import SPEC
from .specialists import RACES, TRAITS, EXPERT_WIDTH


def optimizer_for(policy, args, checkpoint=None, old_policy=None):
    """Keep Adam moments by name across old single-group and new grouped saves."""
    rate = getattr(args, 'specialist_learning_rate', None)
    eps = getattr(args, 'specialist_adam_eps', 1e-5)
    if rate is not None and (not math.isfinite(rate) or rate <= 0 or policy.format_version != 7):
        raise ValueError('specialist learning rate requires a positive finite rate and format 7')
    if not math.isfinite(eps) or eps <= 0:
        raise ValueError('specialist Adam epsilon must be positive and finite')
    groups = {}
    for name, parameter in policy.named_parameters():
        expert = name.startswith(('specialist_races.', 'specialist_traits.'))
        label = 'experts' if expert else 'specialist_shared' if name.startswith('specialist_') else 'inherited'
        group = groups.setdefault(label, dict(params=[], param_names=[], name=label,
                                             lr=rate if expert and rate is not None else args.learning_rate,
                                             eps=eps if expert else 1e-5))
        group['params'].append(parameter)
        group['param_names'].append(name)
    optimizer = torch.optim.Adam(list(groups.values()), lr=args.learning_rate, eps=1e-5)
    if not checkpoint:
        return optimizer
    previous = checkpoint['optimizer']
    named = {}
    if all('param_names' in group for group in previous['param_groups']):
        for group in previous['param_groups']:
            if len(group['param_names']) != len(group['params']):
                raise ValueError('saved optimizer name count differs')
            for name, index in zip(group['param_names'], group['params']):
                if name in named:
                    raise ValueError('duplicate saved optimizer parameter')
                named[name] = index
    else:
        if len(previous['param_groups']) != 1:
            raise ValueError('unnamed optimizer must have one parameter group')
        names = list(dict((old_policy or policy).named_parameters()))
        indices = previous['param_groups'][0]['params']
        if len(names) != len(indices):
            raise ValueError('legacy optimizer parameter count differs')
        named = dict(zip(names, indices))
    saved = optimizer.state_dict()
    parameters = dict(policy.named_parameters())
    for group in saved['param_groups']:
        for name, index in zip(group['param_names'], group['params']):
            moment = previous['state'].get(named.get(name))
            if moment is not None:
                if moment['exp_avg'].shape != parameters[name].shape:
                    raise ValueError('optimizer parameter shape differs: '+name)
                saved['state'][index] = copy.deepcopy(moment)
    optimizer.load_state_dict(saved)
    return optimizer


class ExpertCredit(nn.Module):
    def __init__(self):
        super().__init__()
        # Nonzero readouts provide immediate gradients to expert embeddings,
        # bypassing the near-zero actor coordinator's gradient bottleneck.
        self.races = nn.ModuleList([nn.Linear(EXPERT_WIDTH, 1) for _ in RACES])
        self.traits = nn.ModuleList([nn.Linear(EXPERT_WIDTH, 1) for _ in TRAITS])

    def forward(self, policy, features, batch, actions, advantage):
        rows = torch.arange(actions.shape[0], device=actions.device)
        race_ids, trait_ids = batch[7][rows, actions], batch[8][rows, actions]
        market = batch[1][rows, actions, ACTION_FEATURES+SPEC['action_kinds'].index('pick')] > 0
        total = features.sum()*0
        count = 0
        for index in race_ids[~market].unique().tolist():
            if index < 0:
                continue
            selected = (race_ids == index) & ~market
            prediction = self.races[index](policy.specialist_races[index](features[selected])).squeeze(-1)
            total = total+F.smooth_l1_loss(prediction, advantage[selected].detach(), reduction='sum')
            count += selected.sum()
        for index in (trait_ids[~market].sum(0) > 0).nonzero().flatten().tolist():
            selected = (trait_ids[:, index] > 0) & ~market
            prediction = self.traits[index](policy.specialist_traits[index](features[selected])).squeeze(-1)
            total = total+F.smooth_l1_loss(prediction, advantage[selected].detach(), reduction='sum')
            count += selected.sum()
        return total/torch.as_tensor(count, device=features.device).clamp_min(1)


@torch.no_grad()
def contribution_metrics(policy, batch, samples=128):
    """Ablate actual race/trait inputs, not just the combined new branch."""
    size = batch[0].shape[0]
    indices = torch.linspace(0, size-1, min(samples, size), device=batch[0].device).long()
    counters = {key: 0. for key in ('race', 'trait', 'general', 'all')}
    divergences = counters.copy()
    was_training = policy.training
    policy.eval()
    try:
        for chosen in indices.split(8):
            data = tuple(value.index_select(0, chosen) for value in batch)
            distribution = policy.action_distribution(*data)
            decisions = distribution.logits.argmax(-1)
            for component in counters:
                ablated = policy.specialist_forward(data, value=False, ablate=component)[0]
                counters[component] += float((decisions != ablated.logits.argmax(-1)).sum())
                divergences[component] += float((distribution.probs*(distribution.logits-ablated.logits)).sum())
    finally:
        policy.train(was_training)
    result = {'specialist_probe_decisions': len(indices)}
    for key in counters:
        result['specialist_'+key+'_choice_change'] = counters[key]/len(indices)
        result['specialist_'+key+'_kl'] = max(0., divergences[key]/len(indices))
    return result
