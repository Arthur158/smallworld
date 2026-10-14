import json
import os
from pathlib import Path
import re

import torch
from torch import nn
from torch.distributions import Categorical

from . import ACTION_FEATURES, SCHEMA, STATE_FEATURES


class Policy(nn.Module):
    """Score each legal candidate with shared weights; mask only batch padding."""
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
        return Categorical(logits=logits), self.critic(states).squeeze(-1)

    @torch.inference_mode()
    def choose(self, decision):
        batch = batch_decisions([decision], next(self.parameters()).device)
        distribution, _ = self(*batch)
        return int(distribution.logits.argmax(dim=-1).item())


def batch_decisions(decisions, device):
    if not decisions:
        raise ValueError("empty decision batch")
    max_actions = max(len(d["candidates"]) for d in decisions)
    if max_actions == 0:
        raise ValueError("terminal state cannot be sampled")
    states = torch.empty((len(decisions), STATE_FEATURES), dtype=torch.float32)
    candidates = torch.zeros((len(decisions), max_actions, ACTION_FEATURES), dtype=torch.float32)
    mask = torch.zeros((len(decisions), max_actions), dtype=torch.bool)
    for i, d in enumerate(decisions):
        if d["schema"] != SCHEMA or len(d["state"]) != STATE_FEATURES or not d["candidates"]:
            raise ValueError("incompatible or terminal game decision")
        states[i] = torch.tensor(d["state"])
        features = [a["features"] for a in d["candidates"]]
        if any(len(f) != ACTION_FEATURES for f in features):
            raise ValueError("incompatible action feature width")
        candidates[i, :len(features)] = torch.tensor(features)
        mask[i, :len(features)] = True
    return states.to(device), candidates.to(device), mask.to(device)


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
    if checkpoint.get("schema") != SCHEMA or checkpoint.get("format_version") != 1:
        raise ValueError("checkpoint schema is incompatible")
    policy = Policy(checkpoint["config"]["hidden"])
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
    payload = {"format_version": 1, "schema": SCHEMA, "id": model_id, "name": name,
               "description": description, "state_features": STATE_FEATURES,
               "action_features": ACTION_FEATURES, "layers": layers}
    path = Path(path)
    path.parent.mkdir(parents=True, exist_ok=True)
    temporary = path.with_suffix(path.suffix + ".tmp")
    temporary.write_text(json.dumps(payload, separators=(",", ":"), allow_nan=False) + "\n")
    temporary.replace(path)
