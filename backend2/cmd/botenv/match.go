package main

import (
	"backend/internal/gamestate"
	"backend/internal/store"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math/rand"
	"sort"
)

type matchResult struct {
	Players           int              `json:"players"`
	SuccessfulActions []map[string]int `json:"successful_actions"`
	Scores            []int            `json:"scores"`
	Steps             int              `json:"steps"`
	Actions           []int            `json:"actions"`
	Rejections        []int            `json:"rejections"`
	InitialHash       string           `json:"initial_sha256"`
	TrajectoryHash    string           `json:"trajectory_sha256"`
}

// Encoding maps is stable, but the save format also contains slices built from
// map iteration. Sort those before hashing/storing a comparison fixture.
func matchSnapshot(state *gamestate.GameState) (json.RawMessage, error) {
	data, err := store.EncodeGameState(state)
	if err != nil {
		return nil, err
	}
	var saved store.GameStateCopy
	if err := json.Unmarshal(data, &saved); err != nil {
		return nil, err
	}
	sort.Slice(saved.TileList, func(i, j int) bool { return saved.TileList[i].Id < saved.TileList[j].Id })
	sort.Slice(saved.Powers, func(i, j int) bool { return saved.Powers[i].Name < saved.Powers[j].Name })
	for i := range saved.TileList {
		sort.Strings(saved.TileList[i].AdjacentTiles)
	}
	return json.Marshal(saved)
}

func (e *environment) matchSetup(req request) (response, error) {
	if req.Players < 2 || req.Players > 6 || req.Focus {
		return response{}, fmt.Errorf("comparison setup requires 2–6 players and no forced curriculum")
	}
	req.Op = "reset"
	if _, err := e.handle(req); err != nil {
		return response{}, err
	}
	data, err := matchSnapshot(e.state)
	if err != nil {
		return response{}, err
	}
	// Normalize through the same restoration path used by every match.
	e.state, err = store.DecodeGameState(data)
	if err != nil {
		return response{}, err
	}
	data, err = matchSnapshot(e.state)
	if err != nil {
		return response{}, err
	}
	return response{InitialState: data, InitialHash: fmt.Sprintf("%x", sha256.Sum256(data))}, nil
}

func (e *environment) playMatch(req request) (response, error) {
	players := len(req.Policies)
	if players < 2 || players > 6 || len(req.InitialState) == 0 {
		return response{}, fmt.Errorf("match needs 2–6 policies and an initial snapshot")
	}
	if req.Players != 0 && req.Players != players {
		return response{}, fmt.Errorf("player count differs from policy count")
	}
	if req.Rules != "legacy" && req.Rules != "surface" {
		return response{}, fmt.Errorf("match rules must be legacy or surface")
	}
	for _, id := range req.Policies {
		if !e.policies.Has(id) {
			return response{}, fmt.Errorf("unknown model %q", id)
		}
		if req.Rules == "surface" && id != "random" && id != "greedy" && e.policies.Schema(id) == gamestate.BotSchema {
			return response{}, fmt.Errorf("legacy model %q cannot play expanded surface rules", id)
		}
	}
	state, err := store.DecodeGameState(req.InitialState)
	if err != nil {
		return response{}, err
	}
	if len(state.Players) != players || state.TurnInfo == nil {
		return response{}, fmt.Errorf("snapshot player count must match policies and include turn information")
	}
	if req.Rules == "surface" {
		err = state.ValidateBotStateV2()
	} else {
		err = state.ValidateBotState()
	}
	if err != nil {
		return response{}, err
	}
	initial, err := matchSnapshot(state)
	if err != nil {
		return response{}, err
	}
	result := &matchResult{Players: players, InitialHash: fmt.Sprintf("%x", sha256.Sum256(initial)), Actions: make([]int, players), Rejections: make([]int, players)}
	if req.InitialHash == "" || result.InitialHash != req.InitialHash {
		return response{}, fmt.Errorf("restored initial snapshot checksum differs")
	}
	result.SuccessfulActions = make([]map[string]int, players)
	for i := range result.SuccessfulActions {
		result.SuccessfulActions[i] = map[string]int{}
	}
	rngs := make([]*rand.Rand, players)
	for seat := range rngs {
		rngs[seat] = rand.New(rand.NewSource(int64(req.Seed) + int64(seat)*1000003))
	}
	trace := sha256.New()
	for state.TurnInfo.Phase != gamestate.GameFinished {
		if result.Steps >= e.limit {
			return response{}, fmt.Errorf("match reached %d actions without finishing", e.limit)
		}
		seat := state.TurnInfo.PlayerIndex
		if seat < 0 || seat >= players {
			return response{}, fmt.Errorf("invalid current player")
		}
		id := req.Policies[seat]
		surface := e.policies.Schema(id) == gamestate.BotSchemaV2 || ((id == "random" || id == "greedy") && req.Rules == "surface")
		var decision gamestate.BotDecision
		if surface {
			decision, err = state.BotDecisionV2()
		} else {
			decision, err = state.BotDecision()
		}
		if err != nil {
			return response{}, fmt.Errorf("seat %d model %s decision: %w", seat, id, err)
		}
		choice, err := e.policies.Select(id, decision, rngs[seat])
		if err != nil {
			return response{}, err
		}
		rejected := false
		if surface {
			rejected, err = state.ApplyBotActionV2(choice)
		} else {
			err = state.ApplyBotAction(choice)
		}
		if err != nil {
			return response{}, fmt.Errorf("seat %d model %s action: %w", seat, id, err)
		}
		result.Actions[seat]++
		if rejected {
			result.Rejections[seat]++
		} else {
			a := decision.Candidates[choice].Action
			result.SuccessfulActions[seat][a.Kind+":"+a.Stack]++
		}
		result.Steps++
		json.NewEncoder(trace).Encode(struct {
			Seat     int
			Action   gamestate.BotAction
			Rejected bool
			RNG      uint64
		}{seat, decision.Candidates[choice].Action, rejected, state.RNGState})
		state.Messages = nil
	}
	for _, player := range state.Players {
		result.Scores = append(result.Scores, player.CoinPile)
	}
	result.TrajectoryHash = fmt.Sprintf("%x", trace.Sum(nil))
	return response{Done: true, Match: result}, nil
}
