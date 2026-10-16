"""Batched masked PPO against random, greedy and frozen self-play opponents."""
import argparse
from collections import Counter, deque
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
from .cuda_graph import GPU_LOCK, CapturedInference
from .lifecycle import JsonlLog, prune_checkpoints
from .league import add_snapshot, OpponentLeague, policy_payload
from .exploration import PieceExploration, training_distribution, sample_training_actions
from .feedback import rejection_loss, ResponseHead
from .engine import DEFAULT_ENGINE, LearnerEnv, outcome
from .model import Policy, atomic_torch_save, batch_decisions, load_checkpoint, merge_decision_batches, policy_from_checkpoint


def advantages(rewards, dones, values, last_values, gamma, gae_lambda, turn_ends=None):
    """GAE follows the learner's decisions; an opponent never receives its credit."""
    result = torch.zeros_like(rewards)
    carry = torch.zeros_like(last_values)
    for t in reversed(range(rewards.shape[0])):
        next_values = last_values if t == rewards.shape[0]-1 else values[t+1]
        alive = 1.0-dones[t]
        delta = rewards[t] + gamma*next_values*alive - values[t]
        trace = gae_lambda if turn_ends is None else torch.where(turn_ends[t].bool(), gae_lambda, 1.0)
        carry = delta + gamma*trace*alive*carry
        result[t] = carry
    return result, result+values


def update_policy(policy, optimizer, observations, actions, old_logprobs, returns, advantage, args, device, acceptance=None, acceptance_head=None, acceptance_optimizer=None, exploration_context=None, survival_targets=None, response_targets=None, response_head=None, response_optimizer=None, expert_credit=None, expert_optimizer=None):
    # Raw observations remain accepted for callers outside the rollout loop.
    # The trainer supplies already encoded device tensors, reused for all epochs.
    packed = batch_decisions(observations, device, policy) if isinstance(observations, list) else observations
    count = packed[0].shape[0]
    advantage = (advantage-advantage.mean()) / (advantage.std(unbiased=False)+1e-8)
    actions, old_logprobs = actions.to(device), old_logprobs.to(device)
    returns, advantage = returns.to(device), advantage.to(device)
    metrics = []
    response_counts = []
    credit_losses = []
    expert_norms = []
    stop = False
    for _ in range(args.epochs):
        for index in torch.randperm(count).to(device).split(args.batch_size):
            batch = tuple(value.index_select(0, index) for value in packed)
            relational = getattr(policy, 'format_version', 0) in (5, 6, 7)
            if acceptance_head is not None or relational:
                outputs = (policy.training_forward(batch, actions[index], expert_credit=True) if expert_credit is not None
                           else policy.training_forward(batch, actions[index]))
                distribution, values, hidden = outputs[:3]
            else:
                distribution, values = policy(*batch)
            credit_loss = (expert_credit(policy, outputs[5], batch, actions[index], advantage[index])
                           if expert_credit is not None else values.new_zeros(()))
            survival_loss = values.new_zeros(())
            if relational:
                targets = survival_targets[index] if survival_targets is not None else torch.full_like(outputs[3], -1)
                valid = targets >= 0
                per_tile = torch.nn.functional.binary_cross_entropy_with_logits(outputs[3], targets.clamp_min(0), reduction='none')
                survival_loss = (per_tile*valid).sum()/valid.sum().clamp_min(1)
            response_loss = values.new_zeros(())
            if response_head is not None:
                response_loss, counts = response_head(outputs[4], response_targets[index])
                response_counts.append(counts)
            if acceptance_head is not None:
                prediction = acceptance_head(hidden).squeeze(-1)
                targets = acceptance[index]
                acceptance_loss = torch.nn.functional.binary_cross_entropy_with_logits(prediction, targets)
                acceptance_accuracy = ((prediction >= 0) == (targets >= .5)).float().mean()
            else:
                acceptance_loss = acceptance_accuracy = values.new_zeros(())
            rejection = (rejection_loss(distribution.logits, actions[index], acceptance[index], batch[2])
                         if getattr(args, 'rejection_loss', 0) else values.new_zeros(()))
            distribution = training_distribution(policy, distribution, batch, args.exploration_fraction, args.exploration_temperature, exploration_context[index] if exploration_context is not None else None,
                                                 getattr(args, 'exploration_learned_groups', .75), getattr(args, 'exploration_piece_prior', 2.))
            logprobs = distribution.log_prob(actions[index])
            logratio = logprobs-old_logprobs[index]
            ratio = logratio.exp()
            clipped = ratio.clamp(1-args.clip, 1+args.clip)
            actor_loss = -torch.minimum(ratio*advantage[index], clipped*advantage[index]).mean()
            critic_loss = (values-returns[index]).square().mean()
            entropy = distribution.entropy().mean()
            group_entropy = policy.group_entropy(distribution, batch[1]).mean() if hasattr(policy,"group_entropy") else 0
            loss = actor_loss+0.5*critic_loss-args.entropy*entropy-getattr(args,"group_entropy",0)*group_entropy+getattr(args,"acceptance_loss",0)*acceptance_loss+getattr(args,"survival_loss",0)*survival_loss+getattr(args,"rejection_loss",0)*rejection+getattr(args,"response_loss",0)*response_loss+getattr(args,"specialist_credit_loss",0)*credit_loss
            if not torch.isfinite(loss):
                raise RuntimeError("non-finite PPO loss; training stopped before saving invalid weights")
            optimizer.zero_grad(set_to_none=True)
            if acceptance_optimizer is not None:
                acceptance_optimizer.zero_grad(set_to_none=True)
            if response_optimizer is not None:
                response_optimizer.zero_grad(set_to_none=True)
            if expert_optimizer is not None:
                expert_optimizer.zero_grad(set_to_none=True)
            loss.backward()
            if getattr(args, "specialist_learning_rate", None) is not None:
                # Clip groups independently so critic or auxiliary gradients cannot
                # suppress the much smaller expert gradients.
                for group in optimizer.param_groups:
                    norm = torch.nn.utils.clip_grad_norm_(group["params"], 0.5)
                    if group.get("name") == "experts":
                        expert_norms.append(norm.detach())
            else:
                torch.nn.utils.clip_grad_norm_(policy.parameters(), 0.5)
            if expert_optimizer is not None:
                torch.nn.utils.clip_grad_norm_(expert_credit.parameters(), .5)
                expert_optimizer.step()
                credit_losses.append(credit_loss.detach())
            if acceptance_optimizer is not None:
                torch.nn.utils.clip_grad_norm_(acceptance_head.parameters(), .5)
                acceptance_optimizer.step()
            if response_optimizer is not None:
                torch.nn.utils.clip_grad_norm_(response_head.parameters(), .5)
                response_optimizer.step()
            optimizer.step()
            with torch.no_grad():
                kl = ((ratio-1)-logratio).mean()
                metrics.append(torch.stack((actor_loss, critic_loss, entropy, kl, acceptance_loss, acceptance_accuracy, survival_loss, rejection, response_loss)).detach())
                # Keep the same per-minibatch KL stop, but copy loss metrics only
                # once at the end instead of synchronizing for each scalar.
                if kl.item() > args.target_kl:
                    stop = True
                    break
        if stop:
            break
    result = dict(zip(("actor_loss", "critic_loss", "entropy", "approx_kl", "acceptance_loss", "acceptance_accuracy", "survival_loss", "rejection_loss", "response_loss"),
                    torch.stack(metrics).mean(0).cpu().tolist()))
    if credit_losses:
        result["specialist_credit_loss"] = float(torch.stack(credit_losses).mean())
    if expert_norms:
        result["specialist_expert_gradient_norm"] = float(torch.stack(expert_norms).mean())
    if response_counts:
        total, correct, changed, changed_correct, pieces_correct, pieces_changed, pieces_changed_correct = torch.stack(response_counts).sum(0).cpu().tolist()
        def fraction(n, d): return n/d if d else None
        result.update(response_owner_accuracy=fraction(correct,total),
                      response_owner_persistence_accuracy=fraction(total-changed,total),
                      response_changed_owner_accuracy=fraction(changed_correct,changed),
                      response_pieces_accuracy=fraction(pieces_correct,total),
                      response_pieces_persistence_accuracy=fraction(total-pieces_changed,total),
                      response_changed_pieces_accuracy=fraction(pieces_changed_correct,pieces_changed))
    return result


def step_and_reset(pair):
    """Reset finished episodes in their workers, preserving terminal rewards."""
    env, action = pair
    obs, reward, terminated, truncated, info = env.step(action)
    if terminated and not truncated:
        obs = env.reset()
    return obs, reward, terminated, truncated, info


def calibrate_group_head(policy, teacher, environments, observations, pool, args, device, record_map):
    """Transfer the previous learned strategy using experience, not move rules.

    The target mixes the teacher distribution with the student's initial equal
    action/piece-group distribution. No piece name is privileged. Only the new
    group head is fitted; transferred target and value weights stay unchanged.
    """
    if not args.bootstrap_steps:
        return observations
    teacher = copy.deepcopy(teacher).to(device).eval()
    policy.eval()
    batches, targets = [], []
    for _ in range((args.bootstrap_steps+args.envs-1)//args.envs):
        batch = batch_decisions(observations, device, policy)
        with torch.inference_mode():
            old, _ = teacher(*batch)
            initial, _ = policy(*batch)
            target = (1-args.bootstrap_mix)*old.probs + args.bootstrap_mix*initial.probs
            actions = old.sample().cpu().tolist()
        batches.append(batch)
        targets.append(target.clone())
        results = list(pool.map(step_and_reset, zip(environments, actions)))
        observations = []
        for env, (obs, _, done, truncated, _) in zip(environments, results):
            if truncated:
                raise RuntimeError("calibration episode reached the engine limit")
            if done:
                record_map(env)
            observations.append(obs)
    del teacher
    flags = [p.requires_grad for p in policy.parameters()]
    for p in policy.parameters():
        p.requires_grad_(False)
    for p in policy.group_actor.parameters():
        p.requires_grad_(True)
    optimizer = torch.optim.Adam(policy.group_actor.parameters(), lr=1e-3)
    losses = []
    try:
        for epoch in range(args.bootstrap_epochs):
            total = 0.0
            for i in torch.randperm(len(batches)).tolist():
                dist, _ = policy(*batches[i])
                loss = -(targets[i]*dist.logits).sum(-1).mean()
                optimizer.zero_grad(set_to_none=True)
                loss.backward()
                torch.nn.utils.clip_grad_norm_(policy.group_actor.parameters(), 1)
                optimizer.step()
                total += float(loss.detach())
            losses.append(total/len(batches))
    finally:
        for p, enabled in zip(policy.parameters(), flags):
            p.requires_grad_(enabled)
    report = dict(decisions=len(batches)*args.envs, epochs=args.bootstrap_epochs,
                  teacher_fraction=1-args.bootstrap_mix, cross_entropy=losses)
    (args.output/'calibration.json').write_text(json.dumps(report, indent=2)+"\n")
    print(json.dumps({'calibration': report}), flush=True)
    return observations


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
    p.add_argument("--upgrade-specialists", action="store_true", help="expand a format-6 checkpoint into separately trained race and trait experts")
    p.add_argument("--freeze-backbone", action=argparse.BooleanOptionalAction, default=None, help="specialist prototype freezes inherited perception by default")
    p.add_argument("--specialist-curriculum", action=argparse.BooleanOptionalAction, default=None, help="shared race/trait practice quotas; preserved on resume unless explicitly disabled")
    p.add_argument("--specialist-fraction", type=float, default=.75, help="fraction of games with assigned race/trait; remainder trains natural tribe choice")
    p.add_argument("--specialist-learning-rate", type=float, help="separate rate for race/trait experts; preserves named Adam moments on resume")
    p.add_argument("--specialist-adam-eps", type=float, default=1e-5, help="Adam epsilon for race/trait experts")
    p.add_argument("--specialist-credit-loss", type=float, default=0., help="train selected race/trait representations directly from realized rollout advantages")
    p.add_argument("--specialist-probe-size", type=int, default=128, help="positions per update for contribution ablations; 0 disables")
    p.add_argument("--upgrade-entity", action="store_true", help="expand a relational checkpoint, preserving weights and Adam moments")
    p.add_argument("--cuda-graphs", action="store_true", help="bounded captured inference; CPU and unsupported shapes use the ordinary path")
    p.add_argument("--upgrade-relational", action="store_true", help="expand a resumed joint policy without changing its initial actions or discarding Adam moments")
    p.add_argument("--response-loss", type=float, default=0., help="predict public board ownership and garrison changes after opponents play, from the current position")
    p.add_argument("--objective", choices=["score", "win"], default="score", help="score: own income plus outcome; win: final shared win credit with cancelling relative-score shaping")
    p.add_argument("--survival-loss", type=float, default=0., help="learn observed territory retention across opponents turns; no reward shaping")
    p.add_argument("--gae-clock", choices=["decision", "turn"], default="decision", help="apply GAE trace decay per click or per completed learner turn")
    p.add_argument("--warm-start", type=Path, help="transfer a legacy policy to graph, or a graph policy to hierarchical")
    p.add_argument("--bootstrap-steps", type=int, default=2048, help="teacher experience to calibrate a new hierarchical head")
    p.add_argument("--bootstrap-epochs", type=int, default=8)
    p.add_argument("--bootstrap-mix", type=float, default=0.15, help="generic action/piece-group exploration in the calibration targets")
    p.add_argument("--architecture", choices=["graph", "hierarchical", "joint", "relational", "entity", "specialists"], default="graph")
    p.add_argument("--focus-fraction", type=float, default=0.25, help="fraction of episodes assigning a curriculum tribe to a random learner seat")
    p.add_argument("--exploration-bonus", type=float, default=0.0, help="initial curiosity reward for a successful first use of each piece type per game; training only")
    p.add_argument("--exploration-budget", type=float, default=0.04, help="maximum curiosity reward per game")
    p.add_argument("--exploration-fraction", type=float, default=0.0, help="permanent training mixture fraction exploring action/piece groups")
    p.add_argument("--exploration-temperature", type=float, default=1.5, help="soften learned destination preferences during exploratory choices")
    p.add_argument("--exploration-learned-groups", type=float, default=.75, help="learned fraction of exploratory group selection; the rest is a permanent sampling floor")
    p.add_argument("--exploration-piece-prior", type=float, default=2., help="generic training-only prior for trying actions with a stack; never changes exported inference")
    p.add_argument("--acceptance-loss", type=float, default=0.0, help="joint-model auxiliary loss predicting engine acceptance of the attempted action; no rule labels or inference mask")
    p.add_argument("--rejection-penalty", type=float, default=0.001, help="generic feedback for an engine-rejected action")
    p.add_argument("--rejection-loss", type=float, default=0.0, help="directly learn lower base-policy probability for observed rejected actions; no tactical labels or inference mask")
    p.add_argument("--rules", choices=["legacy", "surface"], default="legacy")
    p.add_argument("--total-steps", type=int, default=200_000, help="target total learner decisions, including resume; 0 runs until interrupted")
    p.add_argument("--async-rollouts", action=argparse.BooleanOptionalAction, default=False, help="advance ready games without waiting for the slowest opponent turn")
    p.add_argument("--envs", type=int, default=8)
    p.add_argument("--rollout-steps", type=int, default=256)
    p.add_argument("--players", type=int, default=2)
    p.add_argument("--player-counts", type=int, nargs="+", help="vary table size between episodes")
    p.add_argument("--maps", nargs="+", default=["generated"])
    p.add_argument("--opponent-device", choices=["auto", "cpu", "cuda"], default="cpu", help="optional batched CUDA opponents; benchmark before enabling")
    p.add_argument("--opponent", choices=["random", "greedy", "mixed", "selfplay"], default="mixed")
    p.add_argument("--league", type=Path, help="manifest of immutable, permanent opponent checkpoints; embedded in future training checkpoints")
    p.add_argument("--league-fraction", type=float, default=.75, help="fraction of self-play opponents drawn from permanent rivals instead of changing snapshots")
    p.add_argument("--seed", type=int, default=1)
    p.add_argument("--hidden", type=int, default=128)
    p.add_argument("--batch-size", type=int, default=256)
    p.add_argument("--epochs", type=int, default=4)
    p.add_argument("--learning-rate", type=float, default=3e-4)
    p.add_argument("--gamma", type=float, default=1.0)
    p.add_argument("--gae-lambda", type=float, default=0.99)
    p.add_argument("--clip", type=float, default=0.2)
    p.add_argument("--entropy", type=float, default=0.01)
    p.add_argument("--group-entropy", type=float, default=0.02, help="exploration across action/piece groups without named-piece preferences")
    p.add_argument("--target-kl", type=float, default=0.03)
    p.add_argument("--snapshot-every", type=int, default=10, help="updates between frozen self-play snapshots")
    p.add_argument("--save-every", type=int, default=10, help="updates between numbered checkpoints; latest saved each update")
    p.add_argument("--keep-checkpoints", type=int, default=0, help="retain this many numbered checkpoints; 0 keeps all (latest, parents and exports are always retained)")
    p.add_argument("--torch-threads", type=int, default=1)
    return p


def run(args):
    for key in ("envs", "rollout_steps", "batch_size", "epochs", "snapshot_every", "save_every", "torch_threads"):
        if getattr(args, key) < 1:
            raise ValueError(f"{key} must be positive")
    if args.total_steps < 0 or args.keep_checkpoints < 0:
        raise ValueError("total_steps and keep_checkpoints must be non-negative")
    if not 2 <= args.players <= 6 or args.envs > 128 or any(not 2 <= n <= 6 for n in (args.player_counts or [])):
        raise ValueError("players must be 2..6 and envs at most 128")
    if not 0 < args.gamma <= 1 or not 0 <= args.gae_lambda <= 1 or not 0 < args.clip < 1:
        raise ValueError("invalid gamma, GAE lambda or clip")
    if args.objective == 'win' and (args.gamma != 1 or args.exploration_bonus != 0 or args.rejection_penalty != 0):
        raise ValueError('win objective requires gamma=1, exploration-bonus=0 and rejection-penalty=0; sampling exploration stays available')
    if not 0 <= args.response_loss < float('inf'):
        raise ValueError('response loss must be nonnegative and finite')
    if args.bootstrap_steps < 0 or args.bootstrap_epochs < 1 or not 0 <= args.bootstrap_mix <= 1:
        raise ValueError("invalid calibration parameters")
    if not 0 <= args.focus_fraction <= 1 or args.rejection_penalty < 0:
        raise ValueError("invalid focus fraction or rejection penalty")
    if args.exploration_bonus < 0 or args.exploration_budget < 0 or not 0 <= args.exploration_fraction <= 1 or args.exploration_temperature < 1:
        raise ValueError("invalid exploration options")
    if not 0 <= args.exploration_learned_groups < 1 or args.exploration_piece_prior < 1 or not 0 <= args.league_fraction <= 1:
        raise ValueError('invalid exploration prior or opponent league fraction')
    if args.architecture in ("hierarchical", "joint", "relational", "entity", "specialists") and args.rules != "surface":
        raise ValueError("hierarchical policies require surface observations")
    if args.acceptance_loss < 0 or not 0 <= args.rejection_loss < float('inf'):
        raise ValueError("acceptance and rejection losses must be nonnegative and finite")
    if args.learning_rate <= 0 or args.target_kl <= 0 or args.entropy < 0 or args.group_entropy < 0:
        raise ValueError("invalid optimizer parameters")
    if args.cuda_graphs and not args.async_rollouts:
        raise ValueError("--cuda-graphs requires --async-rollouts to coordinate capture")
    torch.set_num_threads(args.torch_threads)
    torch.manual_seed(args.seed)
    random.seed(args.seed)
    device, device_report = select_device(args.device)
    print(json.dumps({"device": device_report}), flush=True)
    if args.rules == "surface":
        from .graph import GraphPolicy
        if args.architecture in ("hierarchical", "joint", "relational", "entity", "specialists"):
            from .hierarchical import HierarchicalPolicy
            if args.architecture == "specialists":
                from .specialists import SpecialistPolicy
                policy = SpecialistPolicy(args.hidden).to(device)
            elif args.architecture == "entity":
                from .entity import EntityPolicy
                policy = EntityPolicy(args.hidden).to(device)
            elif args.architecture == "relational":
                from .relational import RelationalPolicy
                policy = RelationalPolicy(args.hidden).to(device)
            else:
                policy = HierarchicalPolicy(args.hidden, joint=args.architecture == "joint").to(device)
        else:
            policy = GraphPolicy(args.hidden).to(device)
    else:
        policy = Policy(args.hidden).to(device)
    if args.warm_start:
        if args.rules != "surface" or args.resume:
            raise ValueError("warm-start requires surface rules and no resume")
        teacher, _ = load_checkpoint(args.warm_start)
        policy.warm_start(teacher.to(device))
    checkpoint = None
    upgrade_teacher = None
    step, iteration = 0, 0
    if args.resume:
        policy, checkpoint = load_checkpoint(args.resume, device)
        if (policy.schema == SCHEMA) != (args.rules == "legacy"):
            raise ValueError("resume schema does not match --rules")
        args.hidden = policy.hidden
        if args.upgrade_relational and getattr(policy, 'format_version', 0) not in (5, 6, 7):
            from .relational import RelationalPolicy
            upgrade_teacher = policy
            policy = RelationalPolicy(policy.hidden).to(device)
            policy.warm_start(upgrade_teacher)
        if args.upgrade_entity and getattr(policy, 'format_version', 0) != 6:
            if upgrade_teacher is not None:
                raise ValueError('upgrade relational and entity in separate checkpointed steps')
            from .entity import EntityPolicy
            upgrade_teacher = policy
            policy = EntityPolicy(policy.hidden).to(device)
            policy.warm_start(upgrade_teacher)
        if args.upgrade_specialists and getattr(policy, 'format_version', 0) != 7:
            if upgrade_teacher is not None:
                raise ValueError('upgrade to specialists from a saved format-6 checkpoint')
            from .specialists import SpecialistPolicy
            upgrade_teacher = policy
            policy = SpecialistPolicy(policy.hidden).to(device)
            policy.warm_start(upgrade_teacher)
        args.architecture = "specialists" if getattr(policy, "format_version", 0) == 7 else "entity" if getattr(policy, "format_version", 0) == 6 else "relational" if getattr(policy, "format_version", 0) == 5 else "joint" if getattr(policy,"format_version",0)==4 else "hierarchical" if hasattr(policy, "group_actor") else "graph"
        step, iteration = checkpoint["step"], checkpoint["iteration"]
        torch.set_rng_state(checkpoint["torch_rng"])
        if device.type == "cuda" and checkpoint.get("cuda_rng"):
            torch.cuda.set_rng_state_all(checkpoint["cuda_rng"])
        random.setstate(checkpoint["python_rng"])
    if args.upgrade_specialists and not args.resume:
        raise ValueError('--upgrade-specialists requires --resume')
    if getattr(policy, 'format_version', 0) == 7:
        freeze = policy.backbone_frozen if args.resume or args.warm_start else False
        policy.freeze_backbone(freeze if args.freeze_backbone is None else args.freeze_backbone)
        args.freeze_backbone = policy.backbone_frozen
    elif args.freeze_backbone:
        raise ValueError('--freeze-backbone requires a specialist policy')
    if args.specialist_curriculum is None:
        args.specialist_curriculum = bool((checkpoint or {}).get('specialist_curriculum'))
    if args.specialist_curriculum and args.rules != 'surface':
        raise ValueError('specialist curriculum requires surface rules')
    if args.upgrade_entity and not args.resume:
        raise ValueError("--upgrade-entity requires --resume")
    if args.upgrade_relational and not args.resume:
        raise ValueError('--upgrade-relational requires --resume')
    if args.survival_loss < 0 or (args.survival_loss and getattr(policy, 'format_version', 0) not in (5, 6, 7)):
        raise ValueError('survival learning requires a relational policy and a nonnegative weight')
    from .specialist_learning import optimizer_for, ExpertCredit
    if not 0 <= args.specialist_credit_loss < float('inf') or args.specialist_probe_size < 0:
        raise ValueError('specialist credit loss and probe size must be nonnegative and finite')
    if args.specialist_credit_loss and getattr(policy, 'format_version', 0) != 7:
        raise ValueError('specialist credit learning requires format 7')
    optimizer = optimizer_for(policy, args, checkpoint, upgrade_teacher)
    expert_credit = expert_optimizer = None
    if args.specialist_credit_loss:
        expert_credit = ExpertCredit().to(device)
        rate = args.specialist_learning_rate or args.learning_rate
        expert_optimizer = torch.optim.Adam(expert_credit.parameters(), lr=rate, eps=args.specialist_adam_eps)
        if checkpoint and checkpoint.get('specialist_credit'):
            expert_credit.load_state_dict(checkpoint['specialist_credit']['model'])
            expert_optimizer.load_state_dict(checkpoint['specialist_credit']['optimizer'])
            for group in expert_optimizer.param_groups:
                group.update(lr=rate, eps=args.specialist_adam_eps)
    acceptance_head = acceptance_optimizer = None
    if args.acceptance_loss:
        if getattr(policy, "format_version", 0) not in (4, 5, 6, 7):
            raise ValueError("acceptance learning requires a joint policy")
        acceptance_head = torch.nn.Linear(policy.hidden, 1).to(device)
        acceptance_optimizer = torch.optim.Adam(acceptance_head.parameters(), lr=args.learning_rate, eps=1e-5)
        if checkpoint and checkpoint.get("acceptance"):
            acceptance_head.load_state_dict(checkpoint["acceptance"]["model"])
            acceptance_optimizer.load_state_dict(checkpoint["acceptance"]["optimizer"])
            for group in acceptance_optimizer.param_groups:
                group["lr"] = args.learning_rate
    response_head = response_optimizer = None
    if args.response_loss:
        if getattr(policy, 'format_version', 0) not in (5, 6, 7):
            raise ValueError('opponent response learning requires a relational policy')
        from .relational import WIDTH
        response_head = ResponseHead(WIDTH).to(device)
        response_optimizer = torch.optim.Adam(response_head.parameters(), lr=args.learning_rate, eps=1e-5)
        if checkpoint and checkpoint.get('opponent_response'):
            response_head.load_state_dict(checkpoint['opponent_response']['model'])
            response_optimizer.load_state_dict(checkpoint['opponent_response']['optimizer'])
            for group in response_optimizer.param_groups:
                group['lr'] = args.learning_rate
    config = {k: str(v) if isinstance(v, Path) else v for k, v in vars(args).items()}
    config.update(schema=policy.schema, git_revision=git_revision(), map_split="train")
    config["trainer_sha256"] = hashlib.sha256(b"".join(
        Path(__file__).with_name(name).read_bytes() for name in ("train.py", "model.py", "engine.py", "graph.py", "hierarchical.py", "inference.py", "rollout.py", "lifecycle.py", "exploration.py", "league.py", "relational.py", "feedback.py", "entity.py", "cuda_graph.py", "specialists.py", "curriculum.py", "specialist_learning.py")
    )).hexdigest()
    try:
        config["map_generator_revision"] = subprocess.check_output(["git", "rev-parse", "HEAD"], cwd=Path(__file__).resolve().parents[2]/"smallworld-map-generator", text=True).strip()
    except (OSError, subprocess.CalledProcessError):
        config["map_generator_revision"] = "unknown"
    args.output.mkdir(parents=True, exist_ok=True)
    # Preserve the initial config; each resume gets its own invocation record.
    (args.output / f"config-{step}-{time.time_ns()}.json").write_text(json.dumps(config, indent=2)+"\n")
    environments = []
    inference = None
    captured = CapturedInference(enabled=args.cuda_graphs)
    if args.opponent_device == "cuda" or (args.opponent_device == "auto" and device.type == "cuda"):
        from .inference import BatchedInference
        inference = BatchedInference(device if device.type == "cuda" else "cuda", captured=captured)
    started = time.monotonic()
    start_step = step
    snapshots = [copy.deepcopy(policy).cpu().eval()]
    if checkpoint and checkpoint.get("opponent_pool"):
        snapshots = [policy_from_checkpoint(saved).eval() for saved in checkpoint["opponent_pool"]]
    elif args.warm_start and teacher.schema == policy.schema:
        snapshots.insert(0, teacher.cpu().eval())
    league_snapshots_seen = (checkpoint or {}).get("league_snapshots_seen", max(0, len(snapshots)-1))
    league = OpponentLeague(args.league, (checkpoint or {}).get('permanent_league'), policy.schema)
    episode_stats = deque(maxlen=100)
    episodes_completed = (checkpoint or {}).get("episodes_completed", 0)
    start_episodes = episodes_completed
    coverage = {key: set() for key in ("races", "traits", "powers", "actions")}
    if checkpoint:
        for key, values in checkpoint.get("coverage", {}).items():
            coverage[key].update(values)
    action_counts = {key:Counter((checkpoint or {}).get("action_counts", {}).get(key, {})) for key in ("available", "chosen", "successful", "rejected")}
    exploration = PieceExploration(args.exploration_bonus, args.exploration_budget,
                                   step=step, state=(checkpoint or {}).get("exploration"),
                                   successful=action_counts["successful"]) if args.exploration_bonus else None
    rejected_total = checkpoint.get("rejected_total", 0) if checkpoint else 0
    player_count_stats = copy.deepcopy((checkpoint or {}).get('player_count_stats', {}))
    player_count_stats_start = (checkpoint or {}).get('player_count_stats_start', step)
    specialist_curriculum = None
    if args.specialist_curriculum:
        from .curriculum import SpecialistCurriculum
        specialist_curriculum = SpecialistCurriculum(args.seed, args.player_counts or [args.players], args.specialist_fraction,
                                                     (checkpoint or {}).get('specialist_curriculum'))
    cursors = checkpoint.get("curriculum_cursors", []) if checkpoint else []
    try:
        for i in range(args.envs):
            env = LearnerEnv(args.engine, args.seed+step*101+i*1009, args.player_counts or args.players, args.maps, args.opponent,
                             max_steps=8192 if args.rules == "surface" else 4096, rules=args.rules,
                             curriculum_start=cursors[i] if len(cursors)==args.envs else max(cursors, default=0)+i, curriculum_stride=args.envs, focus_fraction=args.focus_fraction, rejection_penalty=args.rejection_penalty, objective=args.objective)
            if checkpoint and len(checkpoint.get("environment_episodes", [])) == args.envs:
                env.episodes = checkpoint["environment_episodes"][i]
            env.specialist_curriculum = specialist_curriculum
            env.exploration = exploration
            env.snapshots = snapshots
            env.opponent_inference = inference
            env.league, env.league_fraction = league, args.league_fraction
            environments.append(env)
        with ThreadPoolExecutor(max_workers=args.envs) as pool, JsonlLog(args.output / "metrics.jsonl") as log, JsonlLog(args.output / "maps.jsonl") as map_log:
            def record_map(env):
                map_log.write(json.dumps({"step": step, "seat": env.seat, "map": env.engine.map_info})+"\n")
            observations = list(pool.map(lambda env: env.reset(), environments))
            for env in environments:
                record_map(env)
            if args.warm_start and getattr(policy, "format_version", 0) == 3:
                observations = calibrate_group_head(policy, teacher, environments, observations, pool, args, device, record_map)
                snapshots.append(copy.deepcopy(policy).cpu().eval())
            while args.total_steps == 0 or step < args.total_steps:
                rollout_started = time.monotonic()
                collection_metrics = {}
                if device.type == "cuda":
                    torch.cuda.reset_peak_memory_stats(device)
                policy.eval()
                rollout_batches, rollout_actions, rollout_logprobs = [], [], []
                rollout_values, rollout_rewards, rollout_dones = [], [], []
                accepted_by_env = {id(env): [] for env in environments}
                accepted_in_order = []
                feedback_by_env = {id(env): [] for env in environments}
                feedback_in_order = []
                exploration_by_env = {id(env): [] for env in environments}
                exploration_in_order = []
                def record_exploration(env, group):
                    exploration_by_env[id(env)].append(group)
                    exploration_in_order.append(group)
                def record_result(env, result):
                    obs, reward, terminated, truncated, info = result
                    nonlocal rejected_total, step, episodes_completed
                    if truncated:
                        raise RuntimeError(f"episode hit engine limit: {info}; investigate rules before training")
                    if specialist_curriculum:
                        specialist_curriculum.record(info, terminated)
                    accepted = float(not info.get("rejected", False))
                    accepted_by_env[id(env)].append(accepted)
                    accepted_in_order.append(accepted)
                    feedback = (info.get('turn_ended', False), info.get('survival'), info.get('opponent_response'))
                    feedback_by_env[id(env)].append(feedback)
                    feedback_in_order.append(feedback)
                    rejected_total += int(info.get("rejected", False))
                    action = info["action"]; key = action["kind"]+":"+action.get("stack", "")
                    action_counts["chosen"][key] += 1
                    action_counts["rejected" if info.get("rejected") else "successful"][key] += 1
                    action_counts["available"].update(info["available"])
                    for key, items in info.get("coverage", {}).items():
                        coverage[key].update(items)
                    step += 1
                    if terminated:
                        episodes_completed += 1
                        scores, seat = info['scores'], info['seat']
                        bucket = player_count_stats.setdefault(str(len(scores)), dict(games=0, win_credit=0., score_total=0., return_total=0.))
                        bucket['games'] += 1
                        bucket['win_credit'] += 1/scores.count(scores[seat]) if scores[seat] == max(scores) else 0.
                        bucket['score_total'] += scores[seat]
                        bucket['return_total'] += info['episode_return']
                        episode_stats.append({"outcome": outcome(info["scores"], info["seat"]),
                                              "score": info["scores"][info["seat"]], "return": info["episode_return"]})
                        record_map(env)

                if args.async_rollouts:
                    from .rollout import collect_async
                    sequences = collect_async(policy, environments, observations, pool,
                                              args.envs*args.rollout_steps, device, step_and_reset, record_result,
                                              args.exploration_fraction, args.exploration_temperature, record_exploration,
                                              args.exploration_learned_groups, args.exploration_piece_prior, captured=captured)
                    with GPU_LOCK, torch.inference_mode():
                        _, bootstrap = policy(*batch_decisions(observations, device, policy))
                    collection_metrics["environment_decisions"] = [len(seq) for seq in sequences]
                    advs, rets = [], []
                    for i, sequence in enumerate(sequences):
                        if not sequence:
                            continue
                        batches, actions, logprobs, values, rewards, dones = zip(*sequence)
                        advantage, returns = advantages(torch.tensor(rewards).unsqueeze(1), torch.tensor(dones).unsqueeze(1),
                                                        torch.stack(values).cpu().unsqueeze(1), bootstrap[i:i+1].cpu(), args.gamma, args.gae_lambda,
                                                        torch.tensor([item[0] for item in feedback_by_env[id(environments[i])]]).unsqueeze(1) if args.gae_clock == "turn" else None)
                        rollout_batches.extend(batches)
                        rollout_actions.extend(actions)
                        rollout_logprobs.extend(logprobs)
                        advs.append(advantage.flatten()); rets.append(returns.flatten())
                    advantage, returns = torch.cat(advs), torch.cat(rets)
                    del sequences
                else:
                    # Keep a complete rectangular rollout for GAE, including the last update.
                    for _ in range(args.rollout_steps):
                        # Create normal tensors outside inference_mode: the same
                        # cached inputs will later be used for gradient computation.
                        with GPU_LOCK:
                            batch = batch_decisions(observations, device, policy)
                            with torch.inference_mode():
                                distribution, values = captured(policy,batch)
                                retry_groups = torch.tensor([env.exploration_retry_group for env in environments], device=device)
                                actions, logprobs, retry_after_rejection, context = sample_training_actions(policy, distribution, batch, args.exploration_fraction, args.exploration_temperature, retry_groups,
                                                                                                           args.exploration_learned_groups, args.exploration_piece_prior)
                        for env, mode in zip(environments, context.cpu().tolist()):
                            record_exploration(env, mode)
                        retry_after_rejection = retry_after_rejection.cpu().tolist()
                        rollout_batches.append(batch)
                        rollout_actions.append(actions)
                        rollout_logprobs.append(logprobs)
                        rollout_values.append(values)
                        # Actions are the only per-step GPU result the CPU engines
                        # need. Values are copied together when computing GAE.
                        results = list(pool.map(step_and_reset, zip(environments, actions.cpu().tolist())))
                        rewards, dones, next_obs = [], [], []
                        for env, result, retry in zip(environments, results, retry_after_rejection):
                            env.exploration_retry_group = retry if result[4].get("rejected") and not result[2] and not result[3] else -1
                            record_result(env, result)
                            obs, reward, terminated, truncated, info = result
                            rewards.append(reward)
                            dones.append(float(terminated))
                            next_obs.append(obs)
                        rollout_rewards.append(torch.tensor(rewards))
                        rollout_dones.append(torch.tensor(dones))
                        observations = next_obs
                    with torch.inference_mode():
                        _, last_values = policy(*batch_decisions(observations, device, policy))
                    advantage, returns = advantages(torch.stack(rollout_rewards), torch.stack(rollout_dones),
                                                    torch.stack(rollout_values).cpu(), last_values.cpu(), args.gamma, args.gae_lambda,
                                                    torch.tensor([item[0] for item in feedback_in_order]).reshape(args.rollout_steps, args.envs) if args.gae_clock == "turn" else None)
                rollout_seconds = time.monotonic() - rollout_started
                policy.train()
                update_started = time.monotonic()
                packed = merge_decision_batches(rollout_batches)
                # Release the per-step buffers once the contiguous rollout owns
                # the inputs, keeping the extra CUDA memory bounded to one rollout.
                rollout_batches.clear()
                accepted = ([value for env in environments for value in accepted_by_env[id(env)]] if args.async_rollouts else accepted_in_order)
                acceptance = torch.tensor(accepted, device=device) if acceptance_head is not None or args.rejection_loss else None
                contexts = ([value for env in environments for value in exploration_by_env[id(env)]] if args.async_rollouts else exploration_in_order)
                exploration_context = torch.tensor(contexts, dtype=torch.long, device=device)
                feedback = ([item for env in environments for item in feedback_by_env[id(env)]] if args.async_rollouts else feedback_in_order)
                survival_targets = None
                if getattr(policy, 'format_version', 0) in (5, 6, 7):
                    survival_targets = torch.full(packed[5].shape, -1., device=device)
                    for row, (_, labels, _) in enumerate(feedback):
                        if labels is not None:
                            survival_targets[row, :len(labels)] = torch.tensor(labels, device=device)
                response_targets = None
                if response_head is not None:
                    response_targets = torch.full((*packed[5].shape, 3), -1, dtype=torch.long, device=device)
                    for row, (_, _, labels) in enumerate(feedback):
                        if labels is not None:
                            values = list(zip(labels['owner'], labels['pieces'], labels['owner_before']))
                            response_targets[row, :len(values)] = torch.tensor(values, device=device)
                metrics = update_policy(policy, optimizer, packed, torch.stack(rollout_actions).flatten(),
                                        torch.stack(rollout_logprobs).flatten(), returns.flatten(), advantage.flatten(), args, device, acceptance, acceptance_head, acceptance_optimizer, exploration_context, survival_targets, response_targets, response_head, response_optimizer, expert_credit, expert_optimizer)
                if getattr(policy, "format_version", 0) == 7 and args.specialist_probe_size:
                    from .specialist_learning import contribution_metrics
                    metrics.update(contribution_metrics(policy, packed, args.specialist_probe_size))
                metrics.update(collection_metrics)
                metrics.update(captured.stats())
                metrics['response_tiles'] = sum(len(labels['owner']) for _, _, labels in feedback if labels is not None)
                metrics['completed_learner_turns'] = sum(item[0] for item in feedback)
                metrics['survival_tiles'] = sum(sum(v >= 0 for v in labels) for _, labels, _ in feedback if labels is not None)
                metrics['survival_lost_tiles'] = sum(sum(v == 0 for v in labels) for _, labels, _ in feedback if labels is not None)
                if device.type == "cuda":
                    metrics["peak_cuda_memory_mb"] = round(torch.cuda.max_memory_allocated(device)/1e6, 1)
                metrics.update(rollout_seconds=round(rollout_seconds, 4),
                               update_seconds=round(time.monotonic() - update_started, 4))
                del packed
                iteration += 1
                if iteration % args.snapshot_every == 0:
                    league_snapshots_seen = add_snapshot(snapshots, copy.deepcopy(policy).cpu().eval(), league_snapshots_seen)
                metrics["exploration_successes"] = sum(c != -1 and a > 0 for c, a in zip(contexts, accepted))
                metrics["exploration_starts"] = sum(context == -2 for context in contexts)
                metrics["exploration_retry_decisions"] = sum(context >= 0 for context in contexts)
                metrics["opponent_pool_size"] = len(snapshots)
                metrics['results_by_player_count'] = {
                    players: dict(games=row['games'], win_credit=row['win_credit']/row['games'],
                                  mean_score=row['score_total']/row['games'], mean_return=row['return_total']/row['games'])
                    for players, row in sorted(player_count_stats.items()) if row['games']}
                metrics['player_count_stats_start_step'] = player_count_stats_start
                metrics['permanent_league'] = league.metrics()
                metrics["rollout_rejection_rate"] = 1-sum(accepted)/len(accepted)
                for mode, predicate in (("policy", lambda c: c == -1), ("exploration", lambda c: c == -2), ("retry", lambda c: c >= 0)):
                    outcomes = [a for c, a in zip(contexts, accepted) if predicate(c)]
                    metrics[mode+"_decisions"] = len(outcomes)
                    metrics[mode+"_rejection_rate"] = 1-sum(outcomes)/len(outcomes) if outcomes else None
                if inference:
                    metrics.update(inference.stats())
                if exploration:
                    metrics.update(exploration.metrics())
                metrics.update(step=step, iteration=iteration, episodes=episodes_completed,
                               decisions_per_second=round((step-start_step)/(time.monotonic()-started), 1),
                               games_per_second=round((episodes_completed-start_episodes)/(time.monotonic()-started), 3),
                               elapsed_seconds=round(time.monotonic()-started, 3), timestamp=time.time())
                recent = list(episode_stats)
                if recent:
                    metrics.update(recent_win_rate=sum(e["outcome"] == 1 for e in recent)/len(recent),
                                   recent_mean_score=sum(e["score"] for e in recent)/len(recent))
                if args.rules == "surface":
                    metrics.update(rejected_actions=rejected_total, coverage={k:len(v) for k,v in coverage.items()})
                    (args.output / "coverage.json").write_text(json.dumps({k:sorted(v) for k,v in coverage.items()}, indent=2)+"\n")
                if specialist_curriculum:
                    specialist_report = specialist_curriculum.report()
                    (args.output / 'specialists.json').write_text(json.dumps(specialist_report, indent=2)+'\n')
                    metrics['specialist_coverage'] = {kind: sum(row['decisions'] > 0 for row in rows.values()) for kind, rows in specialist_report.items()}
                metrics['decisions_by_phase'] = {phase: sum(count for key, count in action_counts['chosen'].items() if key.startswith(phase+':'))
                                                 for phase in ('pick','conquer','abandon','redeploy','redeploy_out','finish','decline')}
                print(json.dumps(metrics), flush=True)
                log.write(json.dumps(metrics)+"\n")
                (args.output / "actions.json").write_text(json.dumps(action_counts, indent=2)+"\n")
                pool_payload = [policy_payload(p) for p in snapshots]
                payload = {"format_version": getattr(policy, "format_version", 2 if args.rules == "surface" else 1), "schema": policy.schema, "config": config, "device_report": device_report,
                           "step": step, "iteration": iteration, "episodes_completed": episodes_completed, "model": policy.state_dict(),
                           "exploration": exploration.state_dict() if exploration else (checkpoint or {}).get("exploration"),
                           "acceptance": {"model": acceptance_head.state_dict(), "optimizer": acceptance_optimizer.state_dict()} if acceptance_head is not None else (checkpoint or {}).get("acceptance"),
                           "opponent_response": {"model": response_head.state_dict(), "optimizer": response_optimizer.state_dict()} if response_head is not None else (checkpoint or {}).get("opponent_response"),
                           "specialist_credit": {"model": expert_credit.state_dict(), "optimizer": expert_optimizer.state_dict()} if expert_credit is not None else (checkpoint or {}).get("specialist_credit"),
                           "league_snapshots_seen": league_snapshots_seen,
                           "player_count_stats": player_count_stats, "player_count_stats_start": player_count_stats_start,
                           "permanent_league": league.state_dict(),
                           "optimizer": optimizer.state_dict(), "torch_rng": torch.get_rng_state(),
                           "coverage": {k: sorted(v) for k,v in coverage.items()}, "rejected_total": rejected_total,
                           "action_counts": {k:dict(v) for k,v in action_counts.items()}, "opponent_pool": pool_payload,
                           "specialist_curriculum": specialist_curriculum.state_dict() if specialist_curriculum else None,
                           "curriculum_cursors": [e.curriculum for e in environments],
                           "environment_episodes": [e.episodes for e in environments],
                           "python_rng": random.getstate(),
                           "cuda_rng": torch.cuda.get_rng_state_all() if device.type == "cuda" else []}
                atomic_torch_save(payload, args.output / "latest.pt")
                if iteration % args.save_every == 0:
                    atomic_torch_save(payload, args.output / f"step-{step}.pt")
                    prune_checkpoints(args.output, args.keep_checkpoints)
    finally:
        if inference:
            inference.close()
        captured.clear()
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
