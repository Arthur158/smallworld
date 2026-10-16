"""Persistent CPU inference over private stdin/stdout pipes; JSON weights only.

The worker owns the game and selects an action from these scores. Models load
once before readiness. No checkpoint/pickle loading, HTTP listener or GPU use.
"""
import argparse
import hashlib
import json
from pathlib import Path
import sys
import time

import torch
from torch import nn

from .model import Policy, batch_decisions
from .graph import GraphPolicy, N
from .hierarchical import HierarchicalPolicy
from .relational import RelationalPolicy
from .entity import EntityPolicy

MAX_MODEL_BYTES = 384 << 20
MAX_REQUEST_BYTES = 32 << 20


def load_export(path):
    path = Path(path)
    if path.stat().st_size > MAX_MODEL_BYTES:
        raise ValueError('model exceeds 384 MiB')
    raw = path.read_bytes()
    digest = hashlib.sha256(raw).hexdigest()
    data = json.loads(raw)
    del raw
    version = data['format_version']
    hidden = len(data['layers'][0]['bias'])
    if version == 1:
        model = Policy(hidden)
    elif version == 2:
        model = GraphPolicy(hidden)
    elif version in (3, 4):
        model = HierarchicalPolicy(hidden, joint=version == 4)
    elif version == 5:
        model = RelationalPolicy(hidden)
    elif version == 7:
        from .specialists import SpecialistPolicy
        model = SpecialistPolicy(hidden)
    elif version in (6, 7):
        model = EntityPolicy(hidden)
    else:
        raise ValueError('unsupported exported model format')
    if (data['schema'], data['state_features'], data['action_features']) != (model.schema, model.state_features, model.action_features):
        raise ValueError('export schema mismatch')
    if version >= 2 and data['node_features'] != N:
        raise ValueError('node schema mismatch')

    def tensor(target, values):
        value = torch.tensor(values, dtype=target.dtype)
        if value.shape != target.shape or not torch.isfinite(value).all() or (value.abs() > 1e6).any():
            raise ValueError('invalid exported tensor')
        target.copy_(value)

    def layer(target, source):
        tensor(target.weight, source['weights'])
        tensor(target.bias, source['bias'])

    def network(target, source):
        layers = [m for m in target if isinstance(m, nn.Linear)]
        if len(layers) != len(source):
            raise ValueError('invalid exported layer count')
        for dst, src in zip(layers, source):
            layer(dst, src)

    with torch.no_grad():
        network(model.actor, data['layers'])
        if version >= 2:
            network(model.graph, data['graph_layers'])
        if version >= 3:
            network(model.group_actor, data['group_layers'])
        if version >= 5:
            rel = data['relational']
            network(model.relation_state, rel['state'])
            layer(model.relation_node, rel['node'])
            network(model.relation_actor, rel['actor'])
            if len(rel['blocks']) != len(model.relations):
                raise ValueError('invalid relational block count')
            for dst, src in zip(model.relations, rel['blocks']):
                for name in ('norm1', 'qkv', 'projection', 'norm2', 'ff1', 'ff2'):
                    layer(getattr(dst, name), src[name])
        if version in (6, 7):
            ent = data['entity']
            for name in ('state', 'tile', 'player', 'stack', 'action'):
                layer(getattr(model.entity, name), ent[name])
            tensor(model.entity.stack_id.weight, ent['stack_id'])
            network(model.entity.actor, ent['actor'])
            if len(ent['blocks']) != len(model.entity.blocks):
                raise ValueError('invalid entity block count')
            for dst, src in zip(model.entity.blocks, ent['blocks']):
                for name in ('norm1', 'norm2', 'qkv', 'out'):
                    layer(getattr(dst, name), src[name])
                network(dst.ff, src['ff'])
                tensor(dst.edge_bias, src['edge_bias'])
        if version == 7:
            from .specialists import RACES, TRAITS
            if data.get('specialist_races') != list(RACES) or data.get('specialist_traits') != list(TRAITS):
                raise ValueError('specialist vocabulary mismatch')
            expected = {name: value for name, value in model.state_dict().items() if name.startswith('specialist_')}
            if set(data.get('specialists', {})) != set(expected):
                raise ValueError('invalid specialist tensor names')
            for name, target in expected.items():
                tensor(target, data['specialists'][name])
    model.eval().requires_grad_(False)
    return data['id'], model, digest


@torch.inference_mode()
def scores(model, decision):
    if decision['schema'] != model.schema:
        raise ValueError('decision schema mismatch')
    batch = batch_decisions([decision], 'cpu', model)
    if hasattr(model, 'action_distribution'):
        distribution = model.action_distribution(*batch)
    else:
        distribution = model(*batch)[0]
    logits = distribution.logits[0]
    if not torch.isfinite(logits).all():
        raise ValueError('non-finite policy output')
    return logits.tolist()


def write(value):
    print(json.dumps(value, separators=(',', ':'), allow_nan=False), flush=True)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--model-dir', type=Path, required=True)
    parser.add_argument('--threads', type=int, default=1)
    args = parser.parse_args()
    if not 1 <= args.threads <= 16:
        parser.error('--threads must be between 1 and 16')
    torch.set_num_threads(args.threads)
    torch.set_num_interop_threads(1)
    started = time.monotonic()
    models, manifest = {}, {}
    for path in sorted(args.model_dir.glob('*.json')):
        identifier, model, digest = load_export(path)
        if identifier in models or identifier in ('random', 'greedy'):
            raise ValueError('duplicate or reserved model ID')
        models[identifier], manifest[identifier] = model, digest
    print(f'CPU evaluator: loaded {len(models)} models in {time.monotonic()-started:.2f}s; torch={torch.__version__}; threads={args.threads}', file=sys.stderr, flush=True)
    write({'ready': True, 'protocol': 1, 'models': manifest})
    while True:
        raw = sys.stdin.buffer.readline(MAX_REQUEST_BYTES+1)
        if not raw:
            break
        if len(raw) > MAX_REQUEST_BYTES or not raw.endswith(b'\n'):
            raise ValueError('oversized or incomplete evaluator request')
        request = {}
        try:
            request = json.loads(raw)
            result = scores(models[request['model']], request['decision'])
            write({'id': request['id'], 'scores': result})
        except (ValueError, KeyError, TypeError, IndexError, RuntimeError) as error:
            write({'id': request.get('id', 0), 'error': str(error)})


if __name__ == '__main__':
    main()
