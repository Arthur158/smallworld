package gateway

import "sync"

type ConnectionManager struct {
	mu sync.RWMutex

	byID   map[string]*Client
	byRoom map[string]map[string]*Client
}

func newConnectionManager() *ConnectionManager {
	return &ConnectionManager{
		byID:   make(map[string]*Client),
		byRoom: make(map[string]map[string]*Client),
	}
}

func (m *ConnectionManager) Add(c *Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.byID[c.ID] = c
}

// Remove returns the room subscriptions that this connection held. The caller
// releases those subscriptions after dropping the connection from the manager.
func (m *ConnectionManager) Remove(c *Client) (rooms []string, removed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.byID[c.ID]; !ok {
		return nil, false
	}
	delete(m.byID, c.ID)

	s := c.snapshot()
	seen := make(map[string]struct{})
	for _, roomID := range []string{s.RoomID, s.DisplayRoomID} {
		if roomID == "" {
			continue
		}
		if clients := m.byRoom[roomID]; clients != nil {
			delete(clients, c.ID)
			if len(clients) == 0 {
				delete(m.byRoom, roomID)
			}
		}
		if _, exists := seen[roomID]; !exists {
			rooms = append(rooms, roomID)
			seen[roomID] = struct{}{}
		}
	}
	return rooms, true
}

// Bind changes one of the client's two room bindings. It returns the old room
// and whether the new room represents a new subscription for this connection.
func (m *ConnectionManager) Bind(c *Client, scope, roomID string, spectator bool) (oldRoomID string, changed bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	oldRoomID = c.setBinding(scope, roomID, spectator)
	if oldRoomID == roomID {
		return oldRoomID, false
	}

	if oldRoomID != "" {
		if clients := m.byRoom[oldRoomID]; clients != nil {
			delete(clients, c.ID)
			if len(clients) == 0 {
				delete(m.byRoom, oldRoomID)
			}
		}
	}

	if roomID != "" {
		if m.byRoom[roomID] == nil {
			m.byRoom[roomID] = make(map[string]*Client)
		}
		m.byRoom[roomID][c.ID] = c
	}
	return oldRoomID, true
}

func (m *ConnectionManager) Get(id string) (*Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.byID[id]
	return c, ok
}

func (m *ConnectionManager) All() []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Client, 0, len(m.byID))
	for _, c := range m.byID {
		out = append(out, c)
	}
	return out
}

func (m *ConnectionManager) InRoom(roomID string) []*Client {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clients := m.byRoom[roomID]
	out := make([]*Client, 0, len(clients))
	for _, c := range clients {
		out = append(out, c)
	}
	return out
}
