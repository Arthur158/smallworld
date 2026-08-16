package roomworker

import (
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"log"
)

// Client is a logical player/session owned by a room worker. It deliberately
// contains no websocket connection. GatewayID + ConnectionID are only routing
// information for targeted responses.
type Client struct {
	Username      string
	GatewayID     string
	ConnectionID  string
	Index         int
	Room          *Room
	DisplayRoom   *Room
	IsSpectator   bool
	Connected     bool
	CommandRoomID string

	worker *Worker
}

func (c *Client) updateRoute(cmd protocol.Command) {
	c.GatewayID = cmd.GatewayID
	c.ConnectionID = cmd.ConnectionID
	c.Connected = true
	c.CommandRoomID = cmd.RoomID
}

func (c *Client) sendMessage(msgType string, msgData json.RawMessage) {
	if c == nil || c.worker == nil || c.GatewayID == "" || c.ConnectionID == "" {
		return
	}
	msg := messages.Message{Type: msgType, Data: msgData}
	if err := c.worker.publishDirect(context.Background(), c.GatewayID, protocol.DirectEvent{
		ConnectionID: c.ConnectionID,
		Messages:     []messages.Message{msg},
	}); err != nil {
		log.Printf("roomworker: direct send %s to %s: %v", msgType, c.Username, err)
	}
}

func (c *Client) sendMessages(binding *protocol.Binding, msgs ...messages.Message) error {
	if c == nil || c.worker == nil || c.GatewayID == "" || c.ConnectionID == "" {
		return nil
	}
	return c.worker.publishDirect(context.Background(), c.GatewayID, protocol.DirectEvent{
		ConnectionID: c.ConnectionID,
		Binding:      binding,
		Messages:     msgs,
	})
}

func (c *Client) sendError(errorMsg string) {
	raw, _ := json.Marshal(map[string]string{"message": errorMsg})
	c.sendMessage("error", raw)
}

func (c *Client) bindRoom(room *Room, spectator bool, msgs ...messages.Message) error {
	c.Room = room
	c.IsSpectator = spectator
	binding := &protocol.Binding{
		Action:      protocol.BindingSet,
		Scope:       protocol.ScopeRoom,
		RoomID:      room.ID,
		IsSpectator: spectator,
	}
	if err := c.worker.setMembership(context.Background(), c.Username, room.ID, spectator); err != nil {
		return err
	}
	return c.sendMessages(binding, msgs...)
}

func (c *Client) clearRoom(msgs ...messages.Message) error {
	c.Room = nil
	c.IsSpectator = false
	if err := c.worker.clearMembership(context.Background(), c.Username); err != nil {
		return err
	}
	return c.sendMessages(&protocol.Binding{Action: protocol.BindingClear, Scope: protocol.ScopeRoom}, msgs...)
}

func (c *Client) bindDisplay(room *Room, msgs ...messages.Message) error {
	c.DisplayRoom = room
	return c.sendMessages(&protocol.Binding{
		Action: protocol.BindingSet,
		Scope:  protocol.ScopeDisplay,
		RoomID: room.ID,
	}, msgs...)
}

func (c *Client) clearDisplay(msgs ...messages.Message) error {
	c.DisplayRoom = nil
	return c.sendMessages(&protocol.Binding{Action: protocol.BindingClear, Scope: protocol.ScopeDisplay}, msgs...)
}

func (c *Client) sendUserSaves() {
	if c == nil || c.worker == nil || c.Username == "" {
		return
	}
	saves, err := c.worker.store.GetUserSaves(context.Background(), c.Username)
	if err != nil {
		log.Printf("roomworker: get saves for %s: %v", c.Username, err)
		return
	}
	values := make([]map[string]any, 0, len(saves)+1)
	values = append(values, map[string]any{"saveId": int64(-1), "summary": "New game"})
	for _, save := range saves {
		values = append(values, map[string]any{"saveId": save.SaveID, "summary": save.Summary})
	}
	raw, _ := json.Marshal(map[string]any{"saves": values})
	c.sendMessage("loadSaves", raw)
}
