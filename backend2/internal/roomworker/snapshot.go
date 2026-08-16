package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/store"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

const snapshotVersion = 1

type snapshotPlayer struct {
	Username string `json:"username"`
	Index    int    `json:"index"`
}

type generatedMapSnapshot struct {
	Map     Map                 `json:"map"`
	Visuals GeneratedMapVisuals `json:"visuals"`
}

type roomSnapshot struct {
	Version             int               `json:"version"`
	ID                  string            `json:"id"`
	Name                string            `json:"name"`
	HostUsername        string            `json:"hostUsername"`
	Players             []*snapshotPlayer `json:"players"`
	Spectators          []string          `json:"spectators"`
	InProgress          bool              `json:"inProgress"`
	Map                 Map               `json:"map"`
	SaveID              int64             `json:"saveId"`
	AutoSaveID          int64             `json:"autoSaveId"`
	PlayerStatuses      []string          `json:"playerStatuses"`
	IsDisplayRoom       bool              `json:"isDisplayRoom"`
	ExtensionChoices    []Extension       `json:"extensionChoices"`
	GlobalToggle        bool              `json:"globalToggle"`
	ProcessedCommandIDs []string          `json:"processedCommandIds,omitempty"`
	GameState           json.RawMessage   `json:"gameState,omitempty"`
}

func roomSnapshotKey(roomID string) string  { return "game:room:" + roomID + ":snapshot" }
func generatedMapKey(mapName string) string { return "game:generated-map:" + mapName }

func (w *Worker) persistCommandRoom(ctx context.Context, roomID string) error {
	room, ok := rooms[roomID]
	if !ok || room == nil {
		_ = w.redis.Del(ctx, roomSnapshotKey(roomID)).Err()
		return nil
	}
	return w.persistRoom(ctx, room)
}

func (w *Worker) persistRoom(ctx context.Context, room *Room) error {
	snap := roomSnapshot{
		Version:             snapshotVersion,
		ID:                  room.ID,
		Name:                room.Name,
		HostUsername:        room.HostUsername,
		Players:             make([]*snapshotPlayer, len(room.Players)),
		Spectators:          make([]string, 0, len(room.Spectators)),
		InProgress:          room.InProgress,
		Map:                 room.Map,
		SaveID:              room.saveId,
		AutoSaveID:          room.autoSaveId,
		PlayerStatuses:      append([]string{}, room.playerStatuses...),
		IsDisplayRoom:       room.IsDisplayRoom,
		ExtensionChoices:    room.ExtensionChoices,
		GlobalToggle:        room.GlobalToggle,
		ProcessedCommandIDs: append([]string(nil), room.processedCommandIDs...),
	}
	for i, p := range room.Players {
		if p != nil {
			snap.Players[i] = &snapshotPlayer{Username: p.Username, Index: p.Index}
		}
	}
	for _, s := range room.Spectators {
		if s != nil {
			snap.Spectators = append(snap.Spectators, s.Username)
		}
	}
	if room.Gamestate.TurnInfo != nil {
		data, err := store.EncodeGameState(&room.Gamestate)
		if err != nil {
			return err
		}
		snap.GameState = data
	}

	data, err := json.Marshal(snap)
	if err != nil {
		return err
	}
	ttl := w.cfg.SnapshotTTL
	if room.IsDisplayRoom {
		ttl = w.cfg.DisplayRoomTTL
	}
	if err := w.redis.Set(ctx, roomSnapshotKey(room.ID), data, ttl).Err(); err != nil {
		return err
	}

	if visuals, ok := GetGeneratedMapVisuals(room.Map.Name); ok {
		v, _ := json.Marshal(generatedMapSnapshot{Map: room.Map, Visuals: visuals})
		_ = w.redis.Set(ctx, generatedMapKey(room.Map.Name), v, w.cfg.SnapshotTTL).Err()
	}
	return nil
}

func (w *Worker) ensureRoomLoaded(ctx context.Context, roomID string) (*Room, error) {
	if room, ok := rooms[roomID]; ok && room != nil {
		return room, nil
	}

	data, err := w.redis.Get(ctx, roomSnapshotKey(roomID)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var snap roomSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return nil, err
	}
	if snap.Version != snapshotVersion {
		return nil, fmt.Errorf("unsupported room snapshot version %d", snap.Version)
	}

	_ = w.restoreGeneratedMap(ctx, snap.Map.Name)

	room := &Room{
		ID:                  snap.ID,
		Name:                snap.Name,
		HostUsername:        snap.HostUsername,
		Players:             make([]*Client, len(snap.Players)),
		Spectators:          make([]*Client, 0, len(snap.Spectators)),
		InProgress:          snap.InProgress,
		Map:                 snap.Map,
		saveId:              snap.SaveID,
		autoSaveId:          snap.AutoSaveID,
		playerStatuses:      append([]string{}, snap.PlayerStatuses...),
		IsDisplayRoom:       snap.IsDisplayRoom,
		ExtensionChoices:    snap.ExtensionChoices,
		GlobalToggle:        snap.GlobalToggle,
		processedCommandIDs: append([]string(nil), snap.ProcessedCommandIDs...),
	}
	if len(snap.GameState) > 0 {
		gs, err := store.DecodeGameState(snap.GameState)
		if err != nil {
			return nil, err
		}
		room.Gamestate = *gs
	}

	for i, sp := range snap.Players {
		if sp == nil {
			continue
		}
		c := w.clientByUsername(sp.Username)
		c.Index = sp.Index
		if room.IsDisplayRoom {
			c.DisplayRoom = room
		} else {
			c.Room = room
		}
		room.Players[i] = c
	}
	for _, username := range snap.Spectators {
		c := w.clientByUsername(username)
		c.Room = room
		c.IsSpectator = true
		room.Spectators = append(room.Spectators, c)
	}

	rooms[room.ID] = room
	return room, nil
}

func (w *Worker) clientByUsername(username string) *Client {
	w.clientsMu.Lock()
	defer w.clientsMu.Unlock()
	c, ok := w.clients[username]
	if !ok {
		c = &Client{Username: username, worker: w}
		w.clients[username] = c
	}
	return c
}

func (w *Worker) restoreGeneratedMap(ctx context.Context, mapName string) error {
	if !strings.HasPrefix(mapName, "generated-") {
		return nil
	}
	data, err := w.redis.Get(ctx, generatedMapKey(mapName)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}

	var saved generatedMapSnapshot
	if err := json.Unmarshal(data, &saved); err != nil || saved.Visuals.MapName == "" {
		// Backward-compatible with the first gateway/worker draft, which stored
		// just GeneratedMapVisuals under this key.
		var visuals GeneratedMapVisuals
		if err := json.Unmarshal(data, &visuals); err != nil {
			return err
		}
		saved.Visuals = visuals
		saved.Map = Map{Name: mapName, Offset: visuals.Offset, FontSize: 60, Capacity: 6}
	}

	visuals := saved.Visuals
	specs := make([]gamestate.RuntimeTileSpec, 0, len(visuals.ServerTiles))
	for _, tile := range visuals.ServerTiles {
		biome, err := generatedBiomeToGameBiome(tile.Biome)
		if err != nil {
			return err
		}
		attrs := make([]gamestate.Attribute, 0, len(tile.Attributes))
		for _, name := range tile.Attributes {
			a, err := generatedAttributeToGameAttribute(name)
			if err != nil {
				return err
			}
			attrs = append(attrs, a)
		}
		specs = append(specs, gamestate.RuntimeTileSpec{
			ID: tile.ID.String(), Biome: biome, Attributes: attrs, IsEdge: tile.IsEdge,
			AdjacentIDs: generatedIDsToStrings(tile.AdjacentIDs),
		})
	}
	gamestate.RegisterRuntimeMap(mapName, specs)
	StoreGeneratedMapVisuals(visuals)
	if saved.Map.Name == "" {
		saved.Map.Name = mapName
	}
	mapMap[mapName] = saved.Map
	return nil
}
