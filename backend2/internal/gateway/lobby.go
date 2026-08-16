package gateway

import (
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"fmt"
	"sort"
)

func (g *Gateway) loadLobby(ctx context.Context) (waiting, inProgress []protocol.RoomMetadata, err error) {
	values, err := g.redis.HVals(ctx, protocol.RoomsHashKey).Result()
	if err != nil {
		return nil, nil, fmt.Errorf("read room metadata: %w", err)
	}

	waiting = make([]protocol.RoomMetadata, 0)
	inProgress = make([]protocol.RoomMetadata, 0)
	for _, value := range values {
		var room protocol.RoomMetadata
		if err := json.Unmarshal([]byte(value), &room); err != nil {
			// A single malformed room entry should not prevent the entire lobby
			// from refreshing.
			continue
		}
		if room.InProgress {
			inProgress = append(inProgress, room)
		} else {
			waiting = append(waiting, room)
		}
	}

	less := func(rooms []protocol.RoomMetadata) {
		sort.Slice(rooms, func(i, j int) bool {
			if rooms[i].Name == rooms[j].Name {
				return rooms[i].ID < rooms[j].ID
			}
			return rooms[i].Name < rooms[j].Name
		})
	}
	less(waiting)
	less(inProgress)
	return waiting, inProgress, nil
}

func (g *Gateway) sendLobbyToClient(ctx context.Context, c *Client) error {
	waiting, inProgress, err := g.loadLobby(ctx)
	if err != nil {
		return err
	}

	waitingJSON, err := json.Marshal(waiting)
	if err != nil {
		return err
	}
	progressJSON, err := json.Marshal(inProgress)
	if err != nil {
		return err
	}

	if err := c.Send(messages.Message{
		Type: "roomEntriesUpdate",
		Data: waitingJSON,
	}); err != nil {
		return err
	}
	return c.Send(messages.Message{
		Type: "roomsInProgress",
		Data: progressJSON,
	})
}

func (g *Gateway) broadcastLobby(ctx context.Context) {
	waiting, inProgress, err := g.loadLobby(ctx)
	if err != nil {
		return
	}

	waitingJSON, err := json.Marshal(waiting)
	if err != nil {
		return
	}
	progressJSON, err := json.Marshal(inProgress)
	if err != nil {
		return
	}

	waitingMsg := messages.Message{Type: "roomEntriesUpdate", Data: waitingJSON}
	progressMsg := messages.Message{Type: "roomsInProgress", Data: progressJSON}

	for _, c := range g.clients.All() {
		_ = c.Send(waitingMsg)
		_ = c.Send(progressMsg)
	}
}
