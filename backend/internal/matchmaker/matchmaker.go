package matchmaker

import "sync"

type Matchmaker struct {
	Players []string
	mutex   sync.Mutex
}

func New() *Matchmaker {
	return &Matchmaker{Players: []string{}}
}

func (m *Matchmaker) AddPlayer(player string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Players = append(m.Players, player)
}

func (m *Matchmaker) GetPlayers() []string {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	copyPlayers := make([]string, len(m.Players))
	copy(copyPlayers, m.Players)
	return copyPlayers
}
