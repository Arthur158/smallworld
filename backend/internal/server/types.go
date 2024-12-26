package server
import (
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn     *websocket.Conn
	Stop chan struct{}
	Username string
	RoomID   string
	IsHost   bool
}

type Room struct {
	ID           string
	Name         string
	HostUsername string
	Players      []*Client
	MaxPlayers   int
	InProgress   bool
}
