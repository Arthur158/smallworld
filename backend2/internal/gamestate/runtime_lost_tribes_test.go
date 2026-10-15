package gamestate_test

import (
	"backend/internal/gamestate"
	"backend/internal/store"
	"reflect"
	"testing"
)

func runtimeLostTribeState(t *testing.T, specs []gamestate.RuntimeTileSpec, powers []string) *gamestate.GameState {
	t.Helper()
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	gamestate.RegisterRuntimeMap(t.Name(), specs)
	gs, err := gamestate.NewSeeded([]string{"one", "two"}, t.Name(), []string{"Humans"}, []string{"Alchemist"}, powers, 47)
	if err != nil {
		t.Fatal(err)
	}
	return gs
}

func assertRuntimeLostTribe(t *testing.T, tile *gamestate.Tile) {
	t.Helper()
	if tile == nil || tile.OwningTribe == nil {
		t.Fatal("flagged land has no neutral tribe")
	}
	tribe := tile.OwningTribe
	if tribe.Race != "Lost Tribe" || tribe.Trait != "Lost" || tribe.IsActive || tile.CheckPresence() != gamestate.Passive {
		t.Fatalf("noncanonical neutral tribe on %s: race=%q trait=%q active=%v", tile.Id, tribe.Race, tribe.Trait, tribe.IsActive)
	}
	if tribe.Owner == nil || tribe.Owner.Index != -1 || tribe.Owner.ActiveTribe != tribe {
		t.Fatalf("neutral owner on %s is not independent of the real players", tile.Id)
	}
	if len(tile.PieceStacks) != 1 || tile.PieceStacks[0].Type != "Lost Tribe" || tile.PieceStacks[0].Amount != 1 {
		t.Fatalf("expected exactly one Lost Tribe piece on %s, got %+v", tile.Id, tile.PieceStacks)
	}
}

func TestRuntimeLostTribesDefaultRemainsEmpty(t *testing.T) {
	specs := []gamestate.RuntimeTileSpec{
		{ID: "field", Biome: gamestate.Field, IsEdge: true, Attributes: []gamestate.Attribute{gamestate.Magic}, AdjacentIDs: []string{"mountain"}},
		{ID: "mountain", Biome: gamestate.Mountain, AdjacentIDs: []string{"field"}},
		{ID: "forest", Biome: gamestate.Forest},
		{ID: "swamp", Biome: gamestate.Swamp},
		{ID: "water", Biome: gamestate.Water},
	}
	gs := runtimeLostTribeState(t, specs, nil)
	for _, tile := range gs.TileList {
		if tile.OwningTribe != nil || len(tile.PieceStacks) != 0 || tile.CheckPresence() != gamestate.None {
			t.Fatalf("default runtime map populated %s, changing existing training maps", tile.Id)
		}
	}
	field := gs.TileList["field"]
	if !field.IsEdge || !reflect.DeepEqual(field.Attributes, []gamestate.Attribute{gamestate.Magic}) || len(field.AdjacentTiles) != 1 || field.AdjacentTiles[0] != gs.TileList["mountain"] {
		t.Fatal("default map terrain or adjacency changed")
	}
	// Building the same default map must neither consume game RNG nor create
	// shared neutral state for callers that never opt into Lost Tribes.
	beforeRNG := gs.RNGState
	if _, ok := gamestate.BuildRuntimeMap(t.Name(), gs); !ok || gs.RNGState != beforeRNG {
		t.Fatal("runtime map construction changed the existing game random stream")
	}
}

func TestRuntimeLostTribesOnlyPopulateFlaggedLand(t *testing.T) {
	specs := []gamestate.RuntimeTileSpec{
		{ID: "field", Biome: gamestate.Field, LostTribe: true},
		{ID: "mountain", Biome: gamestate.Mountain, LostTribe: true},
		{ID: "forest", Biome: gamestate.Forest, LostTribe: true},
		{ID: "unflagged", Biome: gamestate.Hill},
		{ID: "water", Biome: gamestate.Water, LostTribe: true},
		{ID: "river", Biome: gamestate.River, LostTribe: true},
		{ID: "chasm", Biome: gamestate.AbyssalChasm, LostTribe: true},
		{ID: "island", Biome: gamestate.Forest, IsSkyIsland: true, LostTribe: true},
	}
	gs := runtimeLostTribeState(t, specs, nil)
	for _, id := range []string{"field", "mountain", "forest"} {
		assertRuntimeLostTribe(t, gs.TileList[id])
		if gs.TileList[id].OwningTribe != gs.TileList["field"].OwningTribe {
			t.Fatal("neutral regions do not share the canonical neutral owner")
		}
	}
	for _, id := range []string{"unflagged", "water", "river", "chasm", "island"} {
		tile := gs.TileList[id]
		if tile.OwningTribe != nil || len(tile.PieceStacks) != 0 {
			t.Fatalf("ineligible region %s received Lost Tribes", id)
		}
	}
	for _, player := range gs.Players {
		if player == gs.TileList["field"].OwningTribe.Owner || player.ActiveTribe != nil || len(player.PieceStacks) != 0 {
			t.Fatal("neutral placement changed a real player")
		}
	}
	beforeRNG := gs.RNGState
	other, ok := gamestate.BuildRuntimeMap(t.Name(), gs)
	if !ok || gs.RNGState != beforeRNG {
		t.Fatal("flagged construction consumed game RNG")
	}
	if other["field"].OwningTribe == gs.TileList["field"].OwningTribe || other["field"].OwningTribe.Owner == gs.TileList["field"].OwningTribe.Owner {
		t.Fatal("separate runtime maps share mutable neutral owners")
	}
	other["field"].PieceStacks[0].Amount = 9
	assertRuntimeLostTribe(t, gs.TileList["field"])
}

func TestRuntimeLostTribesConquestUsesNeutralDefense(t *testing.T) {
	for _, tc := range []struct {
		name    string
		flagged bool
		cost    int
	}{{"empty", false, 2}, {"lost_tribe", true, 3}} {
		t.Run(tc.name, func(t *testing.T) {
			gs := runtimeLostTribeState(t, []gamestate.RuntimeTileSpec{
				{ID: "target", Biome: gamestate.Field, IsEdge: true, LostTribe: tc.flagged},
				{ID: "other_neutral", Biome: gamestate.Forest, LostTribe: true},
			}, nil)
			if err := gs.HandleTribeChoice(0, 0); err != nil {
				t.Fatal(err)
			}
			player := gs.Players[0]
			player.PieceStacks = []gamestate.PieceStack{{Type: "Humans", Amount: 10}}
			beforeRNG := gs.RNGState
			if err := gs.HandleConquest("target", 0, "Humans"); err != nil {
				t.Fatal(err)
			}
			target := gs.TileList["target"]
			if target.OwningTribe != player.ActiveTribe || target.CheckPresence() != gamestate.Active || len(target.PieceStacks) != 1 || target.PieceStacks[0].Type != "Humans" || target.PieceStacks[0].Amount != tc.cost {
				t.Fatalf("conquest did not replace the neutral defenders with %d attackers: %+v", tc.cost, target.PieceStacks)
			}
			if len(player.PieceStacks) != 1 || player.PieceStacks[0].Amount != 10-tc.cost || gs.RNGState != beforeRNG {
				t.Fatal("neutral defense spent the wrong number of pieces or required a reinforcement roll")
			}
			if gs.Players[1].CoinPile != 5 || gs.Players[1].ActiveTribe != nil || len(gs.Players[1].PieceStacks) != 0 {
				t.Fatal("neutral conquest modified an unrelated real player")
			}
			assertRuntimeLostTribe(t, gs.TileList["other_neutral"])
		})
	}
}

func TestRuntimeLostTribesPowerSpawnSurvivesSaveLoad(t *testing.T) {
	for _, roundTrip := range []bool{false, true} {
		name := "new_seeded"
		if roundTrip {
			name = "saved_and_loaded"
		}
		t.Run(name, func(t *testing.T) {
			gs := runtimeLostTribeState(t, []gamestate.RuntimeTileSpec{
				{ID: "neutral", Biome: gamestate.Field, IsEdge: true, LostTribe: true, AdjacentIDs: []string{"empty"}},
				{ID: "empty", Biome: gamestate.Forest, AdjacentIDs: []string{"neutral"}},
				{ID: "water", Biome: gamestate.Water, LostTribe: true},
			}, []string{"Scepter of Avarice"})
			// The public constructor must populate neutrals before it initializes
			// selected powers, otherwise this spawn hook has no eligible region.
			assertRuntimeLostTribe(t, gs.TileList["neutral"])
			if gs.TileList["neutral"].ModifierAfterConquest["Scepter of Avarice Spawn"] == nil {
				t.Fatal("NewSeeded initialized selected powers before Lost Tribes existed")
			}
			if len(gs.TileList["empty"].ModifierAfterConquest) != 0 || len(gs.TileList["water"].ModifierAfterConquest) != 0 {
				t.Fatal("selected power was placed outside the neutral land region")
			}
			if err := gs.HandleTribeChoice(0, 0); err != nil {
				t.Fatal(err)
			}
			gs.Players[0].PieceStacks = []gamestate.PieceStack{{Type: "Humans", Amount: 10}}
			if roundTrip {
				data, err := store.EncodeGameState(gs)
				if err != nil {
					t.Fatal(err)
				}
				gs, err = store.DecodeGameState(data)
				if err != nil {
					t.Fatal(err)
				}
				assertRuntimeLostTribe(t, gs.TileList["neutral"])
				if gs.TileList["water"].OwningTribe != nil || len(gs.TileList["water"].PieceStacks) != 0 || gs.TileList["empty"].OwningTribe != nil {
					t.Fatal("save/load invented neutral defenders on unoccupied regions")
				}
				if len(gs.TileList["neutral"].AdjacentTiles) != 1 || gs.TileList["neutral"].AdjacentTiles[0] != gs.TileList["empty"] {
					t.Fatal("save/load lost runtime-map adjacency")
				}
			}
			if err := gs.HandleConquest("neutral", 0, "Humans"); err != nil {
				t.Fatal(err)
			}
			power := gs.Powers["Scepter of Avarice"]
			if power == nil || power.Owner != gs.Players[0] || gs.TileList["neutral"].ModifierAfterConquest["Scepter of Avarice Spawn"] != nil {
				t.Fatal("conquest did not activate and consume the neutral region's power hook")
			}
			amount := 0
			for _, stack := range gs.Players[0].PieceStacks {
				if stack.Type == "Scepter of Avarice" {
					amount += stack.Amount
				}
			}
			if amount != 1 {
				t.Fatalf("power reward amount=%d, want one", amount)
			}
		})
	}
}
