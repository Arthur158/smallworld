import json
import os
from pathlib import Path
import re

import numpy as np
import torch
from torch import nn
from torch.distributions import Categorical

from . import ACTION_FEATURES, SCHEMA, STATE_FEATURES


class Policy(nn.Module):
    """Score each legal candidate with shared weights; mask only batch padding."""
    schema = SCHEMA
    state_features, action_features = STATE_FEATURES, ACTION_FEATURES

    def __init__(self, hidden=128):
        super().__init__()
        if hidden < 8 or hidden > 256:
            raise ValueError("hidden size must be in [8, 256]")
        self.hidden = hidden
        self.actor = nn.Sequential(nn.Linear(STATE_FEATURES + ACTION_FEATURES, hidden),
                                   nn.ReLU(), nn.Linear(hidden, hidden), nn.ReLU(),
                                   nn.Linear(hidden, 1))
        self.critic = nn.Sequential(nn.Linear(STATE_FEATURES, hidden), nn.ReLU(),
                                    nn.Linear(hidden, hidden), nn.ReLU(), nn.Linear(hidden, 1))

    def forward(self, states, candidates, mask):
        expanded = states.unsqueeze(1).expand(-1, candidates.shape[1], -1)
        logits = self.actor(torch.cat((expanded, candidates), dim=-1)).squeeze(-1)
        logits = logits.masked_fill(~mask, -1e9)
        # Per-call distribution validation synchronizes CUDA with the host.
        # Packing checks shapes/actions; sampling and the PPO loss check reject
        # invalid numeric outputs without these redundant checks on every move.
        return Categorical(logits=logits, validate_args=False), self.critic(states).squeeze(-1)

    @torch.inference_mode()
    def choose(self, decision):
        states, candidates, _ = batch_decisions([decision], next(self.parameters()).device)
        expanded = states.unsqueeze(1).expand(-1, candidates.shape[1], -1)
        # Frozen opponents only need the best action, not a critic or softmax.
        scores = self.actor(torch.cat((expanded, candidates), dim=-1)).squeeze(-1)
        return int(scores.argmax(dim=-1).item())


def batch_decisions(decisions, device):
    if not decisions:
        raise ValueError("empty decision batch")
    if decisions[0]["schema"] == "smallworld-bot-v2":
        from .graph import batch_graph
        return batch_graph(decisions, device)
    max_actions = max(len(d["candidates"]) for d in decisions)
    if max_actions == 0:
        raise ValueError("terminal state cannot be sampled")
    states = np.empty((len(decisions), STATE_FEATURES), dtype=np.float32)
    candidates = np.zeros((len(decisions), max_actions, ACTION_FEATURES), dtype=np.float32)
    mask = np.zeros((len(decisions), max_actions), dtype=np.bool_)
    for i, d in enumerate(decisions):
        if d["schema"] != SCHEMA or len(d["state"]) != STATE_FEATURES or not d["candidates"]:
            raise ValueError("incompatible or terminal game decision")
        states[i] = d["state"]
        features = [a["features"] for a in d["candidates"]]
        if any(len(f) != ACTION_FEATURES for f in features):
            raise ValueError("incompatible action feature width")
        candidates[i, :len(features)] = features
        mask[i, :len(features)] = True
    # Three bulk tensor views/copies replace hundreds of tiny tensor allocations
    # and indexed assignments. CPU tensors keep their NumPy storage alive.
    return tuple(torch.from_numpy(value).to(device) for value in (states, candidates, mask))


def merge_decision_batches(batches):
    """Keep rollout order while padding cached device inputs once for PPO."""
    if not batches:
        raise ValueError("empty rollout")
    if len(batches[0]) == 7:
        from .graph import merge_graph
        return merge_graph(batches)
    count = sum(states.shape[0] for states, _, _ in batches)
    max_actions = max(candidates.shape[1] for _, candidates, _ in batches)
    states = torch.cat([batch[0] for batch in batches])
    candidates = batches[0][1].new_zeros((count, max_actions, ACTION_FEATURES))
    mask = batches[0][2].new_zeros((count, max_actions))
    offset = 0
    for state_batch, candidate_batch, mask_batch in batches:
        end = offset + state_batch.shape[0]
        width = candidate_batch.shape[1]
        candidates[offset:end, :width] = candidate_batch
        mask[offset:end, :width] = mask_batch
        offset = end
    return states, candidates, mask


def atomic_torch_save(value, path):
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    with temporary.open("wb") as stream:
        torch.save(value, stream)
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)


def load_checkpoint(path, device="cpu"):
    checkpoint = torch.load(path, map_location="cpu", weights_only=True)
    if checkpoint.get("schema") == "smallworld-bot-v2" and checkpoint.get("format_version") == 2:
        from .graph import GraphPolicy
        policy = GraphPolicy(checkpoint["config"]["hidden"])
    elif checkpoint.get("schema") == SCHEMA and checkpoint.get("format_version") == 1:
        policy = Policy(checkpoint["config"]["hidden"])
    else:
        raise ValueError("checkpoint schema is incompatible")
    policy.load_state_dict(checkpoint["model"], strict=True)
    policy.to(device)
    return policy, checkpoint


def export_policy(policy, path, model_id, name, description="Trained PPO policy"):
    if not re.fullmatch(r"[a-z0-9][a-z0-9_-]{0,63}", model_id) or model_id in ("random", "greedy"):
        raise ValueError("use a unique lowercase model id (letters, digits, _ and -; max 64)")
    if not name or len(name.encode()) > 80 or len(description.encode()) > 300:
        raise ValueError("model name/description exceeds server limits")
    layers = []
    for layer in policy.actor:
        if isinstance(layer, nn.Linear):
            layers.append({"weights": layer.weight.detach().cpu().tolist(),
                           "bias": layer.bias.detach().cpu().tolist()})
    payload = {"format_version": 2 if policy.schema != SCHEMA else 1, "schema": policy.schema, "id": model_id, "name": name,
               "description": description, "state_features": policy.state_features,
               "action_features": policy.action_features, "layers": layers}
    if policy.schema != SCHEMA:
        from .graph import N
        payload["node_features"] = N
        payload["graph_layers"] = [{"weights": layer.weight.detach().cpu().tolist(), "bias": layer.bias.detach().cpu().tolist()} for layer in policy.graph]
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(json.dumps(payload, separators=(",", ":"), allow_nan=False) + "\n")
    temporary.replace(path)
