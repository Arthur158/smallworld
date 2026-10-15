"""Evaluate a deployed Go policy on held-out seeds with rotating player seats."""
import argparse
import json
import math
from pathlib import Path

from .engine import DEFAULT_ENGINE, Engine, outcome


def evaluate(engine, model, opponent, games, seed, players, maps, distribution="standard", excluded_hashes=None):
    results = []
    for i in range(games):
        seat = i % players
        map_seed = seed+i//players
        response = engine.request(op="reset", players=players, map=maps[(i//players) % len(maps)], seed=map_seed,
                                  map_split="eval", map_distribution=distribution)
        if excluded_hashes and engine.map_info.get("hash") in excluded_hashes:
            raise RuntimeError("evaluation map overlaps the training map log")
        while not (response["done"] or response["truncated"]):
            policy = model if response["decision"]["player"] == seat else opponent
            response = engine.request(op="step", policy=policy)
        if response["truncated"]:
            raise RuntimeError(f"evaluation game {i} did not finish")
        scores = response["scores"]
        results.append({"seed": map_seed, "seat": seat, "scores": scores, "map": engine.map_info,
                        "outcome": outcome(scores, seat), "steps": response["steps"]})
    wins = sum(r["outcome"] == 1 for r in results)
    ties = sum(r["outcome"] == 0 for r in results)
    rate, z = wins/games, 1.96
    center = (rate+z*z/(2*games))/(1+z*z/games)
    radius = z*math.sqrt(rate*(1-rate)/games+z*z/(4*games*games))/(1+z*z/games)
    return {"model": model, "opponent": opponent, "games": games, "players": players, "maps": maps, "map_split": "eval", "distribution": distribution,
            "wins": wins, "ties": ties, "losses": games-wins-ties, "win_rate": rate,
            "win_rate_95_interval": [center-radius, center+radius],
            "mean_score": sum(r["scores"][r["seat"]] for r in results)/games,
            "mean_score_margin": sum(r["scores"][r["seat"]]-max(s for j,s in enumerate(r["scores"]) if j != r["seat"]) for r in results)/games,
            "matches": results}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--engine", type=Path, default=DEFAULT_ENGINE)
    parser.add_argument("--model", default="random")
    parser.add_argument("--rules", choices=["legacy", "surface"], default="legacy")
    parser.add_argument("--model-dir", type=Path)
    parser.add_argument("--opponent", default="greedy")
    parser.add_argument("--games", type=int, default=100)
    parser.add_argument("--seed", type=int, default=1_000_000)
    parser.add_argument("--players", type=int, default=2)
    parser.add_argument("--maps", nargs="+", default=["generated"])
    parser.add_argument("--map-distribution", choices=["standard", "stress", "surface", "surface-stress"], default="standard")
    parser.add_argument("--exclude-map-log", type=Path, help="reject any exact map hash already recorded during training")
    parser.add_argument("--output", type=Path)
    args = parser.parse_args()
    if args.games < 1 or not 2 <= args.players <= 6:
        parser.error("games must be positive and players must be 2..6")
    excluded_hashes = set()
    if args.exclude_map_log:
        with args.exclude_map_log.open() as stream:
            for line in stream:
                value = json.loads(line).get("map", {}).get("hash")
                if value:
                    excluded_hashes.add(value)
    with Engine(args.engine, args.model_dir, rules=args.rules, max_steps=8192 if args.rules == "surface" else 4096) as engine:
        result = evaluate(engine, args.model, args.opponent, args.games, args.seed, args.players, args.maps, args.map_distribution, excluded_hashes)
    print(json.dumps({k:v for k,v in result.items() if k != "matches"}, indent=2))
    if args.output:
        args.output.parent.mkdir(parents=True, exist_ok=True)
        args.output.write_text(json.dumps(result, indent=2)+"\n")


if __name__ == "__main__":
    main()
