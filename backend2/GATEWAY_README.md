# Distributed Gateway

This gateway is the WebSocket-facing half of the backend split.

It deliberately does **not** import `internal/server` or `internal/gamestate`.
That boundary is the point of the refactor: websocket connections live in
Gateway pods; authoritative `Room`/`Gamestate` objects live in room-worker pods.

## Files

- `cmd/gateway/main.go` - process entry point and signal handling
- `internal/gateway/server.go` - HTTP/WebSocket server and graceful shutdown
- `internal/gateway/client.go` - one local WebSocket connection
- `internal/gateway/connections.go` - gateway-local connection/room index
- `internal/gateway/commands.go` - frontend message routing and Redis Stream producer
- `internal/gateway/events.go` - worker/lobby Redis Pub/Sub consumer
- `internal/gateway/subscriptions.go` - dynamic per-room subscriptions
- `internal/gateway/auth.go` - register/login/logout/save-list/delete-save
- `internal/gateway/presence.go` - distributed single-login presence lease
- `internal/gateway/lobby.go` - replacement for `sendRoomsUpdateToAll`
- `internal/protocol/protocol.go` - shared gateway/worker wire contract
- `internal/store/store.go` - PostgreSQL gateway store
- `migrations/001_distributed_backend.sql` - target PostgreSQL schema

## Environment

```bash
export LISTEN_ADDR=:8080
export REDIS_ADDR=localhost:6379
export DATABASE_URL='postgres://game:game@localhost:5432/game?sslmode=disable'
export ROOM_WORKER_COUNT=2
export GATEWAY_ID=gateway-dev-1
export WS_ALLOWED_ORIGINS='*'
```

`GATEWAY_ID` can be omitted locally. In Kubernetes, set it from the Pod name.

## Run

```bash
go run ./cmd/gateway
```

Endpoints:

- `GET /livez`
- `GET /readyz`
- `GET /ws` (WebSocket upgrade)

## Redis contract expected from the room worker

### Gateway -> worker

The gateway hashes `roomId` to a fixed worker and executes:

```text
XADD game:worker:<N>:commands ... command=<protocol.Command JSON>
```

The worker should consume only its own stream.

### Worker -> one connection

Publish `protocol.DirectEvent` to:

```text
game:gateway:<gatewayId>:events
```

When create/join/spectate is accepted, include a binding before messages:

```json
{
  "connectionId": "connection UUID",
  "binding": {
    "action": "set",
    "scope": "room",
    "roomId": "room UUID",
    "isSpectator": false
  },
  "messages": [
    {"type":"roomid","data":{"roomid":"room UUID"}}
  ]
}
```

When leave is accepted:

```json
{
  "connectionId": "connection UUID",
  "binding": {
    "action": "clear",
    "scope": "room"
  },
  "messages": [
    {"type":"lobby","data":null}
  ]
}
```

For display rooms use `"scope":"display"`.

### Worker -> everybody in a room

Publish `protocol.RoomEvent` to:

```text
game:room:<roomId>:events
```

Example:

```json
{
  "roomId":"room UUID",
  "message": {"type":"megaUpdate","data":{}}
}
```

Set `recipients` when the message should only go to particular usernames.

### Lobby metadata

Workers maintain a Redis hash:

```text
game:rooms
```

Each field is a room ID and each value is `protocol.RoomMetadata` JSON. After a
create/join/leave/kick/map/start mutation, update the hash and publish any value
to:

```text
game:lobby:changed
```

The gateways then rebuild the existing frontend messages
`roomEntriesUpdate` and `roomsInProgress`.

### Reconnect mapping

Workers maintain:

```text
game:user-rooms
```

field: username
value:

```json
{"roomId":"room UUID","isSpectator":false}
```

Do not delete this merely because a WebSocket disconnects. Delete it when the
player explicitly leaves or when the room is removed.

After login, the gateway reads this mapping, subscribes to the room and sends a
synthetic frontend-style message with type `reconnect` to the owning worker.

### Internal disconnect message

When a WebSocket disappears, the gateway emits a command whose message type is
`gatewaydisconnect`. The worker can use it for transient connection state and
for destroying display rooms. It should **not** remove a normal player from a
room just because the TCP/WebSocket connection disappeared.

## Important migration note

The gateway code uses PostgreSQL and the normalized `user_savegames` table. It
does not use the current SQLite `users.savegameids` JSON column. Migrate existing
users/saves before replacing the old server in production.
