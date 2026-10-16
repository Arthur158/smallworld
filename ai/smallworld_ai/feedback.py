"""Observed consequences, used as learning targets and never as action rules."""
import torch
from torch import nn
from torch.nn import functional as F

from .graph import SPEC

# V2 tile layout: biome[10], feature[3], border[1], relative owner[8].
SELF_OWNER = 14
OWNER_COUNT = 8
STACK_START = SELF_OWNER+OWNER_COUNT+1+len(SPEC['races'])+2*len(SPEC['traits'])
STACK_END = STACK_START+len(SPEC['stacks'])


def response_labels(before, after):
    """Public board outcomes at the learner's next turn, in its perspective.

    These are future training targets, never observation features. No opponent
    identity, private score, move history or rule-derived threat score is used.
    """
    if before.get('schema') != 'smallworld-bot-v2':
        return None
    if before['player'] != after['player']:
        raise ValueError('response target has a different player perspective')
    old, new = before['graph']['nodes'], after['graph']['nodes']
    if len(old) != len(new):
        raise ValueError('response target crossed maps')
    owner, pieces, owner_before = [], [], []
    for a, b in zip(old, new):
        owner.append(max(range(OWNER_COUNT), key=lambda i: b[SELF_OWNER+i]))
        owner_before.append(max(range(OWNER_COUNT), key=lambda i: a[SELF_OWNER+i]))
        delta = sum(b[STACK_START:STACK_END])-sum(a[STACK_START:STACK_END])
        pieces.append(2 if delta > 1e-5 else 0 if delta < -1e-5 else 1)
    return dict(owner=owner, pieces=pieces, owner_before=owner_before)


class ResponseHead(nn.Module):
    """Training-only forecasts sharing the actor's current-position encoder."""
    def __init__(self, width):
        super().__init__()
        self.network = nn.Sequential(nn.Linear(width, 64), nn.ReLU(), nn.Linear(64, OWNER_COUNT+3))

    def forward(self, nodes, targets):
        scores = self.network(nodes)
        owner, pieces = scores[..., :OWNER_COUNT], scores[..., OWNER_COUNT:]
        valid = targets[..., 0] >= 0
        owner_target, pieces_target = targets[..., 0].clamp_min(0), targets[..., 1].clamp_min(0)
        owner_loss = F.cross_entropy(owner.transpose(1, 2), owner_target, reduction='none')
        pieces_loss = F.cross_entropy(pieces.transpose(1, 2), pieces_target, reduction='none')
        loss = ((owner_loss+pieces_loss)*valid).sum()/(2*valid.sum().clamp_min(1))
        # Counts, not minibatch averages: rare changed tiles remain visible and
        # are compared with the simple "nothing changes" diagnostic baseline.
        with torch.no_grad():
            changed = valid & (owner_target != targets[..., 2])
            pieces_changed = valid & (pieces_target != 1)
            correct = (owner.argmax(-1) == owner_target) & valid
            piece_correct = (pieces.argmax(-1) == pieces_target) & valid
            counts = torch.stack((valid.sum(), correct.sum(), changed.sum(), (correct & changed).sum(),
                                  piece_correct.sum(), pieces_changed.sum(), (piece_correct & pieces_changed).sum()))
        return loss, counts


def rejection_loss(logits, actions, accepted, mask):
    """Teach the policy itself from observed engine rejections.

    Penalize probability of the exact rejected stack/destination/context, not
    an entire piece type. Successful moves receive no tactical preference.
    This is -log(1-p(rejected_action)) under the base policy, before training
    exploration/retry conditioning. Log-sum-exp avoids saturation near p=1.
    There is no rule lookup, prediction-based mask or change to game rewards.
    """
    import torch
    alternatives = mask.scatter(1, actions[:, None], False)
    usable = (accepted < .5) & alternatives.any(-1)
    original = logits.masked_fill(~mask, -torch.inf)
    # A rejected singleton has no alternative to prefer. Reuse its original
    # support to obtain exactly zero loss and avoid log(0) and NaN gradients.
    remaining = logits.masked_fill(~alternatives, -torch.inf)
    remaining = torch.where(usable[:, None], remaining, original)
    return (original.logsumexp(-1)-remaining.logsumexp(-1)).mean()


def survival_labels(before, after):
    """Did each previously owned tile remain ours after opponents played?

    Graph node order is the engine's sorted stable tile IDs within a game.
    Call before resetting a finished game; terminal states have no next round.
    Labels for unowned tiles are masked, so this is not an expansion heuristic.
    """
    if before.get('schema') != 'smallworld-bot-v2':
        return None
    if before['player'] != after['player']:
        raise ValueError('survival target has a different player perspective')
    old, new = before['graph']['nodes'], after['graph']['nodes']
    if len(old) != len(new):
        raise ValueError('survival target crossed maps')
    return [float(b[SELF_OWNER] > .5) if a[SELF_OWNER] > .5 else -1. for a, b in zip(old, new)]
