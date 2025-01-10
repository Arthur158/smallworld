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
	Map	     Map
}

type Map struct {
	Name	string
	populateMap   func() []TileData
}

type TilePolygon struct {
    Coords []int `json:"coords"`
    StackX int   `json:"stackX"`
    StackY int   `json:"stackY"`
}

type TileData struct {
    ID      int         `json:"id"`
    Polygon TilePolygon `json:"polygon"`
    // ...
}
