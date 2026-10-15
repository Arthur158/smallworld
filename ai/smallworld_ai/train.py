"""Batched masked PPO against random, greedy and frozen self-play opponents."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import copy
import hashlib
import json
from pathlib import Path
import random
import subprocess
import time

import torch

from . import SCHEMA
from .device import select_device
from .engine import DEFAULT_ENGINE, LearnerEnv, outcome
from .model import Policy, atomic_torch_save, batch_decisions, load_checkpoint, merge_decision_batches


def advantages(rewards, dones, values, last_values, gamma, gae_lambda):
    """GAE follows the learner's decisions; an opponent never receives its credit."""
    result = torch.zeros_like(rewards)
    carry = torch.zeros_like(last_values)
    for t in reversed(range(rewards.shape[0])):
        next_values = last_values if t == rewards.shape[0]-1 else values[t+1]
        alive = 1.0-dones[t]
        delta = rewards[t] + gamma*next_values*alive - values[t]
        carry = delta + gamma*gae_lambda*alive*carry
        result[t] = carry
    return result, result+values


def update_policy(policy, optimizer, observations, actions, old_logprobs, returns, advantage, args, device):
    # Raw observations remain accepted for callers outside the rollout loop.
    # The trainer supplies already encoded device tensors, reused for all epochs.
    packed = batch_decisions(observations, device) if isinstance(observations, list) else observations
    count = packed[0].shape[0]
    advantage = (advantage-advantage.mean()) / (advantage.std(unbiased=False)+1e-8)
    actions, old_logprobs = actions.to(device), old_logprobs.to(device)
    returns, advantage = returns.to(device), advantage.to(device)
    metrics = []
    stop = False
    for _ in range(args.epochs):
        for index in torch.randperm(count).to(device).split(args.batch_size):
            batch = tuple(value.index_select(0, index) for value in packed)
            distribution, values = policy(*batch)
            logprobs = distribution.log_prob(actions[index])
            logratio = logprobs-old_logprobs[index]
            ratio = logratio.exp()
            clipped = ratio.clamp(1-args.clip, 1+args.clip)
            actor_loss = -torch.minimum(ratio*advantage[index], clipped*advantage[index]).mean()
            critic_loss = (values-returns[index]).square().mean()
            entropy = distribution.entropy().mean()
            loss = actor_loss+0.5*critic_loss-args.entropy*entropy
            if not torch.isfinite(loss):
                raise RuntimeError("non-finite PPO loss; training stopped before saving invalid weights")
            optimizer.zero_grad(set_to_none=True)
            loss.backward()
            torch.nn.utils.clip_grad_norm_(policy.parameters(), 0.5)
            optimizer.step()
            with torch.no_grad():
                kl = ((ratio-1)-logratio).mean()
                metrics.append(torch.stack((actor_loss, critic_loss, entropy, kl)).detach())
                # Keep the same per-minibatch KL stop, but copy loss metrics only
                # once at the end instead of synchronizing for each scalar.
                if kl.item() > args.target_kl:
                    stop = True
                    break
        if stop:
            break
    return dict(zip(("actor_loss", "critic_loss", "entropy", "approx_kl"),
                    torch.stack(metrics).mean(0).cpu().tolist()))


def step_and_reset(pair):
    """Reset finished episodes in their workers, preserving terminal rewards."""
    env, action = pair
    obs, reward, terminated, truncated, info = env.step(action)
    if terminated and not truncated:
        obs = env.reset()
    return obs, reward, terminated, truncated, info


def git_revision():
    try:
        return subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=Path(__file__).resolve().parents[2],
                                       text=True, stderr=subprocess.DEVNULL).strip()
    except (OSError, subprocess.CalledProcessError):
        return "unknown"


def parser():
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument("--engine", type=Path, default=DEFAULT_ENGINE)
    p.add_argument("--device", default="auto", help="auto, cpu, cuda, or cuda:N")
    p.add_argument("--output", type=Path, default=Path("ai/runs/first"))
    p.add_argument("--resume", type=Path)
    p.add_argument("--warm-start", type=Path, help="transfer a legacy policy into the graph model")
    p.add_argument("--rules", choices=["legacy", "surface"], default="legacy")
    p.add_argument("--total-steps", type=int, default=200_000, help="target total learner decisions, including a resumed run")
    p.add_argument("--envs", type=int, default=8)
    p.add_argument("--rollout-steps", type=int, default=128)
    p.add_argument("--players", type=int, default=2)
    p.add_argument("--player-counts", type=int, nargs="+", help="vary table size between episodes")
    p.add_argument("--maps", nargs="+", default=["generated"])
    p.add_argument("--opponent", choices=["random", "greedy", "mixed", "selfplay"], default="mixed")
    p.add_argument("--seed", type=int, default=1)
    p.add_argument("--hidden", type=int, default=128)
    p.add_argument("--batch-size", type=int, default=256)
    p.add_argument("--epochs", type=int, default=4)
    p.add_argument("--learning-rate", type=float, default=3e-4)
    p.add_argument("--gamma", type=float, default=0.995)
    p.add_argument("--gae-lambda", type=float, default=0.95)
    p.add_argument("--clip", type=float, default=0.2)
    p.add_argument("--entropy", type=float, default=0.01)
    p.add_argument("--target-kl", type=float, default=0.03)
    p.add_argument("--snapshot-every", type=int, default=10, help="updates between frozen self-play snapshots")
    p.add_argument("--save-every", type=int, default=10, help="updates between numbered checkpoints; latest saved each update")
    p.add_argument("--torch-threads", type=int, default=1)
    return p


def run(args):
    for key in ("total_steps", "envs", "rollout_steps", "batch_size", "epochs", "snapshot_every", "save_every", "torch_threads"):
        if getattr(args, key) < 1:
            raise ValueError(f"{key} must be positive")
    if not 2 <= args.players <= 6 or args.envs > 128 or any(not 2 <= n <= 6 for n in (args.player_counts or [])):
        raise ValueError("players must be 2..6 and envs at most 128")
    if not 0 < args.gamma <= 1 or not 0 <= args.gae_lambda <= 1 or not 0 < args.clip < 1:
        raise ValueError("invalid gamma, GAE lambda or clip")
    if args.learning_rate <= 0 or args.target_kl <= 0 or args.entropy < 0:
        raise ValueError("invalid optimizer parameters")
    torch.set_num_threads(args.torch_threads)
    torch.manual_seed(args.seed)
    random.seed(args.seed)
    device, device_report = select_device(args.device)
    print(json.dumps({"device": device_report}), flush=True)
    if args.rules == "surface":
        from .graph import GraphPolicy
        policy = GraphPolicy(args.hidden).to(device)
    else:
        policy = Policy(args.hidden).to(device)
    if args.warm_start:
        if args.rules != "surface" or args.resume:
            raise ValueError("warm-start requires surface rules and no resume")
        teacher, _ = load_checkpoint(args.warm_start)
        policy.warm_start(teacher.to(device))
    checkpoint = None
    step, iteration = 0, 0
    if args.resume:
        policy, checkpoint = load_checkpoint(args.resume, device)
        if (policy.schema == SCHEMA) != (args.rules == "legacy"):
            raise ValueError("resume schema does not match --rules")
        args.hidden = policy.hidden
        step, iteration = checkpoint["step"], checkpoint["iteration"]
        torch.set_rng_state(checkpoint["torch_rng"])
        if device.type == "cuda" and checkpoint.get("cuda_rng"):
            torch.cuda.set_rng_state_all(checkpoint["cuda_rng"])
        random.setstate(checkpoint["python_rng"])
    optimizer = torch.optim.Adam(policy.parameters(), lr=args.learning_rate, eps=1e-5)
    if checkpoint:
        optimizer.load_state_dict(checkpoint["optimizer"])
        for group in optimizer.param_groups:
            group["lr"] = args.learning_rate
    config = {k: str(v) if isinstance(v, Path) else v for k, v in vars(args).items()}
    config.update(schema=policy.schema, git_revision=git_revision(), map_split="train")
    config["trainer_sha256"] = hashlib.sha256(b"".join(
        Path(__file__).with_name(name).read_bytes() for name in ("train.py", "model.py", "engine.py")
    )).hexdigest()
    try:
        config["map_generator_revision"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=Path(__file__).resolve().parents[2]/"smallworld-map-generator", text=True).strip()
    except (OSError, subprocess.CalledProcessError):
        config["map_generator_revision"] = "unknown"
    args.output.mkdir(parents=True, exist_ok=True)
    # Preserve the initial config; each resume gets its own invocation record.
    (args.output / f"config-{step}-{time.time_ns()}.json").write_text(json.dumps(config, indent=2)+"\n")
    environments = []
    started = time.monotonic()
    start_step = step
    snapshots = [copy.deepcopy(policy).cpu().eval()]
    episode_stats = []
    coverage = {key: set() for key in ("races", "traits", "powers", "actions")}
    if checkpoint:
        for key, values in checkpoint.get("coverage", {}).items():
            coverage[key].update(values)
    rejected_total = checkpoint.get("rejected_total", 0) if checkpoint else 0
    cursors = checkpoint.get("curriculum_cursors", []) if checkpoint else []
    try:
        for i in range(args.envs):
            env = LearnerEnv(args.engine, args.seed+step*101+i*1009, args.player_counts or args.players, args.maps, args.opponent,
                             max_steps=8192 if args.rules == "surface" else 4096, rules=args.rules,
                             curriculum_start=cursors[i] if len(cursors)==args.envs else max(cursors, default=0)+i, curriculum_stride=args.envs)
            if checkpoint and len(checkpoint.get("environment_episodes", [])) == args.envs:
                env.episodes = checkpoint["environment_episodes"][i]
            env.snapshots = snapshots
            environments.append(env)
        with ThreadPoolExecutor(max_workers=args.envs) as pool, (args.output / "metrics.jsonl").open("a", buffering=1) as log, (args.output / "maps.jsonl").open("a", buffering=1) as map_log:
            def record_map(env):
                map_log.write(json.dumps({"step": step, "seat": env.seat, "map": env.engine.map_info})+"\n")
            observations = list(pool.map(lambda env: env.reset(), environments))
            for env in environments:
                record_map(env)
            while step < args.total_steps:
                rollout_started = time.monotonic()
                policy.eval()
                rollout_batches, rollout_actions, rollout_logprobs = [], [], []
                rollout_values, rollout_rewards, rollout_dones = [], [], []
                # Keep a complete rectangular rollout for GAE, including the last update.
                for _ in range(args.rollout_steps):
                    # Create normal tensors outside inference_mode: the same
                    # cached inputs will later be used for gradient computation.
                    batch = batch_decisions(observations, device)
                    with torch.inference_mode():
                        distribution, values = policy(*batch)
                        actions = distribution.sample()
                        logprobs = distribution.log_prob(actions)
                    rollout_batches.append(batch)
                    rollout_actions.append(actions)
                    rollout_logprobs.append(logprobs)
                    rollout_values.append(values)
                    # Actions are the only per-step GPU result the CPU engines
                    # need. Values are copied together when computing GAE.
                    results = list(pool.map(step_and_reset, zip(environments, actions.cpu().tolist())))
                    rewards, dones, next_obs = [], [], []
                    for env, (obs, reward, terminated, truncated, info) in zip(environments, results):
                        if truncated:
                            # A limit should never be reached by this finite action set.
                            # Do not quietly train on incomplete games or count them as losses.
                            raise RuntimeError(f"episode hit engine limit: {info}; investigate rules before training")
                        rejected_total += int(info.get("rejected", False))
                        for key, items in info.get("coverage", {}).items():
                            coverage[key].update(items)
                        rewards.append(reward)
                        dones.append(float(terminated))
                        if terminated:
                            episode_stats.append({"outcome": outcome(info["scores"], info["seat"]),
                                                  "score": info["scores"][info["seat"]], "return": info["episode_return"]})
                            record_map(env)
                        next_obs.append(obs)
                    rollout_rewards.append(torch.tensor(rewards))
                    rollout_dones.append(torch.tensor(dones))
                    observations = next_obs
                    step += args.envs
                with torch.inference_mode():
                    _, last_values = policy(*batch_decisions(observations, device))
                advantage, returns = advantages(torch.stack(rollout_rewards), torch.stack(rollout_dones),
                                                torch.stack(rollout_values).cpu(), last_values.cpu(), args.gamma, args.gae_lambda)
                rollout_seconds = time.monotonic() - rollout_started
                policy.train()
                update_started = time.monotonic()
                packed = merge_decision_batches(rollout_batches)
                # Release the per-step buffers once the contiguous rollout owns
                # the inputs, keeping the extra CUDA memory bounded to one rollout.
                rollout_batches.clear()
                metrics = update_policy(policy, optimizer, packed, torch.stack(rollout_actions).flatten(),
                                        torch.stack(rollout_logprobs).flatten(), returns.flatten(), advantage.flatten(), args, device)
                metrics.update(rollout_seconds=round(rollout_seconds, 4),
                               update_seconds=round(time.monotonic() - update_started, 4))
                del packed
                iteration += 1
                if iteration % args.snapshot_every == 0:
                    snapshots.append(copy.deepcopy(policy).cpu().eval())
                    snapshots[:] = snapshots[-8:]
                metrics.update(step=step, iteration=iteration, episodes=len(episode_stats),
                               decisions_per_second=round((step-start_step)/(time.monotonic()-started), 1))
                recent = episode_stats[-100:]
                if recent:
                    metrics.update(recent_win_rate=sum(e["outcome"] == 1 for e in recent)/len(recent),
                                   recent_mean_score=sum(e["score"] for e in recent)/len(recent))
                if args.rules == "surface":
                    metrics.update(rejected_actions=rejected_total, coverage={k:len(v) for k,v in coverage.items()})
                    (args.output / "coverage.json").write_text(json.dumps({k:sorted(v) for k,v in coverage.items()}, indent=2)+"\n")
                print(json.dumps(metrics), flush=True)
                log.write(json.dumps(metrics)+"\n")
                payload = {"format_version": 2 if args.rules == "surface" else 1, "schema": policy.schema, "config": config, "device_report": device_report,
                           "step": step, "iteration": iteration, "model": policy.state_dict(),
                           "optimizer": optimizer.state_dict(), "torch_rng": torch.get_rng_state(),
                           "coverage": {k: sorted(v) for k,v in coverage.items()}, "rejected_total": rejected_total,
                           "curriculum_cursors": [e.curriculum for e in environments],
                           "environment_episodes": [e.episodes for e in environments],
                           "python_rng": random.getstate(),
                           "cuda_rng": torch.cuda.get_rng_state_all() if device.type == "cuda" else []}
                atomic_torch_save(payload, args.output / "latest.pt")
                if iteration % args.save_every == 0:
                    atomic_torch_save(payload, args.output / f"step-{step}.pt")
    finally:
        for env in environments:
            env.close()
    print(f"Checkpoint: {args.output / 'latest.pt'}", flush=True)


def main():
    args = parser().parse_args()
    try:
        run(args)
    except KeyboardInterrupt:
        print("Stopped. The last completed update is in latest.pt.")
        raise SystemExit(130)


if __name__ == "__main__":
    main()
