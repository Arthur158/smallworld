package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func assertRulesPreset(t *testing.T, room *Room, preset string) {
	t.Helper()
	gameMap, choices, err := rulesPresetSettings(preset)
	if err != nil {
		t.Fatal(err)
	}
	if room.Map != gameMap || len(room.Players) != gameMap.Capacity || room.GlobalToggle || !reflect.DeepEqual(room.ExtensionChoices, choices) {
		t.Fatalf("room does not match %s preset: map=%s seats=%d global=%v choices=%+v", preset, room.Map.Name, len(room.Players), room.GlobalToggle, room.ExtensionChoices)
	}
}

func presetState(t *testing.T, room *Room) string {
	t.Helper()
	names := make([]string, len(room.Players))
	indexes := make([]int, len(room.Players))
	for i, player := range room.Players {
		if player != nil {
			names[i], indexes[i] = player.Username, player.Index
		}
	}
	raw, err := json.Marshal(struct {
		Map      Map
		Choices  []Extension
		Global   bool
		Save     int64
		Statuses []string
		Players  []string
		Indexes  []int
		Host     string
	}{room.Map, room.ExtensionChoices, room.GlobalToggle, room.saveId, room.playerStatuses, names, indexes, room.HostUsername})
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func TestRulesPresetSelections(t *testing.T) {
	for _, preset := range []string{"standard", "underground"} {
		t.Run(preset, func(t *testing.T) {
			gameMap, choices, err := rulesPresetSettings(preset)
			if err != nil {
				t.Fatal(err)
			}
			if (preset == "standard" && gameMap != GeneratedMap) || (preset == "underground" && gameMap != Underground2) {
				t.Fatalf("wrong map for %s: %+v", preset, gameMap)
			}
			if len(choices) != len(extensions) {
				t.Fatal("preset omitted expansions")
			}
			for i, extension := range extensions {
				choice := choices[i]
				want := (extension.Name == "Underground" || extension.Name == "Powers") == (preset == "underground")
				if choice.ExtensionName != extension.Name || choice.IsChecked != want || len(choice.RaceChoices) != len(extension.Races) || len(choice.TraitChoices) != len(extension.Traits) {
					t.Fatalf("wrong extension selection: %+v", choice)
				}
				for j, race := range choice.RaceChoices {
					if race.Choice != extension.Races[j] || race.IsChecked != (want && gamestate.DefaultRaceEnabled(race.Choice)) {
						t.Fatalf("wrong race selection: %+v", race)
					}
				}
				for j, trait := range choice.TraitChoices {
					if trait.Choice != extension.Traits[j] || trait.IsChecked != (want && gamestate.DefaultTraitEnabled(trait.Choice)) {
						t.Fatalf("wrong power selection: %+v", trait)
					}
				}
			}
		})
	}
	_, choices, _ := rulesPresetSettings("standard")
	choices[0].RaceChoices[0].IsChecked = false
	_, other, _ := rulesPresetSettings("standard")
	if !other[0].RaceChoices[0].IsChecked {
		t.Fatal("preset choices share mutable slices")
	}
}

func TestRulesPresetAuthorizationAndValidation(t *testing.T) {
	w, room := botTestRoom(t)
	room.saveId, room.playerStatuses = 42, []string{"saved host", "saved guest"}
	before := presetState(t, room)
	for _, user := range []string{"guest", "outsider"} {
		if command(t, w, room, user, "setRulesPreset", map[string]string{"preset": "standard"}) == nil {
			t.Fatalf("%s changed rules", user)
		}
	}
	for _, data := range []any{nil, map[string]string{}, map[string]string{"preset": "unknown"}, map[string]string{"preset": "Standard"}, map[string]int{"preset": 1}} {
		if command(t, w, room, "host", "setRulesPreset", data) == nil {
			t.Fatalf("accepted invalid preset: %#v", data)
		}
	}
	room.InProgress = true
	if command(t, w, room, "host", "setRulesPreset", map[string]string{"preset": "standard"}) == nil {
		t.Fatal("accepted preset after game start")
	}
	room.InProgress = false
	other := &Room{ID: "another-room"}
	rooms[other.ID] = other
	if command(t, w, other, "host", "setRulesPreset", map[string]string{"preset": "standard"}) == nil {
		t.Fatal("accepted a preset for another room")
	}
	if presetState(t, room) != before {
		t.Fatal("rejected preset changed lobby state")
	}
}

func TestRulesPresetCapacityIsAtomic(t *testing.T) {
	w, room := botTestRoom(t)
	if err := room.addBot(w, "greedy"); err != nil {
		t.Fatal(err)
	}
	room.saveId, room.playerStatuses = 42, []string{"saved host", "saved guest", "saved bot"}
	before := presetState(t, room)
	if command(t, w, room, "host", "setRulesPreset", map[string]string{"preset": "underground"}) == nil {
		t.Fatal("preset dropped an occupied human or bot seat")
	}
	if presetState(t, room) != before {
		t.Fatal("rejected capacity change partially changed lobby")
	}
	gameMap := mapMap["generatedmap"]
	delete(mapMap, "generatedmap")
	t.Cleanup(func() { mapMap["generatedmap"] = gameMap })
	if command(t, w, room, "host", "setRulesPreset", map[string]string{"preset": "standard"}) == nil || presetState(t, room) != before {
		t.Fatal("missing preset map changed the lobby")
	}
}

func TestRulesPresetCompactsSeatsClearsSaveAndPreservesBotRules(t *testing.T) {
	w, room := botTestRoom(t)
	host, guest := room.Players[0], room.Players[1]
	host.IsSpectator = true
	room.Spectators = []*Client{host}
	room.Players = []*Client{nil, guest, nil}
	if err := room.addBot(w, "greedy"); err != nil {
		t.Fatal(err)
	}
	bot := room.Players[0]
	room.Players = []*Client{nil, guest, nil, bot, nil, nil}
	room.Map = GeneratedMap
	room.saveId, room.playerStatuses = 42, []string{"saved"}
	if err := command(t, w, room, "host", "setRulesPreset", map[string]string{"preset": "underground"}); err != nil {
		t.Fatal(err)
	}
	assertRulesPreset(t, room, "underground")
	if room.Players[0] != guest || room.Players[1] != bot || guest.Index != 0 || bot.Index != 1 || room.hostClient() != host || bot.BotModel != "greedy" {
		t.Fatal("preset did not preserve and compact human/bot roster with spectator host")
	}
	if room.saveId != -1 || len(room.playerStatuses) != 0 {
		t.Fatal("preset left a conflicting selected save")
	}
	if err := command(t, w, room, "host", "setRulesPreset", map[string]string{"preset": "standard"}); err != nil {
		t.Fatal(err)
	}
	assertRulesPreset(t, room, "standard")
	if room.Players[0] != guest || room.Players[1] != bot || room.Players[5] != nil {
		t.Fatal("expanding preset lost occupants")
	}
	if len(gamestate.BotRaces()) != 12 || len(gamestate.BotTraits()) != 12 {
		t.Fatal("preset changed trained bot policy compatibility")
	}
	room.useBotRules()
	for _, extension := range room.ExtensionChoices {
		for _, race := range extension.RaceChoices {
			if race.IsChecked && !containsChoice(gamestate.BotRaces(), race.Choice) {
				t.Fatalf("bot rules included %s", race.Choice)
			}
		}
	}
}

func containsChoice(choices []string, want string) bool {
	for _, choice := range choices {
		if choice == want {
			return true
		}
	}
	return false
}

func TestNewRoomDefaultsToStandardGeneratedMap(t *testing.T) {
	w, oldRoom := spectatorTestRoom(t)
	host := oldRoom.Players[0]
	host.CommandRoomID = oldRoom.ID + "-new"
	t.Cleanup(func() { w.redis.Del(context.Background(), roomSnapshotKey(host.CommandRoomID)) })
	createRoom(host, "New room", host.Username)
	if host.Room == oldRoom || host.Room == nil {
		t.Fatal("new room was not created")
	}
	assertRulesPreset(t, host.Room, "standard")
	if host.Room.Players[0] != host || host.Room.saveId != -1 {
		t.Fatal("default changed host or selected save")
	}
	if oldRoom.Map != Map3 {
		t.Fatal("new default changed an existing room")
	}
}

func TestRulesPresetSnapshotAndCommandReplay(t *testing.T) {
	w, room := spectatorTestRoom(t)
	w.cfg.WorkerCount, w.cfg.ConsumerGroup, w.cfg.ConsumerName = 1, room.ID, room.ID
	t.Cleanup(func() { w.redis.Del(context.Background(), room.ID+":commands") })
	processPlaybackCommand(t, w, room, "preset-standard", "setRulesPreset", map[string]string{"preset": "standard"})
	assertRulesPreset(t, room, "standard")
	delete(rooms, room.ID)
	w.clients = make(map[string]*Client)
	loaded, err := w.ensureRoomLoaded(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertRulesPreset(t, loaded, "standard")
	if !loaded.hasProcessedCommand("preset-standard") {
		t.Fatal("preset command journal not persisted")
	}
	// A delayed duplicate must not overwrite later custom lobby edits.
	loaded.toggleRace("Base Game", "Humans", false)
	before := presetState(t, loaded)
	processPlaybackCommand(t, w, loaded, "preset-standard", "setRulesPreset", map[string]string{"preset": "standard"})
	if presetState(t, loaded) != before {
		t.Fatal("duplicate preset overwrote newer custom choices")
	}
}

func TestRulesPresetBroadcastsCompleteLobbyState(t *testing.T) {
	w, room := spectatorTestRoom(t)
	currentWorker = w
	t.Cleanup(func() { w.redis.HDel(context.Background(), protocol.RoomsHashKey, room.ID) })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gatewayID := "preset-gateway-" + room.ID
	sub := w.redis.Subscribe(ctx, protocol.GatewayEventsChannel(gatewayID), protocol.RoomEventsChannel(room.ID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	room.saveId, room.playerStatuses = 42, []string{"saved"}
	err := w.HandleCommand(ctx, protocol.Command{
		ID: "preset", Username: room.HostUsername, RoomID: room.ID, Scope: protocol.ScopeRoom,
		GatewayID: gatewayID, ConnectionID: "connection",
		Message: messages.Message{Type: "setRulesPreset", Data: json.RawMessage(`{"preset":"underground"}`)},
	})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]json.RawMessage{}
	for range 4 {
		msg, err := sub.ReceiveMessage(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if msg.Channel == protocol.RoomEventsChannel(room.ID) {
			var event protocol.RoomEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				t.Fatal(err)
			}
			seen[event.Message.Type] = event.Message.Data
		} else {
			var event protocol.DirectEvent
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				t.Fatal(err)
			}
			for _, m := range event.Messages {
				seen[m.Type] = m.Data
			}
		}
	}
	for _, kind := range []string{"smallmapupdate", "choices", "playerStatuses", "saveSelection"} {
		if _, ok := seen[kind]; !ok {
			t.Fatalf("missing %s: %v", kind, seen)
		}
	}
	var selection struct {
		Index int `json:"index"`
	}
	if err := json.Unmarshal(seen["saveSelection"], &selection); err != nil || selection.Index != -1 {
		t.Fatal("selected save was not cleared")
	}
	var meta protocol.RoomMetadata
	data, err := w.redis.HGet(ctx, protocol.RoomsHashKey, room.ID).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.MapName != "underground2players" || meta.MaxPlayers != 2 || len(meta.Players) != 2 {
		t.Fatalf("lobby metadata not updated atomically: %+v", meta)
	}
}
