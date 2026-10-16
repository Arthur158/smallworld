package gamestate

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestV2RegistryCoverage(t *testing.T) {
	InitRaceMap()
	InitTraitMap()
	if len(botV2Races) != 35 || len(botV2Traits) != 38 || len(botV2Powers) != 0 {
		t.Fatal("surface preset changed")
	}
	for _, key := range botV2Races {
		if _, ok := RaceMap[Race(key)]; !ok {
			t.Fatal(key)
		}
	}
	for _, key := range botV2Traits {
		if _, ok := TraitMap[Trait(key)]; !ok {
			t.Fatal(key)
		}
	}
	if ValidateBotChoicesV2([]string{"Drows"}, []string{"Alchemist"}, nil) == nil || ValidateBotChoicesV2([]string{"Humans"}, []string{"Tomb"}, nil) == nil || ValidateBotChoicesV2([]string{"Humans"}, []string{"Alchemist"}, []string{"Shiny Orb"}) == nil {
		t.Fatal("underground/powers accepted")
	}
}
func v2TestState(t *testing.T) *GameState {
	t.Helper()
	InitRaceMap()
	InitTraitMap()
	gs, err := NewSeeded([]string{"one", "two"}, "map2players", BotV2Races(), BotV2Traits(), nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	return gs
}
func TestV2PrivacyAndPurity(t *testing.T) {
	gs := v2TestState(t)
	first, err := gs.BotDecisionV2()
	if err != nil {
		t.Fatal(err)
	}
	rng := gs.RNGState
	gs.Players[1].CoinPile = 999999
	gs.TribeList[6].Race = Race(botV2Races[1])
	second, err := gs.BotDecisionV2()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(first, second) || gs.RNGState != rng {
		t.Fatal("hidden information or RNG leaked")
	}
	// Halflings used to change state merely while testing adjacency.
	tribe, _ := CreateTribe("Halflings", "Alchemist")
	tribe.Owner = gs.Players[0]
	gs.Players[0].ActiveTribe = tribe
	gs.Players[0].PieceStacks = []PieceStack{{Type: "Halflings", Amount: 10}}
	gs.TurnInfo.Phase = TileAbandonment
	before, _ := json.Marshal(tribe.State)
	d, err := gs.BotDecisionV2()
	if err != nil {
		t.Fatal(err)
	}
	after, _ := json.Marshal(tribe.State)
	if string(before) != string(after) || gs.RNGState != rng {
		t.Fatal("enumeration mutated game")
	}
	if len(d.State) != BotV2StateWidth() {
		t.Fatal("state width")
	}
	for _, n := range d.Graph.Nodes {
		if len(n) != BotV2NodeWidth() {
			t.Fatal("node width")
		}
	}
	for _, a := range d.Candidates {
		if len(a.Features) != BotV2ActionWidth() {
			t.Fatal("action width")
		}
	}
}
func TestAtomicStackSubtraction(t *testing.T) {
	original := []PieceStack{{Type: "Storm", Amount: 1}}
	after, ok := SubtractPieceStacks(original, []PieceStack{{Type: "Storm", Amount: 1}, {Type: "Storm Giants", Amount: 1}})
	if ok || !reflect.DeepEqual(after, original) {
		t.Fatal("failed expense destroyed reserves")
	}
	after, ok = SubtractPieceStacks(original, []PieceStack{{Type: "missing", Amount: 0}})
	if !ok || !reflect.DeepEqual(after, original) {
		t.Fatal("zero expense")
	}
	_, ok = SubtractPieceStacks(original, []PieceStack{{Type: "Storm", Amount: 1}, {Type: "Storm", Amount: 1}})
	if ok {
		t.Fatal("duplicate expense overspent")
	}
}
func TestStormRequiresBothResources(t *testing.T) {
	gs := v2TestState(t)
	p := gs.Players[0]
	tribe, _ := CreateTribe("Storm Giants", "Alchemist")
	tribe.Owner = p
	p.ActiveTribe = tribe
	p.PieceStacks = []PieceStack{{Type: "Storm", Amount: 2}}
	gs.TurnInfo.Phase = Conquest
	for id, tile := range gs.TileList {
		if tile.Biome == Mountain {
			before := tile.OwningTribe
			err := gs.HandleConquest(id, 0, "Storm")
			if err == nil || tile.OwningTribe != before || stackAmount(p.PieceStacks, "Storm") != 2 {
				t.Fatal("storm created a free army")
			}
			return
		}
	}
	t.Fatal("fixture lacks mountain")
}
func TestAmazonsCanWithdrawLastTemporaryPiece(t *testing.T) {
	gs := v2TestState(t)
	p := gs.Players[0]
	tribe, _ := CreateTribe("Amazons", "Alchemist")
	tribe.Owner = p
	p.ActiveTribe = tribe
	p.PieceStacks = []PieceStack{{Type: "Amazons", Amount: 3}}
	gs.TurnInfo.Phase = Redeployment
	for id, tile := range gs.TileList {
		if tile.OwningTribe == nil {
			tile.OwningTribe = tribe
			tile.PieceStacks = []PieceStack{{Type: "Amazons", Amount: 1}}
			if err := gs.HandleRedeploymentOut(0, id, "Amazons"); err != nil {
				t.Fatal(err)
			}
			if tile.OwningTribe != nil || tribe.canEndTurn(gs) != nil {
				t.Fatal("temporary piece could not return")
			}
			return
		}
	}
}

func TestScavengersDoNotDelegateToThemselves(t *testing.T) {
	gs := v2TestState(t)
	p := gs.Players[0]
	tribe, _ := CreateTribe("Scavengers", "Fortified")
	tribe.Owner = p
	p.ActiveTribe = tribe
	tile := &Tile{Id: "self", OwningTribe: tribe, PieceStacks: []PieceStack{{Type: "Scavengers", Amount: 2, Tribe: tribe}}}
	if tribe.canBeRedeployedIn(tile, "unavailable", gs) {
		t.Fatal("accepted an unknown stack")
	}
	if _, err := tribe.getStacksOutRedeployment(tile, "unavailable"); err == nil {
		t.Fatal("withdrew an unknown stack")
	}
}
func TestAmazonsCanReturnPiecesFromPlacedTile(t *testing.T) {
	gs := v2TestState(t)
	p := gs.Players[0]
	tribe, _ := CreateTribe("Amazons", "Alchemist")
	tribe.Owner = p
	p.ActiveTribe = tribe
	p.PieceStacks = []PieceStack{{Type: "Amazons", Amount: 3}}
	gs.TurnInfo.Phase = Redeployment
	for id, tile := range gs.TileList {
		if tile.OwningTribe == nil && tile.Biome != Water {
			tile.OwningTribe = tribe
			tile.PieceStacks = []PieceStack{{Type: "Amazons", Amount: 1}}
			gs.BotV2Memory = &BotV2Memory{Turn: "1:0", Placed: map[string]bool{id: true}}
			d, err := gs.BotDecisionV2()
			if err != nil {
				t.Fatal(err)
			}
			for _, a := range d.Candidates {
				if a.Action.Kind == "redeploy_out" && a.Action.Tile == id {
					return
				}
			}
			t.Fatal("temporary Amazons cannot return from a tile used in redeployment")
		}
	}
	t.Fatal("no fixture tile")
}

func TestStoutObservationDoesNotConsumeScoringState(t *testing.T) {
	gs := v2TestState(t)
	tribe, _ := CreateTribe("Humans", "Stout")
	tribe.Owner = gs.Players[0]
	gs.Players[0].ActiveTribe = tribe
	gs.Players[0].PieceStacks = []PieceStack{}
	gs.TurnInfo.Phase = Redeployment
	tribe.State["extrapoints"] = 7
	tribe.State["pluspoints"] = 11
	before, _ := json.Marshal(tribe.State)
	for i := 0; i < 3; i++ {
		if _, err := gs.BotDecisionV2(); err != nil {
			t.Fatal(err)
		}
	}
	after, _ := json.Marshal(tribe.State)
	if string(before) != string(after) {
		t.Fatalf("Stout observation mutated state: %s -> %s", before, after)
	}
	// The real decline action still runs its scoring preparation.
	if err := gs.HandleDecline(0); err != nil {
		t.Fatal(err)
	}
	if tribe.State["pluspoints"] == 11 {
		t.Fatal("decline did not capture its score")
	}
}
