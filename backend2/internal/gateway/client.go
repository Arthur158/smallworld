package gateway

import (
	"backend/internal/messages"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type Client struct {
	Conn *websocket.Conn
	ID   string

	writeMu sync.Mutex
	stateMu sync.RWMutex
	closeMu sync.Once
	done    chan struct{}

	Username        string
	IsAuthenticated bool
	RoomID          string
	DisplayRoomID   string
	IsSpectator     bool

	gateway *Gateway
}

func newClient(g *Gateway, conn *websocket.Conn, id string) *Client {
	return &Client{
		Conn:    conn,
		ID:      id,
		done:    make(chan struct{}),
		gateway: g,
	}
}

type clientSnapshot struct {
	Username        string
	IsAuthenticated bool
	RoomID          string
	DisplayRoomID   string
	IsSpectator     bool
}

func (c *Client) snapshot() clientSnapshot {
	c.stateMu.RLock()
	defer c.stateMu.RUnlock()
	return clientSnapshot{
		Username:        c.Username,
		IsAuthenticated: c.IsAuthenticated,
		RoomID:          c.RoomID,
		DisplayRoomID:   c.DisplayRoomID,
		IsSpectator:     c.IsSpectator,
	}
}

func (c *Client) setAuthenticated(username string, authenticated bool) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	c.Username = username
	c.IsAuthenticated = authenticated
}

func (c *Client) setBinding(scope, roomID string, spectator bool) (oldRoomID string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()

	switch scope {
	case "display":
		oldRoomID = c.DisplayRoomID
		c.DisplayRoomID = roomID
	case "room":
		oldRoomID = c.RoomID
		c.RoomID = roomID
		c.IsSpectator = spectator
	}
	return oldRoomID
}

func (c *Client) clearBindings() (roomID, displayRoomID string) {
	c.stateMu.Lock()
	defer c.stateMu.Unlock()
	roomID = c.RoomID
	displayRoomID = c.DisplayRoomID
	c.RoomID = ""
	c.DisplayRoomID = ""
	c.IsSpectator = false
	return roomID, displayRoomID
}

func (c *Client) Send(msg messages.Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	if err := c.Conn.SetWriteDeadline(time.Now().Add(c.gateway.cfg.WriteWait)); err != nil {
		return err
	}
	return c.Conn.WriteJSON(msg)
}

func (c *Client) sendMessage(msgType string, data any) {
	var raw json.RawMessage
	if data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			log.Printf("gateway: marshal outgoing %s: %v", msgType, err)
			return
		}
		raw = b
	}

	if err := c.Send(messages.Message{Type: msgType, Data: raw}); err != nil {
		log.Printf("gateway: write to connection %s: %v", c.ID, err)
	}
}

func (c *Client) sendError(message string) {
	c.sendMessage("error", map[string]string{"message": message})
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(c.gateway.cfg.PingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.writeMu.Lock()
			err := c.Conn.WriteControl(
				websocket.PingMessage,
				nil,
				time.Now().Add(c.gateway.cfg.WriteWait),
			)
			c.writeMu.Unlock()
			if err != nil {
				_ = c.Close()
				return
			}
		case <-c.done:
			return
		}
	}
}

func (c *Client) Close() error {
	var err error
	c.closeMu.Do(func() {
		close(c.done)
		c.writeMu.Lock()
		_ = c.Conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseGoingAway, "gateway shutting down"),
			time.Now().Add(time.Second),
		)
		err = c.Conn.Close()
		c.writeMu.Unlock()
	})
	return err
}
