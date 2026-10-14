package bot_test

import (
	"backend/internal/bot"
	"backend/internal/gamestate"
	"backend/internal/store"
	"encoding/json"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"testing"
)

func newGame(t *testing.T, seed uint64, count int, gameMap string) *gamestate.GameState {
	t.Helper()
	names := make([]string, count)
	gs, err := gamestate.NewSeeded(names, gameMap, gamestate.BotRaces(), gamestate.BotTraits(), nil, seed)
	if err != nil {
		t.Fatal(err)
	}
	return gs
}

func TestFeaturesDoNotDependOnTileLabelsOrOrder(t *testing.T) {
	gs := newGame(t, 198, 3, "map3players")
	rng := rand.New(rand.NewSource(51))
	for step := 0; step < 200 && gs.TurnInfo.Phase != gamestate.GameFinished; step++ {
		d, err := gs.BotDecision()
		if err != nil {
			t.Fatal(err)
		}
		data, err := store.EncodeGameState(gs)
		if err != nil {
			t.Fatal(err)
		}
		clone, err := store.DecodeGameState(data)
		if err != nil {
			t.Fatal(err)
		}
		ids := make([]string, 0, len(clone.TileList))
		for id := range clone.TileList {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		renames := map[string]string{"": ""}
		tiles := make(map[string]*gamestate.Tile)
		for i, old := range ids {
			id := fmt.Sprintf("arbitrary-%03d", len(ids)-i)
			renames[old] = id
			tile := clone.TileList[old]
			tile.Id = id
			for a, b := 0, len(tile.AdjacentTiles)-1; a < b; a, b = a+1, b-1 {
				tile.AdjacentTiles[a], tile.AdjacentTiles[b] = tile.AdjacentTiles[b], tile.AdjacentTiles[a]
			}
			tiles[id] = tile
		}
		clone.TileList = tiles
		renamed, err := clone.BotDecision()
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(d.State, renamed.State) || len(d.Candidates) != len(renamed.Candidates) {
			t.Fatal("renaming tiles changed state features or available move count")
		}
		byAction := make(map[gamestate.BotAction][]float64)
		for _, c := range renamed.Candidates {
			byAction[c.Action] = c.Features
		}
		for _, c := range d.Candidates {
			a := c.Action
			a.Tile = renames[a.Tile]
			if !reflect.DeepEqual(c.Features, byAction[a]) {
				t.Fatalf("labels leaked into features for %+v", c.Action)
			}
		}
		if err := gs.ApplyBotAction(rng.Intn(len(d.Candidates))); err != nil {
			t.Fatal(err)
		}
	}
}

func canonical(t *testing.T, gs *gamestate.GameState) store.GameStateCopy {
	t.Helper()
	data, err := store.EncodeGameState(gs)
	if err != nil {
		t.Fatal(err)
	}
	var state store.GameStateCopy
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	sort.Slice(state.TileList, func(i, j int) bool { return state.TileList[i].Id < state.TileList[j].Id })
	return state
}

func TestCompleteMatchesAndPureDecisions(t *testing.T) {
	registry, _ := bot.NewRegistry("")
	maps := []string{"map2players", "underground2players", "map3players", "map4players", "map5players", "map6players3islands"}
	for seed := uint64(1); seed <= 120; seed++ {
		players := 2 + int(seed%5)
		gs := newGame(t, seed, players, maps[seed%uint64(len(maps))])
		rng := rand.New(rand.NewSource(int64(seed)))
		for step := 0; ; step++ {
			if step > 4096 {
				t.Fatalf("seed %d did not finish", seed)
			}
			before := canonical(t, gs)
			d, err := gs.BotDecision()
			if err != nil {
				t.Fatalf("seed %d step %d: %v", seed, step, err)
			}
			if !reflect.DeepEqual(before, canonical(t, gs)) {
				t.Fatalf("decision mutated state (seed %d)", seed)
			}
			if len(d.State) != gamestate.BotStateFeatures {
				t.Fatal("state schema width mismatch", len(d.State))
			}
			for _, c := range d.Candidates {
				if len(c.Features) != gamestate.BotActionFeatures {
					t.Fatal("action schema width mismatch", len(c.Features))
				}
			}
			if gs.TurnInfo.Phase == gamestate.GameFinished {
				if len(d.Candidates) != 0 {
					t.Fatal("terminal has actions")
				}
				break
			}
			policy := "random"
			if seed%2 == 0 {
				policy = "greedy"
			}
			a, err := registry.Select(policy, d, rng)
			if err != nil {
				t.Fatal(err)
			}
			if err := gs.ApplyBotAction(a); err != nil {
				t.Fatalf("seed %d step %d action %+v: %v", seed, step, d.Candidates[a].Action, err)
			}
			gs.Messages = nil
		}
	}
}

func TestEveryOfferedMoveIsAcceptedAfterSnapshot(t *testing.T) {
	gs := newGame(t, 42, 3, "map3players")
	rng := rand.New(rand.NewSource(13))
	for step := 0; step < 500 && gs.TurnInfo.Phase != gamestate.GameFinished; step++ {
		d, err := gs.BotDecision()
		if err != nil {
			t.Fatal(err)
		}
		data, err := store.EncodeGameState(gs)
		if err != nil {
			t.Fatal(err)
		}
		for i, c := range d.Candidates {
			copy, err := store.DecodeGameState(data)
			if err != nil {
				t.Fatal(err)
			}
			copyDecision, err := copy.BotDecision()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(d, copyDecision) {
				t.Fatal("snapshot changed policy inputs")
			}
			if err := copy.ApplyBotAction(i); err != nil {
				t.Fatalf("offered action %+v failed: %v", c.Action, err)
			}
		}
		if err := gs.ApplyBotAction(rng.Intn(len(d.Candidates))); err != nil {
			t.Fatal(err)
		}
	}
}

func TestDeterminismPrivateScoresAndInvalidActions(t *testing.T) {
	a, b := newGame(t, 99, 2, "map2players"), newGame(t, 99, 2, "map2players")
	rng := rand.New(rand.NewSource(7))
	for a.TurnInfo.Phase != gamestate.GameFinished {
		x, _ := a.BotDecision()
		y, _ := b.BotDecision()
		if !reflect.DeepEqual(x, y) {
			t.Fatal("seeded games diverged")
		}
		before := canonical(t, a)
		if a.ApplyBotAction(-1) == nil || a.ApplyBotAction(len(x.Candidates)) == nil {
			t.Fatal("accepted invalid action")
		}
		if !reflect.DeepEqual(before, canonical(t, a)) {
			t.Fatal("invalid action mutated state")
		}
		other := 1 - x.Player
		a.Players[other].CoinPile += 77
		private, _ := a.BotDecision()
		if !reflect.DeepEqual(x, private) {
			t.Fatal("opponent's coins leaked into features")
		}
		a.Players[other].CoinPile -= 77
		i := rng.Intn(len(x.Candidates))
		if err := a.ApplyBotAction(i); err != nil {
			t.Fatal(err)
		}
		if err := b.ApplyBotAction(i); err != nil {
			t.Fatal(err)
		}
	}
}
