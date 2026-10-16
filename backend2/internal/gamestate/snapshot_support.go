package gamestate

import (
	"encoding/json"
	"sort"
)

// JSON cannot encode maps keyed by BotAction. Store feedback as ordered records
// so a worker restart or playback rewind preserves rejected intents exactly.
func (m BotV2Memory) MarshalJSON() ([]byte, error) {
	type record struct {
		Action   BotAction
		Rejected bool
		Uses     int
	}
	records := []record{}
	keys := map[BotAction]bool{}
	for a := range m.Rejected {
		keys[a] = true
	}
	for a := range m.Uses {
		keys[a] = true
	}
	for a := range keys {
		records = append(records, record{a, m.Rejected[a], m.Uses[a]})
	}
	sort.Slice(records, func(i, j int) bool {
		a, _ := json.Marshal(records[i].Action)
		b, _ := json.Marshal(records[j].Action)
		return string(a) < string(b)
	})
	return json.Marshal(struct {
		Turn    string
		Placed  map[string]bool
		Actions []record
	}{m.Turn, m.Placed, records})
}
func (m *BotV2Memory) UnmarshalJSON(data []byte) error {
	var saved struct {
		Turn    string
		Placed  map[string]bool
		Actions []struct {
			Action   BotAction
			Rejected bool
			Uses     int
		}
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	m.Turn = saved.Turn
	m.Placed = saved.Placed
	if m.Placed == nil {
		m.Placed = map[string]bool{}
	}
	m.Rejected = map[BotAction]bool{}
	m.Uses = map[BotAction]int{}
	for _, a := range saved.Actions {
		if a.Rejected {
			m.Rejected[a.Action] = true
		}
		if a.Uses > 0 {
			m.Uses[a.Action] = a.Uses
		}
	}
	return nil
}

// Queued surface-rule turns either have no callback or clear a Skag attack.
// Persist the callback's identity, never a Go closure.
func (e TurninfoEntry) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		TurnInfo    *TurnInfo
		Player      int
		CleanupTile string
	}{e.TurnInfo, e.player, e.cleanupTile})
}
func (e *TurninfoEntry) UnmarshalJSON(data []byte) error {
	var saved struct {
		TurnInfo    *TurnInfo
		Player      int
		CleanupTile string
	}
	if err := json.Unmarshal(data, &saved); err != nil {
		return err
	}
	e.TurnInfo, e.player, e.cleanupTile = saved.TurnInfo, saved.Player, saved.CleanupTile
	if e.cleanupTile != "" {
		e.actionBefore = func(gs *GameState) {
			tile := gs.TileList[e.cleanupTile]
			if tile == nil {
				return
			}
			delete(tile.ModifierDefenses, "Skag Attack")
			for i := len(tile.PieceStacks) - 1; i >= 0; i-- {
				if tile.PieceStacks[i].Type == "Skag Attack" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					break
				}
			}
		}
	}
	return nil
}

// State fields are typed by the game rules, while JSON decodes generic numbers
// and arrays. Recover those types before rule handlers inspect them.
func NormalizeSavedState(state map[string]interface{}) map[string]interface{} {
	if state == nil {
		return map[string]interface{}{}
	}
	for key, value := range state {
		switch v := value.(type) {
		case float64:
			state[key] = int(v)
		case []interface{}:
			if key == "mountains" || key == "abandonedTiles" {
				out := []string{}
				for _, x := range v {
					if s, ok := x.(string); ok {
						out = append(out, s)
					}
				}
				state[key] = out
			} else if key == "loots" || key == "playersAttacked" {
				out := []int{}
				for _, x := range v {
					if n, ok := x.(float64); ok {
						out = append(out, int(n))
					}
				}
				state[key] = out
			}
		case map[string]interface{}:
			if key == "deploy" {
				out := map[string]int{}
				for id, x := range v {
					if n, ok := x.(float64); ok {
						out[id] = int(n)
					}
				}
				state[key] = out
			}
		}
	}
	return state
}

// Rebind persistent market effects without replaying turn-start rewards.
func (gs *GameState) RestoreMarketHooks(keys []string) {
	for _, key := range keys {
		for _, p := range gs.Players {
			for _, t := range append(append([]*Tribe{}, p.PassiveTribes...), p.ActiveTribe) {
				if t == nil {
					continue
				}
				owns := string(t.Trait) == key
				for _, extra := range t.AdditionalPowers {
					owns = owns || string(extra) == key
				}
				if !owns {
					continue
				}
				switch key {
				case "Racketeering":
					gs.ModifierAfterPick[key] = func(i int, entry *TribeEntry) {
						if (gs.TurnInfo.PlayerIndex != t.Owner.Index) == t.IsActive {
							t.Owner.CoinPile += i
						}
					}
				case "Copycat":
					gs.ModifierAfterPick[key] = func(i int, entry *TribeEntry) {
						name, _ := t.State["additionalpower"].(string)
						if name != "" && string(entry.Trait) == name {
							t.State["additionalpower"] = ""
							t.DeletePower(name, gs)
							t.Owner.PieceStacks = append(t.Owner.PieceStacks, PieceStack{Type: "Mirror", Amount: 1})
						}
					}
				}
			}
		}
	}
}
