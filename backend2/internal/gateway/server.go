package gateway

import (
	"backend/internal/messages"
	"backend/internal/store"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/redis/go-redis/v9"
)

type Gateway struct {
	cfg Config

	redis *redis.Client
	store *store.Store

	clients  *ConnectionManager
	pubsub   *redis.PubSub
	roomSubs *roomSubscriptions

	httpServer *http.Server
	upgrader   websocket.Upgrader
}

func New(cfg Config) (*Gateway, error) {
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

	g := &Gateway{
		cfg:     cfg,
		redis:   rdb,
		store:   dbStore,
		clients: newConnectionManager(),
	}
	g.upgrader = websocket.Upgrader{
		CheckOrigin: g.checkOrigin,
	}
	return g, nil
}

func newRedisClient(cfg Config) (*redis.Client, error) {
	var options *redis.Options
	var err error
	if cfg.RedisURL != "" {
		options, err = redis.ParseURL(cfg.RedisURL)
		if err != nil {
			return nil, err
		}
	} else {
		options = &redis.Options{Addr: cfg.RedisAddr}
	}

	rdb := redis.NewClient(options)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, err
	}
	return rdb, nil
}

func (g *Gateway) Run(ctx context.Context) error {
	if err := g.startEventBus(ctx); err != nil {
		return err
	}
	go g.presenceLoop(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/ws", g.handleWebSocket)
	mux.HandleFunc("/livez", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", g.handleReady)

	g.httpServer = &http.Server{
		Addr:              g.cfg.ListenAddr,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("gateway %s listening on %s", g.cfg.GatewayID, g.cfg.ListenAddr)
		errCh <- g.httpServer.ListenAndServe()
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		g.closeAllClients()
		_ = g.httpServer.Shutdown(shutdownCtx)
		if g.pubsub != nil {
			_ = g.pubsub.Close()
		}
		_ = g.redis.Close()
		_ = g.store.Close()
		return nil
	case err := <-errCh:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	}
}

func (g *Gateway) checkOrigin(r *http.Request) bool {
	if g.cfg.AllowAllOrigins {
		return true
	}
	origin := r.Header.Get("Origin")
	_, ok := g.cfg.AllowedOrigins[origin]
	return ok
}

func (g *Gateway) handleReady(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), time.Second)
	defer cancel()
	if err := g.redis.Ping(ctx).Err(); err != nil {
		http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
		return
	}
	if err := g.store.Ping(ctx); err != nil {
		http.Error(w, "database unavailable", http.StatusServiceUnavailable)
		return
	}
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

func (g *Gateway) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := g.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("gateway: websocket upgrade: %v", err)
		return
	}

	client := newClient(g, conn, uuid.NewString())
	g.clients.Add(client)
	defer g.disconnectClient(client)

	conn.SetReadLimit(g.cfg.MaxMessageBytes)
	_ = conn.SetReadDeadline(time.Now().Add(g.cfg.PongWait))
	conn.SetPongHandler(func(string) error {
		return conn.SetReadDeadline(time.Now().Add(g.cfg.PongWait))
	})
	go client.pingLoop()

	lobbyCtx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	if err := g.sendLobbyToClient(lobbyCtx, client); err != nil {
		log.Printf("gateway: initial lobby for %s: %v", client.ID, err)
	}
	cancel()

	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			if !websocket.IsCloseError(err, websocket.CloseNormalClosure, websocket.CloseGoingAway) {
				log.Printf("gateway: read %s: %v", client.ID, err)
			}
			return
		}

		var msg messages.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			client.sendError("invalid message format")
			continue
		}
		if strings.TrimSpace(msg.Type) == "" {
			client.sendError("message type is required")
			continue
		}

		g.handleClientMessage(r.Context(), client, msg)
	}
}

func (g *Gateway) disconnectClient(c *Client) {
	disconnectCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	s := c.snapshot()
	g.notifyDisconnect(disconnectCtx, c)
	if s.IsAuthenticated {
		if err := g.releasePresence(disconnectCtx, c, s.Username); err != nil {
			log.Printf("gateway: release presence on disconnect for %q: %v", s.Username, err)
		}
	}

	rooms, removed := g.clients.Remove(c)
	if !removed {
		return
	}
	for _, roomID := range rooms {
		if g.roomSubs != nil {
			if err := g.roomSubs.Release(disconnectCtx, roomID); err != nil {
				log.Printf("gateway: release room subscription %s: %v", roomID, err)
			}
		}
	}
	_ = c.Close()
}

func (g *Gateway) closeAllClients() {
	for _, c := range g.clients.All() {
		_ = c.Close()
	}
}
