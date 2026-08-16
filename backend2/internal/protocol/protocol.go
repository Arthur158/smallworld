package protocol

import (
	"backend/internal/messages"
	"encoding/json"
	"fmt"
	"hash/fnv"
)

const (
	ScopeRoom    = "room"
	ScopeDisplay = "display"

	BindingSet   = "set"
	BindingClear = "clear"

	LobbyChangedChannel = "game:lobby:changed"
	RoomsHashKey        = "game:rooms"
	UserRoomsHashKey    = "game:user-rooms"
)

// Command is the internal message sent by a gateway to the authoritative
// room worker. Message is the exact frontend message, so the frontend protocol
// does not need to change when the backend is split into multiple services.
type Command struct {
	ID           string           `json:"id"`
	GatewayID    string           `json:"gatewayId"`
	ConnectionID string           `json:"connectionId"`
	Username     string           `json:"username"`
	RoomID       string           `json:"roomId"`
	Scope        string           `json:"scope"`
	IsSpectator  bool             `json:"isSpectator"`
	CreatedAtMS  int64            `json:"createdAtMs"`
	Message      messages.Message `json:"message"`
}

// Binding tells a gateway that a worker accepted a room-membership change.
// Gateways must not optimistically bind join/create requests: a worker can
// reject them because a room is missing, full, or already in progress.
type Binding struct {
	Action      string `json:"action"`
	Scope       string `json:"scope"`
	RoomID      string `json:"roomId,omitempty"`
	IsSpectator bool   `json:"isSpectator,omitempty"`
}

// DirectEvent is published to one gateway. It is used for command errors,
// targeted responses, and room-binding acknowledgements.
type DirectEvent struct {
	ConnectionID string             `json:"connectionId"`
	Binding      *Binding           `json:"binding,omitempty"`
	Messages     []messages.Message `json:"messages,omitempty"`
}

// RoomEvent is published to game:room:<room-id>:events. An empty Recipients
// slice means every locally connected client subscribed to that room.
type RoomEvent struct {
	RoomID     string           `json:"roomId"`
	Recipients []string         `json:"recipients,omitempty"`
	Message    messages.Message `json:"message"`
}

// RoomMembership is stored by room workers in the game:user-rooms Redis hash.
// It survives gateway reconnects and tells the new gateway which room to
// resubscribe to.
type RoomMembership struct {
	RoomID      string `json:"roomId"`
	IsSpectator bool   `json:"isSpectator"`
}

// RoomMetadata is the distributed equivalent of the room list that used to be
// built by sendRoomsUpdateToAll from the in-process rooms map.
type RoomMetadata struct {
	ID         string   `json:"id"`
	Name       string   `json:"name"`
	Players    []string `json:"players"`
	MaxPlayers int      `json:"maxPlayers"`
	MapName    string   `json:"mapName"`
	Creator    string   `json:"creator"`
	InProgress bool     `json:"inProgress"`
}

func WorkerForRoom(roomID string, workerCount uint32) (uint32, error) {
	if roomID == "" {
		return 0, fmt.Errorf("room id is empty")
	}
	if workerCount == 0 {
		return 0, fmt.Errorf("worker count must be greater than zero")
	}

	h := fnv.New32a()
	_, _ = h.Write([]byte(roomID))
	return h.Sum32() % workerCount, nil
}

func WorkerCommandStream(roomID string, workerCount uint32) (string, error) {
	worker, err := WorkerForRoom(roomID, workerCount)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("game:worker:%d:commands", worker), nil
}

func GatewayEventsChannel(gatewayID string) string {
	return "game:gateway:" + gatewayID + ":events"
}

func RoomEventsChannel(roomID string) string {
	return "game:room:" + roomID + ":events"
}

func OnlineUserKey(username string) string {
	return "game:online:" + username
}

func MarshalMembership(m RoomMembership) (string, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
