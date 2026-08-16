package gateway

import (
	"backend/internal/messages"
	"backend/internal/protocol"
	"backend/internal/store"
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/redis/go-redis/v9"
)

func (g *Gateway) handleRegister(ctx context.Context, c *Client, msg messages.Message) {
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.sendError("invalid register data")
		return
	}

	if err := g.store.AddUser(ctx, data.Username, data.Password); err != nil {
		if errors.Is(err, store.ErrUserExists) {
			c.sendError("username already exists")
			return
		}
		log.Printf("gateway: register %q: %v", data.Username, err)
		c.sendError("error adding user")
		return
	}

	g.login(ctx, c, data.Username, data.Password)
}

func (g *Gateway) handleLogin(ctx context.Context, c *Client, msg messages.Message) {
	var data struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.sendError("invalid login data")
		return
	}
	g.login(ctx, c, data.Username, data.Password)
}

func (g *Gateway) login(ctx context.Context, c *Client, username, password string) {
	current := c.snapshot()
	if current.IsAuthenticated {
		c.sendError("already authenticated")
		return
	}

	if err := g.store.AuthenticateUser(ctx, username, password); err != nil {
		if errors.Is(err, store.ErrUserNotFound) || errors.Is(err, store.ErrInvalidPassword) {
			c.sendError("invalid username or password")
			return
		}
		log.Printf("gateway: authenticate %q: %v", username, err)
		c.sendError("authentication failed")
		return
	}

	reserved, err := g.reservePresence(ctx, c, username)
	if err != nil {
		log.Printf("gateway: reserve presence for %q: %v", username, err)
		c.sendError("authentication temporarily unavailable")
		return
	}
	if !reserved {
		c.sendError("user with that username already active")
		return
	}

	c.setAuthenticated(username, true)
	c.sendMessage("auth", map[string]string{"name": username})
	g.sendUserSaves(ctx, c)

	membership, ok, err := g.getRoomMembership(ctx, username)
	if err != nil {
		log.Printf("gateway: get reconnect room for %q: %v", username, err)
		return
	}
	if !ok || membership.RoomID == "" {
		return
	}

	if err := g.bindClient(ctx, c, protocol.Binding{
		Action:      protocol.BindingSet,
		Scope:       protocol.ScopeRoom,
		RoomID:      membership.RoomID,
		IsSpectator: membership.IsSpectator,
	}); err != nil {
		log.Printf("gateway: subscribe reconnect room %s: %v", membership.RoomID, err)
		return
	}

	reconnectData, _ := json.Marshal(map[string]bool{"spectator": membership.IsSpectator})
	cmdMsg := messages.Message{Type: "reconnect", Data: reconnectData}
	if err := g.publishCommand(ctx, c, membership.RoomID, protocol.ScopeRoom, cmdMsg); err != nil {
		_ = g.bindClient(ctx, c, protocol.Binding{
			Action: protocol.BindingClear,
			Scope:  protocol.ScopeRoom,
		})
		log.Printf("gateway: publish reconnect for %q: %v", username, err)
		c.sendError("could not reconnect to room")
	}
}

func (g *Gateway) handleLogout(ctx context.Context, c *Client) {
	s := c.snapshot()
	if !s.IsAuthenticated {
		c.sendMessage("unauth", map[string]string{"name": ""})
		return
	}

	g.notifyDisconnect(ctx, c)
	if err := g.releasePresence(ctx, c, s.Username); err != nil {
		log.Printf("gateway: release presence for %q: %v", s.Username, err)
	}

	_ = g.bindClient(ctx, c, protocol.Binding{Action: protocol.BindingClear, Scope: protocol.ScopeRoom})
	_ = g.bindClient(ctx, c, protocol.Binding{Action: protocol.BindingClear, Scope: protocol.ScopeDisplay})
	c.setAuthenticated("", false)
	c.sendMessage("unauth", map[string]string{"name": ""})
}

func (g *Gateway) handleDeleteSave(ctx context.Context, c *Client, msg messages.Message) {
	s := c.snapshot()
	if !s.IsAuthenticated {
		c.sendError("not authenticated")
		return
	}

	var data struct {
		SaveID int64 `json:"saveId"`
	}
	if err := json.Unmarshal(msg.Data, &data); err != nil {
		c.sendError("invalid save id")
		return
	}

	if err := g.store.RemoveSaveFromUser(ctx, s.Username, data.SaveID); err != nil {
		if !errors.Is(err, store.ErrSaveNotFound) {
			log.Printf("gateway: delete save %d for %q: %v", data.SaveID, s.Username, err)
		}
		c.sendError("unable to delete save")
		return
	}
	g.sendUserSaves(ctx, c)
}

func (g *Gateway) sendUserSaves(ctx context.Context, c *Client) {
	s := c.snapshot()
	if !s.IsAuthenticated {
		return
	}

	dbSaves, err := g.store.GetUserSaves(ctx, s.Username)
	if err != nil {
		log.Printf("gateway: load saves for %q: %v", s.Username, err)
		return
	}

	saves := make([]store.SaveInfo, 0, len(dbSaves)+1)
	saves = append(saves, store.SaveInfo{SaveID: -1, Summary: "New game"})
	saves = append(saves, dbSaves...)
	c.sendMessage("loadSaves", map[string]any{"saves": saves})
}

func (g *Gateway) getRoomMembership(ctx context.Context, username string) (protocol.RoomMembership, bool, error) {
	value, err := g.redis.HGet(ctx, protocol.UserRoomsHashKey, username).Result()
	if errors.Is(err, redis.Nil) {
		return protocol.RoomMembership{}, false, nil
	}
	if err != nil {
		return protocol.RoomMembership{}, false, err
	}

	var membership protocol.RoomMembership
	if err := json.Unmarshal([]byte(value), &membership); err == nil {
		return membership, true, nil
	}

	// Backward-compatible fallback while migrating from the earlier proposed
	// hash format where the value was only the room ID.
	return protocol.RoomMembership{RoomID: value}, true, nil
}
