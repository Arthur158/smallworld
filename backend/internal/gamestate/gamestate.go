package gamestate

import (
	"sync"
	"fmt"
)

type GameState struct {
	Actions []string
	Players []Player
	TribeList []TribeEntry
	TileList []Tile
	Turn int
	PlayerTurn int
	Mutex   sync.Mutex
}

func New(playerCount int) (*GameState, error) {
	// Create a list of initialized players
	players := make([]Player, playerCount)
	for i := 0; i < playerCount; i++ {
		players[i] = Player{
			ActiveTribe:    nil,
			PassiveTribes:  []*Tribe{}, // Initialize as empty slice
			OwnedTiles:     []*Tile{},  // Initialize as empty slice
			CoinPile: 5,
		}
	}

	tribelist, err := createTribeList()

	tilelist := Map1()

        if err != nil {
            return nil, fmt.Errorf("failed to create list of tribe entries", err)
        }

	return &GameState{
		Actions:	[]string{},
		Players:	players,
		TribeList:	tribelist,
		TileList:	tilelist,
		Turn:		0,
		PlayerTurn:	0,
		Mutex:		sync.Mutex{},
	}, nil
}

func GetTribeEntries(gs *GameState) []TribeEntry {
	return gs.TribeList[:5]
}

func (gs *GameState) ApplyAction(action string) {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	gs.Actions = append(gs.Actions, action)
}

func (gs *GameState) GetActions() []string {
	gs.Mutex.Lock()
	defer gs.Mutex.Unlock()
	copyActions := make([]string, len(gs.Actions))
	copy(copyActions, gs.Actions)
	return copyActions
}
