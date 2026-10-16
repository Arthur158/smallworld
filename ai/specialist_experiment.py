#!/usr/bin/env python3
"""Run isolated, equal-budget baseline, curriculum, and specialist experiments.

No service changes, remote installation, model-catalog writes or deployment.
All arms start from the same immutable format-6 checkpoint and frozen rival
weights. Training counters differ from held-out strength measurements.
"""
import argparse
import hashlib
import json
import os
from pathlib import Path
import shlex
import shutil
import subprocess
import sys

import torch

ROOT = Path(__file__).resolve().parents[1]
ARMS = ('baseline', 'curriculum', 'specialists')


def commands(args, parent):
    if parent.get('format_version') != 6:
        raise ValueError('use the saved format-6 Moneta checkpoint as the common parent')
    rollout = args.envs*args.rollout_steps
    if args.steps <= 0 or args.steps % rollout:
        raise ValueError('--steps must be a positive multiple of envs * rollout-steps')
    if len(set(args.arms)) != len(args.arms):
        raise ValueError('each arm can be selected only once')
    # The next snapshot update lies beyond the last update of this experiment.
    snapshot_every = parent['iteration']+args.steps//rollout+1
    inherited = parent.get('config', {})
    shared = [sys.executable, '-m', 'smallworld_ai.train', '--rules', 'surface',
              '--resume', str(args.checkpoint.resolve()), '--engine', str(args.engine.resolve()),
              '--device', args.device, '--opponent-device', args.opponent_device,
              '--total-steps', str(parent['step']+args.steps), '--envs', str(args.envs),
              '--rollout-steps', str(args.rollout_steps), '--batch-size', str(args.batch_size),
              '--epochs', str(args.epochs), '--seed', str(args.seed), '--maps', 'generated',
              '--player-counts', '2', '3', '4', '5', '6', '--opponent', 'mixed',
              '--league-fraction', '0', '--snapshot-every', str(snapshot_every),
              '--objective', 'win', '--gamma', '1', '--rejection-penalty', '0',
              '--exploration-bonus', '0', '--gae-clock', 'turn', '--keep-checkpoints', '2']
    # Keep optimizer/exploration/feedback settings equal across all arms.
    defaults = dict(learning_rate=1e-4, gae_lambda=.99, entropy=.01, group_entropy=.02,
                    exploration_fraction=.1, exploration_temperature=1.5,
                    exploration_learned_groups=.75, exploration_piece_prior=2.,
                    acceptance_loss=.05, survival_loss=.05, response_loss=.05, rejection_loss=.05)
    for key, default in defaults.items():
        value = inherited.get(key, default)
        # The prototype's threat head needs observed outcomes to learn.
        if key == 'survival_loss': value = max(float(value), .05)
        shared.extend(['--'+key.replace('_','-'), str(value)])
    result = {}
    for arm in args.arms:
        command = shared+['--output', str((args.output/arm).resolve()),
                          '--focus-fraction', str(inherited.get('focus_fraction', .25))]
        if arm != 'baseline':
            command += ['--specialist-curriculum', '--specialist-fraction', str(args.fraction)]
        if arm == 'specialists':
            command += ['--upgrade-specialists', '--freeze-backbone' if args.freeze_backbone else '--no-freeze-backbone']
        result[arm] = command
    return result


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--checkpoint', required=True, type=Path)
    p.add_argument('--output', required=True, type=Path, help='new directory; existing directories are rejected')
    p.add_argument('--steps', type=int, default=262144, help='additional learner decisions per arm')
    p.add_argument('--envs', type=int, default=16)
    p.add_argument('--rollout-steps', type=int, default=128)
    p.add_argument('--batch-size', type=int, default=128)
    p.add_argument('--epochs', type=int, default=2)
    p.add_argument('--seed', type=int, default=170016)
    p.add_argument('--device', default='auto')
    p.add_argument('--opponent-device', choices=['auto','cpu','cuda'], default='cpu')
    p.add_argument('--engine', type=Path, default=ROOT/'ai/bin/botenv')
    p.add_argument('--arms', choices=ARMS, nargs='+', default=list(ARMS))
    p.add_argument('--fraction', type=float, default=.75)
    p.add_argument('--freeze-backbone', action=argparse.BooleanOptionalAction, default=False,
                   help='freeze inherited weights for an adapter-only pilot; default trains all perception too')
    p.add_argument('--dry-run', action='store_true', help='print commands without creating files or training')
    args = p.parse_args()
    if min(args.envs,args.rollout_steps,args.batch_size,args.epochs) < 1 or not 0 < args.fraction < 1:
        p.error('positive batch sizes and 0 < fraction < 1 are required')
    source = args.checkpoint.resolve()
    if not args.dry_run:
        args.output.mkdir(parents=True, exist_ok=False)
        args.checkpoint = args.output.resolve()/'parent.pt'
        # Opening the source once also freezes a trainer's atomically replaced
        # latest.pt. Every arm must read this copy, never the live path again.
        shutil.copyfile(source, args.checkpoint)
    parent = torch.load(args.checkpoint, map_location='cpu', weights_only=False)
    planned = commands(args, parent)
    for arm, command in planned.items():print(f'{arm}: {shlex.join(command)}', flush=True)
    if args.dry_run:return
    with args.checkpoint.open('rb') as stream:digest = hashlib.file_digest(stream,'sha256').hexdigest()
    manifest = dict(parent_source=str(source),frozen_parent=str(args.checkpoint),parent_sha256=digest, parent_step=parent['step'], additional_steps=args.steps,
                    commands=planned, completed=[], frozen_rival_weights=True,
                    note='Opponents sample the same frozen parent pool; trajectories can diverge. Evaluate on held-out maps.')
    del parent
    for arm, command in planned.items():
        (args.output/'experiment.json').write_text(json.dumps(manifest,indent=2)+'\n')
        with (args.output/(arm+'.log')).open('w') as log:
            subprocess.run(command,cwd=ROOT,stdout=log,stderr=subprocess.STDOUT,check=True,
                           env=os.environ | {'PYTHONPATH':str(ROOT/'ai')})
        manifest['completed'].append(arm)
    (args.output/'experiment.json').write_text(json.dumps(manifest,indent=2)+'\n')


if __name__ == '__main__':main()
