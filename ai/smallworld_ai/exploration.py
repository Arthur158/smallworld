"""Generic piece-use curiosity and persistent exploration for training only.

Counts measure familiarity, not tactical quality or a calibrated uncertainty.
Every stack name follows the same rule; nothing encodes what a dragon does.
"""
from collections import Counter
from math import sqrt
from threading import Lock


class PieceExploration:
    def __init__(self, bonus, budget, step=0, state=None, successful=None):
        if bonus < 0 or budget < 0:
            raise ValueError('invalid exploration bonus or budget')
        self.bonus, self.budget = bonus, budget
        self.lock = Lock()
        state = state or {}
        self.step = step
        self.counts = Counter(state.get('counts', {}))
        if not state:
            # Respect experience already present in the resumed checkpoint.
            for key, count in (successful or {}).items():
                _, _, stack = key.partition(':')
                if stack:
                    self.counts[stack] += count
        self.rewarded_actions = state.get('rewarded_actions', 0)
        self.total_reward = state.get('total_reward', 0.0)

    def reward(self, action, rejected, seen, earned):
        with self.lock:
            self.step += 1
            stack = action.get('stack', '')
            if rejected or not stack:
                return 0.0
            count = self.counts[stack]
            self.counts[stack] += 1
            # One bonus per piece type per game, regardless of action kind or
            # tile. Moving it back and forth cannot collect repeated bonuses.
            if stack in seen:
                return 0.0
            seen.add(stack)
            amount = min(max(0.0, self.budget-earned), self.bonus/sqrt(1+count/100))
            self.total_reward += amount
            self.rewarded_actions += int(amount > 0)
            return amount

    def state_dict(self):
        with self.lock:
            return dict(step=self.step, counts=dict(self.counts),
                        rewarded_actions=self.rewarded_actions, total_reward=self.total_reward)

    def metrics(self):
        state = self.state_dict()
        return dict(exploration_reward=round(state['total_reward'], 6),
                    exploration_rewarded_actions=state['rewarded_actions'])


def training_distribution(policy, distribution, batch, fraction, temperature=1.5, retry_groups=None,
                          learned_groups=.75, piece_prior=2.):
    """A persistent mixture, used consistently for collection and PPO updates.

    Exploration mixes softened learned group scores with a persistent group
    floor. Unlike fixed uniform group selection, its likelihood has a gradient
    for choosing the piece itself, so successful experiments can teach the
    exported policy to select that piece. A generic training-only prior favors
    actions carrying a stack. It says nothing about any named piece or move.
    Destination scores remain joint stack-and-tile scores; rejected attempts
    retry the same group on another available destination.
    """
    if not fraction:
        return distribution
    if not 0 <= fraction <= 1 or temperature < 1 or not 0 <= learned_groups < 1 or piece_prior < 1:
        raise ValueError('invalid exploration mixture or temperature')
    import torch
    from torch.distributions import Categorical
    mask = batch[2]
    if policy.schema == 'smallworld-bot-v2':
        from .hierarchical import HierarchicalPolicy, GROUPS
        groups = HierarchicalPolicy.group_ids(HierarchicalPolicy.group_features(batch[1]))
        scores = distribution.logits/temperature
        maxima = scores.new_full((scores.shape[0], GROUPS), -1e9)
        maxima.scatter_reduce_(1, groups, scores.detach().masked_fill(~mask, -1e9), reduce='amax', include_self=True)
        weights = (scores-maxima.gather(1, groups)).masked_fill(~mask, -1e9).exp()*mask
        totals = scores.new_zeros((scores.shape[0], GROUPS))
        totals.scatter_add_(1, groups, weights)
        counts = scores.new_zeros(totals.shape)
        counts.scatter_add_(1, groups, mask.to(scores.dtype))
        present = counts > 0
        # Mean, not sum: a piece with many destinations gets no accidental
        # advantage over a piece with just one. Group scores still depend on
        # the joint network, including the proposed tiles and their owners.
        group_scores = maxima + totals.clamp_min(1e-30).log() - counts.clamp_min(1).log()
        from .hierarchical import STACKS
        has_stack = torch.arange(GROUPS, device=scores.device) % (STACKS+1) != 0
        prior = torch.where(has_stack, piece_prior, 1.).to(scores.dtype)[None, :]*present
        floor = prior/prior.sum(-1, keepdim=True).clamp_min(1e-30)
        learned = (group_scores + prior.clamp_min(1e-30).log()).masked_fill(~present, -1e9).softmax(-1)
        group_mass = learned_groups*learned + (1-learned_groups)*floor
        exploratory = weights/totals.gather(1, groups).clamp_min(1e-30)*group_mass.gather(1, groups)
    else:
        exploratory = mask/mask.sum(-1, keepdim=True)
    probabilities = ((1-fraction)*distribution.probs+fraction*exploratory).masked_fill(~mask, 0)
    if retry_groups is not None and policy.schema == 'smallworld-bot-v2':
        # A failed exploratory move keeps its action/piece group. The real
        # engine removes rejected candidates, so another destination is tried.
        selected = (groups == retry_groups[:, None]) & mask & (retry_groups[:, None] >= 0)
        active = selected.any(-1, keepdim=True)
        retry_weights = exploratory*selected
        retry_probs = retry_weights/retry_weights.sum(-1, keepdim=True).clamp_min(1e-30)
        # The exploration coin is independent of model weights. Conditioning
        # both collection and PPO on it gives the correct behavior likelihood.
        fresh = torch.where((retry_groups == -2)[:, None], exploratory, distribution.probs)
        probabilities = torch.where(active, retry_probs, fresh)
    return Categorical(probs=probabilities, validate_args=False)


def sample_training_actions(policy, base_distribution, batch, fraction, temperature=1.5, retry_groups=None,
                            learned_groups=.75, piece_prior=2.):
    """Choose a training exploration mode, then sample its exact distribution.

    Context -1 uses the learned policy, -2 explores action/piece groups, and a
    nonnegative context retries that group. The independent exploration coin
    and retry context are recorded, so PPO uses the same conditional likelihood
    as collection. These modes never change exported inference.
    """
    import torch
    count = batch[0].shape[0]
    context = torch.full((count,), -1, dtype=torch.long, device=batch[0].device)
    if fraction and policy.schema == 'smallworld-bot-v2':
        from .hierarchical import HierarchicalPolicy
        groups = HierarchicalPolicy.group_ids(HierarchicalPolicy.group_features(batch[1]))
        context = torch.where(torch.rand(count, device=context.device) < fraction, -2, context)
        if retry_groups is not None:
            active = ((groups == retry_groups[:, None]) & batch[2] & (retry_groups[:, None] >= 0)).any(-1)
            context = torch.where(active, retry_groups, context)
    distribution = training_distribution(policy, base_distribution, batch, fraction, temperature, context, learned_groups, piece_prior)
    actions = distribution.sample()
    retry_after_rejection = torch.full_like(actions, -1)
    if fraction and policy.schema == 'smallworld-bot-v2':
        retry_after_rejection = torch.where(context != -1, groups.gather(1,actions[:, None]).squeeze(1), retry_after_rejection)
    return actions, distribution.log_prob(actions), retry_after_rejection, context
