package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const maxGeneratedMapRequests = 2

type generatedMapPreview struct {
	PreviewID   string               `json:"previewId"`
	PlayerCount int                  `json:"playerCount"`
	Settings    MapGeneratorSettings `json:"settings"`
	Visuals     GeneratedMapVisuals  `json:"visuals"`
}

type pendingGeneratedMap struct {
	token, requestID, username string
	players                    int
	cancel                     context.CancelFunc
}

type generatedMapResult struct {
	roomID, token, requestID, username string
	players                            int
	settings                           MapGeneratorSettings
	response                           *mapGeneratorResponse
	err                                error
}

func (room *Room) generatedMapPlayerCount() int {
	count := 0
	for _, p := range room.Players {
		if p != nil {
			count++
		}
	}
	return max(2, min(6, count))
}

func (room *Room) generatedMapSelection() *protocol.GeneratedMapSelection {
	if room.generatedSelected == nil || !IsGeneratedMapChoice(room.Map.Name) {
		return nil
	}
	seed, _ := room.generatedSelected.Settings["seed"].(string)
	return &protocol.GeneratedMapSelection{PreviewID: room.generatedSelected.PreviewID, PlayerCount: room.generatedSelected.PlayerCount, Seed: seed}
}

func publicGeneratedPreview(preview *generatedMapPreview) map[string]any {
	if preview == nil {
		return nil
	}
	return map[string]any{"previewId": preview.PreviewID, "playerCount": preview.PlayerCount, "settings": preview.Settings, "imageBase64": preview.Visuals.ImageBase64}
}

func (room *Room) clearGeneratedMapCustomization() {
	if room.generatedPending != nil {
		room.generatedPending.cancel()
		room.generatedPending = nil
	}
	if room.generatedDraft != nil || room.generatedSelected != nil {
		room.generatedSelectionDirty = true
	}
	room.generatedDraft = nil
	room.generatedSelected = nil
}

func (room *Room) sendGeneratedMapSelection(client *Client) {
	data := map[string]any{"roomId": room.ID}
	if selection := room.generatedMapSelection(); selection != nil {
		data["previewId"] = selection.PreviewID
		data["playerCount"] = selection.PlayerCount
	}
	raw, _ := json.Marshal(data)
	if client != nil {
		client.sendMessage("generatedmapselection", raw)
	} else {
		room.sendToRoomPlayers(messages.Message{Type: "generatedmapselection", Data: raw})
	}
}

func (room *Room) publishGeneratedMapSelection() {
	if !room.generatedSelectionDirty {
		return
	}
	room.generatedSelectionDirty = false
	sendRoomsUpdateToAll()
	room.sendGeneratedMapSelection(nil)
}

func generatedMapCommandError(client *Client, cmd protocol.Command, err error) {
	var data struct {
		RequestID string `json:"requestId"`
	}
	_ = json.Unmarshal(cmd.Message.Data, &data)
	raw, _ := json.Marshal(map[string]any{"requestId": data.RequestID, "roomId": cmd.RoomID, "message": err.Error()})
	client.sendMessage("generatedmaperror", raw)
}

func (w *Worker) handleGeneratedMapCommand(ctx context.Context, client *Client, cmd protocol.Command) error {
	room, err := requireHost(client)
	if err != nil {
		return err
	}
	if cmd.Scope != protocol.ScopeRoom || room.ID != cmd.RoomID || room.IsDisplayRoom || !roomHasUser(room, client.Username) {
		return errors.New("generated map settings require the host of this waiting room")
	}
	if !IsGeneratedMapChoice(room.Map.Name) {
		return errors.New("choose Generated map before customizing its settings")
	}
	if room.saveId != -1 {
		return errors.New("choose New game before customizing a generated map")
	}
	var data struct {
		RequestID string               `json:"requestId"`
		PreviewID string               `json:"previewId"`
		Settings  MapGeneratorSettings `json:"settings"`
	}
	if len(cmd.Message.Data) > 32*1024 {
		return errors.New("map settings are too large")
	}
	if len(cmd.Message.Data) > 0 {
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid generated map request")
		}
	}
	if len(data.RequestID) > 128 {
		return errors.New("request ID is too long")
	}
	switch cmd.Message.Type {
	case "getGeneratedMapSettings":
		preview := room.generatedDraft
		if preview == nil {
			preview = room.generatedSelected
		}
		settings, err := normalizedGeneratedMapSettings(room.generatedMapPlayerCount(), nil)
		if err != nil {
			return err
		}
		response := map[string]any{"requestId": data.RequestID, "roomId": room.ID, "playerCount": room.generatedMapPlayerCount(), "settings": settings}
		if room.generatedSelected != nil {
			response["selectedPreviewId"] = room.generatedSelected.PreviewID
		}
		if preview != nil {
			response["preview"] = publicGeneratedPreview(preview)
		}
		raw, _ := json.Marshal(response)
		client.sendMessage("generatedmapsettings", raw)
		return nil
	case "previewGeneratedMap":
		if room.generatedPending != nil {
			return errors.New("a preview is already being generated for this room")
		}
		if w.generatedMapRequests >= maxGeneratedMapRequests {
			return errors.New("map generation is busy; try again shortly")
		}
		if data.Settings == nil {
			return errors.New("settings are required")
		}
		players := room.generatedMapPlayerCount()
		settings, err := normalizedGeneratedMapSettings(players, data.Settings)
		if err != nil {
			return err
		}
		if w.generatedMapResults == nil {
			w.generatedMapResults = make(chan generatedMapResult, maxGeneratedMapRequests)
		}
		token := uuid.NewString()
		requestCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		room.generatedPending = &pendingGeneratedMap{token: token, requestID: data.RequestID, username: client.Username, players: players, cancel: cancel}
		w.generatedMapRequests++
		result := generatedMapResult{roomID: room.ID, token: token, requestID: data.RequestID, username: client.Username, players: players, settings: settings}
		// The goroutine owns only immutable request data. Registration, room state,
		// persistence and publication all happen back on the command consumer.
		go func() {
			defer cancel()
			result.response, result.err = requestGeneratedMapContext(requestCtx, settings)
			select {
			case w.generatedMapResults <- result:
			case <-ctx.Done():
			}
		}()
		return nil
	case "selectGeneratedMap":
		preview := room.generatedDraft
		if preview == nil || preview.PreviewID != data.PreviewID {
			preview = room.generatedSelected
		}
		if preview == nil || preview.PreviewID != data.PreviewID {
			return errors.New("that preview is no longer available; generate a new preview")
		}
		if preview.PlayerCount != room.generatedMapPlayerCount() {
			return errors.New("the player count changed; generate a new preview")
		}
		if room.generatedPending != nil {
			room.generatedPending.cancel()
			room.generatedPending = nil
		}
		room.generatedSelected = preview
		room.Map = GeneratedMap
		for len(room.Players) < GeneratedMap.Capacity {
			room.Players = append(room.Players, nil)
		}
		room.generatedSelectionDirty = true
		return nil
	case "resetGeneratedMap":
		room.clearGeneratedMapCustomization()
		room.Map = GeneratedMap
		for len(room.Players) < GeneratedMap.Capacity {
			room.Players = append(room.Players, nil)
		}
		room.generatedSelectionDirty = true
		return nil
	}
	return errors.New("unsupported generated map command")
}

// drainGeneratedMaps is called only by the authoritative worker goroutine.
func (w *Worker) drainGeneratedMaps(ctx context.Context) {
	for {
		select {
		case result := <-w.generatedMapResults:
			w.generatedMapRequests--
			room := rooms[result.roomID]
			if room == nil || room.generatedPending == nil || room.generatedPending.token != result.token {
				continue
			}
			room.generatedPending = nil
			client := room.hostClient()
			if client == nil || client.Username != result.username {
				continue
			}
			requestData, _ := json.Marshal(map[string]string{"requestId": result.requestID})
			cmd := protocol.Command{RoomID: room.ID, Message: messages.Message{Data: requestData}}
			if room.InProgress || room.IsDisplayRoom || room.saveId != -1 || room.generatedMapPlayerCount() != result.players {
				generatedMapCommandError(client, cmd, errors.New("the room changed while generating; generate a new preview"))
				continue
			}
			if result.err != nil {
				generatedMapCommandError(client, cmd, result.err)
				continue
			}
			preview, err := makeGeneratedMapPreview(result.players, result.settings, result.response)
			if err != nil {
				generatedMapCommandError(client, cmd, err)
				continue
			}
			previous := room.generatedDraft
			room.generatedDraft = preview
			if err := w.persistRoom(ctx, room); err != nil {
				room.generatedDraft = previous
				generatedMapCommandError(client, cmd, fmt.Errorf("could not save map preview: %w", err))
				continue
			}
			data := publicGeneratedPreview(preview)
			data["requestId"] = result.requestID
			data["roomId"] = room.ID
			raw, _ := json.Marshal(data)
			client.sendMessage("generatedmappreview", raw)
		default:
			return
		}
	}
}

func makeGeneratedMapPreview(players int, settings MapGeneratorSettings, response *mapGeneratorResponse) (*generatedMapPreview, error) {
	if response == nil {
		return nil, errors.New("map generator returned no preview")
	}
	settings, err := normalizedGeneratedMapSettings(players, settings)
	if err != nil {
		return nil, err
	}
	specs, err := convertGeneratedTiles(response)
	if err != nil {
		return nil, err
	}
	expected := int(settings["tileCount"].(float64))
	if len(specs) != expected {
		return nil, fmt.Errorf("map generator returned %d regions; expected %d", len(specs), expected)
	}
	ids := make(map[string]map[string]bool, len(specs))
	for _, spec := range specs {
		if spec.ID == "" || ids[spec.ID] != nil {
			return nil, errors.New("map generator returned duplicate or empty regions")
		}
		ids[spec.ID] = make(map[string]bool, len(spec.AdjacentIDs))
		for _, adj := range spec.AdjacentIDs {
			if adj == spec.ID || ids[spec.ID][adj] {
				return nil, errors.New("map generator returned invalid adjacency")
			}
			ids[spec.ID][adj] = true
		}
	}
	for id, neighbors := range ids {
		for neighbor := range neighbors {
			if !ids[neighbor][id] {
				return nil, errors.New("map generator returned nonreciprocal adjacency")
			}
		}
	}
	if len(response.Settings) > 0 {
		var normalized MapGeneratorSettings
		if err := json.Unmarshal(response.Settings, &normalized); err != nil {
			return nil, errors.New("map generator returned invalid settings")
		}
		settings, err = normalizedGeneratedMapSettings(players, normalized)
		if err != nil {
			return nil, err
		}
		if int(settings["tileCount"].(float64)) != expected {
			return nil, errors.New("map generator changed the requested tile count")
		}
	}
	id := "generated-" + uuid.NewString()
	rawSettings, _ := json.Marshal(settings)
	return &generatedMapPreview{PreviewID: id, PlayerCount: players, Settings: settings, Visuals: GeneratedMapVisuals{
		MapName: id, ImageBase64: response.ImageBase64, Graphics: response.Metadata.Graphics, Settings: rawSettings,
		ServerTiles: response.Metadata.Server.Tiles, LostTribeTileIDs: generatedLostTribeTileIDs(specs, id), Offset: 0.808,
	}}, nil
}

// useSelectedGeneratedMap installs the exact preview only when starting play.
// Selection itself never populates the process-wide runtime map registry.
func (room *Room) useSelectedGeneratedMap(players int) (Map, error) {
	preview := room.generatedSelected
	if preview == nil {
		return Map{}, errors.New("no generated map selected")
	}
	if preview.PlayerCount != players {
		return Map{}, errors.New("the player count changed; generate a new preview or return to automatic maps")
	}
	response := &mapGeneratorResponse{}
	response.Metadata.Server.Tiles = preview.Visuals.ServerTiles
	specs, err := convertGeneratedTiles(response)
	if err != nil {
		return Map{}, err
	}
	markGeneratedLostTribes(specs, preview.Visuals.LostTribeTileIDs)
	gamestate.RegisterRuntimeMap(preview.PreviewID, specs)
	StoreGeneratedMapVisuals(preview.Visuals)
	return Map{Name: preview.PreviewID, Capacity: players, Offset: preview.Visuals.Offset, FontSize: 60}, nil
}
