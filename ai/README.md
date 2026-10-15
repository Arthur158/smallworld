# Smallworld bots and GPU training

This is an end-to-end first training setup for **backend2**. The real Go game
engine supplies observations and available moves to a PyTorch PPO trainer.
Exported actors run directly inside the Go room worker. The game server needs
neither Python nor a GPU. Random and greedy bots are built in and need no model
files or training dependencies.

## Play with a bot

Rebuild the backend2 gateway, room workers and frontend using your normal
[deployment](../README.md). Create a room, choose a model under **Bot players**,
and click **Add bot**. Only the room host can add a bot, select its model, remove
it with the existing × button, or apply the bot rules preset. These checks also
run on the server. Set the player order with the existing arrows, then start.

For a game with only bots, choose **Watch as host** in the room, add a bot to
each empty seat, then start watching. You retain the map, model, and rules
controls without taking a player seat. Before starting, you can return to
playing whenever a seat is free. Existing training runs and exported models
use the same observations, actions, rules, and checkpoint format.

While watching a bots-only match, the host can adjust the speed from **0.25×**
to **3×**, **Pause** or **Resume** play, and use **Previous move** / **Next move**
while paused. Each step moves one bot decision, not an entire turn. Rewinding
keeps the game paused, and moving forward replays the same decision. Up to 100
previous moves are available, including the final move from the results dialog. Playback settings and history survive
reconnects and worker restarts; mixed human/bot games keep their usual pacing.

Adding the first bot applies the preset described below. Everyone plays with
those rules. The lobby explains this before adding a bot. The host can choose a
subset of supported races/traits; unsupported changes prevent the match from
starting and **Use bot rules** restores the preset. Loaded saves are validated
against the same rules. Bot seats and model IDs survive room-worker snapshots.
Bot actions pause while no human player or spectator is connected, and resume
after reconnect. Bots cannot become room hosts or keep a room alive after the
last human player or spectator leaves.

Built-in opponents:

- **Random bot** uniformly chooses an available move. This is the placeholder
  and baseline, not a claim of playing strength.
- **Greedy bot** prefers cheap conquests, larger armies and distributing spare
  troops, with a simple decline threshold. Use it as a second benchmark.
- Exported models appear by their configured names after restarting the workers.

## Prepare the training computer

Use Linux or WSL2, Git, Go 1.24+, Node.js 18+, and Python **3.10–3.13** with `venv` support.
Copy/clone the same Smallworld revision on the training computer. Redis,
PostgreSQL, Docker and the frontend are not needed to train. Setup initializes
the existing map-generator submodule. Its pure JavaScript generator runs locally;
no map server, browser, image generation or npm installation is needed.

For an NVIDIA machine, install a working NVIDIA driver first, then:

```bash
cd ~/github/smallworld
nvidia-smi
./ai/setup.sh --cuda
ai/.venv/bin/smallworld-doctor --device cuda
```

The setup uses PyTorch 2.8.0 and its CUDA 12.8 wheel. This is a pinned, tested
software baseline, not a requirement to install the CUDA toolkit on the host.
Choose a compatible wheel for older hardware/drivers using the
[official PyTorch installation instructions](https://pytorch.org/get-started/locally/)
and [version-specific wheel commands](https://pytorch.org/get-started/previous-versions/).
For example, a different supported wheel index can be passed explicitly:

```bash
./ai/setup.sh --cuda --torch-index https://download.pytorch.org/whl/cu126
```

Setup also supports `--cpu`. Without either flag, it checks `nvidia-smi` to
choose a wheel. `SMALLWORLD_PYTHON=/path/to/python3.12` selects a Python binary.
On Ubuntu/Debian, install `python3-venv` if creating the environment fails.

The doctor prints the GPU name, driver, PyTorch/CUDA versions, compute capability
and VRAM. It performs an allocation, matrix multiply, backward pass and device
synchronization. `--device cuda` fails if that check fails; it never silently
trains on CPU. `--device auto` selects usable CUDA when available and explicitly
reports CPU fallback if CUDA is unavailable. `--device cuda:1` selects a specific
GPU. One training process uses one GPU; distributed multi-GPU training is not
implemented.

## Launch training

First test the complete pipeline on CPU or GPU with a small run:

```bash
ai/.venv/bin/smallworld-train \
  --device auto --envs 2 --rollout-steps 64 --total-steps 2048 \
  --output ai/runs/smoke
```

A longer NVIDIA run:

```bash
ai/.venv/bin/smallworld-train \
  --device cuda --envs 16 --players 2 \
  --maps generated \
  --opponent mixed --total-steps 2000000 \
  --output ai/runs/ppo-v1
```

Keep a long run in your preferred terminal session manager. The command runs in
the foreground. `latest.pt` is saved atomically after each PPO update; numbered
checkpoints are retained every ten updates. Each update prints JSON metrics and
appends to `metrics.jsonl`, including throughput, losses, entropy and recent
training win rate. Those training wins are against a changing opponent mix;
use the separate evaluation command to measure strength.

The CPU runs one independent Go engine per environment. The GPU batches policy
inference and gradient updates. The network is small, so GPU utilization may be
low and CPU can be competitive. Increase `--envs` only while measured decisions
per second improves. There is no browser/image rendering during training.

Observations are encoded once per rollout step and reused on the selected device
throughout PPO's passes. Finished games reset in parallel, and frozen self-play
opponents compute only action scores. `rollout_seconds` and `update_seconds` in
each metrics record distinguish time gathering game experience from time
updating the network. Use decisions per second to compare configurations after
startup; GPU utilization alone does not measure useful training progress.
Invocation records include `trainer_sha256` to identify the training code even
when local changes have not been committed. Existing checkpoints remain usable
with the same resume command after a trainer update.

Resume up to a new **total** learner-decision count:

```bash
ai/.venv/bin/smallworld-train \
  --device cuda --envs 16 --players 2 \
  --maps generated --opponent mixed \
  --resume ai/runs/ppo-v1/latest.pt --total-steps 4000000 \
  --output ai/runs/ppo-v1
```

Resume restores actor, critic, optimizer, counters and Python/PyTorch RNGs. Live
Go episodes and the historical self-play pool restart; resuming is not bitwise
identical to an uninterrupted run. Pass the same training options when resuming.
The target rounds up to a complete rollout of `envs × rollout-steps` decisions.
Ctrl-C preserves the last completed update. Do not run two trainers against the
same output directory.

## Generalizing across maps

Every new training episode uses a fresh, seeded **Voronoi map from the same
generator as live rooms**. `--maps generated` is the default. The sampler varies
region count, aspect ratio, land shape, islands, water, terrain proportions and
cave/mine/magic placement. It checks connectivity, symmetric borders, enough land
and an accessible edge in every land component before the game starts. The
learner's seat, tribe market and game randomness also vary.

The policy sees terrain, ownership, armies, local neighbors and board summaries,
never a map name, seed, tile ID, coordinate, or candidate index. Its shared
candidate scorer handles variable board sizes without learning a fixed board
position. Tile labels are shuffled independently on each generated layout.
Tests verify that renaming tiles and reordering candidates preserves scores.

`maps.jsonl` records each episode's seed, generator settings, map hash and learner
seat. Invocation files also record the generator revision. Training and evaluation
use separate seed namespaces, even if the same numeric seed is supplied. Evaluation
plays the same generated map from each seat before advancing its seed. Pass
`--exclude-map-log ai/runs/ppo-v1/maps.jsonl` to reject exact training-map overlap.

Use fresh evaluation seeds and test the wider `stress` distribution, plus the
built-in boards that were absent from generated-only training:

```bash
ai/.venv/bin/smallworld-evaluate --model ppo-v1 --model-dir ai/models \
  --opponent greedy --games 200 --seed 2000000 --map-distribution stress \
  --exclude-map-log ai/runs/ppo-v1/maps.jsonl --output ai/runs/ppo-v1/stress.json
ai/.venv/bin/smallworld-evaluate --model ppo-v1 --model-dir ai/models \
  --opponent greedy --games 200 --maps map2players underground2players \
  --output ai/runs/ppo-v1/builtin-transfer.json
```

This reduces dependence on particular boards; it does **not guarantee no
overfitting**. A policy can still overfit the generator, rules, player count or
opponents. Keep an untouched final test seed set, compare against both baselines,
and check additional player counts. The stress sampler widens size and terrain
ranges; it is a deliberate distribution shift, not a proof of universal play.

## Evaluate, export and deploy

Export a versioned model. The exporter reads checkpoints with PyTorch's
[`weights_only` loader](https://docs.pytorch.org/docs/stable/generated/torch.load.html)
and writes data-only JSON; the live server never unpickles a checkpoint.

```bash
ai/.venv/bin/smallworld-export ai/runs/ppo-v1/latest.pt \
  --id ppo-v1 --name 'PPO v1' \
  --description 'Trained on fresh generated two-player maps' \
  --output ai/models/ppo-v1.json
```

Compare against **both** baselines on the same held-out seeds:

```bash
ai/.venv/bin/smallworld-evaluate --model ppo-v1 --model-dir ai/models \
  --opponent random --games 200 --seed 1000000 \
  --output ai/runs/ppo-v1/vs-random.json
ai/.venv/bin/smallworld-evaluate --model ppo-v1 --model-dir ai/models \
  --opponent greedy --games 200 --seed 1000000 \
  --output ai/runs/ppo-v1/vs-greedy.json
ai/.venv/bin/smallworld-evaluate --model random --opponent greedy \
  --games 200 --seed 1000000 --output ai/runs/random-vs-greedy.json
```

Evaluation runs the **exported Go actor**, rotates seats and reports wins, ties,
score margin and a 95% Wilson interval for strict win rate. All games must finish;
truncated matches raise an error. Use additional maps and player counts before
advertising a model for them. Two-player training does not establish competence
in six-player games. A successful short smoke run proves the pipeline, not bot
strength. Keep evaluation seeds separate from training and avoid tuning against
the same final test set indefinitely.

Copy the JSON file to `ai/models/` on the game server, then use the optional
Compose overlay on that server:

```bash
docker compose -f docker-compose.production.yml -f docker-compose.bots.yml up --build -d
```

After adding later models to the mounted directory, restart **both** workers:

```bash
docker compose -f docker-compose.production.yml -f docker-compose.bots.yml restart worker-0 worker-1
```

The model catalog is loaded and validated at startup. Keep immutable IDs such as
`ppo-v1`, `ppo-v2`; retain older files while rooms reference them. A missing model
in a restored room produces a visible message and switches that seat to the
random policy. Invalid files fail startup with a precise error. Room users send
catalog IDs, never file paths or URLs. Files are limited to 8 MiB and checked for
schema, tensor dimensions and non-finite weights.

For direct/systemd/NixOS deployment, set `BOT_MODEL_DIR` to an operator-managed,
readable directory on **every** worker and restart them. On NixOS the existing
`services.smallworld.environmentFile` supports this setting. Use a path such as
`/var/lib/smallworld/bot-models`, since the service protects home directories.
`BOT_MOVE_DELAY=350ms` controls visible pacing (allowed range: 100ms to 10s).
The Nix backend package also builds `botenv`.

For the existing local kind cluster, bake the same model set into the worker
image instead of using a size-limited ConfigMap. Whenever `ai/models` contains
exported JSON models, include the model layer on **every worker rebuild**, not
just the first bot deployment. The base worker image contains no trained models.
Keep all model files referenced by existing rooms; missing models fall back to
the random policy when they next move.

Run from the repository root, using a fresh release tag for each deployment:

```bash
worker_release=20261015-1
docker build -t "smallworld-roomworker:${worker_release}-base" \
  -f backend2/Dockerfile.worker backend2
docker build -t "smallworld-roomworker:${worker_release}-models" \
  --build-arg "WORKER_IMAGE=smallworld-roomworker:${worker_release}-base" \
  -f ai/deploy/Dockerfile.models ai/models
kind load docker-image "smallworld-roomworker:${worker_release}-models" --name game-local
kubectl --context kind-game-local -n game set image statefulset/room-worker \
  room-worker="smallworld-roomworker:${worker_release}-models"
kubectl --context kind-game-local -n game rollout status statefulset/room-worker
```

Load and deploy the final `-models` image. The explicit `WORKER_IMAGE` argument
ensures the model layer uses the newly built worker when using custom tags.

Rebuild/deploy the gateway and frontend too for the first bot-enabled release.
These are deployment instructions; the training tools do not modify a cluster.
The older root `docker-compose.yml` / `backend` stack does not support bots.

## Rules and learning design

Version `smallworld-bot-v1` supports these existing engine rules:

| Races | Traits |
| --- | --- |
| Dwarves, Elves, Giants, Goblins, Humans, Orcs | Alchemist, Commando, Flying, Forest, Hill, Hordes of |
| Ratmen, Shrubmen, Tritons, Trolls, White Ladies, Wizards | Merchant, Pillaging, Seafaring, Swamp, Underworld, Wealthy |

Powers and other expansion combinations are excluded from bot matches. Several
existing expansion hooks perform stateful checks or require special actions;
allowing them without explicit support would produce invalid training data.
The first adapter covers picking from the five visible tribes, conquest
(including a reinforcement-die attempt), abandonment, starting redeployment,
placing one unit, ending a turn and declining. Redeploy-out and special
opponent/entry/movement actions are outside this action abstraction.

Legality is computed without speculative moves. Every selected action is checked
again and executed by the existing `Handle*` methods. When the finite market has
no spare tribe, decline is refused to reserve entries for players who already
need one. A game has the engine's existing nine rounds. Per-game random state
makes this supported ruleset reproducible and survives room snapshots.

The observation is from the acting player's perspective: own coins, public
market entries, army/territory summaries and candidate-local terrain, ownership,
conquest cost and neighbor information. Opponent coins and hidden tribe identities
are excluded. A shared two-layer MLP scores each candidate, so no fixed tile IDs
or map dimensions are required. Padding is masked in training. The critic sees
state features. This is a compact feature policy, with limited global spatial
context; it is not a game-tree search or full-board graph network.

The trainer uses clipped [PPO](https://arxiv.org/abs/1707.06347), GAE, entropy
regularization, gradient clipping and a KL stop. One learner seat is randomized
per episode. `mixed` opponents combine random, greedy and frozen historical
self-play actors; the self-play pool retains eight snapshots. Credit is assigned
between consecutive decisions by the **same learner**, including all intervening
opponent turns. Reward is own coin change / 100 and terminal +1/-1/0 for a
strict win/loss/tie. The episode cap is an error diagnostic, never a fake victory.

The room worker advances one bot decision per room/tick on its existing serial
command loop. It persists before publishing a bot move or accepting another
command. Bots have no websocket, account, user-save links, or authentication
route. Only the worker can act for a bot seat. Removing the last human player
cleans up the room. Model inference is bounded local computation with no external
request that could hold up a game.

## Development and validation

```bash
(cd backend2 && go build -o ../ai/bin/botenv ./cmd/botenv)
(cd backend2 && go test -race ./...)
ai/.venv/bin/python -m unittest discover -s ai/tests -v
(cd frontend && CI=true npm test -- --watchAll=false --watchman=false --runInBand RoomBotControls)
(cd frontend && npm run build)
```

For the optional real Redis snapshot test, set `SMALLWORLD_TEST_REDIS_ADDR` to an
isolated test Redis. The test uses a unique snapshot key and does not flush Redis.
The Go tests exercise complete games, pure move enumeration, every offered move
from sampled states, snapshot consistency, seeded replay, hidden-score isolation,
permissions, seat management and end-of-game handling. Python tests exercise
padding masks, terminal credit assignment, device failure behavior, real training,
checkpoint resume, export, Python/Go logit agreement, generated-map diversity,
separate train/evaluation layouts, tile-label invariance and complete generated
games with 2–6 players.

`backend2/cmd/botenv` accepts one JSON request per line on stdin and returns one
JSON response per line. It has no HTTP listener. Operations are `reset`, `step`,
`observe` and `scores` (exported actor diagnostics). `step` accepts an action
index from the current decision or a registered policy ID. Responses include
`decision`, `done`, `truncated`, `scores`, and `steps`. Scores are trainer/evaluator
bookkeeping; only `decision.state` and candidate features enter the policy.

Add expansion support by auditing its checks for side effects, adding the missing
actions, and extending complete-match/property tests first. Changing feature
order, vocabulary, scaling or action semantics requires a new schema and new
models; do not reinterpret old model weights.

## Surface graph training (v2)

The current expanded curriculum is **surface only**: 35 surface races, 38 surface
traits, and no powers. Underground maps, races and traits are rejected by the v2
boundary. The base-game Underworld trait is a surface trait. Fresh generated maps
vary topology, biomes, labels, sky-island flags and neutral defenders. Held-out
map seeds remain separate from training seeds.

`--rules surface --hidden 256 --warm-start PATH` creates a graph policy with two
message-passing layers, per-region features, adjacency, the visible market,
public player reserves and special-action intents. A legacy checkpoint transfers
its shared learned weights; added features and graph weights then learn from PPO.
Invalid intents receive a penalty and are excluded until the next successful
action. Opponent scores and hidden future randomness are excluded from inputs.

```bash
PYTHONPATH=ai ai/.venv/bin/python -m smallworld_ai.train \
  --rules surface --device cuda --hidden 256 --envs 16 --rollout-steps 64 \
  --batch-size 128 --player-counts 2 3 4 --maps generated --opponent mixed \
  --warm-start ai/runs/previous/latest.pt --output ai/runs/surface-v2
```

Use `--resume ai/runs/surface-v2/latest.pt` instead of `--warm-start` to continue
v2 with optimizer state and curriculum progress. `latest.pt` is atomically saved
every update. Export with `python -m smallworld_ai.export`; the JSON is data-only.
The deployed Training 1–4 room bots still use the v1 adapter. V2 exports currently
run through botenv with `--rules surface`; integrating v2 room action intents and
validating playing strength are required before publishing a v2 selectable bot.

The October 15 GPU run lives at `~/github/smallworld/ai/runs/surface-v2-20261015`
on the training host, uses isolated sources in `ai/workspaces/surface-v2-20261015`,
and runs in tmux session `smallworld-surface-v2` until 23:00 Europe/Amsterdam.
`run.log`, `metrics.jsonl`, `coverage.json`, `maps.jsonl`, and `latest.pt` record its
progress. Its launcher inhibits idle/suspend and restarts failed trainer processes
from the latest saved update. The warm-start checkpoint is retained separately.
