"""Batched masked PPO against random, greedy and frozen self-play opponents."""
import argparse
from concurrent.futures import ThreadPoolExecutor
import copy
import json
from pathlib import Path
import random
import subprocess
import time

import torch

from . import SCHEMA
from .device import select_device
from .engine import DEFAULT_ENGINE, LearnerEnv, outcome
from .model import Policy, atomic_torch_save, batch_decisions, load_checkpoint


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
    count = len(observations)
    advantage = (advantage-advantage.mean()) / (advantage.std(unbiased=False)+1e-8)
    actions, old_logprobs = actions.to(device), old_logprobs.to(device)
    returns, advantage = returns.to(device), advantage.to(device)
    metrics = []
    stop = False
    for _ in range(args.epochs):
        for indices in torch.randperm(count).split(args.batch_size):
            batch = batch_decisions([observations[i] for i in indices.tolist()], device)
            distribution, values = policy(*batch)
            index = indices.to(device)
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
                kl = ((ratio-1)-logratio).mean().item()
                metrics.append((actor_loss.item(), critic_loss.item(), entropy.item(), kl))
                if kl > args.target_kl:
                    stop = True
                    break
        if stop:
            break
    return dict(zip(("actor_loss", "critic_loss", "entropy", "approx_kl"),
                    [sum(m[i] for m in metrics)/len(metrics) for i in range(4)]))


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
    p.add_argument("--total-steps", type=int, default=200_000, help="target total learner decisions, including a resumed run")
    p.add_argument("--envs", type=int, default=8)
    p.add_argument("--rollout-steps", type=int, default=128)
    p.add_argument("--players", type=int, default=2)
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
    if not 2 <= args.players <= 6 or args.envs > 128:
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
    policy = Policy(args.hidden).to(device)
    checkpoint = None
    step, iteration = 0, 0
    if args.resume:
        policy, checkpoint = load_checkpoint(args.resume, device)
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
    config.update(schema=SCHEMA, git_revision=git_revision(), map_split="train")
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
    try:
        for i in range(args.envs):
            env = LearnerEnv(args.engine, args.seed+step*101+i*1009, args.players, args.maps, args.opponent)
            env.snapshots = snapshots
            environments.append(env)
        with ThreadPoolExecutor(max_workers=args.envs) as pool, (args.output / "metrics.jsonl").open("a", buffering=1) as log, (args.output / "maps.jsonl").open("a", buffering=1) as map_log:
            def record_map(env):
                map_log.write(json.dumps({"step": step, "seat": env.seat, "map": env.engine.map_info})+"\n")
            observations = list(pool.map(lambda env: env.reset(), environments))
            for env in environments:
                record_map(env)
            while step < args.total_steps:
                policy.eval()
                rollout_obs, rollout_actions, rollout_logprobs = [], [], []
                rollout_values, rollout_rewards, rollout_dones = [], [], []
                # Keep a complete rectangular rollout for GAE, including the last update.
                for _ in range(args.rollout_steps):
                    with torch.inference_mode():
                        distribution, values = policy(*batch_decisions(observations, device))
                        actions = distribution.sample()
                        logprobs = distribution.log_prob(actions)
                    rollout_obs.extend(observations)
                    rollout_actions.append(actions.cpu())
                    rollout_logprobs.append(logprobs.cpu())
                    rollout_values.append(values.cpu())
                    results = list(pool.map(lambda pair: pair[0].step(pair[1]), zip(environments, actions.cpu().tolist())))
                    rewards, dones, next_obs = [], [], []
                    for env, (obs, reward, terminated, truncated, info) in zip(environments, results):
                        if truncated:
                            # A limit should never be reached by this finite action set.
                            # Do not quietly train on incomplete games or count them as losses.
                            raise RuntimeError(f"episode hit engine limit: {info}; investigate rules before training")
                        rewards.append(reward)
                        dones.append(float(terminated))
                        if terminated:
                            episode_stats.append({"outcome": outcome(info["scores"], info["seat"]),
                                                  "score": info["scores"][info["seat"]], "return": info["episode_return"]})
                            obs = env.reset()
                            record_map(env)
                        next_obs.append(obs)
                    rollout_rewards.append(torch.tensor(rewards))
                    rollout_dones.append(torch.tensor(dones))
                    observations = next_obs
                    step += args.envs
                with torch.inference_mode():
                    _, last_values = policy(*batch_decisions(observations, device))
                advantage, returns = advantages(torch.stack(rollout_rewards), torch.stack(rollout_dones),
                                                torch.stack(rollout_values), last_values.cpu(), args.gamma, args.gae_lambda)
                policy.train()
                metrics = update_policy(policy, optimizer, rollout_obs, torch.stack(rollout_actions).flatten(),
                                        torch.stack(rollout_logprobs).flatten(), returns.flatten(), advantage.flatten(), args, device)
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
                print(json.dumps(metrics), flush=True)
                log.write(json.dumps(metrics)+"\n")
                payload = {"format_version": 1, "schema": SCHEMA, "config": config, "device_report": device_report,
                           "step": step, "iteration": iteration, "model": policy.state_dict(),
                           "optimizer": optimizer.state_dict(), "torch_rng": torch.get_rng_state(),
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
