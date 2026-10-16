#!/usr/bin/env python3
"""Reproducible 2–6-player free-for-all comparison using the production bot evaluator."""
import argparse
from collections import Counter
from concurrent.futures import ThreadPoolExecutor, as_completed
import csv
from datetime import datetime, timezone
import hashlib
from functools import lru_cache
from itertools import combinations, permutations
import json
import os
from pathlib import Path
import random
import selectors
import shutil
import subprocess
import sys
import threading
import time

from smallworld_ai.maps import MapGenerator

ROOT = Path(__file__).resolve().parents[1]


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_json(path, value):
    path.write_text(json.dumps(value, indent=2) + '\n')


def catalog(directory):
    entries = [dict(id='random', name='Random bot', schema='builtin'),
               dict(id='greedy', name='Greedy bot', schema='builtin')]
    for path in sorted(directory.glob('*.json')):
        data = json.loads(path.read_bytes())
        entries.append({k: data[k] for k in ('id', 'name', 'schema')} | dict(path=str(path.resolve()), sha256=sha(path)))
    if len({e['id'] for e in entries}) != len(entries):
        raise ValueError('duplicate or reserved model IDs in catalog')
    return entries


def select_bots(entries, names):
    selected = []
    for name in names:
        hits = [e for e in entries if e['id'] == name or e['name'].casefold() == name.casefold()]
        if len(hits) != 1:
            raise ValueError(f'{name!r} matches {len(hits)} bots; use an exact ID from --list')
        if hits[0] in selected:
            raise ValueError(f'duplicate selection: {name}')
        selected.append(hits[0])
    return selected


@lru_cache(maxsize=None)
def blocks(n, exhaustive=False, players=4):
    """Equal appearances and equal unordered opponent-pair exposure."""
    if n < 2:
        raise ValueError('select at least two bots')
    if not 2 <= players <= 6:
        raise ValueError('players must be between 2 and 6')
    if n < players:
        copies, extra = divmod(players, n)
        base = tuple(i for i in range(n) for _ in range(copies))
        return [tuple(sorted(base + more)) for more in combinations(range(n), extra)]
    if exhaustive:
        return list(combinations(range(n), players))
    if players == 2 or players >= n-1:
        return list(combinations(range(n), players))
    if n == 8 and players == 4:
        return [tuple(i for i in range(8) if (i & mask).bit_count() % 2 == parity)
                for mask in range(1, 8) for parity in (0, 1)]
    # Translation orbits automatically balance individual appearances. Select
    # orbits whose cyclic distances have equal pair counts. Fall back to all
    # lineups when a small cyclic design is unavailable.
    if n > 12:
        raise ValueError('for more than 12 bots, explicitly request --matchups exhaustive')
    seen, orbits = set(), []
    for combo in combinations(range(n), players):
        if combo in seen:
            continue
        orbit = sorted({tuple(sorted((i+d) % n for i in combo)) for d in range(n)})
        seen.update(orbit)
        pairs = Counter(p for b in orbit for p in combinations(b, 2))
        orbits.append((orbit, tuple(pairs[(0, d)] for d in range(1, n//2+1))))
    attempts = 0
    for size in range(1, min(4, len(orbits)) + 1):
        best = None
        for indices in combinations(range(len(orbits)), size):
            attempts += 1
            if attempts > 20000:
                return list(combinations(range(n), players))
            counts = [sum(orbits[i][1][d] for i in indices) for d in range(n//2)]
            if min(counts) > 0 and len(set(counts)) == 1:
                candidate = [b for i in indices for b in orbits[i][0]]
                if best is None or len(candidate) < len(best):
                    best = candidate
        if best is not None:
            return best
    return list(combinations(range(n), players))


def rotated_seatings(seats):
    seats = tuple(seats)
    return sorted({order[i:] + order[:i] for order in (seats, seats[::-1]) for i in range(len(seats))})


@lru_cache(maxsize=None)
def compact_duplicate_seatings(block):
    # Different repeated-seat patterns have different numbers of unique
    # rotations. Pick among equally sized complete orbits so no lineup gets
    # extra weight; vary the ordering rather than always grouping copies.
    remaining = set(permutations(block))
    orbits = []
    while remaining:
        orbit = tuple(rotated_seatings(min(remaining)))
        remaining.difference_update(orbit)
        orbits.append(orbit)
    maximum = max(map(len, orbits))
    return tuple(orbit for orbit in orbits if len(orbit) == maximum)


def schedule(n, seeds, base_seed, matchups='balanced', orders='balanced', players=4):
    result = []
    for seed in seeds:
        rng = random.Random(base_seed ^ seed)
        for block in blocks(n, matchups == 'exhaustive', players):
            b = list(block)
            rng.shuffle(b)
            if (players == 4 and n < 4) or orders == 'all':
                seats = sorted(set(permutations(b)))
            elif n < players:
                seats = rng.choice(compact_duplicate_seatings(block))
            else:
                seats = rotated_seatings(b)
            for seating in seats:
                result.append(dict(index=len(result), seed=seed, seats=seating, players=players))
    return result


def comparison_plan(n, player_counts, maps, seed, matchups='balanced', orders='balanced'):
    """Keep whole balanced rounds, adding fresh maps to equalize game totals."""
    if maps < 1 or seed < 0:
        raise ValueError('maps must be positive and seed nonnegative')
    sizes = {count: len(schedule(n, [seed], seed, matchups, orders, count)) for count in player_counts}
    target = max(sizes.values()) * maps
    while True:
        # Round to the nearest whole map round, with ties rounded upward.
        rounds = {count: max(maps, (2*target+size)//(2*size)) for count, size in sizes.items()}
        totals = [rounds[count]*size for count, size in sizes.items()]
        if 5*max(totals) <= 6*min(totals):
            break
        # Move to the next rounding boundary instead of scanning every integer.
        target = min(((2*rounds[count]+1)*size+1)//2 for count, size in sizes.items())
    if seed + max(rounds.values()) > 2**63:
        raise ValueError('additional maps would exceed the signed 64-bit seed range')
    return dict(target_games=target, games_per_map=sizes,
                seeds_by_player_count={str(count): list(range(seed, seed+rounds[count])) for count in player_counts})


def exposure(jobs, n):
    players = len(jobs[0]['seats'])
    if any(len(job['seats']) != players for job in jobs):
        raise ValueError('audit exposure separately for each player count')
    seats, pairs = [[0]*players for _ in range(n)], Counter()
    for job in jobs:
        for seat, bot in enumerate(job['seats']):
            seats[bot][seat] += 1
        for a, b in combinations(job['seats'], 2):
            if a != b:
                pairs[tuple(sorted((a, b)))] += 1
    if len({tuple(s) for s in seats}) != 1 or len(set(seats[0])) != 1:
        raise AssertionError('unbalanced seat schedule')
    if len(pairs) != n*(n-1)//2 or len(set(pairs.values())) != 1:
        raise AssertionError('unbalanced opponent schedule')
    return dict(seats_per_bot=seats, games_per_pair=dict((f'{a}:{b}', c) for (a,b), c in sorted(pairs.items())))


def resolve_evaluator(mode='auto', executable=None):
    """Choose once; every match in this comparison uses the same runtime."""
    if mode == 'go':
        return dict(backend='go', python=None)
    explicit = executable or os.environ.get('BOT_INFERENCE_PYTHON')
    candidates = [explicit] if explicit else [str(ROOT/'ai/.venv/bin/python'), sys.executable]
    errors = []
    for candidate in dict.fromkeys(candidates):
        resolved = shutil.which(candidate)
        if not resolved:
            errors.append(f'{candidate}: executable not found')
            continue
        try:
            check = subprocess.run([resolved, '-c', 'import torch,numpy; print(torch.__version__)'],
                                   check=True, capture_output=True, text=True, timeout=30)
            return dict(backend='pytorch', python=resolved, torch=check.stdout.strip())
        except (OSError, subprocess.SubprocessError) as error:
            errors.append(f'{candidate}: PyTorch/NumPy unavailable ({type(error).__name__})')
    message = '; '.join(errors)
    if mode == 'pytorch' or explicit:
        raise ValueError(f'Cannot start PyTorch evaluator: {message}. Run ./ai/setup.sh --cpu or pass --inference-python PATH.')
    print('PyTorch unavailable; using Go. Run ./ai/setup.sh --cpu for faster neural bots.', flush=True)
    return dict(backend='go', python=None, reason=message)


class Bridge:
    def __init__(self, engine, models, timeout=300, max_steps=8192, python=None):
        self.timeout = timeout
        command = [str(engine), '-model-dir', str(models), '-max-steps', str(max_steps)]
        if python:
            command.extend(['-inference-python', python])
        self.process = subprocess.Popen(command,
            stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, bufsize=1,
            env=os.environ | {'GOMAXPROCS': '1', 'BOT_INFERENCE_PYTHON': python or '',
                                'PYTHONPATH': str(ROOT/'ai'), 'OMP_NUM_THREADS': '1',
                                'MKL_NUM_THREADS': '1', 'OPENBLAS_NUM_THREADS': '1'})

    def request(self, **data):
        self.process.stdin.write(json.dumps(data, separators=(',', ':')) + '\n')
        self.process.stdin.flush()
        with selectors.DefaultSelector() as selector:
            selector.register(self.process.stdout, selectors.EVENT_READ)
            if not selector.select(self.timeout):
                self.process.kill()
                raise RuntimeError(f'game engine timed out after {self.timeout}s')
        line = self.process.stdout.readline()
        if not line:
            raise RuntimeError('game engine exited; see stderr')
        value = json.loads(line)
        if value.get('error'):
            raise RuntimeError(value['error'])
        return value

    def close(self):
        self.process.stdin.close()
        try:
            self.process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()
        self.process.stdout.close()


def summarize(bots, rows):
    players = len(rows[0]['seats'])
    if any(len(row['seats']) != players or len(row['scores']) != players for row in rows):
        raise ValueError('summarize each player count separately')
    stats = [dict(id=b['id'], name=b['name'], appearances=0, wins=0, tied_first=0, win_credit=0.,
                  points=0, rank=0., margin=0., actions=0, rejections=0, successful_actions=Counter(), by_seed={}) for b in bots]
    pairwise = [[dict(games=0, credit=0.) for _ in bots] for _ in bots]
    for row in rows:
        scores, seats = row['scores'], row['seats']
        winners = scores.count(max(scores))
        for seat, i in enumerate(seats):
            s, score = stats[i], scores[seat]
            credit = (1/winners) if score == max(scores) else 0.
            s['appearances'] += 1
            s['wins'] += int(credit == 1)
            s['tied_first'] += int(0 < credit < 1)
            s['win_credit'] += credit
            s['points'] += score
            s['rank'] += 1 + sum(x > score for x in scores) + (scores.count(score)-1)/2
            s['margin'] += score - (sum(scores)-score)/(players-1)
            s['actions'] += row['actions'][seat]
            s['rejections'] += row['rejections'][seat]
            s['successful_actions'].update(row.get('successful_actions', [{}]*players)[seat])
            seed = s['by_seed'].setdefault(str(row['seed']), dict(appearances=0, win_credit=0.))
            seed['appearances'] += 1
            seed['win_credit'] += credit
            for other, j in enumerate(seats):
                if i != j:
                    pairwise[i][j]['games'] += 1
                    pairwise[i][j]['credit'] += (1. if score > scores[other] else .5 if score == scores[other] else 0.)
    for s in stats:
        count = s['appearances']
        s.update(win_share=s['win_credit']/count, mean_points=s['points']/count, mean_rank=s['rank']/count,
                 mean_margin=s['margin']/count, rejection_rate=s['rejections']/max(1,s['actions']))
    return dict(players=players, leaderboard=sorted(stats, key=lambda s: (-s['win_share'], s['mean_rank'], -s['mean_points'])), pairwise=pairwise)


def report(output, summary, manifest, print_output=True):
    columns = ['name','id','appearances','wins','tied_first','win_share','mean_points','mean_rank','mean_margin','rejection_rate']
    with (output/'leaderboard.csv').open('w') as file:
        writer = csv.DictWriter(file, columns, extrasaction='ignore')
        writer.writeheader()
        writer.writerows(summary['leaderboard'])
    lines = [f"# {manifest['players']}-player bot comparison", '', f"{manifest['games']} completed games; {len(manifest['seeds'])} generated evaluation maps; {manifest['rules']} rules.", '',
             '| Bot | Appearances | Win share | Mean points | Mean rank | Rejected actions |',
             '|---|---:|---:|---:|---:|---:|']
    for s in summary['leaderboard']:
        lines.append(f"| {s['name']} | {s['appearances']} | {s['win_share']:.1%} | {s['mean_points']:.1f} | {s['mean_rank']:.2f} | {s['rejection_rate']:.1%} |")
    lines.extend(['', 'Tied first places split one win credit. Lower mean rank is better. Seat rotations share a starting state and are correlated, so these are descriptive results, not independent-game confidence intervals.',
                  'Scores describe this opponent pool and rule palette. Legacy bots restrict a mixed tournament to the shared races and traits. Models still use their production observation/action adapters.',
                  'When fewer bots than seats are selected, evenly distributed copies fill the seats and compete individually; no teams or shared rewards.', ''])
    (output/'report.md').write_text('\n'.join(lines))
    if print_output:
        print('\n'.join(lines), flush=True)



def summarize_counts(bots, rows, player_counts):
    by_count = {str(count): summarize(bots, [r for r in rows if r['players'] == count])
                for count in player_counts}
    leaderboard = []
    for bot in bots:
        results = {count: next(s for s in by_count[str(count)]['leaderboard'] if s['id'] == bot['id'])
                   for count in player_counts}
        leaderboard.append(dict(id=bot['id'], name=bot['name'],
            mean_relative_win_share=sum(count*s['win_share'] for count, s in results.items())/len(results),
            mean_normalized_rank=sum((count-s['mean_rank'])/(count-1) for count, s in results.items())/len(results),
            win_share_by_players={str(count): s['win_share'] for count, s in results.items()}))
    leaderboard.sort(key=lambda s: (-s['mean_relative_win_share'], -s['mean_normalized_rank'], s['name']))
    return dict(player_counts=player_counts, by_player_count=by_count, leaderboard=leaderboard,
                aggregation='Equal weight per player count; relative win share = players * win share, with 1.0 an even-chance baseline.')


def report_counts(output, summary, manifest):
    counts = summary['player_counts']
    columns = ['name', 'id', 'mean_relative_win_share', 'mean_normalized_rank']
    with (output/'leaderboard.csv').open('w') as file:
        writer = csv.DictWriter(file, columns, extrasaction='ignore')
        writer.writeheader()
        writer.writerows(summary['leaderboard'])
    columns = ['players', 'name', 'id', 'appearances', 'wins', 'tied_first', 'win_share',
               'mean_points', 'mean_rank', 'mean_margin', 'rejection_rate']
    with (output/'leaderboard-by-player-count.csv').open('w') as file:
        writer = csv.DictWriter(file, columns, extrasaction='ignore')
        writer.writeheader()
        for count in counts:
            writer.writerows(dict(s, players=count) for s in summary['by_player_count'][str(count)]['leaderboard'])
    lines = ['# Bot comparison across player counts', '',
             f"{manifest['games']} completed games; {manifest['rules']} rules.", '',
             '| Bot | ' + ' | '.join(f'{count} players: win share' for count in counts) + ' | Mean relative win share |',
             '|---|' + '---:|'*(len(counts)+1)]
    map_counts = {str(count): len(manifest.get('seeds_by_player_count', {}).get(str(count), manifest['seeds'])) for count in counts}
    lines[3:3] = ['Games / fresh maps by player count: ' + '; '.join(f"{count} players: {manifest['games_by_player_count'][str(count)]} games / {map_counts[str(count)]} maps" for count in counts) + '.', '']
    for bot in summary['leaderboard']:
        values = ' | '.join(f"{bot['win_share_by_players'][str(count)]:.1%}" for count in counts)
        lines.append(f"| {bot['name']} | {values} | {bot['mean_relative_win_share']:.2f}x |")
    lines.extend(['', summary['aggregation'],
                  'The combined score does not pool raw wins or points: every player count has equal weight regardless of schedule size. Opponent-pool effects and matchup cycles still matter.', ''])
    for count in counts:
        target = output/f'players-{count}'
        target.mkdir()
        current = summary['by_player_count'][str(count)]
        write_json(target/'summary.json', current)
        report(target, current, dict(manifest, players=count, games=manifest['games_by_player_count'][str(count)],
               seeds=manifest.get('seeds_by_player_count', {}).get(str(count), manifest['seeds'])), print_output=False)
        lines.extend(['', f'## {count} players', '', *(target/'report.md').read_text().splitlines()[2:]])
    (output/'report.md').write_text('\n'.join(lines))
    print('\n'.join(lines), flush=True)


def main():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument('--list', action='store_true')
    group = p.add_mutually_exclusive_group()
    group.add_argument('--all', action='store_true', help='all local exports plus Random and Greedy')
    group.add_argument('--bots', nargs='+', help='menu names or exact IDs; quote names with spaces')
    p.add_argument('--model-dir', type=Path, default=ROOT/'ai/models')
    counts = p.add_mutually_exclusive_group()
    counts.add_argument('--players', type=int, nargs='+', choices=range(2, 7), help='player count(s), default: 4')
    counts.add_argument('--all-player-counts', action='store_true', help='compare at every supported size: 2, 3, 4, 5 and 6 players')
    p.add_argument('--maps', type=int, default=3, help='minimum fresh maps per player count; multi-size runs add maps to equalize game totals')
    p.add_argument('--seed', type=int, default=9152000)
    p.add_argument('--workers', type=int, default=min(8, os.cpu_count() or 1))
    p.add_argument('--rules', choices=['auto','legacy','surface'], default='auto')
    p.add_argument('--matchups', choices=['balanced','exhaustive'], default='balanced')
    p.add_argument('--seat-orders', choices=['balanced','all'], default='balanced')
    p.add_argument('--output', type=Path)
    p.add_argument('--engine', type=Path, help='use an already-built botenv instead of building')
    p.add_argument('--timeout', type=int, default=300, help='seconds allowed for one complete game')
    p.add_argument('--evaluator', choices=['auto','pytorch','go'], default='auto', help='auto prefers local PyTorch CPU; go forces the original evaluator')
    p.add_argument('--inference-python', help='Python executable with PyTorch/NumPy installed')
    p.add_argument('--dry-run', action='store_true', help='show exact schedule size without building or playing')
    args = p.parse_args()
    entries = catalog(args.model_dir)
    if args.list:
        for bot in entries:
            print(f"{bot['name']:18} {bot['id']:40} {bot['schema']}")
        return
    if not args.all and not args.bots:
        p.error('choose --all, --bots, or --list')
    if min(args.maps, args.workers, args.timeout) < 1 or args.seed < 0 or args.seed+args.maps >= 2**63:
        p.error('maps, workers and timeout must be positive; seed must fit a nonnegative signed 64-bit integer')
    bots = entries if args.all else select_bots(entries, args.bots)
    legacy = any(b['schema'] == 'smallworld-bot-v1' for b in bots)
    rules = ('legacy' if legacy else 'surface') if args.rules == 'auto' else args.rules
    if legacy and rules == 'surface':
        p.error('legacy models cannot play expanded surface rules; use auto or legacy')
    player_counts = list(range(2, 7)) if args.all_player_counts else sorted(set(args.players or [4]))
    jobs, balances, games_by_count = [], {}, {}
    try:
        plan = comparison_plan(len(bots), player_counts, args.maps, args.seed, args.matchups, args.seat_orders)
        seeds_by_count = plan['seeds_by_player_count']
        seeds = sorted({seed for group in seeds_by_count.values() for seed in group})
        for count in player_counts:
            current = schedule(len(bots), seeds_by_count[str(count)], args.seed, args.matchups, args.seat_orders, count)
            balances[str(count)] = exposure(current, len(bots))
            by_seed = {seed: [] for seed in seeds_by_count[str(count)]}
            for job in current:
                by_seed[job['seed']].append(job)
            for group in by_seed.values():
                exposure(group, len(bots))
            games_by_count[str(count)] = len(current)
            offset = len(jobs)
            jobs.extend(dict(job, index=offset+i) for i, job in enumerate(current))
    except ValueError as error:
        p.error(str(error))
    print(f"{len(bots)} bots, {len(jobs)} games, {rules} rules; target {plan['target_games']} games per player count (whole balanced rounds, totals within 20%)", flush=True)
    for count in player_counts:
        appearances = sum(balances[str(count)]['seats_per_bot'][0])
        print(f"  {count} players: {games_by_count[str(count)]} games on {len(seeds_by_count[str(count)])} fresh maps; {appearances} appearances per bot", flush=True)
    if args.dry_run:
        return
    try:
        runtime = resolve_evaluator(args.evaluator, args.inference_python)
    except ValueError as error:
        p.error(str(error))
    print(f"Evaluator: {runtime['backend']}" + (f" CPU ({runtime['python']}, torch {runtime['torch']})" if runtime['python'] else ''), flush=True)
    output = (args.output or ROOT/'ai/runs/comparisons'/datetime.now(timezone.utc).strftime('%Y%m%dT%H%M%S%fZ')).resolve()
    output.mkdir(parents=True, exist_ok=False)
    models = output/'models'
    models.mkdir()
    for b in bots:
        if 'path' in b:
            destination = models/Path(b['path']).name
            shutil.copyfile(b['path'], destination)
            if sha(destination) != b['sha256']:
                raise RuntimeError(f"model changed while freezing: {b['name']}")
    engine = output/'botenv'
    if args.engine:
        shutil.copy2(args.engine, engine)
    else:
        subprocess.run(['go','build','-o',str(engine),'./cmd/botenv'], cwd=ROOT/'backend2', check=True)
    manifest = dict(status='running', bots=bots, rules=rules, seeds=seeds, games=len(jobs), players=player_counts[0] if len(player_counts) == 1 else None,
        player_counts=player_counts, games_by_player_count=games_by_count,
        target_games_per_player_count=plan['target_games'], minimum_maps=args.maps,
        seeds_by_player_count=seeds_by_count, maps_by_player_count={count: len(values) for count, values in seeds_by_count.items()},
        matchups=args.matchups, seat_orders=args.seat_orders, exposure=balances[str(player_counts[0])] if len(player_counts) == 1 else balances, engine_sha256=sha(engine),
        generator_sha256=sha(ROOT/'ai/generate-map.mjs'), workers=args.workers, evaluator=runtime,
        created_utc=datetime.now(timezone.utc).isoformat(), command=os.sys.argv)
    write_json(output/'manifest.json', manifest)
    write_json(output/'schedule.json', jobs)
    fixtures, bridges, local = {}, [], threading.local()
    lock = threading.Lock()
    generator = MapGenerator()
    setup = Bridge(engine, models, args.timeout, python=runtime['python'])
    start = time.monotonic()
    try:
        (output/'fixtures').mkdir()
        for count in player_counts:
            directory = output/'fixtures'
            if len(player_counts) > 1:
                directory /= f'players-{count}'
                directory.mkdir()
            for seed in seeds_by_count[str(count)]:
                tiles, info = generator.generate(seed, count, 'eval', 'standard' if rules == 'legacy' else 'surface')
                fixture = setup.request(op='match_setup', rules=rules, seed=seed, players=count, map='generated', tiles=tiles)
                fixture['map_info'], fixture['tiles'], fixture['players'] = info, tiles, count
                fixtures[count, seed] = fixture
                write_json(directory/f'{seed}.json', fixture)
        # Check complete deterministic replay independently for every size.
        manifest['replay_verified_by_player_count'] = {}
        for count in player_counts:
            probe = next(job for job in jobs if job['players'] == count)
            f = fixtures[count, probe['seed']]
            request = dict(op='match', rules=rules, players=count, seed=probe['seed'],
                           policies=[bots[i]['id'] for i in probe['seats']],
                           initial_state=f['initial_state'], initial_sha256=f['initial_sha256'])
            if setup.request(**request)['match'] != setup.request(**request)['match']:
                raise RuntimeError(f'deterministic replay check failed for {count} players')
            manifest['replay_verified_by_player_count'][str(count)] = True
        manifest['replay_verified'] = True
        write_json(output/'manifest.json', manifest)
        def play(job):
            if not hasattr(local, 'bridge'):
                local.bridge = Bridge(engine, models, args.timeout, python=runtime['python'])
                with lock:
                    bridges.append(local.bridge)
            f = fixtures[job['players'], job['seed']]
            begin = time.monotonic()
            r = local.bridge.request(op='match', rules=rules, players=job['players'], seed=job['seed'], policies=[bots[i]['id'] for i in job['seats']],
                                     initial_state=f['initial_state'], initial_sha256=f['initial_sha256'])
            if (not r.get('done') or r.get('truncated') or r['match']['initial_sha256'] != f['initial_sha256']
                    or len(r['match']['scores']) != job['players'] or r['match'].get('players') != job['players']):
                raise RuntimeError('unfinished game or changed initial state')
            return job | r['match'] | dict(elapsed_seconds=time.monotonic()-begin)
        rows = []
        with (output/'matches.jsonl').open('w', buffering=1) as file, ThreadPoolExecutor(max_workers=args.workers) as pool:
            futures = [pool.submit(play, job) for job in jobs]
            try:
                for future in as_completed(futures):
                    row = future.result()
                    rows.append(row)
                    file.write(json.dumps(row)+'\n')
                    if len(rows) % 20 == 0 or len(rows) == len(jobs):
                        print(f"Completed {len(rows)}/{len(jobs)} games in {time.monotonic()-start:.0f}s", flush=True)
            except BaseException:
                for future in futures:
                    future.cancel()
                raise
        ordered = sorted(rows, key=lambda row: row['index'])
        summary = (summarize_counts(bots, ordered, player_counts) if len(player_counts) > 1
                   else summarize(bots, ordered))
        manifest.update(status='complete', elapsed_seconds=time.monotonic()-start)
        write_json(output/'summary.json', summary)
        if len(player_counts) > 1:
            report_counts(output, summary, manifest)
        else:
            report(output, summary, manifest)
        print(f'Results: {output}', flush=True)
    except BaseException as error:
        manifest.update(status='failed', error=str(error))
        raise
    finally:
        generator.close()
        setup.close()
        for bridge in bridges:
            bridge.close()
        write_json(output/'manifest.json', manifest)


if __name__ == '__main__':
    main()
