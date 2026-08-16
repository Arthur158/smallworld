package gateway

import (
	"backend/internal/protocol"
	"context"
	"sync"

	"github.com/redis/go-redis/v9"
)

// roomSubscriptions keeps one Redis Pub/Sub subscription per room per gateway,
// regardless of how many local websocket clients are in that room.
type roomSubscriptions struct {
	mu     sync.Mutex
	refs   map[string]int
	pubsub *redis.PubSub
}

func newRoomSubscriptions(pubsub *redis.PubSub) *roomSubscriptions {
	return &roomSubscriptions{
		refs:   make(map[string]int),
		pubsub: pubsub,
	}
}

func (s *roomSubscriptions) Acquire(ctx context.Context, roomID string) error {
	if roomID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.refs[roomID] > 0 {
		s.refs[roomID]++
		return nil
	}

	if err := s.pubsub.Subscribe(ctx, protocol.RoomEventsChannel(roomID)); err != nil {
		return err
	}
	s.refs[roomID] = 1
	return nil
}

func (s *roomSubscriptions) Release(ctx context.Context, roomID string) error {
	if roomID == "" {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	count := s.refs[roomID]
	if count <= 1 {
		delete(s.refs, roomID)
		return s.pubsub.Unsubscribe(ctx, protocol.RoomEventsChannel(roomID))
	}

	s.refs[roomID] = count - 1
	return nil
}
