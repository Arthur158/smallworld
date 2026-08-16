package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"backend/internal/store"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	currentWorker *Worker
	rooms         = make(map[string]*Room)
	roomsMu       sync.Mutex
)

type Worker struct {
	cfg   Config
	redis *redis.Client
	store *store.Store

	clientsMu sync.Mutex
	clients   map[string]*Client

	publishedMu    sync.Mutex
	publishedRooms map[string]struct{}
	ready          atomic.Bool
}

func New(cfg Config) (*Worker, error) {
	rdb, err := newRedisClient(cfg)
	if err != nil {
		return nil, err
	}
	dbStore, err := store.Open(cfg.DatabaseURL)
	if err != nil {
		_ = rdb.Close()
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := dbStore.EnsureSchema(ctx); err != nil {
		_ = dbStore.Close()
		_ = rdb.Close()
		return nil, err
	}

	gamestate.InitTraitMap()
	gamestate.InitRaceMap()

	w := &Worker{
		cfg:            cfg,
		redis:          rdb,
		store:          dbStore,
		clients:        make(map[string]*Client),
		publishedRooms: make(map[string]struct{}),
	}
	currentWorker = w
	return w, nil
}

func newRedisClient(cfg Config) (*redis.Client, error) {
	var opts *redis.Options
	var err error
	if cfg.RedisURL != "" {
		opts, err = redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, err
		}
	} else {
		opts = &redis.Options{Addr: cfg.RedisAddr}
	}
	rdb := redis.NewClient(opts)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}

func (w *Worker) Close() error {
	var first error
	if err := w.store.Close(); err != nil {
		first = err
	}
	if err := w.redis.Close(); err != nil && first == nil {
		first = err
	}
	return first
}

func (w *Worker) Run(ctx context.Context) error {
	w.startHealthServer(ctx)
	defer func() {
		w.ready.Store(false)
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := w.persistAllRooms(shutdownCtx); err != nil {
			log.Printf("roomworker: final snapshot: %v", err)
		}
	}()
	stream := fmt.Sprintf("game:worker:%d:commands", w.cfg.WorkerID)

	err := w.redis.XGroupCreateMkStream(ctx, stream, w.cfg.ConsumerGroup, "0").Err()
	if err != nil && !stringsContains(err.Error(), "BUSYGROUP") {
		return fmt.Errorf("create redis consumer group: %w", err)
	}

	log.Printf("room worker %d/%d consuming %s", w.cfg.WorkerID, w.cfg.WorkerCount, stream)

	// A StatefulSet restarts with the same worker/consumer identity. Drain
	// messages that were delivered to this consumer but not ACKed before a
	// crash, then switch to new entries.
	if err := w.drainPending(ctx, stream); err != nil && !errors.Is(err, context.Canceled) {
		log.Printf("roomworker: drain pending: %v", err)
	}
	w.ready.Store(true)

	for {
		if err := ctx.Err(); err != nil {
			return nil
		}

		results, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.cfg.ConsumerGroup,
			Consumer: w.cfg.ConsumerName,
			Streams:  []string{stream, ">"},
			Count:    32,
			Block:    w.cfg.CommandBlock,
		}).Result()
		if errors.Is(err, redis.Nil) {
			continue
		}
		if err != nil {
			if ctx.Err() != nil {
				return nil
			}
			log.Printf("roomworker: XREADGROUP: %v", err)
			time.Sleep(500 * time.Millisecond)
			continue
		}

		if err := w.processResults(ctx, stream, results); err != nil {
			log.Printf("roomworker: process commands: %v", err)
		}
	}
}

func (w *Worker) drainPending(ctx context.Context, stream string) error {
	for {
		results, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    w.cfg.ConsumerGroup,
			Consumer: w.cfg.ConsumerName,
			Streams:  []string{stream, "0"},
			Count:    64,
		}).Result()
		if errors.Is(err, redis.Nil) {
			return nil
		}
		if err != nil {
			return err
		}
		count := 0
		for _, sr := range results {
			count += len(sr.Messages)
		}
		if count == 0 {
			return nil
		}
		if err := w.processResults(ctx, stream, results); err != nil {
			return err
		}
	}
}

func (w *Worker) processResults(ctx context.Context, stream string, results []redis.XStream) error {
	for _, sr := range results {
		for _, entry := range sr.Messages {
			raw, ok := entry.Values["command"]
			if !ok {
				_ = w.redis.XAck(ctx, stream, w.cfg.ConsumerGroup, entry.ID).Err()
				continue
			}
			rawString, ok := raw.(string)
			if !ok {
				rawString = fmt.Sprint(raw)
			}

			var cmd protocol.Command
			if err := json.Unmarshal([]byte(rawString), &cmd); err != nil {
				log.Printf("roomworker: invalid command %s: %v", entry.ID, err)
				_ = w.redis.XAck(ctx, stream, w.cfg.ConsumerGroup, entry.ID).Err()
				continue
			}

			expected, err := protocol.WorkerForRoom(cmd.RoomID, w.cfg.WorkerCount)
			if err != nil || expected != w.cfg.WorkerID {
				log.Printf("roomworker: command %s routed to wrong worker (room=%s expected=%d got=%d)", cmd.ID, cmd.RoomID, expected, w.cfg.WorkerID)
				_ = w.redis.XAck(ctx, stream, w.cfg.ConsumerGroup, entry.ID).Err()
				continue
			}

			room, loadErr := w.ensureRoomLoaded(ctx, cmd.RoomID)
			if loadErr != nil {
				log.Printf("roomworker: load room for dedupe %s: %v", cmd.RoomID, loadErr)
			}
			if room != nil && room.hasProcessedCommand(cmd.ID) {
				_ = w.redis.XAck(ctx, stream, w.cfg.ConsumerGroup, entry.ID).Err()
				continue
			}

			commandErr := w.HandleCommand(ctx, cmd)
			if commandErr != nil {
				// Rejected commands are terminal: tell the originating socket and ACK
				// them so a bad move is not retried forever.
				log.Printf("roomworker: command %s (%s): %v", cmd.ID, cmd.Message.Type, commandErr)
				w.sendCommandError(ctx, cmd, commandErr.Error())
			} else {
				if room := rooms[cmd.RoomID]; room != nil {
					room.markProcessedCommand(cmd.ID)
				}

				// For an accepted command, do not ACK until its new authoritative
				// room state has reached Redis. If persistence fails, leaving the
				// stream entry pending lets the same StatefulSet consumer retry it.
				// The in-memory command journal prevents a duplicate application in
				// this process; after a crash, replay from the older snapshot is what
				// we want.
				for {
					if err := w.persistCommandRoom(ctx, cmd.RoomID); err == nil {
						break
					} else {
						log.Printf("roomworker: persist room %s: %v", cmd.RoomID, err)
					}
					select {
					case <-ctx.Done():
						// Leave the stream entry pending. On restart the StatefulSet
						// consumer will replay it against the last durable snapshot.
						return nil
					case <-time.After(500 * time.Millisecond):
					}
				}
			}

			if err := w.redis.XAck(ctx, stream, w.cfg.ConsumerGroup, entry.ID).Err(); err != nil {
				log.Printf("roomworker: XACK %s: %v", entry.ID, err)
			}
		}
	}
	return nil
}

func (w *Worker) persistAllRooms(ctx context.Context) error {
	for _, room := range rooms {
		if room == nil {
			continue
		}
		if err := w.persistRoom(ctx, room); err != nil {
			return err
		}
	}
	return nil
}

func (w *Worker) logicalClient(cmd protocol.Command) *Client {
	key := cmd.Username
	if key == "" {
		key = "connection:" + cmd.ConnectionID
	}

	w.clientsMu.Lock()
	defer w.clientsMu.Unlock()
	c, ok := w.clients[key]
	if !ok {
		c = &Client{Username: cmd.Username, worker: w}
		w.clients[key] = c
	}
	c.Username = cmd.Username
	c.updateRoute(cmd)
	return c
}

func (w *Worker) publishDirect(ctx context.Context, gatewayID string, event protocol.DirectEvent) error {
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return w.redis.Publish(ctx, protocol.GatewayEventsChannel(gatewayID), data).Err()
}

func (w *Worker) publishRoom(ctx context.Context, roomID string, msg messages.Message, recipients ...string) error {
	event := protocol.RoomEvent{RoomID: roomID, Recipients: recipients, Message: msg}
	data, err := json.Marshal(event)
	if err != nil {
		return err
	}
	return w.redis.Publish(ctx, protocol.RoomEventsChannel(roomID), data).Err()
}

func (w *Worker) sendCommandError(ctx context.Context, cmd protocol.Command, text string) {
	raw, _ := json.Marshal(map[string]string{"message": text})
	_ = w.publishDirect(ctx, cmd.GatewayID, protocol.DirectEvent{
		ConnectionID: cmd.ConnectionID,
		Messages:     []messages.Message{{Type: "error", Data: raw}},
	})
}

func (w *Worker) setMembership(ctx context.Context, username, roomID string, spectator bool) error {
	value, err := protocol.MarshalMembership(protocol.RoomMembership{RoomID: roomID, IsSpectator: spectator})
	if err != nil {
		return err
	}
	return w.redis.HSet(ctx, protocol.UserRoomsHashKey, username, value).Err()
}

func (w *Worker) clearMembership(ctx context.Context, username string) error {
	if username == "" {
		return nil
	}
	return w.redis.HDel(ctx, protocol.UserRoomsHashKey, username).Err()
}

func stringsContains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
