package gateway

import (
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var workerMessageTypes = map[string]struct{}{
	"tribepick":         {},
	"entryaction":       {},
	"abandonment":       {},
	"Conquest":          {},
	"startredeployment": {},
	"deploymentin":      {},
	"deploymentout":     {},
	"deploymentthrough": {},
	"movement":          {},
	"opponentaction":    {},
	"finishturn":        {},
	"decline":           {},
	"createRoom":        {},
	"enterdisplayroom":  {},
	"leaveroom":         {},
	"joinRoom":          {},
	"spectateRoom":      {},
	"startGame":         {},
	"moveUp":            {},
	"moveDown":          {},
	"changeRoomMap":     {},
	"kickPlayer":        {},
	"savegame":          {},
	"rollback":          {},
	"loadgame":          {},
	"loadgamedisplay":   {},
	"loadmapdisplay":    {},
	"leavedisplayroom":  {},
	"toggleRace":        {},
	"toggleTrait":       {},
	"toggleExtension":   {},
	"toggleAll":         {},
}

func (g *Gateway) handleClientMessage(ctx context.Context, c *Client, msg messages.Message) {
	switch msg.Type {
	case "register":
		g.handleRegister(ctx, c, msg)
		return
	case "login":
		g.handleLogin(ctx, c, msg)
		return
	case "logout":
		g.handleLogout(ctx, c)
		return
	case "requestrefresh":
		if err := g.sendLobbyToClient(ctx, c); err != nil {
			log.Printf("gateway: refresh lobby: %v", err)
		}
		return
	case "deletesave":
		g.handleDeleteSave(ctx, c, msg)
		return
	}

	if _, ok := workerMessageTypes[msg.Type]; !ok {
		log.Printf("gateway: unknown message type %q", msg.Type)
		c.sendError("unknown message type")
		return
	}

	s := c.snapshot()
	if !s.IsAuthenticated {
		c.sendError("not authenticated")
		return
	}

	roomID, scope, err := resolveCommandRoom(c, msg)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	if err := g.publishCommand(ctx, c, roomID, scope, msg); err != nil {
		log.Printf("gateway: publish %s for %s: %v", msg.Type, c.ID, err)
		c.sendError("unable to send command to game server")
	}
}

func resolveCommandRoom(c *Client, msg messages.Message) (roomID, scope string, err error) {
	s := c.snapshot()

	switch msg.Type {
	case "createRoom":
		return uuid.NewString(), protocol.ScopeRoom, nil
	case "enterdisplayroom":
		return "display-" + uuid.NewString(), protocol.ScopeDisplay, nil
	case "joinRoom", "spectateRoom", "startGame":
		var data struct {
			RoomID string `json:"roomId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			return "", "", errors.New("invalid room id")
		}
		if data.RoomID == "" && msg.Type == "startGame" {
			data.RoomID = s.RoomID
		}
		if data.RoomID == "" {
			return "", "", errors.New("room id is required")
		}
		return data.RoomID, protocol.ScopeRoom, nil
	case "loadgamedisplay", "loadmapdisplay", "leavedisplayroom":
		if s.DisplayRoomID == "" {
			return "", "", errors.New("client not in a display room")
		}
		return s.DisplayRoomID, protocol.ScopeDisplay, nil
	default:
		if s.RoomID == "" {
			return "", "", errors.New("client not in a room")
		}
		return s.RoomID, protocol.ScopeRoom, nil
	}
}

func (g *Gateway) publishCommand(
	ctx context.Context,
	c *Client,
	roomID string,
	scope string,
	msg messages.Message,
) error {
	stream, err := protocol.WorkerCommandStream(roomID, g.cfg.RoomWorkerCount)
	if err != nil {
		return err
	}

	s := c.snapshot()
	cmd := protocol.Command{
		ID:           uuid.NewString(),
		GatewayID:    g.cfg.GatewayID,
		ConnectionID: c.ID,
		Username:     s.Username,
		RoomID:       roomID,
		Scope:        scope,
		IsSpectator:  s.IsSpectator,
		CreatedAtMS:  time.Now().UnixMilli(),
		Message:      msg,
	}

	data, err := json.Marshal(cmd)
	if err != nil {
		return fmt.Errorf("marshal command: %w", err)
	}

	return g.redis.XAdd(ctx, &redis.XAddArgs{
		Stream: stream,
		MaxLen: 100000,
		Approx: true,
		Values: map[string]any{
			"command": string(data),
		},
	}).Err()
}

func (g *Gateway) notifyDisconnect(ctx context.Context, c *Client) {
	s := c.snapshot()
	if !s.IsAuthenticated {
		return
	}

	if s.RoomID != "" {
		data, _ := json.Marshal(map[string]string{"scope": protocol.ScopeRoom})
		_ = g.publishCommand(ctx, c, s.RoomID, protocol.ScopeRoom, messages.Message{
			Type: "gatewaydisconnect",
			Data: data,
		})
	}
	if s.DisplayRoomID != "" {
		data, _ := json.Marshal(map[string]string{"scope": protocol.ScopeDisplay})
		_ = g.publishCommand(ctx, c, s.DisplayRoomID, protocol.ScopeDisplay, messages.Message{
			Type: "gatewaydisconnect",
			Data: data,
		})
	}
}
