package server
import (
	"github.com/gorilla/websocket"
	"backend/internal/gamestate"
	"sync"
)

type Client struct {
	Conn     *websocket.Conn
	Username string
	Index	int
	Room   *Room
}

type Room struct {
	ID           string
	Name         string
	HostUsername string
	Players      []*Client
	MaxPlayers   int
	InProgress   bool
	Gamestate    gamestate.GameState
	mu	     sync.Mutex
}
