package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"backend/internal/gamestate"
)

func CreateGameStatesTable() {
	query := `
	CREATE TABLE IF NOT EXISTS game_states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state_json TEXT NOT NULL,
                saver_index INTEGER
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Error creating game_states table:", err)
	}
	log.Println("Game_states table created successfully!")
}

type GameStateCopy struct {
	Players []PlayerCopy
	TribeList []TribeEntryCopy
	TileList []TileCopy
	TurnInfo TurnInfoCopy
}

type TribeEntryCopy struct {
	Race string;
	Trait string;
	CoinPile int;
	PiecePile int;
}


type TurnInfoCopy struct {
	TurnIndex int;
	PlayerIndex int;
	Phase string;
}

type PlayerCopy struct {
	ActiveTribe TribeCopy;
	PassiveTribes []TribeCopy;
	CoinPile int
	PieceStacks []gamestate.PieceStack
	HasActiveTribe bool
	PointsEachTurn []int;
}

type TileCopy struct {
	Id string;
	AdjacentTiles []string;
	PieceStacks []gamestate.PieceStack;
	OwningPlayer int;
	OwningTribe string;
	Biome string;
	Attributes []string;
	Presence string;
	IsEdge bool;
}

type TribeCopy struct {
	Race      string                  `json:"race"`
	Trait     string                 `json:"trait"`
	IsActive  bool                  `json:"is_active"`
	State     map[string]interface{} `json:"state"`
}


func transformGameState(state *gamestate.GameState) GameStateCopy {
    // Create player mapping for quick lookups
    playerIndex := make(map[*gamestate.Player]int)
    for i, p := range state.Players {
        playerIndex[p] = i
    }

    // Create tile ID mapping
    tileIDMap := make(map[*gamestate.Tile]string)
    for id, tile := range state.TileList {
        tileIDMap[tile] = id
    }

    // Convert players
    players := make([]PlayerCopy, len(state.Players))
    for i, p := range state.Players {
        // Convert active tribe
        var activeTribe TribeCopy
        if p.ActiveTribe != nil {
            activeTribe = TribeCopy{
                Race:     string(p.ActiveTribe.Race),
                Trait:    string(p.ActiveTribe.Trait),
                IsActive: p.ActiveTribe.IsActive,
                State:    p.ActiveTribe.State,
            }
        }

        // Convert passive tribes
        passiveTribes := make([]TribeCopy, len(p.PassiveTribes))
        for j, pt := range p.PassiveTribes {
            passiveTribes[j] = TribeCopy{
                Race:     string(pt.Race),
                Trait:    string(pt.Trait),
                IsActive: pt.IsActive,
                State:    pt.State,
            }
        }

        players[i] = PlayerCopy{
            ActiveTribe:    activeTribe,
            PassiveTribes:  passiveTribes,
            CoinPile:       p.CoinPile,
            PieceStacks:    p.PieceStacks,
            HasActiveTribe: p.HasActiveTribe,
            PointsEachTurn: p.PointsEachTurn,
        }
    }

    // Convert tribe list
    tribeList := make([]TribeEntryCopy, len(state.TribeList))
    for i, te := range state.TribeList {
        tribeList[i] = TribeEntryCopy{
            Race:      string(te.Race),
            Trait:     string(te.Trait),
            CoinPile:  te.CoinPile,
            PiecePile: te.PiecePile,
        }
    }

    // Convert tiles
    tiles := make([]TileCopy, 0, len(state.TileList))
    for _, t := range state.TileList {
        // Convert adjacent tiles to IDs
        adjacentIDs := make([]string, len(t.AdjacentTiles))
        for i, adjTile := range t.AdjacentTiles {
            adjacentIDs[i] = adjTile.Id
        }

        // Convert attributes to strings
        attributes := make([]string, len(t.Attributes))
        for i, attr := range t.Attributes {
            attributes[i] = attr.String()
        }

        // Convert owning player index
        owningPlayerIndex := -1
        if t.OwningPlayer != nil && t.OwningPlayer.HasActiveTribe && t.OwningPlayer.ActiveTribe.Race != "Lost Tribe" {
            owningPlayerIndex = playerIndex[t.OwningPlayer]
        }

        // Convert owning tribe
        owningTribe := ""
        if t.OwningTribe != nil {
            owningTribe = string(t.OwningTribe.Race)
        }

        tiles = append(tiles, TileCopy{
            Id:             t.Id,
            AdjacentTiles:  adjacentIDs,
            PieceStacks:    t.PieceStacks,
            OwningPlayer:   owningPlayerIndex,
            OwningTribe:    owningTribe,
            Biome:          t.Biome.String(),
            Attributes:     attributes,
            Presence:       t.Presence.String(),
            IsEdge:         t.IsEdge,
        })
    }

    // Convert turn info
    turnInfo := TurnInfoCopy{
        TurnIndex:   state.TurnInfo.TurnIndex,
        PlayerIndex: state.TurnInfo.PlayerIndex,
        Phase:       state.TurnInfo.Phase.String(),
    }

    return GameStateCopy{
        Players:    players,
        TribeList:  tribeList,
        TileList:   tiles,
        TurnInfo:   turnInfo,
    }
}

func reverseTransformGameState(copyState GameStateCopy) *gamestate.GameState {
    // Helper maps for resolving references
    playerMap := make(map[int]*gamestate.Player)
    
    // Convert players first (needed for tile ownership)
    players := make([]*gamestate.Player, len(copyState.Players))
    for i, pc := range copyState.Players {
        // Convert tribes (without functions)
	activeTribe, _ := gamestate.CreateTribe(gamestate.Race(pc.ActiveTribe.Race), gamestate.Trait(pc.ActiveTribe.Trait))
	activeTribe.State = pc.ActiveTribe.State

        passiveTribes := make([]*gamestate.Tribe, len(pc.PassiveTribes))
        for j, pt := range pc.PassiveTribes {
	    passiveTribes[j], _ = gamestate.CreateTribe(gamestate.Race(pt.Race), gamestate.Trait(pt.Trait))
	    passiveTribes[j].State = pt.State
	    passiveTribes[j].IsActive = false
        }

        players[i] = &gamestate.Player{
            ActiveTribe:    activeTribe,
            PassiveTribes:  passiveTribes,
            CoinPile:       pc.CoinPile,
            PieceStacks:    pc.PieceStacks,
            HasActiveTribe: pc.HasActiveTribe,
            PointsEachTurn: pc.PointsEachTurn,
        }
        playerMap[i] = players[i]
    }

    // Convert tribe list
    tribeList := make([]*gamestate.TribeEntry, len(copyState.TribeList))
    for i, te := range copyState.TribeList {
        tribeList[i] = &gamestate.TribeEntry{
            Race:      gamestate.Race(te.Race),
            Trait:     gamestate.Trait(te.Trait),
            CoinPile:  te.CoinPile,
            PiecePile: te.PiecePile,
        }
    }

    // Convert tiles (first pass - create without adjacency)
    tileList := make(map[string]*gamestate.Tile)
    for _, tc := range copyState.TileList {
        tile := &gamestate.Tile{
            Id:           tc.Id,
            PieceStacks:  tc.PieceStacks,
            OwningPlayer: nil,  // Will resolve below
            OwningTribe:  nil,  // Will resolve below
            Biome:        parseBiome(tc.Biome),
            Attributes:   parseAttributes(tc.Attributes),
            Presence:     parsePresence(tc.Presence),
            IsEdge:       tc.IsEdge,
        }
        tileList[tc.Id] = tile
    }

    lostPlayer := gamestate.Player{
        PieceStacks : []gamestate.PieceStack{},
    }
    lostTribe := gamestate.CreateBaseTribe()
    lostTribe.Race = "Lost Tribe"
    lostTribe.Trait = "Lost"

    // Resolve tile ownership and adjacency
    for _, tc := range copyState.TileList {
        tile := tileList[tc.Id]
        
        // Resolve adjacent tiles
        tile.AdjacentTiles = make([]*gamestate.Tile, len(tc.AdjacentTiles))
        for i, adjID := range tc.AdjacentTiles {
            tile.AdjacentTiles[i] = tileList[adjID]
        }

        // Resolve owning player
        if tc.OwningPlayer >= 0 && tc.OwningPlayer < len(players) {
            tile.OwningPlayer = playerMap[tc.OwningPlayer]
        } else if tc.OwningPlayer == -1 && tc.Presence == "Passive" {
            tile.OwningPlayer = &lostPlayer
        }

	if tile.Presence == gamestate.Active {
	    tile.OwningTribe = tile.OwningPlayer.ActiveTribe
	} else if tile.Presence == gamestate.None {
	    tile.OwningTribe = nil
	} else if tile.Presence == gamestate.Passive && tile.OwningPlayer == &lostPlayer {
	    tile.OwningTribe = lostTribe
	} else {
	    for _, t := range tile.OwningPlayer.PassiveTribes {
		for _, s := range tile.PieceStacks {
		    if string(t.Race) == s.Type {
			tile.OwningTribe = t
		    }
		}
	    }
	}
    }

    // Convert turn info
    turnInfo := &gamestate.TurnInfo{
        TurnIndex:   copyState.TurnInfo.TurnIndex,
        PlayerIndex: copyState.TurnInfo.PlayerIndex,
        Phase:       parsePhase(copyState.TurnInfo.Phase),
    }

    return &gamestate.GameState{
        Players:    players,
        TribeList:  tribeList,
        TileList:   tileList,
        TurnInfo:   turnInfo,
    }
}

// Helper functions for enum parsing
func parseBiome(s string) gamestate.Biome {
    switch s {
    case "Forest": return gamestate.Forest
    case "Hill": return gamestate.Hill
    case "Field": return gamestate.Field
    case "Swamp": return gamestate.Swamp
    case "Water": return gamestate.Water
    case "Mountain": return gamestate.Mountain
    default: return gamestate.Forest
    }
}

func parseAttributes(attrs []string) []gamestate.Attribute {
    result := make([]gamestate.Attribute, len(attrs))
    for i, a := range attrs {
        switch a {
        case "Magic": result[i] = gamestate.Magic
        case "Mine": result[i] = gamestate.Mine
        case "Cave": result[i] = gamestate.Cave
        }
    }
    return result
}

func parsePresence(s string) gamestate.Presence {
    switch s {
    case "None": return gamestate.None
    case "Active": return gamestate.Active
    case "Passive": return gamestate.Passive
    default: return gamestate.None
    }
}

func parsePhase(s string) gamestate.Phase {
    switch s {
    case "TribeChoice": return gamestate.TribeChoice
    case "DeclineChoice": return gamestate.DeclineChoice
    case "TileAbandonment": return gamestate.TileAbandonment
    case "Conquest": return gamestate.Conquest
    case "Redeployment": return gamestate.Redeployment
    case "GameFinished": return gamestate.GameFinished
    default: return gamestate.TribeChoice
    }
}

func SaveGameState(state *gamestate.GameState, saverIndex int) (int64, error) {
	copy := transformGameState(state)
	jsonData, err := json.Marshal(copy)
	if err != nil {
		log.Println("are we here")
		log.Println(err)
		return 0, err
	}

	query := "INSERT INTO game_states (state_json, saver_index) VALUES (?, ?);"
	result, err := db.Exec(query, string(jsonData), saverIndex)
	if err != nil {
		log.Println("Error saving game state:", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return 0, err
	}

	log.Println("Game state saved successfully with id:", id)
	return id, nil
}

func LoadGameState(id int64) (*gamestate.GameState, int, error) {
	query := `SELECT state_json, "saver_index" FROM game_states WHERE id = ?;`
	row := db.QueryRow(query, id)

	var jsonStr string
	var index int
	err := row.Scan(&jsonStr, &index)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, 0, errors.New("game state not found")
		}
		return nil, 0, err
	}

	var statecp GameStateCopy
	err = json.Unmarshal([]byte(jsonStr), &statecp)
	if err != nil {
		return nil, 0, err
	}

	var state = reverseTransformGameState(statecp)

	return state, index, nil
}
