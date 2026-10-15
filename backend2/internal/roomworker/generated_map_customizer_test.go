package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func customMapResponse(players int) *mapGeneratorResponse {
	return customMapResponseWithCount(generatedMapSettingsForPlayerCount(players)["tileCount"].(int))
}

func customMapResponseWithCount(count int) *mapGeneratorResponse {
	response := &mapGeneratorResponse{ImageBase64: "cHJldmlldy1waXhlbHM="}
	response.Metadata.Graphics = json.RawMessage(`{"exact":"preview-graphics"}`)
	for i := 0; i < count; i++ {
		response.Metadata.Server.Tiles = append(response.Metadata.Server.Tiles, GeneratedServerTile{
			ID: GeneratedID(fmt.Sprint(i)), Biome: "Field", Attributes: []string{}, IsEdge: i < 9,
			AdjacentIDs: []GeneratedID{GeneratedID(fmt.Sprint((i + count - 1) % count)), GeneratedID(fmt.Sprint((i + 1) % count))},
		})
	}
	return response
}

func TestGeneratedMapCustomizerUsesRequestedTileCount(t *testing.T) {
	for _, players := range []int{2, 6} {
		for _, count := range []int{18, 35, 120} {
			t.Run(fmt.Sprintf("%d-players-%d-tiles", players, count), func(t *testing.T) {
				settings := requireNormalizedSettings(t, players, MapGeneratorSettings{"seed": "custom-count", "tileCount": count})
				response := customMapResponseWithCount(count)
				response.Settings, _ = json.Marshal(settings)
				preview, err := makeGeneratedMapPreview(players, settings, response)
				if err != nil {
					t.Fatal(err)
				}
				room := &Room{generatedSelected: preview}
				gameMap, err := room.useSelectedGeneratedMap(players)
				if err != nil {
					t.Fatal(err)
				}
				tiles, ok := gamestate.BuildRuntimeMap(gameMap.Name, nil)
				if !ok || len(tiles) != count || gameMap.Capacity != players {
					t.Fatalf("selected map has %d tiles / %d players; want %d / %d", len(tiles), gameMap.Capacity, count, players)
				}
				visuals, ok := GetGeneratedMapVisuals(gameMap.Name)
				if !ok || !reflect.DeepEqual(visuals, preview.Visuals) || preview.Settings["tileCount"] != float64(count) {
					t.Fatal("custom tile count lost selected settings or visuals")
				}
				if _, err := makeGeneratedMapPreview(players, settings, customMapResponseWithCount(count+1)); err == nil {
					t.Fatal("accepted a map with a different number of tiles than requested")
				}
				response.Settings = json.RawMessage(`{"seed":"custom-count","tileCount":99}`)
				if _, err := makeGeneratedMapPreview(players, settings, response); err == nil {
					t.Fatal("accepted settings that disagree with the requested tile count")
				}
			})
		}
	}
}

func customPreview(t *testing.T, players int) *generatedMapPreview {
	t.Helper()
	settings, err := normalizedGeneratedMapSettings(players, MapGeneratorSettings{"seed": "custom-map-test"})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := makeGeneratedMapPreview(players, settings, customMapResponse(players))
	if err != nil {
		t.Fatal(err)
	}
	return preview
}

func customRedisRoom(t *testing.T) (*Worker, *Room) {
	t.Helper()
	addr := os.Getenv("SMALLWORLD_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SMALLWORLD_TEST_REDIS_ADDR to an isolated Redis")
	}
	w, room := botTestRoom(t)
	room.Map = GeneratedMap
	delete(rooms, room.ID)
	room.ID = "custom-map-" + uuid.NewString()
	rooms[room.ID] = room
	room.Map = GeneratedMap
	w.redis = redis.NewClient(&redis.Options{Addr: addr})
	if err := w.redis.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		room.clearGeneratedMapCustomization()
		w.redis.Del(context.Background(), roomSnapshotKey(room.ID))
		w.redis.Close()
	})
	return w, room
}

func drainCustomRequest(t *testing.T, w *Worker) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for w.generatedMapRequests > 0 && time.Now().Before(deadline) {
		w.drainGeneratedMaps(context.Background())
		time.Sleep(time.Millisecond)
	}
	if w.generatedMapRequests != 0 {
		t.Fatal("preview did not complete")
	}
}

func TestGeneratedMapCustomizerAuthorizationAndBounds(t *testing.T) {
	w, room := botTestRoom(t)
	room.Map = GeneratedMap
	preview := customPreview(t, 2)
	room.generatedDraft = preview
	for _, kind := range []string{"getGeneratedMapSettings", "previewGeneratedMap", "selectGeneratedMap", "resetGeneratedMap"} {
		for _, user := range []string{"guest", "outsider"} {
			if command(t, w, room, user, kind, map[string]any{"previewId": preview.PreviewID, "settings": preview.Settings}) == nil {
				t.Fatalf("%s accepted from %s", kind, user)
			}
		}
		room.InProgress = true
		if command(t, w, room, "host", kind, nil) == nil {
			t.Fatalf("accepted %s during game", kind)
		}
		room.InProgress = false
		room.saveId = 42
		if command(t, w, room, "host", kind, nil) == nil {
			t.Fatalf("accepted %s while loading save", kind)
		}
		room.saveId = -1
	}
	if err := command(t, w, room, "host", "selectGeneratedMap", map[string]string{"previewId": "another-room"}); err == nil {
		t.Fatal("selected unowned preview")
	}
	room.Map = Map2
	if command(t, w, room, "host", "getGeneratedMapSettings", nil) == nil {
		t.Fatal("fixed map accepted customization")
	}
	room.Map = GeneratedMap
	w.generatedMapRequests = maxGeneratedMapRequests
	if command(t, w, room, "host", "previewGeneratedMap", map[string]any{"settings": preview.Settings}) == nil {
		t.Fatal("exceeded worker cap")
	}
	w.generatedMapRequests = 0
	raw, _ := json.Marshal(map[string]string{"previewId": preview.PreviewID})
	if w.HandleCommand(context.Background(), protocol.Command{Username: "host", RoomID: room.ID, Scope: protocol.ScopeDisplay, Message: messages.Message{Type: "selectGeneratedMap", Data: raw}}) == nil {
		t.Fatal("accepted display scope")
	}
}

func TestGeneratedMapCustomizerSelectionAndExactRuntimeMap(t *testing.T) {
	w, room := botTestRoom(t)
	room.Map = GeneratedMap
	preview := customPreview(t, 2)
	room.generatedDraft = preview
	if _, ok := gamestate.BuildRuntimeMap(preview.PreviewID, nil); ok {
		t.Fatal("preview prematurely registered")
	}
	if err := command(t, w, room, "host", "selectGeneratedMap", map[string]string{"previewId": preview.PreviewID}); err != nil {
		t.Fatal(err)
	}
	if room.Map.Name != "generatedmap" || len(room.Players) != 6 || room.generatedSelected != preview {
		t.Fatal("selection changed lobby semantics")
	}
	if _, ok := gamestate.BuildRuntimeMap(preview.PreviewID, nil); ok {
		t.Fatal("selection prematurely registered")
	}
	if selection := room.generatedMapSelection(); selection == nil || selection.PlayerCount != 2 || selection.Seed != "custom-map-test" {
		t.Fatalf("bad metadata: %+v", selection)
	}
	room.Players[2] = &Client{Username: "third"}
	if err := command(t, w, room, "host", "selectGeneratedMap", map[string]string{"previewId": preview.PreviewID}); err == nil {
		t.Fatal("accepted stale player count")
	}
	room.startLobbyGame(room.Players[0], room.ID)
	if room.InProgress || room.Map.Name != "generatedmap" || len(room.Players) != 6 {
		t.Fatal("stale selected preview mutated lobby at start")
	}
	room.Players[2] = nil
	gameMap, err := room.useSelectedGeneratedMap(2)
	if err != nil {
		t.Fatal(err)
	}
	visuals, ok := GetGeneratedMapVisuals(gameMap.Name)
	if !ok || !reflect.DeepEqual(visuals, preview.Visuals) {
		t.Fatal("runtime map differs from selected image/specs/neutral setup")
	}
	tiles, ok := gamestate.BuildRuntimeMap(gameMap.Name, nil)
	if !ok || len(tiles) != 22 {
		t.Fatal("exact runtime map not installed")
	}
	for id, tile := range tiles {
		want := false
		for _, lostID := range preview.Visuals.LostTribeTileIDs {
			want = want || lostID == id
		}
		if (tile.OwningTribe != nil) != want {
			t.Fatal("neutral setup changed")
		}
	}
	if err := command(t, w, room, "host", "resetGeneratedMap", nil); err != nil {
		t.Fatal(err)
	}
	if room.generatedDraft != nil || room.generatedSelected != nil || room.Map != GeneratedMap {
		t.Fatal("reset did not restore automatic mode")
	}
}

func TestGeneratedMapCustomizerClearsOnMapPresetAndSave(t *testing.T) {
	for _, kind := range []string{"changeRoomMap", "setRulesPreset", "loadgame"} {
		t.Run(kind, func(t *testing.T) {
			w, room := botTestRoom(t)
			room.Map = GeneratedMap
			preview := customPreview(t, 2)
			room.generatedDraft, room.generatedSelected = preview, preview
			canceled := false
			room.generatedPending = &pendingGeneratedMap{cancel: func() { canceled = true }}
			data := map[string]any{"newMap": "map3players", "preset": "standard", "saveId": -1}
			if err := command(t, w, room, "host", kind, data); err != nil {
				t.Fatal(err)
			}
			if room.generatedDraft != nil || room.generatedSelected != nil || room.generatedPending != nil || !canceled {
				t.Fatal("stale preview survived lobby change")
			}
		})
	}
}

func TestGeneratedMapCustomizerPreviewAndRestartPersistence(t *testing.T) {
	w, room := customRedisRoom(t)
	selected := customPreview(t, 2)
	room.generatedSelected = selected
	requestReceived := make(chan struct{}, 1)
	release := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		var settings MapGeneratorSettings
		if err := json.NewDecoder(request.Body).Decode(&settings); err != nil {
			t.Error(err)
			return
		}
		requestReceived <- struct{}{}
		<-release
		response := customMapResponse(2)
		response.Settings, _ = json.Marshal(settings)
		json.NewEncoder(writer).Encode(response)
	}))
	defer server.Close()
	t.Setenv("MAP_GENERATOR_URL", server.URL)
	if err := command(t, w, room, "host", "previewGeneratedMap", map[string]any{"requestId": "draft-1", "settings": MapGeneratorSettings{"seed": "replacement"}}); err != nil {
		t.Fatal(err)
	}
	<-requestReceived
	if room.generatedDraft != nil || room.generatedSelected != selected {
		t.Fatal("pending generation changed selection")
	}
	if command(t, w, room, "host", "previewGeneratedMap", map[string]any{"settings": selected.Settings}) == nil {
		t.Fatal("allowed concurrent same-room generation")
	}
	// Another lobby operation can proceed while rendering is blocked.
	if err := command(t, w, room, "host", "moveDown", map[string]string{"username": "host"}); err != nil {
		t.Fatal(err)
	}
	close(release)
	drainCustomRequest(t, w)
	draft := room.generatedDraft
	if draft == nil || draft.Settings["seed"] != "replacement" || room.generatedSelected != selected {
		t.Fatal("completion replaced applied selection or lost normalized draft")
	}
	if _, ok := gamestate.BuildRuntimeMap(draft.PreviewID, nil); ok {
		t.Fatal("draft polluted runtime map registry")
	}
	saved, err := w.redis.Get(context.Background(), roomSnapshotKey(room.ID)).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	delete(rooms, room.ID)
	restored, err := w.ensureRoomLoaded(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(restored.generatedDraft.Visuals, draft.Visuals) || !reflect.DeepEqual(restored.generatedSelected.Visuals, selected.Visuals) {
		t.Fatal("restart changed exact rendered maps")
	}
	if restored.generatedPending != nil {
		t.Fatal("in-flight request restored as stuck pending")
	}
	// Old room snapshots omit both optional fields and retain automatic defaults.
	var object map[string]json.RawMessage
	if err := json.Unmarshal(saved, &object); err != nil {
		t.Fatal(err)
	}
	delete(object, "generatedDraft")
	delete(object, "generatedSelected")
	old, _ := json.Marshal(object)
	w.redis.Set(context.Background(), roomSnapshotKey(room.ID), old, 0)
	delete(rooms, room.ID)
	legacy, err := w.ensureRoomLoaded(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if legacy.generatedSelected != nil || legacy.generatedDraft != nil || legacy.generatedMapSelection() != nil {
		t.Fatal("old snapshot acquired customization")
	}
}

func TestGeneratedMapCustomizerStaleCompletionAndFailure(t *testing.T) {
	for _, change := range []string{"reset", "players", "failure"} {
		t.Run(change, func(t *testing.T) {
			w, room := customRedisRoom(t)
			old := customPreview(t, 2)
			room.generatedDraft = old
			arrived := make(chan struct{})
			release := make(chan struct{})
			server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
				close(arrived)
				<-release
				if change == "failure" {
					http.Error(writer, "test generation failure", 500)
					return
				}
				json.NewEncoder(writer).Encode(customMapResponse(2))
			}))
			defer server.Close()
			t.Setenv("MAP_GENERATOR_URL", server.URL)
			if err := command(t, w, room, "host", "previewGeneratedMap", map[string]any{"settings": old.Settings}); err != nil {
				t.Fatal(err)
			}
			<-arrived
			if change == "reset" {
				if err := command(t, w, room, "host", "resetGeneratedMap", nil); err != nil {
					t.Fatal(err)
				}
			}
			if change == "players" {
				room.Players[2] = &Client{Username: "third"}
			}
			close(release)
			drainCustomRequest(t, w)
			if change == "reset" && room.generatedDraft != nil {
				t.Fatal("late completion resurrected reset draft")
			}
			if change != "reset" && room.generatedDraft != old {
				t.Fatal("stale or failed request overwrote draft")
			}
		})
	}
}

func TestGeneratedMapCustomizerRejectsInvalidGeneratorGraph(t *testing.T) {
	settings, _ := normalizedGeneratedMapSettings(2, MapGeneratorSettings{"seed": "invalid-graph"})
	for _, mode := range []string{"count", "duplicate", "asymmetric"} {
		response := customMapResponse(2)
		switch mode {
		case "count":
			response.Metadata.Server.Tiles = response.Metadata.Server.Tiles[:21]
		case "duplicate":
			response.Metadata.Server.Tiles[1].ID = "0"
		case "asymmetric":
			response.Metadata.Server.Tiles[1].AdjacentIDs = nil
		}
		if _, err := makeGeneratedMapPreview(2, settings, response); err == nil {
			t.Fatalf("accepted invalid %s graph", mode)
		}
	}
}

func TestGeneratedMapCustomizerStartsExactSelectionAndDropsDrafts(t *testing.T) {
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	w, room := customRedisRoom(t)
	selected := customPreview(t, 2)
	room.generatedDraft = selected
	if err := command(t, w, room, "host", "selectGeneratedMap", map[string]string{"previewId": selected.PreviewID}); err != nil {
		t.Fatal(err)
	}
	// A later unselected draft must never leak into the started match.
	room.generatedDraft = customPreview(t, 2)
	_, choices, err := rulesPresetSettings("standard")
	if err != nil {
		t.Fatal(err)
	}
	room.ExtensionChoices = choices
	t.Setenv("MAP_GENERATOR_URL", "http://127.0.0.1:1")
	room.startLobbyGame(room.hostClient(), room.ID)
	if !room.InProgress || room.Map.Name != selected.PreviewID || room.Gamestate.TurnInfo == nil {
		t.Fatal("start regenerated or ignored selected preview")
	}
	if room.generatedSelected != nil || room.generatedDraft != nil {
		t.Fatal("in-game snapshot retains lobby preview buffers")
	}
	visuals, ok := GetGeneratedMapVisuals(room.Map.Name)
	if !ok || !reflect.DeepEqual(visuals, selected.Visuals) {
		t.Fatal("started image changed")
	}
	if err := w.persistRoom(context.Background(), room); err != nil {
		t.Fatal(err)
	}
	saved, err := w.redis.Get(context.Background(), roomSnapshotKey(room.ID)).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var snapshot map[string]json.RawMessage
	if err := json.Unmarshal(saved, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot["generatedDraft"] != nil || snapshot["generatedSelected"] != nil {
		t.Fatal("lobby images retained in game snapshot")
	}
	raw, err := w.redis.Get(context.Background(), generatedMapKey(room.Map.Name)).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	defer w.redis.Del(context.Background(), generatedMapKey(room.Map.Name))
	var mapSnapshot generatedMapSnapshot
	if err := json.Unmarshal(raw, &mapSnapshot); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(mapSnapshot.Visuals, selected.Visuals) {
		t.Fatal("durable map differs from preview")
	}
}

func TestGeneratedMapCustomizerSelectionCommitAndDefaults(t *testing.T) {
	w, room := customRedisRoom(t)
	preview := customPreview(t, 2)
	room.generatedDraft = preview
	w.cfg.WorkerCount, w.cfg.ConsumerGroup, w.cfg.ConsumerName = 1, room.ID, room.ID
	defer w.redis.Del(context.Background(), room.ID+":commands")
	processPlaybackCommand(t, w, room, "select-map", "selectGeneratedMap", map[string]string{"previewId": preview.PreviewID})
	raw, err := w.redis.Get(context.Background(), roomSnapshotKey(room.ID)).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var snapshot roomSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.GeneratedSelected == nil || snapshot.GeneratedSelected.PreviewID != preview.PreviewID || snapshot.GeneratedDraft != nil {
		t.Fatal("selection was not durably committed once")
	}
	second := customPreview(t, 2)
	room.generatedDraft = second
	processPlaybackCommand(t, w, room, "select-map", "selectGeneratedMap", map[string]string{"previewId": second.PreviewID})
	if room.generatedSelected.PreviewID != preview.PreviewID {
		t.Fatal("duplicate command replaced selection")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	gateway := "custom-map-gateway-" + room.ID
	sub := w.redis.Subscribe(ctx, protocol.GatewayEventsChannel(gateway))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]string{"requestId": "defaults"})
	cmd := protocol.Command{Username: room.HostUsername, RoomID: room.ID, Scope: protocol.ScopeRoom, GatewayID: gateway, ConnectionID: "test", Message: messages.Message{Type: "getGeneratedMapSettings", Data: payload}}
	if err := w.HandleCommand(ctx, cmd); err != nil {
		t.Fatal(err)
	}
	event, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var response protocol.DirectEvent
	if err := json.Unmarshal([]byte(event.Payload), &response); err != nil {
		t.Fatal(err)
	}
	var data struct {
		RequestID         string               `json:"requestId"`
		SelectedPreviewID string               `json:"selectedPreviewId"`
		Settings          MapGeneratorSettings `json:"settings"`
		Preview           struct {
			PreviewID string               `json:"previewId"`
			Settings  MapGeneratorSettings `json:"settings"`
		} `json:"preview"`
	}
	if len(response.Messages) != 1 || response.Messages[0].Type != "generatedmapsettings" {
		t.Fatal("missing targeted settings response")
	}
	if err := json.Unmarshal(response.Messages[0].Data, &data); err != nil {
		t.Fatal(err)
	}
	if data.RequestID != "defaults" || data.SelectedPreviewID != preview.PreviewID || data.Preview.PreviewID != second.PreviewID {
		t.Fatal("settings response lost selected/draft identity")
	}
	if data.Settings["seed"] == data.Preview.Settings["seed"] {
		t.Fatal("Reset defaults accidentally returns custom preview settings")
	}
}

func TestGeneratedMapCustomizerFailedStartKeepsWaitingLobby(t *testing.T) {
	for _, failure := range []string{"selected-bot-rules", "selected-game-creation", "automatic-bot-rules", "automatic-generator"} {
		t.Run(failure, func(t *testing.T) {
			w, room := customRedisRoom(t)
			host, guest := room.Players[0], room.Players[1]
			room.Players = []*Client{host, nil, guest, nil, nil, nil}
			originalPlayers := append([]*Client(nil), room.Players...)
			var selected *generatedMapPreview
			if failure == "selected-bot-rules" || failure == "selected-game-creation" {
				selected = customPreview(t, 2)
				room.generatedDraft, room.generatedSelected = selected, selected
			}
			if failure == "selected-bot-rules" || failure == "automatic-bot-rules" {
				guest.BotModel = "greedy"
			}
			room.ExtensionChoices = []Extension{{ExtensionName: "bad-test-rules", RaceChoices: []ChoiceEntry{{Choice: "not-a-race", IsChecked: true}}, TraitChoices: []ChoiceEntry{{Choice: "not-a-trait", IsChecked: true}}}}
			t.Setenv("MAP_GENERATOR_URL", "http://127.0.0.1:1")
			room.startLobbyGame(host, room.ID)
			if room.InProgress || room.Map != GeneratedMap || !reflect.DeepEqual(room.Players, originalPlayers) || room.generatedSelected != selected || room.saveId != -1 {
				t.Fatal("rejected start mutated the waiting lobby")
			}
			if err := w.persistRoom(context.Background(), room); err != nil {
				t.Fatal(err)
			}
			delete(rooms, room.ID)
			loaded, err := w.ensureRoomLoaded(context.Background(), room.ID)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Map != GeneratedMap || len(loaded.Players) != 6 {
				t.Fatal("failed start persisted a runtime map or compressed roster")
			}
			if err := command(t, w, loaded, loaded.HostUsername, "resetGeneratedMap", nil); err != nil {
				t.Fatalf("failed start made editor inaccessible: %v", err)
			}
		})
	}
}

func TestGeneratedMapCustomizerAuthoritativeHostSaveSelection(t *testing.T) {
	w, room := customRedisRoom(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gateway := "save-selection-" + room.ID
	sub := w.redis.Subscribe(ctx, protocol.GatewayEventsChannel(gateway))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	expectSave := func(connection string, want int64) {
		t.Helper()
		for {
			message, err := sub.ReceiveMessage(ctx)
			if err != nil {
				t.Fatal(err)
			}
			var event protocol.DirectEvent
			if err := json.Unmarshal([]byte(message.Payload), &event); err != nil {
				t.Fatal(err)
			}
			for _, message := range event.Messages {
				if message.Type != "saveSelection" {
					continue
				}
				var selection struct {
					Index int64 `json:"index"`
				}
				if err := json.Unmarshal(message.Data, &selection); err != nil {
					t.Fatal(err)
				}
				if event.ConnectionID != connection || selection.Index != want {
					t.Fatalf("save selection routed incorrectly: %s %+v", event.ConnectionID, selection)
				}
				return
			}
		}
	}
	// A fresh room replaces any save choice retained in the browser from before.
	creator := &Client{Username: "creator-" + room.ID, CommandRoomID: "created-" + room.ID, GatewayID: gateway, ConnectionID: "create", worker: w}
	createRoom(creator, "Created", creator.Username)
	defer w.redis.HDel(context.Background(), protocol.UserRoomsHashKey, creator.Username)
	expectSave("create", -1)
	// A host reconnecting to a loaded-save lobby receives the actual positive ID.
	room.saveId = 42
	if err := w.HandleCommand(ctx, protocol.Command{Username: room.HostUsername, RoomID: room.ID, Scope: protocol.ScopeRoom, GatewayID: gateway, ConnectionID: "reconnect", Message: messages.Message{Type: "reconnect"}}); err != nil {
		t.Fatal(err)
	}
	expectSave("reconnect", 42)
	// The newly promoted host must receive that same saved-game state.
	next := room.Players[1]
	next.GatewayID, next.ConnectionID = gateway, "handover"
	room.Players[0] = nil
	room.updateHostAfterLeave()
	expectSave("handover", 42)
	if room.HostUsername != next.Username {
		t.Fatal("host did not transfer")
	}
}
