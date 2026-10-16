package main

import (
	"backend/internal/bot"
	"backend/internal/gamestate"
	"testing"
)

func TestFocusedCurriculumUsesRequestedSeat(t *testing.T) {
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	registry, err := bot.NewRegistry("")
	if err != nil {
		t.Fatal(err)
	}
	e := environment{policies: registry, limit: 8192}
	r, err := e.handle(request{Op: "reset", Rules: "surface", Players: 3, Map: "map3players", Seed: 1701, Focus: true, FocusPlayer: 2, Curriculum: 11})
	if err != nil {
		t.Fatal(err)
	}
	if e.state.Players[0].ActiveTribe != nil {
		t.Fatal("forced the first seat instead of selected learner")
	}
	for i := 0; e.state.Players[2].ActiveTribe == nil; i++ {
		if i > 1000 || r.Done || r.Truncated {
			t.Fatal("focus seat was not reached")
		}
		r, err = e.handle(request{Op: "step", Policy: "greedy"})
		if err != nil {
			t.Fatal(err)
		}
	}
	tribe := e.state.Players[2].ActiveTribe
	if string(tribe.Race) != gamestate.BotTrainingRaces()[11] || string(tribe.Trait) != gamestate.BotTrainingTraits()[11] || e.focusPending {
		t.Fatal("wrong focused tribe or repeated focus")
	}
}

func TestTrainingDefaultsPreserveVocabularyAndCoverPairings(t *testing.T) {
	races, traits := gamestate.BotTrainingRaces(), gamestate.BotTrainingTraits()
	if len(races) != 32 || len(traits) != 37 || len(gamestate.BotV2Races()) != 35 || len(gamestate.BotV2Traits()) != 38 {
		t.Fatal("training deck or checkpoint vocabulary changed unexpectedly")
	}
	pairs := map[[2]string]bool{}
	e := matchEnvironment(t)
	for i := 0; i < len(races)*len(traits); i++ {
		race, trait := curriculumPair(i, races, traits)
		pairs[[2]string{race, trait}] = true
	}
	if len(pairs) != len(races)*len(traits) {
		t.Fatal("curriculum misses race/trait combinations")
	}
	for seed := uint64(1); seed <= 12; seed++ {
		if _, err := e.handle(request{Op: "reset", Rules: "surface", Players: 4, Map: "map4players", Seed: seed, Focus: true, FocusPlayer: 0, Curriculum: int(seed)}); err != nil {
			t.Fatal(err)
		}
		for _, entry := range e.state.TribeList {
			if !gamestate.DefaultRaceEnabled(string(entry.Race)) || !gamestate.DefaultTraitEnabled(string(entry.Trait)) {
				t.Fatalf("disabled race in training deck: %s", entry.Race)
			}
		}
		if !gamestate.DefaultRaceEnabled(string(e.state.Players[0].ActiveTribe.Race)) || !gamestate.DefaultTraitEnabled(string(e.state.Players[0].ActiveTribe.Trait)) {
			t.Fatal("disabled focused race")
		}
	}
}
