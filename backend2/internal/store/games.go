package store

import (
	"backend/internal/gamestate"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type GameStateCopy struct {
	Players   []PlayerCopy
	TribeList []TribeEntryCopy
	TileList  []TileCopy
	TurnInfo  TurnInfoCopy
	Powers    []PowerCopy
}

type PowerCopy struct {
	Name  string
	State map[string]interface{}
	Owner int
	Tile  string
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
	ActiveTribe    TribeCopy
	PassiveTribes  []TribeCopy
	CoinPile       int
	PieceStacks    []PieceStackCopy
	HasActiveTribe bool
	PointsEachTurn []int
}

type TileCopy struct {
	Id                      string
	AdjacentTiles           []string
	PieceStacks             []PieceStackCopy
	OwningTribe             string
	Biome                   string
	Attributes              []string
	IsEdge                  bool
	TileModifierPoints      []string
	TileModifierDefenses    []string
	ModifierAfterConquest   []string
	ModifierSpecialDefenses []string
	State                   map[string]interface{} `json:"state"`
}

type TribeCopy struct {
	Owner            int                    `json:"owner"`
	Race             string                 `json:"race"`
	Trait            string                 `json:"trait"`
	IsActive         bool                   `json:"is_active"`
	State            map[string]interface{} `json:"state"`
	AdditionalPowers []string               `json:"additional_powers"`
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

	powers := []PowerCopy{}
	for _, p := range state.Powers {
		tileId := ""
		if p.Tile == nil {
			tileId = ""
		} else {
			tileId = p.Tile.Id
		}
		powers = append(powers, PowerCopy{
			Name:  p.Name,
			State: p.State,
			Tile:  tileId,
			Owner: p.Owner.Index,
		})
	}

	players := make([]PlayerCopy, len(state.Players))
	for i, p := range state.Players {
		var activeTribe TribeCopy
		if p.ActiveTribe != nil {
			additionalPowersCp := []string{}
			for _, power := range p.ActiveTribe.AdditionalPowers {
				additionalPowersCp = append(additionalPowersCp, string(power))
			}
			activeTribe = TribeCopy{
				Owner:            p.ActiveTribe.Owner.Index,
				Race:             string(p.ActiveTribe.Race),
				Trait:            string(p.ActiveTribe.Trait),
				IsActive:         p.ActiveTribe.IsActive,
				State:            p.ActiveTribe.State,
				AdditionalPowers: additionalPowersCp,
			}
		}

		passiveTribes := make([]TribeCopy, len(p.PassiveTribes))
		for j, pt := range p.PassiveTribes {
			additionalPowersCp := []string{}
			for _, power := range pt.AdditionalPowers {
				additionalPowersCp = append(additionalPowersCp, string(power))
			}
			passiveTribes[j] = TribeCopy{
				Owner:            pt.Owner.Index,
				Race:             string(pt.Race),
				Trait:            string(pt.Trait),
				IsActive:         pt.IsActive,
				State:            pt.State,
				AdditionalPowers: additionalPowersCp,
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
			PointsEachTurn: p.PointsEachTurn,
			HasActiveTribe: p.ActiveTribe != nil,
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

		ModifierAfterConquest := []string{}
		for key := range t.ModifierAfterConquest {
			ModifierAfterConquest = append(ModifierAfterConquest, key)
		}

		ModifierSpecialDefenses := []string{}
		for key := range t.ModifierSpecialDefenses {
			ModifierSpecialDefenses = append(ModifierSpecialDefenses, key)
		}

		tiles = append(tiles, TileCopy{
			Id:                      t.Id,
			AdjacentTiles:           adjacentIDs,
			PieceStacks:             pieceStacks,
			OwningTribe:             owningTribe,
			Biome:                   t.Biome.String(),
			Attributes:              attributes,
			IsEdge:                  t.IsEdge,
			TileModifierPoints:      modifierPoints,
			TileModifierDefenses:    modifierDefenses,
			ModifierAfterConquest:   ModifierAfterConquest,
			ModifierSpecialDefenses: ModifierSpecialDefenses,
			State:                   t.State,
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
		Powers:    powers,
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
			PointsEachTurn: pc.PointsEachTurn,
			Index:          i,
		}
		playerMap[i] = players[i]
	}

	for i, pc := range copyState.Players {
		if pc.HasActiveTribe {
			activeTribe, _ := gamestate.CreateTribe(gamestate.Race(pc.ActiveTribe.Race), gamestate.Trait(pc.ActiveTribe.Trait))
			activeTribe.Owner = players[i]
			tribeMap[string(activeTribe.Race)] = activeTribe
			players[i].ActiveTribe = activeTribe
			for _, traitcp := range pc.ActiveTribe.AdditionalPowers {
				players[i].ActiveTribe.GiveTrait(gamestate.Trait(traitcp))
			}
			activeTribe.State = pc.ActiveTribe.State
		}

		passiveTribes := make([]*gamestate.Tribe, len(pc.PassiveTribes))
		for j, pt := range pc.PassiveTribes {
			passiveTribes[j], _ = gamestate.CreateTribe(gamestate.Race(pt.Race), gamestate.Trait(pt.Trait))
			passiveTribes[j].IsActive = false
			passiveTribes[j].Owner = players[i]
			tribeMap[string(passiveTribes[j].Race)] = passiveTribes[j]
			for _, traitcp := range pt.AdditionalPowers {
				passiveTribes[j].GiveTrait(gamestate.Trait(traitcp))
			}
			passiveTribes[j].State = pt.State
		}
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

		modifierPoints := make(map[string]func(*gamestate.Tile) int)
		for _, key := range tc.TileModifierPoints {
			modifierPoints[key] = gamestate.TileModifierPoints[key]
		}

		modifierDefenses := make(map[string]func(*gamestate.Tile, *gamestate.GameState) (int, int, int, error))
		for _, key := range tc.TileModifierDefenses {
			modifierDefenses[key] = gamestate.TileModifierDefenses[key]
		}

		ModifierAfterConquest := make(map[string]func(*gamestate.Tile, *gamestate.Tribe, *gamestate.GameState))
		for _, key := range tc.ModifierAfterConquest {
			if strings.HasSuffix(key, " Spawn") {
				powerName := strings.TrimSuffix(key, " Spawn")
				power := gamestate.PowerMap[powerName]()
				ModifierAfterConquest[key] = power.Spawn
			} else {
				ModifierAfterConquest[key] = gamestate.TileModifierAfterConquests[key]
			}
		}

		ModifierSpecialDefenses := make(map[string]func(*gamestate.Tile, *gamestate.GameState, *gamestate.Tribe, string) (bool, error))
		for _, key := range tc.ModifierSpecialDefenses {
			ModifierSpecialDefenses[key] = gamestate.TileModifierSpecialDefenses[key]
		}

		tile := &gamestate.Tile{
			Id:                      tc.Id,
			PieceStacks:             piecestacks,
			OwningTribe:             nil,
			Biome:                   parseBiome(tc.Biome),
			Attributes:              parseAttributes(tc.Attributes),
			IsEdge:                  tc.IsEdge,
			ModifierDefenses:        modifierDefenses,
			ModifierPoints:          modifierPoints,
			ModifierAfterConquest:   ModifierAfterConquest,
			ModifierSpecialDefenses: ModifierSpecialDefenses,
			State:                   tc.State,
		}
		tileList[tc.Id] = tile
	}

	powerList := make(map[string]*gamestate.Power, len(copyState.Powers))
	for _, pc := range copyState.Powers {
		power := gamestate.PowerMap[pc.Name]()
		power.Name = pc.Name
		power.Owner = playerMap[pc.Owner]
		power.State = pc.State
		if pc.Tile == "" {
			power.Tile = nil
		} else {
			power.Tile = tileList[pc.Tile]
		}
		powerList[pc.Name] = power
	}

	lostTribe := gamestate.CreateBaseTribe()
	lostTribe.Race = "Lost Tribe"
	lostTribe.Trait = "Lost"
	lostTribe.IsActive = false
	lostPlayer := gamestate.Player{
		PieceStacks: []gamestate.PieceStack{},
		ActiveTribe: lostTribe,
		Index:       -1,
	}
	lostTribe.Owner = &lostPlayer
	tribeMap["Lost Tribe"] = lostTribe

	for _, tc := range copyState.TileList {
		tile := tileList[tc.Id]
		tile.AdjacentTiles = make([]*gamestate.Tile, len(tc.AdjacentTiles))
		for i, adjID := range tc.AdjacentTiles {
			tile.AdjacentTiles[i] = tileList[adjID]
		}

		if strings.HasPrefix(tc.OwningTribe, "Monster") {
			monster := gamestate.CreateBaseTribe()
			monster.Race = gamestate.Race(tc.OwningTribe)
			monster.Trait = ""
			monster.Owner = &lostPlayer
			tile.OwningTribe = monster
		} else {
			tile.OwningTribe = tribeMap[tc.OwningTribe]
		}
	}

	turnInfo := &gamestate.TurnInfo{
		TurnIndex:   copyState.TurnInfo.TurnIndex,
		PlayerIndex: copyState.TurnInfo.PlayerIndex,
		Phase:       parsePhase(copyState.TurnInfo.Phase),
	}

	return &gamestate.GameState{
		Players:            players,
		TribeList:          tribeList,
		TileList:           tileList,
		Powers:             powerList,
		TurnInfo:           turnInfo,
		ModifierPoints:     make(map[string]func(int, *gamestate.Player) int),
		ModifierTurnsAfter: []gamestate.TurninfoEntry{},
		ModifierAfterPick:  make(map[string]func(int, *gamestate.TribeEntry)),
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
	case "MineArea":
		return gamestate.MineArea
	case "CrystalArea":
		return gamestate.CrystalArea
	case "AbyssalChasm":
		return gamestate.AbyssalChasm
	case "River":
		return gamestate.River
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

func EncodeGameState(state *gamestate.GameState) ([]byte, error) {
	copyState := transformGameState(state)
	return json.Marshal(copyState)
}

func DecodeGameState(data []byte) (*gamestate.GameState, error) {
	var copyState GameStateCopy
	if err := json.Unmarshal(data, &copyState); err != nil {
		return nil, err
	}
	return reverseTransformGameState(copyState), nil
}

func (s *Store) SaveGameState(ctx context.Context, state *gamestate.GameState, saverIndex int, mapName string) (int64, error) {
	copyState := transformGameState(state)
	if saverIndex < 0 || saverIndex >= len(copyState.Players) {
		return 0, fmt.Errorf("invalid saver index %d", saverIndex)
	}

	player := copyState.Players[saverIndex]
	saverActiveTribe := player.ActiveTribe
	tribeString := "No tribe"
	if player.HasActiveTribe {
		tribeString = strings.TrimSpace(fmt.Sprintf("%s %s", saverActiveTribe.Trait, saverActiveTribe.Race))
	} else if len(player.PassiveTribes) > 0 {
		saverActiveTribe = player.PassiveTribes[0]
		tribeString = strings.TrimSpace(fmt.Sprintf("%s %s in decline", saverActiveTribe.Trait, saverActiveTribe.Race))
	}

	summary := fmt.Sprintf("%s | Turn: %d | %s", tribeString, copyState.TurnInfo.TurnIndex, mapName)

	playersTribes := make([]string, len(copyState.Players))
	for i, pc := range copyState.Players {
		if pc.HasActiveTribe {
			playersTribes[i] = strings.TrimSpace(fmt.Sprintf("%s %s", pc.ActiveTribe.Trait, pc.ActiveTribe.Race))
		} else if len(pc.PassiveTribes) > 0 {
			firstPassive := pc.PassiveTribes[0]
			playersTribes[i] = strings.TrimSpace(fmt.Sprintf("%s %s in decline", firstPassive.Trait, firstPassive.Race))
		}
	}

	playersTribesJSON, err := json.Marshal(playersTribes)
	if err != nil {
		return 0, fmt.Errorf("marshal players tribes: %w", err)
	}
	stateJSON, err := json.Marshal(copyState)
	if err != nil {
		return 0, fmt.Errorf("marshal game state: %w", err)
	}

	var id int64
	err = s.db.QueryRowContext(ctx, `
INSERT INTO game_states (state_json, saver_index, summary, map_name, players_tribes)
VALUES ($1::jsonb, $2, $3, $4, $5::jsonb)
RETURNING id
`, stateJSON, saverIndex, summary, mapName, playersTribesJSON).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("save game state: %w", err)
	}
	return id, nil
}

func (s *Store) LoadGameState(ctx context.Context, id int64) (*gamestate.GameState, int, error) {
	var stateJSON []byte
	var saverIndex int
	err := s.db.QueryRowContext(ctx,
		`SELECT state_json, saver_index FROM game_states WHERE id = $1`, id,
	).Scan(&stateJSON, &saverIndex)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, 0, errors.New("game state not found")
		}
		return nil, 0, err
	}

	state, err := DecodeGameState(stateJSON)
	if err != nil {
		return nil, 0, err
	}
	return state, saverIndex, nil
}

func (s *Store) LoadGameInfo(ctx context.Context, id int64) (int, string, []string, error) {
	var saverIndex int
	var mapName string
	var playersTribesJSON []byte
	err := s.db.QueryRowContext(ctx,
		`SELECT saver_index, map_name, players_tribes FROM game_states WHERE id = $1`, id,
	).Scan(&saverIndex, &mapName, &playersTribesJSON)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", nil, errors.New("game state not found")
		}
		return 0, "", nil, err
	}

	var playersTribes []string
	if err := json.Unmarshal(playersTribesJSON, &playersTribes); err != nil {
		return saverIndex, mapName, nil, err
	}
	return saverIndex, mapName, playersTribes, nil
}

func (s *Store) LoadSummary(ctx context.Context, id int64) (string, error) {
	var summary string
	err := s.db.QueryRowContext(ctx, `SELECT summary FROM game_states WHERE id = $1`, id).Scan(&summary)
	return summary, err
}

func (s *Store) DeleteGameState(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM game_states WHERE id = $1`, id)
	if err != nil {
		return err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return errors.New("game state not found")
	}
	return nil
}

func (s *Store) AddSaveToUser(ctx context.Context, username string, gameStateID int64) error {
	result, err := s.db.ExecContext(ctx, `
INSERT INTO user_savegames (user_id, game_state_id)
SELECT id, $2 FROM users WHERE username = $1
ON CONFLICT DO NOTHING
`, username, gameStateID)
	if err != nil {
		return fmt.Errorf("add save to user: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		var exists bool
		if err := s.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`, username).Scan(&exists); err != nil {
			return err
		}
		if !exists {
			return ErrUserNotFound
		}
	}
	return nil
}

func (s *Store) RemoveSaveLink(ctx context.Context, username string, gameStateID int64) error {
	_, err := s.db.ExecContext(ctx, `
DELETE FROM user_savegames
WHERE user_id = (SELECT id FROM users WHERE username = $1)
  AND game_state_id = $2
`, username, gameStateID)
	return err
}
