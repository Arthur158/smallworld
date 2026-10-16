package roomworker

import (
	"backend/internal/bot"
	"backend/internal/gamestate"
	"backend/internal/store"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestGraphRoomModelAndPlayback(t *testing.T) {
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	dir := os.Getenv("SMALLWORLD_GRAPH_MODEL_DIR")
	if dir == "" {
		t.Skip("set SMALLWORLD_GRAPH_MODEL_DIR to validate an exported graph bot")
	}
	registry, err := bot.NewRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	id, legacyID := "", ""
	for _, m := range registry.Catalog() {
		if registry.Schema(m.ID) == gamestate.BotSchemaV2 {
			id = m.ID
			break
		}
	}
	if id == "" {
		t.Fatal("no graph model")
	}
	w, room := botTestRoom(t)
	w.bots = registry
	if err := room.addBot(w, id); err != nil {
		t.Fatal(err)
	}
	if !room.surfaceBotRules() {
		t.Fatal("graph room did not select surface rules")
	}
	races, traits := 0, 0
	for _, e := range room.ExtensionChoices {
		for _, c := range e.RaceChoices {
			if c.IsChecked {
				races++
			}
		}
		for _, c := range e.TraitChoices {
			if c.IsChecked {
				traits++
			}
		}
	}
	if races != 35 || traits != 38 {
		t.Fatalf("wrong surface preset %d/%d", races, traits)
	}
	if err := room.validateBotChoices([]string{"Drows"}, []string{"Alchemist"}, nil); err == nil {
		t.Fatal("accepted underground")
	}
	// Switching one seat to a legacy trained model narrows the preset.
	for _, m := range registry.Catalog() {
		if m.ID != "random" && m.ID != "greedy" && registry.Schema(m.ID) == gamestate.BotSchema {
			legacyID = m.ID
			if err := room.setBotModel(w, room.Players[2].Username, m.ID); err != nil {
				t.Fatal(err)
			}
			if room.surfaceBotRules() {
				t.Fatal("legacy model accepted expanded rules")
			}
			count := 0
			for _, e := range room.ExtensionChoices {
				for _, c := range e.RaceChoices {
					if c.IsChecked {
						count++
					}
				}
			}
			if count != 12 {
				t.Fatal("legacy preset not restored")
			}
			if err := room.setBotModel(w, room.Players[2].Username, id); err != nil {
				t.Fatal(err)
			}
			break
		}
	}
	for seed := uint64(19); seed < 23; seed++ {
		races, traits := gamestate.BotV2Races(), gamestate.BotV2Traits()
		mixed := seed == 22 && legacyID != ""
		if mixed {
			races, traits = gamestate.BotRaces(), gamestate.BotTraits()
		}
		gs, err := gamestate.NewSeeded([]string{"one", "two", "three"}, "map3players", races, traits, nil, seed)
		if err != nil {
			t.Fatal(err)
		}
		room.Gamestate, room.InProgress = *gs, true
		room.botHistory = nil
		room.botFuture = nil
		for i, p := range room.Players {
			p.BotModel = "greedy"
			if i == 0 {
				p.BotModel = id
			} else if i == 1 && mixed {
				p.BotModel = legacyID
			}
		}
		for step := 0; room.InProgress; step++ {
			if step > 8192 {
				t.Fatal("graph room stuck")
			}
			if err := room.applyBotMove(w); err != nil {
				t.Fatalf("seed %d step %d: %v", seed, step, err)
			}
			if step%37 == 0 && room.InProgress {
				before, err := room.Gamestate.BotDecisionV2()
				if err != nil {
					t.Fatal(err)
				}
				data, err := store.EncodeGameState(&room.Gamestate)
				if err != nil {
					t.Fatal(err)
				}
				decoded, err := store.DecodeGameState(data)
				if err != nil {
					t.Fatal(err)
				}
				after, err := decoded.BotDecisionV2()
				if err != nil {
					t.Fatal(err)
				}
				if !reflect.DeepEqual(before, after) {
					b, _ := json.MarshalIndent(before, "", " ")
					a, _ := json.MarshalIndent(after, "", " ")
					dir := t.TempDir()
					os.WriteFile(filepath.Join(dir, "before.json"), b, 0600)
					os.WriteFile(filepath.Join(dir, "after.json"), a, 0600)
					os.WriteFile(filepath.Join(dir, "snapshot.json"), data, 0600)
					t.Fatalf("seed %d step %d: snapshot changed model observation (details in %s)", seed, step, dir)
				}
				room.Gamestate = *decoded
			}
		}
		if err := room.previousBotMove(); err != nil {
			t.Fatal(err)
		}
		if err := room.applyBotMove(w); err != nil {
			t.Fatal(err)
		}
		if room.InProgress {
			t.Fatal("replay did not restore finished game")
		}
	}
}
