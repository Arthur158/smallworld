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
throughout PPO's passes. Finished games reset in parallel. `--async-rollouts` lets ready games continue
while other games finish their opponent turns; rewards and GAE stay within each
learner sequence. All pending moves finish before the policy update. Numeric
observations use float32 arrays over the engine pipe, preserving exactly the
values used by PyTorch while avoiding JSON-number parsing.

`--opponent-device cuda` batches frozen opponent inference on the GPU; `cpu` is
the conservative default. CUDA batching helps with asynchronous games, but can
be slower with a synchronous collector. Benchmark the combination on the host. `rollout_seconds` and `update_seconds` in
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
Live Go episodes restart; the historical opponent pool is restored, including its
anchor model. Resuming is not bitwise identical to an uninterrupted run. Pass the same training options when resuming.
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
  --opponent greedy --games 200 --maps map2players \
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
worker_release=20261016-1
docker build -t "smallworld-roomworker:${worker_release}-base" \
  -f backend2/Dockerfile.worker backend2
docker build -t "smallworld-roomworker:${worker_release}-cpu" \
  --build-arg "WORKER_IMAGE=smallworld-roomworker:${worker_release}-base" \
  -f ai/deploy/Dockerfile.cpu-worker .
docker build -t "smallworld-roomworker:${worker_release}-models" \
  --build-arg "WORKER_IMAGE=smallworld-roomworker:${worker_release}-cpu" \
  -f ai/deploy/Dockerfile.models ai/models
kind load docker-image "smallworld-roomworker:${worker_release}-models" --name game-local
kubectl --context kind-game-local -n game set image statefulset/room-worker \
  room-worker="smallworld-roomworker:${worker_release}-models"
kubectl --context kind-game-local -n game rollout status statefulset/room-worker
```

## Modular specialist prototype (format 7)

This is an opt-in training architecture. Installing this code does not restart
the GPU service, replace the active run, publish a model, or deploy Kubernetes.
The saved Moneta checkpoint remains a format-6 baseline.

The prototype reuses Moneta's entity encoder and actor. It adds independent
weights for each of the 32 default-enabled surface races and 37 traits, an
explicit general-strategy component, and learned action, tribe-choice and
redeployment heads. Experts receive a proposed joint stack-and-target action
and the public board context. A learned coordinator combines their outputs;
there is no fixed vote or tactical rule about a named race, biome or piece.
Market alternatives are evaluated by the appropriate experts, but only the
selector/general components learn from the pick itself. Experts receive their
policy updates when their tribe actually participates in play.

The shared threat head predicts retention of currently owned tiles until the
learner's next turn, using actual outcomes as labels. It sees **global** entity
attention and an explicit global-state input: all tiles and stacks, every
player's public active/passive tribes and reserves, relative seat order, and
the five visible market entries. Adjacency is a learned attention bias, never
a visibility restriction. Distant pieces and players about to select a tribe
can therefore influence every tile's forecast. Unknown future market entries,
private opponent coins and future actions are not inputs. The threat estimate
is uncertain and conditioned on the training opponent pool; it is not an exact
attack calculator or a counterfactual simulation of every proposed placement.

Redeployment uses a dedicated head, repeatedly observing the changed board
after each placement/withdrawal and learning whether to finish. It consumes
the shared strategy/expert features and target/source threat forecasts. This
is sequential allocation, not yet a whole-allocation search. Winning remains
the objective; there is no fixed reward for protecting a particular terrain
or using a particular stack. Rejected actions retain the existing learning
feedback and retry behavior.

`--specialist-curriculum` works with both the old and new architecture. By
default, 75% of newly assigned episodes force a race/trait pair at a random
learner seat. One shared scheduler balances race, trait and player-count
assignments across environment workers, and varies pairings. The remaining
25% use ordinary tribe selection. Generated maps, opponents and seats vary.
Assignment counts are balanced; action counts cannot be guaranteed equal
because different games and tribes require different numbers of decisions.
`specialists.json` reports assignments, actual decisions, rejections and
completed-game exposure for every specialist, including those with zero data.
The scheduler and its RNG are saved and resumed with the checkpoint. Finished
game wins attributed to played races/traits are descriptive exposure statistics,
not causal estimates of each expert's strength.

The surface-only, zero-powers scope and default exclusions remain in force:
Storm Giants, Scarecrows, Escargots and Racketeering are not assigned. Separate
experts do not eliminate overfitting. Evaluate on fresh maps/seeds, unfamiliar
pairings and opponents, and every player count before promotion.

To start an isolated pilot from the saved Moneta, on a machine with the normal
AI dependencies installed:

```bash
cd ~/github/smallworld
(cd backend2 && go build -o ../ai/bin/botenv ./cmd/botenv)
PYTHONPATH=ai ai/.venv/bin/python -m smallworld_ai.train \
  --rules surface --resume ai/runs/imports/moneta-s5660672-81301adf/checkpoint.pt \
  --upgrade-specialists --specialist-curriculum --no-freeze-backbone \
  --device auto --envs 16 --rollout-steps 128 --batch-size 128 --epochs 2 \
  --player-counts 2 3 4 5 6 --maps generated --opponent mixed \
  --objective win --gamma 1 --gae-clock turn --rejection-penalty 0 \
  --exploration-fraction .1 --acceptance-loss .05 --survival-loss .05 \
  --response-loss .05 --rejection-loss .05 --learning-rate .0001 \
  --total-steps 5922816 --output ai/runs/specialists-pilot
```

This is 262,144 additional decisions from that specific parent step; the
trainer's `--total-steps` is an absolute target. `--device auto` performs the
existing CUDA availability/compute check and reports CPU fallback. Use
`--device cuda` to require NVIDIA CUDA and fail clearly if unavailable.
Dynamic expert dispatch currently uses ordinary PyTorch inference, even when
`--cuda-graphs` is enabled; GPU throughput must be measured separately.

The initial expanded policy and critic preserve the parent's outputs because
the residual output heads start at zero. On upgrade the inherited backbone is
frozen by default; pass `--no-freeze-backbone` (as above) to let perception keep
learning. A frozen pilot updates about 4.25M new parameters; the full model has
14,324,829 parameters with Moneta's hidden width of 320. Resume the new
`latest.pt` without `--upgrade-specialists`; the frozen/unfrozen choice and
curriculum state are retained. Use the same player-count list and curriculum
fraction on resume. `--no-specialist-curriculum` explicitly disables quotas.

For an equal-budget three-arm experiment instead:

```bash
PYTHONPATH=ai ai/.venv/bin/python ai/specialist_experiment.py \
  --checkpoint ai/runs/imports/moneta-s5660672-81301adf/checkpoint.pt \
  --output ai/runs/specialist-experiment-1 --device auto --steps 262144 --dry-run
```

Remove `--dry-run` to run the three arms sequentially: existing architecture
with its original focus schedule, existing architecture with balanced practice,
and the specialist model with balanced practice. They share parent weights,
optimizer settings, decision budgets and frozen historical rival weights;
no new rival snapshots are added during these experiments. Opponent sampling
and game trajectories can diverge, so this is not identical training data.
`--freeze-backbone` optionally makes the specialist arm an adapter-only test.
Use a new output directory for each run; existing directories are refused.
Logs and the command/checksum manifest are stored there. Nothing is deployed.

Export each arm into a separate evaluation catalog with `smallworld_ai.export`,
then use the existing `compare-bots.sh --all-player-counts` workflow with
`--evaluator pytorch`. Run multiple fresh evaluation seeds and enough fresh
maps per player count. Check `specialists.json` and phase/action metrics as
well as wins; a short smoke run proves integration, not improved play.

For guaranteed roster coverage on held-out maps, also run:

```bash
PYTHONPATH=ai ai/.venv/bin/python -m smallworld_ai.specialist_evaluate \
  --checkpoints ai/runs/imports/moneta-s5660672-81301adf/checkpoint.pt \
                ai/runs/specialists-pilot/latest.pt \
  --opponents greedy random ai/runs/imports/moneta-s5660672-81301adf/checkpoint.pt \
  --cycles 4 --workers 4 --seed 2140000000 \
  --exclude-map-log ai/runs/specialists-pilot/maps.jsonl \
  --output ai/runs/specialist-evaluation-1
```

Each cycle schedules 37 initial pairs at each of 2–6 players, covering all
enabled races and traits with fresh maps and rotated learner seats. Every
contender replays the same fixture with the same frozen opponent lineup.
Results include per-race, per-trait and per-player-count summaries, actions,
rejections, map hashes and checkpoint hashes. These are full-game results from
assigned starting pairs; they complement ordinary market-choice tournaments.
`--distribution surface-stress` tests a shifted map distribution. Supply all
available training map logs to `--exclude-map-log` to reject exact overlap.
Use fresh seeds for repeated evaluations; repeating a seed replays the same
fixtures. `--limit-fixtures 5` is a deliberately incomplete functional smoke
test, with one fixture at each player count, and is marked as such in its
manifest. These games are parallel Python/Go environments, with one Torch CPU
thread per inference call; frozen policies are shared among worker threads.

Format 7 requires the persistent PyTorch CPU evaluator for games and comparisons.
It fails clearly if that evaluator is absent instead of silently ignoring the
specialists. Formats 1–6 retain their Go fallback. The Go reader, Python reader
and pull script now accept bounded exports up to 384 MiB; this prototype's JSON
export is about 274 MB. Before any future live format-7 publication, rebuild
both the Go worker and `ai/deploy/Dockerfile.cpu-worker`; the CPU image advertises
format-7 support. The pull script validates the new format through that same
Python runtime and refuses publication onto an incompatible worker image.

Load and deploy the final `-models` image. The explicit `WORKER_IMAGE` argument
ensures the model layer uses the newly built worker when using custom tags.

Rebuild/deploy the gateway and frontend too for the first bot-enabled release.
The trainer itself does not modify a cluster; the pull script deploys only when
`--deploy-kind` is explicitly supplied.
The older root `docker-compose.yml` / `backend` stack does not support bots.

## Pull a named bot from the GPU host

From the repository root, export the selected checkpoint and make it playable
in the existing local Kubernetes deployment:

```bash
./ai/pull-model.sh --name "Training 6" --deploy-kind
```

Change the name for each new snapshot. The default host is
`gpu`, and the default run is
`~/github/smallworld/ai/runs/active`, a symlink to the current run. It falls back to `surface-v2-20261015` only when that pointer
is absent. The script freezes a complete
checkpoint while training continues, exports it using that run's matching source,
checks SHA-256, validates all models with the Go server reader, and installs an
immutable JSON under `ai/models/`. It refuses to overwrite an existing label with
different weights. Training 1–5 remain available.

Omit `--deploy-kind` to download only. Add `--with-checkpoint` to download the
resumable PyTorch checkpoint too. Receipts and optional checkpoints are saved in
`ai/runs/imports/<model-id>/`. These paths and model JSON files are Git-ignored;
the script does not commit or push. The JSON alone is the playable agent; `.pt`
also contains the critic, optimizer and other training state.

Local requirements: Python 3, Go, and SSH key access to the host (with its host key
already verified). Publishing additionally needs Docker, kind and kubectl. The
remote checkout needs its training venv in `ai/.venv`; the script uses the existing
optional `~/.local/share/smallworld-training/env.sh`. No local PyTorch or GPU is
needed to pull a model. The remote export runs on CPU.

`--deploy-kind` derives a new image from the currently deployed room worker,
retains its code and existing models, and copies in the downloaded models. It
uses explicit defaults `--context kind-game-local --cluster game-local
--namespace game`, waits for rollout, verifies the model hash on every worker,
and refreshes room-list model metadata. It does not rebuild the application or
change ongoing games. V2 observations have several model architectures: formats 2, 3 and 4 need their
matching worker support. The current worker supports formats 1 through 4; the
script checks the deployed image capability label before publishing formats 3/4. A worker update is needed separately after code changes.

To specify the SSH connection as a command string, including its destination:

```bash
./ai/pull-model.sh --name "Training 7" --ssh-command 'ssh gpu' --deploy-kind
./ai/pull-model.sh --name "Training 7" --ssh-command 'ssh "arthurj"@192.168.27.69' --deploy-kind
# SSH flags and quoted arguments are supported too:
./ai/pull-model.sh --name "Training 7" --ssh-command 'ssh -p 2222 -i "/path/to/my key" gpu'
```

The same connection command is used for exporting, downloading the model, and
optionally downloading its checkpoint. The puller parses the quoted arguments
and appends each remote operation as a separate argument; it does not append
another host. Supply only the SSH connection command, without a remote command
or shell pipelines. Local shell expansion is not performed inside this string.
`--ssh-command` and `--host` are mutually exclusive. Omitting both keeps the
existing `ssh` connection to `gpu` (or `SMALLWORLD_GPU_HOST`).

For another host or run:

```bash
./ai/pull-model.sh --name "Training 7" \
  --host arthurj@192.168.27.69 \
  --remote-run '~/github/smallworld/ai/runs/another-run' \
  --with-checkpoint --deploy-kind
```

Use `--remote-root` for another checkout path, and `--remote-source` to override
the matching remote `ai/` source directory recorded by the checkpoint. If a
download or deployment fails, retry the **same** frozen snapshot using the printed
release directory, so the label cannot silently move to newer weights:

```bash
./ai/pull-model.sh --name "Training 6" \
  --remote-release '/home/arthurj/github/smallworld/ai/runs/surface-v2-20261015/previews/export-REPLACE_ME' \
  --deploy-kind
```

Refresh the browser after publication, then choose the new model in the room's
bot controls. Only the room host can add or change bots.

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
Training 5 and other v2 exports use the graph action adapter in rooms. Rooms with
only v2 trained bots (plus random/greedy or humans) support 35 surface races and
38 traits. Mixing Training 1–4 with Training 5 uses the shared 12-race/12-trait
preset. Choose **Use bot rules** after manually changing selections. V2 rejects
underground maps and powers. Rejected action intents are remembered across
snapshots and playback. Playing strength still requires held-out evaluation.

The October 15 GPU run lives at `~/github/smallworld/ai/runs/surface-v2-20261015`
on the training host and is retained as the graph baseline. The current run is
`ai/runs/joint-20261015`, with isolated source in `ai/workspaces/joint-20261015`
and the user service `smallworld-training.service`. It runs continuously with no
23:00 cutoff or decision limit. Use `systemctl --user stop smallworld-training`
on the GPU host to stop it deliberately.
`journalctl --user -u smallworld-training`, `metrics.jsonl`, `coverage.json`,
`maps.jsonl`, and `latest.pt` record its progress. Its launcher inhibits idle/suspend and restarts failed trainer processes
from the latest saved update. The warm-start checkpoint is retained separately.

## Joint stack-and-tile model (export format 4)

The recommended expanded architecture uses the same v2 observations, a 96-wide
board graph with four message-passing layers, mean and max board pooling, and
320-wide actor/critic layers. Each candidate is still a complete action: stack,
destination, source when applicable, or a selected player/market entry. The actor
sees that candidate's tile embeddings and the entire board summary. A learned
context-dependent action/piece bias is added to its joint candidate score.
There are 2,190,211 parameters with `--hidden 320` (Training 5 has 1,350,658).

Warm-starting this model from a graph checkpoint preserves its action distribution
and value prediction at initialization. New channels and graph depth can then
learn; the new action/piece bias starts at zero. This avoids the strategy loss
seen with the experimental format-3 conditional selection model, which remains
readable for reproducibility. No piece-specific rewards, tactical priorities or
scripted special-piece choices are added.

The extra `--group-entropy` objective encourages exploration across action/piece
groups, independent of how many destination candidates each group offers. It
does not say which piece is useful; game rewards must teach that. The default
`--gamma 1 --gae-lambda .99` gives credit across longer sequences of preparation
moves. A small generic rejection penalty supplies feedback for failed intents.
Coverage records successful race/trait/action appearances; `actions.json` also
counts offered, chosen, successful and rejected action/piece groups. Neither
coverage nor training win rate is proof of improved playing strength.

```bash
PYTHONPATH=ai ai/.venv/bin/python -m smallworld_ai.train \
  --rules surface --architecture joint --hidden 320 --device cuda \
  --envs 32 --rollout-steps 128 --async-rollouts --opponent-device cuda \
  --batch-size 128 --epochs 4 --learning-rate .00003 --target-kl .01 \
  --gamma 1 --gae-lambda .99 --entropy .01 --group-entropy .02 \
  --focus-fraction .25 --rejection-penalty .001 \
  --player-counts 2 2 3 3 4 4 5 6 --maps generated --opponent mixed \
  --warm-start ai/runs/previous-graph/latest.pt \
  --total-steps 2000000 --output ai/runs/joint
```

Replace `--warm-start` with `--resume ai/runs/joint/latest.pt` to continue this
run. Keep the other options. CUDA runs report peak tensor memory, rollout/update
times, and opponent batch counts. Asynchronous runs additionally record how many
learner decisions each environment contributed. Larger tables advance more
slowly, so inspect that distribution when tuning parallelism.

Training remains limited to surface maps, 35 races, 38 traits and no underground
relic powers. Three quarters of new games use natural market choices; the other
quarter assigns a rotating race/trait pair to a random learner seat. All games
use generated training maps; evaluation uses separate map seeds and rotated seats.


## Remove a model

Run locally, from the repository root:

```bash
./ai/remove-model.sh --list --deploy-kind
./ai/remove-model.sh --name "Training 2" --deploy-kind --dry-run
./ai/remove-model.sh --name "Training 2" --deploy-kind
```

The dry run changes nothing. Use an exact `--id` instead of `--name` when needed.
Without `--deploy-kind`, removal affects only `ai/models`, not the running game.
Weights are archived with a checksum receipt under `ai/runs/removed/` before
removal. Training checkpoints, GPU jobs and other models are untouched. Built-in
bots cannot be removed. Kubernetes removal checks active seats and saved room
playback references, refuses in-use models, preserves the current worker binary
and other models, verifies the rollout, and refreshes the lobby catalog. Rollout
failures trigger a guarded rollback. The default target is `kind-game-local`,
namespace `game`, kind cluster `game-local`.

## Persistent training exploration and bounded continuous runs

The live joint run additionally uses:

```text
--total-steps 0 --keep-checkpoints 12
--exploration-fraction .05 --exploration-temperature 1.5
--exploration-bonus .02 --exploration-budget .04
```

Zero total steps means run until interrupted. Numbered checkpoints are bounded;
latest, parents and exports are retained. Metrics/map JSONL logs rotate at 64 MiB
with two backups. `games_per_second` counts completed matches across all learner
environments; `decisions_per_second` counts learner actions only.

Exploration has no time cutoff. On each fresh choice, a 5% exploration coin
selects a distribution balancing available action/piece groups, with softened
learned preferences for their destinations. Otherwise the learned joint policy
chooses the move. If an exploratory action is rejected, the same action/piece
group stays selected while the model tries another remaining destination.
Success, episode end or exhaustion releases this selection; a rejected attempt
does not complete an exploration attempt or earn curiosity reward. The engine
filters already-rejected candidates. Rejections still provide learning feedback
and count as learner decisions. Consequently retries can make more than 5% of
all decisions exploratory. Collection records the exploration mode; PPO uses
that same conditional action probability. Evaluation and exports use the base
policy, without forced exploration or retries.

A generic familiarity bonus is available once per piece type per game after a
successful action, with a per-game cap. It shrinks as lifetime successful-use
counts increase, including uses from the parent checkpoint. Rejected moves earn
nothing; repeating movement/redeployment of a piece cannot farm bonuses. There
are no named-piece priorities. Checkpoints preserve curiosity counts, so resume
does not make familiar pieces unfamiliar again. The bonus eventually becomes
negligible for familiar pieces while the exploration mixture stays enabled.

Shared context projection and packed valid candidates reduce repeated model
work while retaining the same exported architecture and learned parameters.
Playing strength still needs held-out evaluation rather than training coverage.

## Compare playable bots

From the repository root, list the available names and IDs:

```sh
./ai/compare-bots.sh --list
```

Compare every local export plus the Random and Greedy baselines:

```sh
./ai/compare-bots.sh --all --workers 8
```

Choose a list using exact IDs or quoted menu names:

```sh
./ai/compare-bots.sh --bots "Training 4" "Training 5" "Gobble" "Shabble" --maps 10 --workers 8
```

The default still runs four-player games on three fresh generated evaluation
maps. Add `--all-player-counts` to compare the same selected models in **2, 3,
4, 5 and 6-player** games:

```sh
./ai/compare-bots.sh --bots "Moneta" "Dusk" --all-player-counts --maps 3 --workers 4
```

To select particular sizes, use `--players 2 4 6` (or `--players 3` for just
three-player games). It cannot be combined with `--all-player-counts`.
For multiple player counts, the script now aims for **similar game totals**
(within 20% between the smallest and largest total). It starts with a common
target based on the largest schedule, then rounds to whole balanced map/seat
groups. If needed, it raises the target until the totals are close enough.
Extra games use fresh map seeds, not duplicate deterministic matches.

`--maps` is the **minimum** number of independent maps for each count; counts
with fewer games per map get more maps automatically. For example, comparing
two bots with `--all-player-counts --maps 3` produces 36, 36, 36, 40 and 36 games
for 2–6 players, respectively. A one-size run still uses exactly `--maps` maps.
`--seed 12345` selects a different reproducible set. Use `--dry-run` first to
see the actual game and map totals for each size.

These run offline without rooms, Kubernetes, a GPU or the training computer.
Requirements are Python 3.10+, Go and Node 18+; initialize the map-generator
submodule if necessary. PyTorch CPU is selected automatically when available;
see the runtime setup below. Eight isolated Go processes run games in parallel
by default (or fewer on small machines); `--workers 4` adjusts concurrency.
Each process reuses its evaluator across games and player counts. Each game's
turns remain sequential, as required by the game rules.

Fairness is checked **separately for every player count and generated map**:
equal appearances, equal exposure to each opponent, and equal counts in every
starting seat. Each (player count, map seed) has its own saved initial snapshot,
including market, neutral pieces and engine RNG state. Seat rotations at that
size restore this exact snapshot and verify its checksum. Different sizes have
appropriately sized generated maps, not a shared snapshot with an invalid
number of players. Deterministic replay is checked for every requested size.

With enough distinct bots, lineups contain no duplicate models. The balanced
schedule rotates each lineup and its reversed order, using a compact design
when possible. It falls back to all combinations when no compact balanced
design is found. `--matchups exhaustive --seat-orders all` uses every lineup
and every unique seating permutation. For more than 12 selected models,
explicitly request `--matchups exhaustive` for multiplayer comparisons; use
`--dry-run` because this can produce many games.

When fewer models than seats are selected, copies fill the seats as evenly as
possible. Every choice of which models get an extra copy is represented equally.
Copies remain individual competitors, not teams. The default four-player
AABB/all-three-model schedules are preserved. At other sizes the compact
schedule samples an ordering, rotates it and reverses it. It chooses patterns
with the same number of distinct rotations to keep lineup exposure balanced;
use `--seat-orders all` to test every seating. Statistics count each copy's
appearance.

`--rules auto` uses the shared legacy race/trait palette if any selected bot is v1. Otherwise it uses the expanded surface palette. All games in one tournament use the same rules. Each bot uses its production observation/action format. The comparison tests the playable agents as deployed, including their adapter differences; it does not isolate network architecture alone. Underground terrain and relic powers are excluded. Use modern bots alone with `--rules surface` to assess expanded-rules play.

A unique directory under `ai/runs/comparisons/` contains `report.md`,
`leaderboard.csv`, `summary.json`, every game's scores/action counts/player
count in `matches.jsonl`, the schedule, map/state fixtures, frozen selected
models, and the exact engine binary. `manifest.json` records player counts,
game totals, the common target, actual map counts and seeds for each size,
balance/replay checks, evaluator, hashes and completion status.
`--output /absolute/new/directory` changes the destination; an existing
directory is refused. A failed/truncated game fails the run and retains partial
evidence. For repeatability use the same seed, options, engine and models.

For multiple sizes, `players-2/` through `players-6/` contain separate reports,
leaderboards and pairwise matrices. The top-level report compares win shares
side by side, and `leaderboard-by-player-count.csv` contains all per-size rows.
The combined `leaderboard.csv` gives every requested player count equal weight:
its score is the mean of `player count × win share`, where **1.0× means even
chance**. This avoids letting a larger schedule dominate or comparing raw
2-player win rates directly with 6-player win rates. The tie-breaker is mean
normalized rank, `(players − mean rank) / (players − 1)`. Raw points are shown
only within each player count. One-size runs keep the original output layout.

Win share gives one win credit per game, split among tied winners. Mean rank uses midranks for ties (lower is better). The pairwise matrix in `summary.json` compares scores when two bots shared a game. Seat rotations on one map are correlated: 720 games on three maps are still only three independent map/initial-state samples. Treat small runs as exploratory and increase map count before drawing strong conclusions.

## Learning action acceptance and varied opponents

The joint trainer optionally uses `--acceptance-loss .05`. A training-only linear
head predicts whether the attempted action was accepted by the real engine. Its
binary classification loss trains the shared board/action representation alongside
PPO. No legal-move labels are fabricated, no piece receives a special rule, and
predictions do not mask or override playable choices. At hidden width 320 the
head adds 321 training parameters; the format-4 playable policy is unchanged.
The head and its optimizer are saved separately in `latest.pt`, so old checkpoints
load unchanged and existing optimizer history is retained. This objective is an
experiment in learning efficiency, not evidence of stronger play by itself.

Each opponent seat now independently chooses a frozen opponent for the whole
episode. The eight-policy league keeps its initial anchor, newest snapshot and
a reservoir of six historical policies, rather than retaining only recent copies.
Checkpointing preserves the reservoir counter and selected weights. Opponent
assignments remain fixed within each game. Generated surface maps, race/trait
coverage, terminal win rewards and persistent exploration remain enabled.

Metrics include `acceptance_loss`, `acceptance_accuracy`,
`rollout_rejection_rate`, `exploration_starts`, `exploration_retry_decisions`,
`exploration_successes`, and `opponent_pool_size`. Acceptance accuracy is class
imbalanced and is not a win-rate measurement. Export checkpoints and compare
them on fresh held-out seeds before promoting a model as stronger.

### Relational policy, defensive feedback, and permanent opponents

`--architecture relational` (export format 5) adds a state-conditioned spatial
attention branch to the joint actor. Two blocks attend over neighboring tiles
and one over the whole board. Residual connections retain local tile identity;
all candidate scores still describe a complete action, stack, target and source.
At hidden size 320 the model has 4,089,862 parameters, versus 2,190,211 for joint.
No named race, biome or piece receives a tactical preference in the network.

For an existing joint checkpoint, use `--resume RUN/latest.pt
--upgrade-relational`. The old actor/critic weights and Adam moments are retained.
The new actor and critic output layers start at zero, so expansion initially
preserves action probabilities and value predictions. Existing graph/joint
policies remain supported as frozen league opponents and playable models.

`--gae-clock turn --gamma 1 --gae-lambda .99` applies trace decay once per
completed learner turn, rather than once per click. Repositioning several pieces
therefore does not weaken the learning signal simply because it takes more
clicks. Longer rollouts still bootstrap from the critic at their boundaries.
This increases dependence on later outcomes and can increase variance; it does
not guarantee better play.

`--survival-loss .05` adds a training-only tile prediction task: which currently
owned regions remain owned when the player next acts after opponents' turns.
Targets come from observed gameplay. This adds no survival reward, no simulated
look-ahead and no explicit Hero/forest rule. Terminal games and unowned tiles are
masked. The prediction head is omitted from playable exports.

A permanent league manifest has version 1 and a `members` array, each containing
`id`, `name`, `checkpoint` and `sha256`. Pass `--league FILE
--league-fraction .75`. Frozen checkpoint hashes are verified. Half of the league
sampling weight is uniform; the remainder favors opponents the learner struggles
against. Opponent seats avoid repeated pinned rivals when possible, stay frozen
throughout a game, and also include changing snapshots, random and greedy bots.
League weights and statistics are embedded in checkpoints for portable resumes.

Persistent exploration uses `--exploration-fraction .15
--exploration-temperature 2 --exploration-learned-groups .75
--exploration-piece-prior 2`. Exploratory stack selection is partly learned and
partly a permanent coverage floor, with a generic preference to try available
pieces. Destination counts cannot swamp rare stacks. Failed exploratory attempts
retry the same action/piece group at another remaining destination. PPO records
the actual conditional behavior probabilities. These exploration settings apply
to training, not the deterministic exported bot.

### Benchmark gate and pulling a selected checkpoint

Run a separate evaluator, using the same updated engine:

```bash
PYTHONPATH=ai python -m smallworld_ai.benchmark \
  --run ai/runs/league-20261016 \
  --league ai/runs/league-20261016/league.json \
  --baseline PATH_TO_INITIAL_INCUMBENT.pt \
  --maps 36 --workers 2 --interval 1800
```

It compares the newest checkpoint with the incumbent over 288 four-player games:
36 fresh generated maps, four seats, and two contenders. Each paired game starts
from the exact same saved state and faces the same three frozen rivals. It checks
deterministic replay and writes fixtures, all game outcomes, and summaries under
`RUN/benchmark`. Promotion requires a positive lower bound of a paired map
bootstrap 95% interval for win-credit improvement, with no rival's mean relative
finish declining by more than .05. This is a limited evaluation against this
roster, not a guarantee of universal superiority. A non-promoted baseline is
marked as the incumbent, not proof that it is the best available model.

The evaluator runs independently of PPO and never supplies optimizer targets.
`RUN/champion.json` points to an immutable, hash-verified full checkpoint. The
counter reserves fresh evaluation seeds even after interrupted evaluations.

```bash
# From the Smallworld repository, using the ZeroTier SSH alias:
./ai/pull-model.sh --name "My new bot"
# Deliberately test the newest training weights, bypassing the benchmark gate:
./ai/pull-model.sh --name "Experimental bot" --checkpoint latest
# Require an evaluated checkpoint explicitly:
./ai/pull-model.sh --name "Selected bot" --checkpoint champion
```

The default host is `gpu` (override with `--host` or `SMALLWORLD_GPU_HOST`).
Default checkpoint selection is `auto`: prefer the evaluated champion. A new
league run with no completed evaluation fails with a clear message; use explicit
`latest` only when you want an experimental checkpoint. Older runs without a
league retain the previous latest-checkpoint behavior. All existing import,
immutable naming, checksum verification, and deployment options remain intact.
Before importing format 5 into kind, rebuild the worker from the updated source;
its image must advertise `io.smallworld.bot-formats="1,2,3,4,5"`.

### Win objective and predicting opponents' board effects

To optimize shared first-place credit instead of own income plus a win bonus,
use `--objective win --gamma 1 --rejection-penalty 0 --exploration-bonus 0`.
At intermediate steps the reward is the change in
`(own score - highest opponent score) / 100`. The terminal step cancels the
remaining score potential and pays `2 * shared_win_credit - 1`. Centering the
potential at the first learner decision handles opponents who act before it.
Therefore a complete episode sums to +1 for a sole win, -1 for a loss, or
`2/k - 1` for a k-way first-place tie, regardless of absolute scores. A low-score
win is worth as much as a high-score win. This is potential-based reward shaping;
it supplies temporary feedback without giving extra final credit for farming
points or attacking a trailing player. This mode rejects extra curiosity rewards,
rejection reward penalties and gamma below 1 so that this identity stays true.
Private scores are training reward labels only and are not policy inputs.

Persistent exploratory sampling, the generic stack prior and same-stack retries
still work. `--rejection-loss .05` separately teaches the base action scorer to
reduce probability of the exact observed rejected action. It adds no piece rules,
no predicted-legality mask, and no tactical bonus for accepted actions. It uses
`-log(1-p(rejected action))`, computed stably in log space, and ignores rejected
singleton supports that have no alternative. The default is zero for legacy runs.

For relational models, `--response-loss .05` trains the shared board encoder to
forecast each region's next owner and whether its total public piece count will
fall, stay the same, or rise by the learner's next turn. Targets come from actual
play after opponents act, not simulated look-ahead. They cover the whole map,
including opponents' territory, and complement `--survival-loss .05` on owned
regions. The finish action and automatic turn transitions are part of the
observed transition. These are net board outcomes, not exact move-sequence
predictions, remembered histories, or counterfactual simulations for every move.
No race-specific rules, threat formulas or future-state features are added to
policy inputs. The small forecast head is training-only; shared representation
weights receive its gradients and remain in the playable export. Its optimizer
and weights resume in `checkpoint['opponent_response']`. Model format 5 and the
4,089,862-parameter playable/training policy layout are unchanged.

Metrics separate policy, fresh-exploration and retry rejection rates. Forecast
accuracies include a persistence baseline (predict no change) and separate
accuracy on changed tiles, so high accuracy on unchanged territory is not taken
as evidence of strong anticipation. These are training diagnostics, not held-out
playing-strength estimates. The independent four-player benchmark still gates
champion promotion using paired matches against the permanent opponent roster.

Background: [potential-based reward shaping](https://people.eecs.berkeley.edu/~russell/papers/icml99-shaping.pdf)
and [agent modeling as an auxiliary task](https://arxiv.org/abs/1907.09597).
These motivate the design, not a guarantee of improved Smallworld win rate.

The same policy already receives the player count, seat, occupied player slots,
each player's active/declined tribe identities and reserves, and relative tile
ownership. `--player-counts 2 3 4 5 6` samples equal numbers of games by table size;
repeating a count weights that size more heavily. No separate model is needed.
`results_by_player_count` tracks completed training games, shared win credit,
mean score and return separately, with counters retained in the checkpoint.
These mixed-opponent training statistics are not a balanced tournament: the
existing promotion benchmark remains specifically four-player, so it does not
yet certify improvements for every table size.


## Entity expansion and rollback

The format-6 `entity` architecture retains the complete format-5 network and
adds six width-256 attention layers over public tile/player/piece-type tokens.
It has 10,071,800 parameters at hidden=320, including training-only feedback
adapters. New actor, value and feedback contributions start at zero. Upgrade a
relational checkpoint with `--resume ... --upgrade-entity`; the original weights
and Adam moments are retained. The same policy handles 2–6 players. No race
rules, move history, underground content or powers are added.

`--cuda-graphs --async-rollouts` enables bounded inference replay. It uses one
shared CUDA lock during collection, masks padded shapes, returns owned output
buffers, and falls back for uncached/unsupported inputs. This is inference only;
PPO still runs normal autograd. The cache is bounded to 32 entries and 384 MiB
of conservative accounting. Evicted shapes must accumulate new demand before
recapture, avoiding repeated compilation when mixed opponents fill the cache.
The complete trainer must still fit GPU memory.

The GPU trial has its own source workspace and run directory. The old run and
Dusk's frozen full checkpoint remain available. After this setup is installed:

```sh
./ai/switch-training.sh old --ssh-command 'ssh gpu'
./ai/switch-training.sh new --ssh-command 'ssh gpu'
./ai/switch-training.sh status --ssh-command 'ssh gpu'
```

Switching stops the current services and resumes the other run from its latest
completed checkpoint. It preserves both directories; incomplete rollout work
since the last save is not retained. It changes training, not published bots.
The old run uses its preserved trainer source and original settings.

Dusk is the format-5 snapshot at step 5,013,504, exported before the expansion.
Its full checkpoint is under `ai/runs/imports/dusk-s5013504-60fcc0a8/` locally.
The `old` switch resumes the old run's final pre-expansion checkpoint, which may
be newer than Dusk. All playable models, including Dusk, stay installed.

Format-6 exports require worker support for format 6. The model reader and pull
script accept up to 256 MiB because the expanded JSON export exceeds 96 MiB.
The CPU worker image uses a persistent PyTorch evaluator for formats 1–6; see below.

New games and training now exclude Storm Giants, Scarecrows and Escargots by
default, along with the Racketeering trait (32 surface races, 37 traits, zero
powers). They remain in the fixed 35-race / 38-trait observation vocabulary so old checkpoints and explicit lobby selections
remain compatible. The curriculum covers all 32 x 37 pairings, including when
roster sizes share factors. Existing saved rooms are not rewritten.


## Fast CPU inference in playable games

The CPU worker image runs **all trained bots, old and new**, through one persistent
PyTorch CPU child per room worker. Existing JSON exports load once at startup;
no retraining or conversion is needed. Go still owns game rules, turn scheduling,
action selection, and the Random/Greedy policies. The model and its observations
are unchanged. CPU floating-point rounding can change a choice between nearly
tied actions, so this does not promise bit-identical games to the Go scorer.

Use the three-stage worker build above: fresh Go binary, CPU runtime, then model
files. `ai/deploy/Dockerfile.cpu-worker` pins CPU-only PyTorch and uses one compute
thread per worker to avoid competing thread pools. Its Dockerfile-specific ignore
file keeps checkpoints and unrelated files out of the build context. Apply the
worker manifest's startup probe when upgrading an older cluster; it allows five
minutes for loading models before ordinary liveness checks begin.

The Python executable comes from `BOT_INFERENCE_PYTHON`; the image supplies it
and `PYTHONPATH` automatically. The Go worker checks SHA256 hashes against the
Python model catalog before becoming ready. Requests travel through private
stdin/stdout pipes, without a network service. A failed request has a five-second
timeout, falls back to Go, and allows a child restart after thirty seconds.
Errors and fallback are logged. A startup failure prevents worker readiness.
Shutdown kills and reaps the child.

`pull-model.sh --deploy-kind` and `remove-model.sh --deploy-kind` inherit the
currently deployed worker image, preserving this runtime automatically. Pulling
a later checkpoint uses the same command as before. The default 350 ms visible
move delay remains separate from neural computation.

The comparison script automatically prefers `ai/.venv/bin/python` when PyTorch
is available, then its own Python interpreter. Install once with
`./ai/setup.sh --cpu`; subsequent comparison commands need no extra settings.
It prints the selected evaluator and records it in `manifest.json`. Use
`--evaluator pytorch` to require it, `--evaluator go` for the original scorer,
or `--inference-python /path/to/python` for another environment. Games remain
parallel under `--workers`; each worker reuses its own persistent child.

For a local `botenv` run, build the updated engine and enable the same runtime
explicitly (substitute your CPU PyTorch environment):

```bash
(cd backend2 && go build -o ../ai/bin/botenv ./cmd/botenv)
PYTHONPATH="$PWD/ai" BOT_INFERENCE_PYTHON="$PWD/ai/.venv/bin/python" \
  ./ai/bin/botenv -model-dir "$PWD/ai/models"
```

Each comparison subprocess gets its own persistent child. Unset
`BOT_INFERENCE_PYTHON` to use Go locally. Training's existing batched CUDA
inference is unchanged.

For a reversible live switch back to Go, keeping every model installed:

```bash
kubectl --context kind-game-local -n game set env statefulset/room-worker BOT_INFERENCE_PYTHON=
kubectl --context kind-game-local -n game rollout status statefulset/room-worker
```

Restore PyTorch by removing that environment override (the image default applies):

```bash
kubectl --context kind-game-local -n game set env statefulset/room-worker BOT_INFERENCE_PYTHON-
kubectl --context kind-game-local -n game rollout status statefulset/room-worker
```


### Specialist credit learning

The specialist run can freeze the inherited policy and train the dedicated race/trait modules with separate optimizer settings:

```sh
--freeze-backbone --learning-rate .0001 \
--specialist-learning-rate .0005 --specialist-adam-eps 1e-8 \
--specialist-credit-loss .1 --specialist-probe-size 128
```

Credit learning uses the actual rollout advantage of the chosen stack-and-destination action. Separate training-only readouts teach the participating race and trait embeddings even while the actor coordinator is weak. It does not change rewards, encode game tactics, or train experts for tribes merely offered in the market. The format-7 export and production evaluator are unchanged.

Check `specialist_expert_gradient_norm`, `specialist_credit_loss`, and `specialist_{race,trait,general,all}_choice_change` / `_kl` in `metrics.jsonl`. The contribution probes remove one component on the same sampled positions; they measure influence, not playing strength. Compare held-out full games before claiming improvement. Named Adam moments, dedicated credit readouts, and curriculum progress survive checkpoint resume.

The revised GPU run is `ai/runs/specialist-learning-20261016`. The previous specialist run is retained at `ai/runs/specialists-20261016`; the revised run also stores its frozen starting checkpoint and a pre-change snapshot under `rollback/`.

After activation, `./ai/switch-training.sh specialists` selects the revised run and `./ai/switch-training.sh specialists-previous` restores the preserved earlier specialist trainer. Both use `ssh gpu` by default; pass `--ssh-command 'ssh arthurj@192.168.27.69'` to choose a different connection.
