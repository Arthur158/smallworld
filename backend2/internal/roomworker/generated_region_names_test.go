package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

const testRegionName = "The Amber Highlands"

func namedRegionPreview(t *testing.T) *generatedMapPreview {
	t.Helper()
	settings := requireNormalizedSettings(t, 2, MapGeneratorSettings{"seed": "named-region"})
	response := customMapResponse(2)
	response.Metadata.RegionName, response.Metadata.RegionNameVersion = testRegionName, 1
	preview, err := makeGeneratedMapPreview(2, settings, response)
	if err != nil {
		t.Fatal(err)
	}
	return preview
}

func TestGeneratedRegionNameFromAutomaticGenerator(t *testing.T) {
	for _, name := range []string{testRegionName, ""} {
		t.Run(name, func(t *testing.T) {
			response := customMapResponse(2)
			response.Metadata.RegionName = name
			if name != "" {
				response.Metadata.RegionNameVersion = 1
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/api/generate" {
					t.Errorf("unexpected generator path %s", r.URL.Path)
				}
				_ = json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()
			t.Setenv("MAP_GENERATOR_URL", server.URL)
			gameMap, err := GenerateRuntimeMapForPlayerCount(2)
			if err != nil {
				t.Fatal(err)
			}
			visuals, ok := GetGeneratedMapVisuals(gameMap.Name)
			if !ok || visuals.RegionName != name || visuals.RegionNameVersion != response.Metadata.RegionNameVersion {
				t.Fatalf("generator region metadata lost: %+v", visuals)
			}
			if !strings.HasPrefix(gameMap.Name, "generated-") || gameMap.Name == name || visuals.MapName != gameMap.Name {
				t.Fatal("display name replaced internal map ID")
			}
			if name == "" {
				raw, _ := json.Marshal(visuals)
				if strings.Contains(string(raw), "regionName") {
					t.Fatal("older generator acquired region name fields")
				}
			}
		})
	}
}

func TestGeneratedRegionNamePublicPreviewAndSelection(t *testing.T) {
	named := namedRegionPreview(t)
	room := &Room{Map: GeneratedMap, generatedSelected: named}
	public := publicGeneratedPreview(named)
	if public["regionName"] != testRegionName || public["regionNameVersion"] != 1 {
		t.Fatalf("preview omitted display metadata: %v", public)
	}
	selection := room.generatedMapSelection()
	if selection.RegionName != testRegionName || selection.RegionNameVersion != 1 || selection.PreviewID != named.PreviewID || selection.Seed != "named-region" {
		t.Fatalf("selection lost map metadata: %+v", selection)
	}
	legacy := customPreview(t, 2)
	room.generatedSelected = legacy
	for _, value := range []any{publicGeneratedPreview(legacy), room.generatedMapSelection(), legacy.Visuals} {
		raw, err := json.Marshal(value)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(raw), "regionName") {
			t.Fatalf("unnamed legacy map acquired optional metadata: %s", raw)
		}
	}
}

func TestGeneratedRegionNameSurvivesSelectionStartAndRestore(t *testing.T) {
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	w, room := customRedisRoom(t)
	selected := namedRegionPreview(t)
	room.generatedDraft = selected
	if err := command(t, w, room, "host", "selectGeneratedMap", map[string]string{"previewId": selected.PreviewID}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	gatewayID := "region-name-" + room.ID
	host := room.hostClient()
	host.GatewayID, host.ConnectionID = gatewayID, "connection"
	sub := w.redis.Subscribe(ctx, protocol.GatewayEventsChannel(gatewayID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	room.sendGeneratedMapSelection(host)
	msg, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var event protocol.DirectEvent
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		t.Fatal(err)
	}
	if len(event.Messages) != 1 || event.Messages[0].Type != "generatedmapselection" {
		t.Fatalf("unexpected event: %+v", event)
	}
	var selection struct {
		PreviewID         string `json:"previewId"`
		RegionName        string `json:"regionName"`
		RegionNameVersion int    `json:"regionNameVersion"`
		Seed              string `json:"seed"`
	}
	if err := json.Unmarshal(event.Messages[0].Data, &selection); err != nil {
		t.Fatal(err)
	}
	if selection.PreviewID != selected.PreviewID || selection.RegionName != testRegionName || selection.RegionNameVersion != 1 || selection.Seed != "named-region" {
		t.Fatalf("selection event lost region metadata: %+v", selection)
	}
	host.GatewayID, host.ConnectionID = "", ""
	if err := w.persistRoom(ctx, room); err != nil {
		t.Fatal(err)
	}
	delete(rooms, room.ID)
	restored, err := w.ensureRoomLoaded(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if restored.generatedSelected == nil || restored.generatedSelected.Visuals.RegionName != testRegionName || restored.generatedSelected.Visuals.RegionNameVersion != 1 {
		t.Fatal("waiting room snapshot lost selected name")
	}
	_, choices, err := rulesPresetSettings("standard")
	if err != nil {
		t.Fatal(err)
	}
	restored.ExtensionChoices = choices
	// Starting a selected preview must not request a newly named map.
	t.Setenv("MAP_GENERATOR_URL", "http://127.0.0.1:1")
	restored.startLobbyGame(restored.hostClient(), restored.ID)
	if !restored.InProgress || restored.Map.Name != selected.PreviewID {
		t.Fatal("start did not reuse named selected preview")
	}
	if err := w.persistRoom(ctx, restored); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.redis.Del(context.Background(), generatedMapKey(selected.PreviewID)) })
	delete(rooms, restored.ID)
	generatedMapVisualsMu.Lock()
	delete(generatedMapVisuals, selected.PreviewID)
	generatedMapVisualsMu.Unlock()
	restored, err = w.ensureRoomLoaded(ctx, restored.ID)
	if err != nil {
		t.Fatal(err)
	}
	visuals, ok := GetGeneratedMapVisuals(selected.PreviewID)
	if !ok || visuals.RegionName != testRegionName || visuals.RegionNameVersion != 1 || visuals.ImageBase64 != selected.Visuals.ImageBase64 {
		t.Fatal("active map restore lost selected name or pixels")
	}
	// Reconnect and saved-game display share this targeted visual payload.
	host = restored.hostClient()
	host.GatewayID, host.ConnectionID = gatewayID, "connection"
	restored.sendGeneratedMapVisualsToClient(host)
	msg, err = sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
		t.Fatal(err)
	}
	if len(event.Messages) != 1 || event.Messages[0].Type != "generatedmapvisuals" {
		t.Fatalf("unexpected restore event: %+v", event)
	}
	var received GeneratedMapVisuals
	if err := json.Unmarshal(event.Messages[0].Data, &received); err != nil {
		t.Fatal(err)
	}
	if received.RegionName != testRegionName || received.RegionNameVersion != 1 || received.MapName != selected.PreviewID {
		t.Fatal("restored visual event lost region name")
	}
}

func TestGeneratedRegionNameLegacySavedVisuals(t *testing.T) {
	for _, wrapped := range []bool{false, true} {
		label := "raw"
		if wrapped {
			label = "snapshot"
		}
		t.Run(label, func(t *testing.T) {
			w, _ := customRedisRoom(t)
			preview := customPreview(t, 2)
			var value any = preview.Visuals
			if wrapped {
				value = generatedMapSnapshot{Map: Map{Name: preview.PreviewID, Capacity: 2}, Visuals: preview.Visuals}
			}
			raw, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Contains(string(raw), "regionName") {
				t.Fatal("legacy fixture contains region names")
			}
			ctx := context.Background()
			if err := w.redis.Set(ctx, generatedMapKey(preview.PreviewID), raw, 0).Err(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { w.redis.Del(context.Background(), generatedMapKey(preview.PreviewID)) })
			if err := w.restoreGeneratedMap(ctx, preview.PreviewID); err != nil {
				t.Fatal(err)
			}
			visuals, ok := GetGeneratedMapVisuals(preview.PreviewID)
			if !ok || visuals.MapName != preview.PreviewID || visuals.RegionName != "" || visuals.RegionNameVersion != 0 {
				t.Fatalf("legacy map restoration changed optional name: %+v", visuals)
			}
		})
	}
}

func TestGeneratedRegionNameSavedDisplaySendsVisualsAfterMapUpdate(t *testing.T) {
	w, room := customRedisRoom(t)
	preview := namedRegionPreview(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	saved, err := json.Marshal(generatedMapSnapshot{
		Map:     Map{Name: preview.PreviewID, Capacity: 2, Offset: preview.Visuals.Offset, FontSize: 60},
		Visuals: preview.Visuals,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.redis.Set(ctx, generatedMapKey(preview.PreviewID), saved, 0).Err(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { w.redis.Del(context.Background(), generatedMapKey(preview.PreviewID)) })
	if err := w.restoreGeneratedMap(ctx, preview.PreviewID); err != nil {
		t.Fatal(err)
	}
	host := room.hostClient()
	room.IsDisplayRoom, room.Players = true, []*Client{host}
	host.Room, host.DisplayRoom = nil, room
	host.GatewayID, host.ConnectionID = "display-"+room.ID, "connection"
	sub := w.redis.Subscribe(ctx, protocol.RoomEventsChannel(room.ID), protocol.GatewayEventsChannel(host.GatewayID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	currentWorker = w
	room.LoadMap(host, preview.PreviewID)
	first, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var mapEvent protocol.RoomEvent
	if err := json.Unmarshal([]byte(first.Payload), &mapEvent); err != nil {
		t.Fatal(err)
	}
	if first.Channel != protocol.RoomEventsChannel(room.ID) || mapEvent.Message.Type != "smallmapupdate" {
		t.Fatalf("first display message was not map update: %s", first.Payload)
	}
	second, err := sub.ReceiveMessage(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var visualEvent protocol.DirectEvent
	if err := json.Unmarshal([]byte(second.Payload), &visualEvent); err != nil {
		t.Fatal(err)
	}
	if second.Channel != protocol.GatewayEventsChannel(host.GatewayID) || len(visualEvent.Messages) != 1 || visualEvent.Messages[0].Type != "generatedmapvisuals" {
		t.Fatalf("map update not followed by targeted visuals: %s", second.Payload)
	}
	var visuals GeneratedMapVisuals
	if err := json.Unmarshal(visualEvent.Messages[0].Data, &visuals); err != nil {
		t.Fatal(err)
	}
	if visuals.MapName != preview.PreviewID || visuals.RegionName != testRegionName || visuals.RegionNameVersion != 1 || visuals.ImageBase64 != preview.Visuals.ImageBase64 {
		t.Fatalf("saved display lost generated name or pixels: %+v", visuals)
	}
}
