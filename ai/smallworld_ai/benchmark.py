"""Paired four-player evaluation and conservative checkpoint promotion.

Runs independently of PPO. Both contenders start from the exact same saved
games, face the same frozen rivals, and occupy every seat. Evaluation outcomes
never enter optimizer updates. New map seeds are used for each comparison.
"""
import argparse
from collections import Counter, defaultdict
from concurrent.futures import ThreadPoolExecutor, as_completed
import fcntl
import hashlib
import json
import os
from pathlib import Path
import random
import shutil
import statistics
import threading
import time

from compare_bots import Bridge
from .engine import DEFAULT_ENGINE
from .league import OpponentLeague
from .maps import MapGenerator
from .model import export_policy, load_checkpoint


def digest(path):
    with Path(path).open('rb') as stream:
        return hashlib.file_digest(stream, 'sha256').hexdigest()


def atomic_json(path, value):
    path = Path(path)
    temporary = path.with_suffix(path.suffix+'.tmp')
    with temporary.open('w') as stream:
        json.dump(value, stream, indent=2, allow_nan=False)
        stream.write('\n')
        stream.flush()
        os.fsync(stream.fileno())
    temporary.replace(path)


def report_pairs(rows, bootstrap_seed=1, regression_tolerance=.05):
    paired = defaultdict(dict)
    actions = {'challenger': Counter(), 'incumbent': Counter()}
    rejections = Counter()
    totals = Counter()
    for row in rows:
        seat, scores = row['seat'], row['scores']
        score = scores[seat]
        credit = 1/scores.count(score) if score == max(scores) else 0.
        relative = {}
        for other, rival in enumerate(row['policies']):
            if other != seat:
                relative[rival] = float(score > scores[other])+.5*float(score == scores[other])
        if row['contender'] in paired[(row['seed'], seat)]:
            raise ValueError('duplicate paired benchmark game')
        paired[(row['seed'], seat)][row['contender']] = (credit, relative)
        actions[row['contender']].update(row['successful_actions'][seat])
        rejections[row['contender']] += row['rejections'][seat]
        totals[row['contender']] += row['actions'][seat]
    by_seed, rivals, credits = defaultdict(list), defaultdict(list), defaultdict(list)
    for (seed, _), pair in paired.items():
        if set(pair) != {'challenger', 'incumbent'}:
            raise ValueError('incomplete paired benchmark')
        a, b = pair['challenger'], pair['incumbent']
        if set(a[1]) != set(b[1]):
            raise ValueError('paired contenders faced different rivals')
        by_seed[seed].append(a[0]-b[0])
        for kind in pair: credits[kind].append(pair[kind][0])
        for rival in a[1]: rivals[rival].append(a[1][rival]-b[1][rival])
    if not by_seed or any(len(v) != 4 for v in by_seed.values()):
        raise ValueError('benchmark must contain all four seats for every map')
    differences = [statistics.mean(v) for v in by_seed.values()]
    rng = random.Random(bootstrap_seed)
    draws = sorted(statistics.mean(rng.choices(differences, k=len(differences))) for _ in range(5000))
    interval = [draws[125], draws[4874]]
    rival_deltas = {key: statistics.mean(value) for key, value in rivals.items()}
    # A positive paired confidence bound is required; a newer timestamp is
    # never sufficient. Also reject point-estimate regressions against a rival.
    promote = interval[0] > 0 and min(rival_deltas.values(), default=0) >= -regression_tolerance
    return dict(promote=promote, maps=len(differences), paired_games=len(paired),
                win_credit={k: statistics.mean(v) for k,v in credits.items()},
                paired_win_credit_delta=statistics.mean(differences),
                paired_map_bootstrap_95_interval=interval,
                relative_finish_delta_by_rival=rival_deltas,
                regression_tolerance=regression_tolerance,
                successful_actions={k:dict(v) for k,v in actions.items()},
                rejection_rate={k:rejections[k]/max(1,totals[k]) for k in totals})


def export_checkpoint(checkpoint, models, identifier):
    policy, data = load_checkpoint(checkpoint)
    export_policy(policy, models/(identifier+'.json'), identifier, identifier)
    return dict(step=data['step'], iteration=data['iteration'], sha256=digest(checkpoint),
                model_sha256=digest(models/(identifier+'.json')))


def compare(args, rivals, models, challenger, incumbent, seed, output):
    fixtures, bridges, lock, local = {}, [], threading.Lock(), threading.local()
    generator = MapGenerator()
    setup = Bridge(args.engine, models, timeout=getattr(args, 'timeout', 300))
    jobs = []
    excluded = set()
    map_log = args.run/'maps.jsonl'
    if map_log.exists():
        with map_log.open() as file:
            for line in file:
                if line.endswith('\n'):
                    excluded.add(json.loads(line).get('map', {}).get('hash'))
    try:
        ordered = list(rivals)
        random.Random(seed).shuffle(ordered)
        (output/'fixtures').mkdir()
        for i in range(args.maps):
            map_seed = seed+i
            tiles, info = generator.generate(map_seed, 4, 'eval', 'surface')
            if info.get('hash') in excluded:
                raise RuntimeError('benchmark map appeared in training')
            fixture = setup.request(op='match_setup', rules='surface', seed=map_seed,
                                    players=4, map='generated', tiles=tiles)
            fixtures[map_seed] = fixture
            atomic_json(output/'fixtures'/f'{map_seed}.json', fixture | {'map_info': info})
            # Consecutive triples of a shuffled circular roster give every
            # rival coverage. Rotation gives every contender all four seats.
            opponents = [ordered[(i+j) % len(ordered)] for j in range(3)]
            for seat in range(4):
                for contender in ('challenger', 'incumbent'):
                    policies = opponents.copy()
                    policies.insert(0, 'eval-'+contender)
                    policies = policies[-seat:]+policies[:-seat] if seat else policies
                    jobs.append(dict(seed=map_seed, seat=seat, contender=contender, policies=policies))
        probe = jobs[0]
        fixture = fixtures[probe['seed']]
        request = dict(op='match', rules='surface', seed=probe['seed'], policies=probe['policies'],
                       initial_state=fixture['initial_state'], initial_sha256=fixture['initial_sha256'])
        if setup.request(**request)['match'] != setup.request(**request)['match']:
            raise RuntimeError('benchmark deterministic replay failed')
        setup.close()
        setup = None
        def play(job):
            if not hasattr(local, 'bridge'):
                local.bridge = Bridge(args.engine, models, timeout=getattr(args, 'timeout', 300))
                with lock: bridges.append(local.bridge)
            fixture = fixtures[job['seed']]
            response = local.bridge.request(op='match', rules='surface', seed=job['seed'], policies=job['policies'],
                                            initial_state=fixture['initial_state'], initial_sha256=fixture['initial_sha256'])
            if not response.get('done') or response.get('truncated') or response['match']['initial_sha256'] != fixture['initial_sha256']:
                raise RuntimeError('incomplete game or changed benchmark fixture')
            return job | response['match']
        rows = []
        with (output/'matches.jsonl').open('w', buffering=1) as file, ThreadPoolExecutor(max_workers=args.workers) as pool:
            for future in as_completed([pool.submit(play, job) for job in jobs]):
                row = future.result()
                rows.append(row)
                file.write(json.dumps(row)+'\n')
                if len(rows) % 16 == 0:
                    print(json.dumps({'benchmark_games':len(rows), 'total':len(jobs), 'report':str(output)}), flush=True)
        report = report_pairs(rows, bootstrap_seed=seed, regression_tolerance=args.regression_tolerance)
        report.update(challenger=challenger, incumbent=incumbent, seed=seed,
                      rules='surface', powers=0, players=4, map_split='eval',
                      engine_sha256=digest(args.engine), replay_verified=True,
                      generator_sha256=digest(Path(__file__).resolve().parents[1]/'generate-map.mjs'))
        atomic_json(output/'summary.json', report)
        return report
    finally:
        generator.close()
        if setup is not None: setup.close()
        for bridge in bridges: bridge.close()


def run(args):
    import torch
    torch.set_num_threads(1)
    args.run = args.run.resolve()
    output = args.run/'benchmark'
    output.mkdir(exist_ok=True)
    with (output/'.lock').open('w') as lock:
        fcntl.flock(lock, fcntl.LOCK_EX | fcntl.LOCK_NB)
        league = OpponentLeague(args.league)
        if len(league.members) < 3 or args.maps < len(league.members):
            raise ValueError('benchmark needs at least three rivals and enough maps to cover the roster')
        models = output/'models'
        models.mkdir(exist_ok=True)
        for member in league.members:
            export_policy(member['policy'], models/(member['id']+'.json'), member['id'], member['name'])
        rivals = [m['id'] for m in league.members]
        roster = {m['id']:m['sha256'] for m in league.members}
        gate = args.run/'champion.json'
        counter_path = output/'next-seed.json'
        seed = json.loads(counter_path.read_text())['seed'] if counter_path.exists() else args.seed
        last_checked = None
        while True:
            # Reserve new seeds before evaluating, including after an interrupted run.
            atomic_json(counter_path, {'seed':seed+args.maps})
            trial = output/f'comparison-{seed}'
            trial.mkdir(exist_ok=False)
            frozen = output/'challenger.pt'
            shutil.copyfile(args.run/'latest.pt', frozen)
            challenger = export_checkpoint(frozen, models, 'eval-challenger')
            if gate.exists():
                incumbent_record = json.loads(gate.read_text())
                if incumbent_record['roster'] != roster:
                    raise ValueError('champion roster changed; use a new training run')
                incumbent_path = Path(incumbent_record['checkpoint'])
                if digest(incumbent_path) != incumbent_record['sha256']:
                    raise ValueError('champion checkpoint checksum mismatch')
            else:
                incumbent_path = args.baseline.resolve()
                incumbent_record = {'baseline': True}
            incumbent = export_checkpoint(incumbent_path, models, 'eval-incumbent')
            report = compare(args, rivals, models, challenger, incumbent, seed, trial)
            selected = frozen if report['promote'] else incumbent_path
            chosen = challenger if report['promote'] else incumbent
            champions = args.run/'champions'
            champions.mkdir(exist_ok=True)
            saved = champions/(chosen['sha256']+'.pt')
            if not saved.exists():
                temporary = saved.with_suffix('.pt.tmp')
                shutil.copyfile(selected, temporary)
                if digest(temporary) != chosen['sha256']:
                    raise RuntimeError('checkpoint changed while promoting')
                temporary.replace(saved)
            atomic_json(gate, chosen | {'checkpoint':str(saved), 'roster':roster,
                        'report':str(trial/'summary.json'), 'promoted':report['promote'],
                        'status':'evaluated', 'timestamp':time.time()})
            print(json.dumps({'champion':str(saved), 'step':chosen['step'], 'promoted':report['promote'],
                              'paired_win_credit_delta':report['paired_win_credit_delta'],
                              'interval':report['paired_map_bootstrap_95_interval']}), flush=True)
            last_checked = challenger['sha256']
            if args.once: break
            time.sleep(args.interval)
            while digest(args.run/'latest.pt') == last_checked:
                time.sleep(min(60,args.interval))
            seed += args.maps


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--run',type=Path,required=True)
    p.add_argument('--league',type=Path,required=True)
    p.add_argument('--baseline',type=Path,required=True,help='initial benchmark incumbent; not assumed universally best')
    p.add_argument('--engine',type=Path,default=DEFAULT_ENGINE)
    p.add_argument('--maps',type=int,default=36)
    p.add_argument('--workers',type=int,default=2)
    p.add_argument('--timeout',type=int,default=300,help='seconds allowed per whole game; increase for large Go models')
    p.add_argument('--seed',type=int,default=2_000_000_000)
    p.add_argument('--interval',type=int,default=1800)
    p.add_argument('--regression-tolerance',type=float,default=.05)
    p.add_argument('--once',action='store_true')
    args = p.parse_args()
    if min(args.maps,args.workers,args.interval,args.timeout) < 1 or not 0 <= args.regression_tolerance <= 1:
        p.error('invalid benchmark limits')
    run(args)


if __name__ == '__main__': main()
