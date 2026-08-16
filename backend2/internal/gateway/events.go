package gateway

import (
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"log"
	"strings"

	"github.com/redis/go-redis/v9"
)

func (g *Gateway) startEventBus(ctx context.Context) error {
	g.pubsub = g.redis.Subscribe(
		ctx,
		protocol.GatewayEventsChannel(g.cfg.GatewayID),
		protocol.LobbyChangedChannel,
	)

	// Wait until Redis confirms both initial subscriptions before accepting
	// websocket traffic. Receive can also return other Pub/Sub frame types, so
	// do not assume the first two frames are the two confirmations.
	for {
		frame, err := g.pubsub.Receive(ctx)
		if err != nil {
			return err
		}
		sub, ok := frame.(*redis.Subscription)
		if ok && sub.Kind == "subscribe" && sub.Count >= 2 {
			break
		}
	}

	g.roomSubs = newRoomSubscriptions(g.pubsub)
	go g.eventLoop(ctx, g.pubsub.Channel())
	return nil
}

func (g *Gateway) eventLoop(ctx context.Context, ch <-chan *redis.Message) {
	for {
		select {
		case <-ctx.Done():
			return
		case redisMsg, ok := <-ch:
			if !ok {
				return
			}

			switch {
			case redisMsg.Channel == protocol.LobbyChangedChannel:
				g.broadcastLobby(ctx)
			case redisMsg.Channel == protocol.GatewayEventsChannel(g.cfg.GatewayID):
				g.handleDirectEvent(ctx, redisMsg.Payload)
			case strings.HasPrefix(redisMsg.Channel, "game:room:") && strings.HasSuffix(redisMsg.Channel, ":events"):
				g.handleRoomEvent(redisMsg.Payload)
			}
		}
	}
}

func (g *Gateway) handleDirectEvent(ctx context.Context, payload string) {
	var event protocol.DirectEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		log.Printf("gateway: invalid direct event: %v", err)
		return
	}

	c, ok := g.clients.Get(event.ConnectionID)
	if !ok {
		return
	}

	if event.Binding != nil {
		if err := g.bindClient(ctx, c, *event.Binding); err != nil {
			log.Printf("gateway: apply binding for %s: %v", c.ID, err)
			c.sendError("failed to subscribe to room updates")
			return
		}
	}

	for _, msg := range event.Messages {
		if err := c.Send(msg); err != nil {
			log.Printf("gateway: direct send to %s: %v", c.ID, err)
			return
		}
	}
}

func (g *Gateway) handleRoomEvent(payload string) {
	var event protocol.RoomEvent
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		log.Printf("gateway: invalid room event: %v", err)
		return
	}
	if event.RoomID == "" {
		return
	}

	allowed := make(map[string]struct{}, len(event.Recipients))
	for _, username := range event.Recipients {
		allowed[username] = struct{}{}
	}

	for _, c := range g.clients.InRoom(event.RoomID) {
		if len(allowed) > 0 {
			if _, ok := allowed[c.snapshot().Username]; !ok {
				continue
			}
		}
		if err := c.Send(event.Message); err != nil {
			log.Printf("gateway: room send to %s: %v", c.ID, err)
		}
	}
}

func (g *Gateway) bindClient(ctx context.Context, c *Client, binding protocol.Binding) error {
	var roomID string
	switch binding.Action {
	case protocol.BindingSet:
		roomID = binding.RoomID
	case protocol.BindingClear:
		roomID = ""
	default:
		return nil
	}

	oldRoomID, changed := g.clients.Bind(c, binding.Scope, roomID, binding.IsSpectator)
	if !changed {
		return nil
	}

	if oldRoomID != "" {
		if err := g.roomSubs.Release(ctx, oldRoomID); err != nil {
			log.Printf("gateway: unsubscribe room %s: %v", oldRoomID, err)
		}
	}
	if roomID != "" {
		if err := g.roomSubs.Acquire(ctx, roomID); err != nil {
			// Roll back the local binding if the Redis subscription failed.
			_, _ = g.clients.Bind(c, binding.Scope, oldRoomID, binding.IsSpectator)
			if oldRoomID != "" {
				_ = g.roomSubs.Acquire(ctx, oldRoomID)
			}
			return err
		}
	}
	return nil
}
