"""Paired, held-out initial-tribe evaluation across the enabled surface roster.

This complements ordinary market-choice tournaments. Assigned race/trait
results describe complete games starting with that pair, not isolated causal
effects of a module. The evaluator never updates weights or promotes models.
"""
import argparse
from collections import Counter
from concurrent.futures import ThreadPoolExecutor, as_completed
import hashlib
import json
from pathlib import Path
import random

import torch

from .engine import Engine, DEFAULT_ENGINE
from .model import policy_from_checkpoint
from .specialists import RACES, TRAITS


def fixtures(seed, cycles, players, races=RACES, traits=TRAITS):
    rng=random.Random(seed)
    result=[]
    for cycle in range(cycles):
        race_order,trait_order=list(races),list(traits)
        rng.shuffle(race_order);rng.shuffle(trait_order)
        for i in range(max(len(races),len(traits))):
            for count in players:
                result.append(dict(index=len(result),seed=seed+len(result),players=count,
                                   opponent_offset=cycle*max(len(races),len(traits))+i,
                                   seat=(i+cycle)%count,race=race_order[i%len(races)],
                                   trait=trait_order[i%len(traits)]))
    return result


def fingerprint(value):
    return hashlib.sha256(json.dumps(value,sort_keys=True,separators=(',',':')).encode()).hexdigest()


def opponent_lineup(job, opponents):
    # Global fixture indices interleave player counts. Using that index alone
    # pins each count to one offset when the opponent pool has the same size.
    # Keep old manifests reproducible while new fixtures rotate within counts.
    offset=job.get('opponent_offset',job['index'])
    return [opponents[(offset+seat)%len(opponents)] for seat in range(job['players'])]


def load_snapshot(path):
    # Hash the exact opened file, even if latest.pt is replaced during loading.
    with Path(path).open('rb') as stream:
        data=torch.load(stream,map_location='cpu',weights_only=False)
        stream.seek(0);digest=hashlib.file_digest(stream,'sha256').hexdigest()
    model=policy_from_checkpoint(data).eval().requires_grad_(False)
    return model,data,digest


def play_fixture(job, contenders, opponents, engine_path, distribution, excluded):
    # Each opponent seat uses the same immutable rival in both/all contenders'
    # games. All contenders replay the same generated map and engine RNG seed.
    lineup=opponent_lineup(job,opponents)
    results=[]
    with Engine(engine_path,rules='surface',max_steps=16384) as engine:
        initial=None
        for identifier,model in contenders.items():
            response=engine.request(op='reset',players=job['players'],map='generated',seed=job['seed'],
                                    map_split='eval',map_distribution=distribution,focus=True,focus_player=job['seat'],
                                    focus_race=job['race'],focus_trait=job['trait'])
            if engine.map_info.get('hash') in excluded:
                raise RuntimeError('evaluation map overlaps a supplied training map log')
            start=fingerprint(response['decision'])
            if initial is not None and start!=initial:raise RuntimeError('paired starting observations changed')
            initial=start
            counts,rejected=Counter(),Counter()
            learner_start=None
            while not response['done'] and not response['truncated']:
                decision=response['decision'];seat=decision['player'];learner=seat==job['seat']
                policy=model if learner else lineup[seat]
                if learner and learner_start is None:
                    learner_start=fingerprint(decision)
                    if {c['expert_race'] for c in decision['candidates']}!={job['race']}:
                        raise RuntimeError('assigned race was not available at the learner first turn')
                    if not all(job['trait'] in c['expert_traits'] for c in decision['candidates']):
                        raise RuntimeError('assigned trait was not available at the learner first turn')
                response=engine.request(op='step',**({'policy':policy} if isinstance(policy,str) else {'action':policy.choose(decision)}))
                if learner:
                    action=response['action'];key=action['kind']+':'+action.get('stack','')
                    counts[key]+=1;rejected[key]+=int(response.get('rejected',False))
            if response['truncated']:raise RuntimeError(f"fixture {job['index']} reached the step limit")
            scores=response['scores'];score=scores[job['seat']]
            credit=1/scores.count(score) if score==max(scores) else 0.
            results.append(dict(**job,model=identifier,scores=scores,win_credit=credit,
                                map=engine.map_info,initial_observation_sha256=initial,learner_start_sha256=learner_start,
                                actions=dict(counts),rejected=dict(rejected)))
    if len({r['learner_start_sha256'] for r in results})!=1:
        raise RuntimeError('opponents before the first learner turn differed across contenders')
    return results


def summarize(rows):
    result={}
    for row in rows:
        model=result.setdefault(row['model'],{})
        for dimension,name in (('players',str(row['players'])),('race',row['race']),('trait',row['trait'])):
            cell=model.setdefault(dimension,{}).setdefault(name,dict(games=0,win_credit=0.,points=0,actions=0,rejected=0))
            cell['games']+=1;cell['win_credit']+=row['win_credit'];cell['points']+=row['scores'][row['seat']]
            cell['actions']+=sum(row['actions'].values());cell['rejected']+=sum(row['rejected'].values())
    for dimensions in result.values():
        for cells in dimensions.values():
            for cell in cells.values():
                cell['win_share']=cell['win_credit']/cell['games'];cell['mean_points']=cell['points']/cell['games']
                cell['rejection_rate']=cell['rejected']/max(1,cell['actions'])
    return result


def main():
    p=argparse.ArgumentParser(description=__doc__)
    p.add_argument('--checkpoints',type=Path,nargs='+',required=True,help='trusted local training checkpoints')
    p.add_argument('--opponents',nargs='+',default=['greedy','random'],help='random, greedy, or trusted checkpoint paths')
    p.add_argument('--player-counts',type=int,nargs='+',default=[2,3,4,5,6])
    p.add_argument('--cycles',type=int,default=1,help='fresh pairing cycles; one covers every race and trait at each table size')
    p.add_argument('--seed',type=int,default=2140000000)
    p.add_argument('--workers',type=int,default=4)
    p.add_argument('--engine',type=Path,default=DEFAULT_ENGINE)
    p.add_argument('--distribution',choices=['surface','surface-stress'],default='surface')
    p.add_argument('--exclude-map-log',type=Path,nargs='*',default=[])
    p.add_argument('--limit-fixtures',type=int,help='functional smoke test only; intentionally leaves incomplete coverage')
    p.add_argument('--output',type=Path,required=True,help='new directory')
    args=p.parse_args()
    if min(args.workers,args.cycles)<1 or any(n not in range(2,7) for n in args.player_counts) or len(set(args.player_counts))!=len(args.player_counts):
        p.error('positive workers/cycles and distinct player counts 2..6 required')
    if args.limit_fixtures is not None and args.limit_fixtures<1:p.error('limit-fixtures must be positive')
    if len(set(p.resolve() for p in args.checkpoints))!=len(args.checkpoints):p.error('duplicate checkpoints')
    torch.set_num_threads(1)
    contenders,metadata={},{}
    for i,path in enumerate(args.checkpoints):
        model,data,digest=load_snapshot(path)
        if model.schema!='smallworld-bot-v2':p.error('focused evaluation requires surface checkpoints')
        identifier=f'{i}-{path.parent.name}'
        contenders[identifier]=model
        metadata[identifier]=dict(path=str(path.resolve()),sha256=digest,step=data.get('step'))
    opponents=[]
    opponent_metadata=[]
    for source in args.opponents:
        if source in ('random','greedy'):
            opponents.append(source);opponent_metadata.append(dict(builtin=source))
        else:
            model,data,digest=load_snapshot(source)
            if model.schema!='smallworld-bot-v2':p.error('opponents must use surface checkpoints')
            opponents.append(model)
            opponent_metadata.append(dict(path=str(Path(source).resolve()),sha256=digest,step=data.get('step')))
    excluded=set()
    for path in args.exclude_map_log:
        for line in path.read_text().splitlines():
            if line.strip():excluded.add(json.loads(line).get('map',{}).get('hash'))
    jobs=fixtures(args.seed,args.cycles,args.player_counts)
    if args.limit_fixtures:jobs=jobs[:args.limit_fixtures]
    args.output.mkdir(parents=True,exist_ok=False)
    with args.engine.open('rb') as stream:engine_digest=hashlib.file_digest(stream,'sha256').hexdigest()
    manifest=dict(models=metadata,fixtures=jobs,opponents=opponent_metadata,distribution=args.distribution,engine_sha256=engine_digest,
                  smoke_only=args.limit_fixtures is not None,excluded_maps=len(excluded),
                  note='Paired complete games from assigned starting tribes. Use ordinary tournaments to assess tribe selection.')
    (args.output/'manifest.json').write_text(json.dumps(manifest,indent=2)+'\n')
    rows=[]
    with ThreadPoolExecutor(max_workers=args.workers) as pool,(args.output/'games.jsonl').open('w') as log:
        pending=[pool.submit(play_fixture,job,contenders,opponents,args.engine,args.distribution,excluded) for job in jobs]
        for future in as_completed(pending):
            for row in future.result():
                rows.append(row);log.write(json.dumps(row)+'\n');log.flush()
            print(json.dumps(dict(completed_games=len(rows),total_games=len(jobs)*len(contenders))),flush=True)
    (args.output/'report.json').write_text(json.dumps(summarize(rows),indent=2)+'\n')


if __name__=='__main__':main()
