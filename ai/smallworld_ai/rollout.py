"""On-policy collection without a barrier between unrelated games."""
from concurrent.futures import wait, FIRST_COMPLETED
import torch
from .model import batch_decisions
from .cuda_graph import GPU_LOCK
from .exploration import sample_training_actions


def collect_async(policy, environments, observations, pool, count, device, advance, on_result,
                  exploration_fraction=0.0, exploration_temperature=1.5, on_exploration_context=None,
                  exploration_learned_groups=.75, exploration_piece_prior=2., captured=None):
    """Keep each player's sequence ordered while other games advance freely.

    All actions use one frozen learner version. Every submitted action finishes
    before returning, so PPO never mixes pre-update and post-update experience.
    Per-environment sequences let GAE keep rewards with the correct player.
    """
    sequences = [[] for _ in environments]
    pending = {}
    ready = list(range(len(environments)))
    dispatched = 0
    while ready or pending:
        if ready and dispatched < count:
            indices = ready[:count-dispatched]
            ready = []
            with GPU_LOCK:
                batch = batch_decisions([observations[i] for i in indices], device, policy)
                with torch.inference_mode():
                    distribution, values = captured(policy,batch) if captured else policy(*batch)
                    retry_groups = torch.tensor([getattr(environments[i], 'exploration_retry_group', -1) for i in indices], device=device)
                    actions, logprobs, retry_after_rejection, context = sample_training_actions(policy, distribution, batch, exploration_fraction, exploration_temperature, retry_groups, exploration_learned_groups, exploration_piece_prior)
                    choices = torch.stack((actions,retry_after_rejection,context),-1).cpu().tolist()
            for j, (i, (action, retry, mode)) in enumerate(zip(indices, choices)):
                if on_exploration_context is not None:
                    on_exploration_context(environments[i], mode)
                encoded = tuple(x[j:j+1] for x in batch)
                future = pool.submit(advance, (environments[i], action))
                pending[future] = (i, encoded, actions[j], logprobs[j], values[j], retry)
            dispatched += len(indices)
        if not pending:
            break
        completed, _ = wait(pending, return_when=FIRST_COMPLETED)
        for future in completed:
            i, encoded, action, logprob, value, retry = pending.pop(future)
            result = future.result()
            obs, reward, terminated, truncated, info = result
            environments[i].exploration_retry_group = retry if info.get("rejected") and not terminated and not truncated else -1
            on_result(environments[i], result)
            observations[i] = obs
            sequences[i].append((encoded, action, logprob, value, reward, float(terminated)))
            if dispatched < count:
                ready.append(i)
    return sequences
