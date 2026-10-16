package store

import (
	"backend/internal/gamestate"
	"reflect"
	"testing"
)

func TestSurfaceFeedbackAndTypedStateRoundTrip(t *testing.T) {
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	gs, err := gamestate.NewSeeded([]string{"one", "two"}, "map2players", gamestate.BotV2Races(), gamestate.BotV2Traits(), nil, 7)
	if err != nil {
		t.Fatal(err)
	}
	tribe, _ := gamestate.CreateTribe("Ghouls", "Diplomat")
	tribe.Owner = gs.Players[0]
	gs.Players[0].ActiveTribe = tribe
	tribe.State["deploy"] = map[string]int{"a": 2}
	tribe.State["playersAttacked"] = []int{1}
	tribe.State["abandonedTiles"] = []string{"b"}
	tribe.State["pointsfromstart"] = 4
	a := gamestate.BotAction{Kind: "conquer", Tile: "a", Stack: "Ghouls"}
	gs.BotV2Memory = &gamestate.BotV2Memory{Turn: "1:0", Rejected: map[gamestate.BotAction]bool{a: true}, Uses: map[gamestate.BotAction]int{a: 3}, Placed: map[string]bool{"a": true}}
	gs.RealTurninfo = &gamestate.TurnInfo{PlayerIndex: 1, TurnIndex: 1, Phase: gamestate.Conquest}
	data, err := EncodeGameState(gs)
	if err != nil {
		t.Fatal(err)
	}
	copy, err := DecodeGameState(data)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(gs.BotV2Memory, copy.BotV2Memory) || !reflect.DeepEqual(gs.RealTurninfo, copy.RealTurninfo) || !reflect.DeepEqual(tribe.State, copy.Players[0].ActiveTribe.State) {
		t.Fatal("surface snapshot lost typed state or action feedback")
	}
}
