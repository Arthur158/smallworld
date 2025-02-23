package server

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"backend/internal/gamestate"
)

func CreateGameStatesTable() {
	query := `
	CREATE TABLE IF NOT EXISTS game_states (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		state_json TEXT NOT NULL,
		saver_index INTEGER,
		summary TEXT NOT NULL,
		map_name TEXT NOT NULL,
		players_tribes TEXT NOT NULL
	);`
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Error creating game_states table:", err)
	}
	log.Println("Game_states table created successfully!")
}

type GameStateCopy struct {
	Players    []PlayerCopy
	TribeList  []TribeEntryCopy
	TileList   []TileCopy
	TurnInfo   TurnInfoCopy
}

type TribeEntryCopy struct {
	Race      string
	Trait     string
	CoinPile  int
	PiecePile int
}

type TurnInfoCopy struct {
	TurnIndex   int
	PlayerIndex int
	Phase       string
}

type PlayerCopy struct {
	ActiveTribe     TribeCopy
	PassiveTribes   []TribeCopy
	CoinPile        int
	PieceStacks     []PieceStackCopy
	HasActiveTribe  bool
	PointsEachTurn  []int
}

type TileCopy struct {
	Id                  string
	AdjacentTiles       []string
	PieceStacks         []PieceStackCopy
	OwningTribe         string
	Biome               string
	Attributes          []string
	Presence            string
	IsEdge              bool
	TileModifierPoints  []string
	TileModifierDefenses []string
}

type TribeCopy struct {
	Owner    int                    `json:"owner"`
	Race     string                 `json:"race"`
	Trait    string                 `json:"trait"`
	IsActive bool                   `json:"is_active"`
	State    map[string]interface{} `json:"state"`
}

type PieceStackCopy struct {
	Type   string
	Amount int
	Tribe  string
}

func transformGameState(state *gamestate.GameState) GameStateCopy {
	tileIDMap := make(map[*gamestate.Tile]string)
	for id, tile := range state.TileList {
		tileIDMap[tile] = id
	}

	players := make([]PlayerCopy, len(state.Players))
	for i, p := range state.Players {
		var activeTribe TribeCopy
		if p.ActiveTribe != nil {
			activeTribe = TribeCopy{
				Owner:    p.ActiveTribe.Owner.Index,
				Race:     string(p.ActiveTribe.Race),
				Trait:    string(p.ActiveTribe.Trait),
				IsActive: p.ActiveTribe.IsActive,
				State:    p.ActiveTribe.State,
			}
		}

		passiveTribes := make([]TribeCopy, len(p.PassiveTribes))
		for j, pt := range p.PassiveTribes {
			passiveTribes[j] = TribeCopy{
				Owner:    pt.Owner.Index,
				Race:     string(pt.Race),
				Trait:    string(pt.Trait),
				IsActive: pt.IsActive,
				State:    pt.State,
			}
		}

		pieceStacks := make([]PieceStackCopy, len(p.PieceStacks))
		for k, ps := range p.PieceStacks {
			race := ""
			if ps.Tribe != nil {
				race = string(ps.Tribe.Race)
			}
			pieceStacks[k] = PieceStackCopy{
				Type:   ps.Type,
				Amount: ps.Amount,
				Tribe:  race,
			}
		}

		players[i] = PlayerCopy{
			ActiveTribe:    activeTribe,
			PassiveTribes:  passiveTribes,
			CoinPile:       p.CoinPile,
			PieceStacks:    pieceStacks,
			HasActiveTribe: p.HasActiveTribe,
			PointsEachTurn: p.PointsEachTurn,
		}
	}

	tribeList := make([]TribeEntryCopy, len(state.TribeList))
	for i, te := range state.TribeList {
		tribeList[i] = TribeEntryCopy{
			Race:      string(te.Race),
			Trait:     string(te.Trait),
			CoinPile:  te.CoinPile,
			PiecePile: te.PiecePile,
		}
	}

	tiles := make([]TileCopy, 0, len(state.TileList))
	for _, t := range state.TileList {
		adjacentIDs := make([]string, len(t.AdjacentTiles))
		for i, adjTile := range t.AdjacentTiles {
			adjacentIDs[i] = adjTile.Id
		}

		attributes := make([]string, len(t.Attributes))
		for i, attr := range t.Attributes {
			attributes[i] = attr.String()
		}

		owningTribe := ""
		if t.OwningTribe != nil {
			owningTribe = string(t.OwningTribe.Race)
		}

		pieceStacks := make([]PieceStackCopy, len(t.PieceStacks))
		for k, ps := range t.PieceStacks {
			race := ""
			if ps.Tribe != nil {
				race = string(ps.Tribe.Race)
			}
			pieceStacks[k] = PieceStackCopy{
				Type:   ps.Type,
				Amount: ps.Amount,
				Tribe:  race,
			}
		}

		modifierPoints := []string{}
		for key := range t.ModifierPoints {
			modifierPoints = append(modifierPoints, key)
		}

		modifierDefenses := []string{}
		for key := range t.ModifierDefenses {
			modifierDefenses = append(modifierDefenses, key)
		}

		tiles = append(tiles, TileCopy{
			Id:                   t.Id,
			AdjacentTiles:        adjacentIDs,
			PieceStacks:          pieceStacks,
			OwningTribe:          owningTribe,
			Biome:                t.Biome.String(),
			Attributes:           attributes,
			Presence:             t.Presence.String(),
			IsEdge:               t.IsEdge,
			TileModifierPoints:   modifierPoints,
			TileModifierDefenses: modifierDefenses,
		})
	}

	turnInfo := TurnInfoCopy{
		TurnIndex:   state.TurnInfo.TurnIndex,
		PlayerIndex: state.TurnInfo.PlayerIndex,
		Phase:       state.TurnInfo.Phase.String(),
	}

	return GameStateCopy{
		Players:   players,
		TribeList: tribeList,
		TileList:  tiles,
		TurnInfo:  turnInfo,
	}
}

func reverseTransformGameState(copyState GameStateCopy) *gamestate.GameState {
	playerMap := make(map[int]*gamestate.Player)
	tribeMap := make(map[string]*gamestate.Tribe)

	players := make([]*gamestate.Player, len(copyState.Players))
	for i, pc := range copyState.Players {
		players[i] = &gamestate.Player{
			CoinPile:       pc.CoinPile,
			PieceStacks:    nil,
			HasActiveTribe: pc.HasActiveTribe,
			PointsEachTurn: pc.PointsEachTurn,
		}
		playerMap[i] = players[i]
	}

	for i, pc := range copyState.Players {
		activeTribe, _ := gamestate.CreateTribe(gamestate.Race(pc.ActiveTribe.Race), gamestate.Trait(pc.ActiveTribe.Trait))
		activeTribe.State = pc.ActiveTribe.State
		activeTribe.Owner = players[i]
		tribeMap[string(activeTribe.Race)] = activeTribe

		passiveTribes := make([]*gamestate.Tribe, len(pc.PassiveTribes))
		for j, pt := range pc.PassiveTribes {
			passiveTribes[j], _ = gamestate.CreateTribe(gamestate.Race(pt.Race), gamestate.Trait(pt.Trait))
			passiveTribes[j].State = pt.State
			passiveTribes[j].IsActive = false
			passiveTribes[j].Owner = players[i]
			tribeMap[string(passiveTribes[j].Race)] = passiveTribes[j]
		}

		players[i].ActiveTribe = activeTribe
		players[i].PassiveTribes = passiveTribes
	}

	for i := range players {
		players[i].PieceStacks = make([]gamestate.PieceStack, len(copyState.Players[i].PieceStacks))
		for j, stack := range copyState.Players[i].PieceStacks {
			if stack.Tribe == "" {
				players[i].PieceStacks[j] = gamestate.PieceStack{Type: stack.Type, Amount: stack.Amount}
			} else {
				players[i].PieceStacks[j] = gamestate.PieceStack{Type: stack.Type, Amount: stack.Amount, Tribe: tribeMap[stack.Tribe]}
			}
		}
	}

	tribeList := make([]*gamestate.TribeEntry, len(copyState.TribeList))
	for i, te := range copyState.TribeList {
		tribeList[i] = &gamestate.TribeEntry{
			Race:      gamestate.Race(te.Race),
			Trait:     gamestate.Trait(te.Trait),
			CoinPile:  te.CoinPile,
			PiecePile: te.PiecePile,
		}
	}


	tileList := make(map[string]*gamestate.Tile)
	for _, tc := range copyState.TileList {
		piecestacks := make([]gamestate.PieceStack, len(tc.PieceStacks))
		for i, stack := range tc.PieceStacks {
			if stack.Tribe == "" {
				piecestacks[i] = gamestate.PieceStack{Type: stack.Type, Amount: stack.Amount}
			} else {
				piecestacks[i] = gamestate.PieceStack{Type: stack.Type, Amount: stack.Amount, Tribe: tribeMap[stack.Tribe]}
			}
		}

		modifierPoints := make(map[string]func(int) int)
		for _, key := range tc.TileModifierPoints {
			modifierPoints[key] = gamestate.TileModifierPoints[key]
		}

		modifierDefenses := make(map[string]func(int, error) (int, error))
		for _, key := range tc.TileModifierDefenses {
			modifierDefenses[key] = gamestate.TileModifierDefenses[key]
		}

		tile := &gamestate.Tile{
			Id:              tc.Id,
			PieceStacks:     piecestacks,
			OwningTribe:     nil,
			Biome:           parseBiome(tc.Biome),
			Attributes:      parseAttributes(tc.Attributes),
			Presence:        parsePresence(tc.Presence),
			IsEdge:          tc.IsEdge,
			ModifierDefenses: modifierDefenses,
			ModifierPoints:   modifierPoints,
		}
		tileList[tc.Id] = tile
	}

	lostTribe := gamestate.CreateBaseTribe()
	lostTribe.Race = "Lost Tribe"
	lostTribe.Trait = "Lost"
	lostTribe.IsActive = false
	lostPlayer := gamestate.Player{
		PieceStacks : []gamestate.PieceStack{},
		ActiveTribe: lostTribe,
		Index: -1,
	}
	lostTribe.Owner = &lostPlayer
	tribeMap["Lost Tribe"] = lostTribe

	for _, tc := range copyState.TileList {
		tile := tileList[tc.Id]
		tile.AdjacentTiles = make([]*gamestate.Tile, len(tc.AdjacentTiles))
		for i, adjID := range tc.AdjacentTiles {
			tile.AdjacentTiles[i] = tileList[adjID]
		}

		tile.OwningTribe = tribeMap[tc.OwningTribe]
	}

	turnInfo := &gamestate.TurnInfo{
		TurnIndex:   copyState.TurnInfo.TurnIndex,
		PlayerIndex: copyState.TurnInfo.PlayerIndex,
		Phase:       parsePhase(copyState.TurnInfo.Phase),
	}

	return &gamestate.GameState{
		Players:   players,
		TribeList: tribeList,
		TileList:  tileList,
		TurnInfo:  turnInfo,
	}
}

func parseBiome(s string) gamestate.Biome {
	switch s {
	case "Forest":
		return gamestate.Forest
	case "Hill":
		return gamestate.Hill
	case "Field":
		return gamestate.Field
	case "Swamp":
		return gamestate.Swamp
	case "Water":
		return gamestate.Water
	case "Mountain":
		return gamestate.Mountain
	default:
		return gamestate.Forest
	}
}

func parseAttributes(attrs []string) []gamestate.Attribute {
	result := make([]gamestate.Attribute, len(attrs))
	for i, a := range attrs {
		switch a {
		case "Magic":
			result[i] = gamestate.Magic
		case "Mine":
			result[i] = gamestate.Mine
		case "Cave":
			result[i] = gamestate.Cave
		}
	}
	return result
}

func parsePresence(s string) gamestate.Presence {
	switch s {
	case "None":
		return gamestate.None
	case "Active":
		return gamestate.Active
	case "Passive":
		return gamestate.Passive
	default:
		return gamestate.None
	}
}

func parsePhase(s string) gamestate.Phase {
	switch s {
	case "TribeChoice":
		return gamestate.TribeChoice
	case "DeclineChoice":
		return gamestate.DeclineChoice
	case "TileAbandonment":
		return gamestate.TileAbandonment
	case "Conquest":
		return gamestate.Conquest
	case "Redeployment":
		return gamestate.Redeployment
	case "GameFinished":
		return gamestate.GameFinished
	default:
		return gamestate.TribeChoice
	}
}

func SaveGameState(state *gamestate.GameState, saverIndex int, mapName string) (int64, error) {
	copyState := transformGameState(state)

	// Build the summary
	player := copyState.Players[saverIndex]
	saverActiveTribe := player.ActiveTribe
	if !player.HasActiveTribe && len(player.PassiveTribes) > 0 {
		saverActiveTribe = player.PassiveTribes[0]
	}
	tribeString := fmt.Sprintf("%s %s", saverActiveTribe.Trait, saverActiveTribe.Race)
	if !player.HasActiveTribe && len(player.PassiveTribes) > 0 {
		tribeString = fmt.Sprintf("%s %s in decline", saverActiveTribe.Trait, saverActiveTribe.Race)
	}
	turnIndex := copyState.TurnInfo.TurnIndex
	playerCount := len(copyState.Players)
	summary := fmt.Sprintf("%s | TurnIndex: %d | Map: %s | Players: %d",
		tribeString, turnIndex, mapName, playerCount)

	// Build the players_tribes list
	playersTribes := make([]string, len(copyState.Players))
	for i, pc := range copyState.Players {
		if pc.HasActiveTribe {
			playersTribes[i] = fmt.Sprintf("%s %s", pc.ActiveTribe.Trait, pc.ActiveTribe.Race)
		} else if len(pc.PassiveTribes) > 0 {
			firstPassive := pc.PassiveTribes[0]
			playersTribes[i] = fmt.Sprintf("%s %s in decline", firstPassive.Trait, firstPassive.Race)
		} else {
			playersTribes[i] = ""
		}
	}

	playersTribesJSON, err := json.Marshal(playersTribes)
	if err != nil {
		log.Println("Error marshaling playersTribes:", err)
		return 0, err
	}

	jsonData, err := json.Marshal(copyState)
	if err != nil {
		log.Println(err)
		return 0, err
	}

	query := `INSERT INTO game_states (state_json, saver_index, summary, map_name, players_tribes)
			  VALUES (?, ?, ?, ?, ?);`
	result, err := db.Exec(query, string(jsonData), saverIndex, summary, mapName, string(playersTribesJSON))
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

func LoadGameInfo(id int64) (int, string, []string, error) {
	query := `SELECT "saver_index", map_name, players_tribes FROM game_states WHERE id = ?;`
	row := db.QueryRow(query, id)

	var index int
	var mapName string
	var playersTribesStr string

	err := row.Scan(&index, &mapName, &playersTribesStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, "", nil, errors.New("index or mapname not found")
		}
		return 0, "", nil, err
	}

	var playersTribes []string
	if err := json.Unmarshal([]byte(playersTribesStr), &playersTribes); err != nil {
		return index, mapName, nil, err
	}

	return index, mapName, playersTribes, nil
}

func LoadSummary(id int64) (string, error) {
	var summary string
	row := db.QueryRow("SELECT summary FROM game_states WHERE id = ?", id)
	if err := row.Scan(&summary); err != nil {
		return "", err
	}
	return summary, nil
}

func DeleteGameState(id int64) error {
	query := `DELETE FROM game_states WHERE id = ?;`
	result, err := db.Exec(query, id)
	if err != nil {
		log.Println("Error deleting game state:", err)
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return errors.New("game state not found")
	}

	log.Println("Game state deleted successfully with id:", id)
	return nil
}
