# Room worker service

This is the authoritative game/room half of the distributed backend. It is designed to work with the gateway in this repository without changing the browser protocol.

## What the worker owns

The gateway owns WebSocket connections and authentication/session routing. A room worker owns:

- lobby room creation/join/spectate/leave
- room ordering, map choice, extension toggles and kicking
- `Room` and `gamestate.GameState`
- all in-game commands (`movement`, `Conquest`, redeployment, finish turn, etc.)
- save/autosave/load/rollback operations
- generated-map runtime registration and visuals
- Redis room snapshots and lobby metadata

A worker-side `Client` is only a logical player route (`Username`, `GatewayID`, `ConnectionID`). It contains no WebSocket.

## Redis contract

Gateway -> worker uses Redis Streams:

```text
game:worker:<worker-id>:commands
```

The gateway hashes each room ID with the shared `protocol.WorkerForRoom` function, so all commands for one room reach the same worker.

Worker -> one gateway uses Pub/Sub:

```text
game:gateway:<gateway-id>:events
```

Worker -> everyone currently connected to a room uses Pub/Sub:

```text
game:room:<room-id>:events
```

Lobby metadata is stored in:

```text
game:rooms
```

User reconnect membership is stored in:

```text
game:user-rooms
```

Authoritative recovery snapshots are stored at:

```text
game:room:<room-id>:snapshot
```

Generated-map data needed to reconstruct runtime maps is stored at:

```text
game:generated-map:<map-name>
```

For production, Redis should use persistence (for example a managed persistent Redis or AOF-backed Redis). PostgreSQL remains the durable store for user accounts and saved games.

## Command processing and crash recovery

Each worker consumes its own Redis Stream with a consumer group. The worker processes commands serially, which preserves a deterministic order for game mutations assigned to that worker.

After an accepted command:

1. game/room state is mutated in memory;
2. the command ID is added to a bounded processed-command journal;
3. the room snapshot is written to Redis;
4. only then is the Stream entry acknowledged.

If the worker dies after the snapshot but before `XACK`, the restarted StatefulSet pod reloads the snapshot, sees that command ID, and acknowledges the redelivery without applying the action twice.

Saved games are separate from active-room snapshots and are written to PostgreSQL through `internal/store`.

## Required environment

```bash
export REDIS_ADDR=localhost:6379
export DATABASE_URL='postgres://game:game@localhost:5432/game?sslmode=disable'

# Local single worker:
export WORKER_ID=0
export ROOM_WORKER_COUNT=1

# Optional:
export HEALTH_ADDR=:8081
export MAP_GENERATOR_URL=http://localhost:3001
export ROOM_SNAPSHOT_TTL=0
export DISPLAY_ROOM_TTL=2h
export COMMAND_BLOCK=5s
```

`REDIS_URL` can be used instead of `REDIS_ADDR` when you need a full Redis URL with credentials/TLS options understood by go-redis.

Run:

```bash
go run ./cmd/roomworker
```

Health endpoints:

```text
GET :8081/livez
GET :8081/readyz
```

`/readyz` becomes successful only after the worker has created its Stream consumer group, drained its own pending messages, and can reach Redis and PostgreSQL.

## Running multiple workers locally

Use the same Redis/PostgreSQL for all processes and give them stable worker IDs:

```bash
ROOM_WORKER_COUNT=2 WORKER_ID=0 go run ./cmd/roomworker
ROOM_WORKER_COUNT=2 WORKER_ID=1 HEALTH_ADDR=:8082 go run ./cmd/roomworker
```

The gateway must also use:

```bash
ROOM_WORKER_COUNT=2
```

The value **must be identical everywhere**.

## Important scaling rule

The current first-version ownership algorithm is:

```text
worker = hash(roomID) % ROOM_WORKER_COUNT
```

Therefore do not horizontally autoscale room workers by changing `ROOM_WORKER_COUNT` while active rooms exist. The included Kubernetes setup uses a StatefulSet so workers have stable identities:

```text
room-worker-0
room-worker-1
room-worker-2
room-worker-3
```

`POD_NAME` is parsed automatically, so `room-worker-2` becomes worker ID `2`.

Gateway pods *can* scale independently because they hold no authoritative room state.

A later version can replace this fixed-count scheme with many logical shards plus Redis leases, which would allow dynamic worker scaling without remapping every room.

## Docker

Build:

```bash
docker build -f Dockerfile.worker -t your-game-roomworker:dev .
```

Run one worker:

```bash
docker run --rm \
  -e REDIS_ADDR=host.docker.internal:6379 \
  -e DATABASE_URL='postgres://game:game@host.docker.internal:5432/game?sslmode=disable' \
  -e WORKER_ID=0 \
  -e ROOM_WORKER_COUNT=1 \
  -p 8081:8081 \
  your-game-roomworker:dev
```

On Linux, use the appropriate Docker network/service names instead of `host.docker.internal` if it is unavailable.

## Kubernetes

Edit the image and dependency names in:

```text
k8s/roomworker.yaml
```

In particular:

- `YOUR_REGISTRY/YOUR_GAME_ROOMWORKER:latest`
- Redis Service name/address
- `game-database` Secret
- `MAP_GENERATOR_URL`
- `replicas` and `ROOM_WORKER_COUNT` together

Then:

```bash
kubectl apply -f k8s/roomworker.yaml
```

Do not put the room workers behind your HTTP ingress. Gateways communicate with them through Redis, not HTTP.

## Files added for the worker

```text
cmd/roomworker/main.go
internal/roomworker/
internal/store/games.go
Dockerfile.worker
k8s/roomworker.yaml
ROOMWORKER_README.md
```

The worker intentionally reuses your existing `internal/gamestate` package rather than replacing game logic with a second implementation.
