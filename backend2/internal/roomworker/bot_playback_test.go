package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"backend/internal/store"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func playbackTestRoom(t *testing.T) (*Worker, *Room) {
	t.Helper()
	w, room := botTestRoom(t)
	host := room.Players[0]
	host.IsSpectator = true
	room.Spectators = []*Client{host}
	delete(w.clients, room.Players[1].Username)
	names := []string{"Bot 1", "Bot 2", "Bot 3"}
	for i := range room.Players {
		room.Players[i] = &Client{Username: fmt.Sprintf("bot:%d", i), BotName: names[i], BotModel: "greedy", Index: i, Room: room, worker: w}
	}
	state, err := gamestate.NewSeeded(names, room.Map.Name, gamestate.BotRaces(), gamestate.BotTraits(), nil, 123)
	if err != nil {
		t.Fatal(err)
	}
	room.Gamestate, room.InProgress = *state, true
	w.cfg.BotMoveDelay = 350 * time.Millisecond
	return w, room
}

func playbackRedisRoom(t *testing.T) (*Worker, *Room) {
	t.Helper()
	addr := os.Getenv("SMALLWORLD_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SMALLWORLD_TEST_REDIS_ADDR to an isolated Redis")
	}
	w, room := playbackTestRoom(t)
	delete(rooms, room.ID)
	room.ID = "bot-playback-" + uuid.NewString()
	rooms[room.ID] = room
	w.redis = redis.NewClient(&redis.Options{Addr: addr})
	if err := w.redis.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	w.cfg.WorkerCount, w.cfg.ConsumerGroup, w.cfg.ConsumerName = 1, room.ID, room.ID
	t.Cleanup(func() {
		w.redis.Del(context.Background(), roomSnapshotKey(room.ID), room.ID+":commands")
		w.redis.HDel(context.Background(), protocol.RoomsHashKey, room.ID)
		w.redis.Close()
	})
	return w, room
}

func playbackState(t *testing.T, room *Room) string {
	t.Helper()
	data, err := store.EncodeGameState(&room.Gamestate)
	if err != nil {
		t.Fatal(err)
	}
	var state store.GameStateCopy
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatal(err)
	}
	sort.Slice(state.TileList, func(i, j int) bool { return state.TileList[i].Id < state.TileList[j].Id })
	for i := range state.TileList {
		tile := &state.TileList[i]
		sort.Strings(tile.TileModifierPoints)
		sort.Strings(tile.TileModifierDefenses)
		sort.Strings(tile.ModifierAfterConquest)
		sort.Strings(tile.ModifierSpecialDefenses)
	}
	data, err = json.Marshal(state)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func processPlaybackCommand(t *testing.T, w *Worker, room *Room, id, kind string, data any) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	stream := room.ID + ":commands"
	_ = w.redis.XGroupCreateMkStream(ctx, stream, w.cfg.ConsumerGroup, "0").Err()
	payload, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(protocol.Command{ID: id, Username: room.HostUsername, RoomID: room.ID, Scope: protocol.ScopeRoom, Message: messages.Message{Type: kind, Data: payload}})
	if err != nil {
		t.Fatal(err)
	}
	if err := w.redis.XAdd(ctx, &redis.XAddArgs{Stream: stream, Values: map[string]any{"command": string(raw)}}).Err(); err != nil {
		t.Fatal(err)
	}
	results, err := w.redis.XReadGroup(ctx, &redis.XReadGroupArgs{Group: w.cfg.ConsumerGroup, Consumer: w.cfg.ConsumerName, Streams: []string{stream, ">"}, Count: 1, Block: -1}).Result()
	if err != nil {
		t.Fatal(err)
	}
	if err := w.processResults(ctx, stream, results); err != nil {
		t.Fatal(err)
	}
	pending, err := w.redis.XPending(ctx, stream, w.cfg.ConsumerGroup).Result()
	if err != nil || pending.Count != 0 {
		t.Fatalf("command not acknowledged: %+v / %v", pending, err)
	}
}

func TestBotPlaybackHostAuthorization(t *testing.T) {
	for _, kind := range []string{"setBotPlayback", "stepBot", "previousBotMove"} {
		t.Run(kind, func(t *testing.T) {
			w, room := playbackTestRoom(t)
			room.botPaused = true
			guest := &Client{Username: "watcher", Room: room, IsSpectator: true, worker: w}
			room.Spectators = append(room.Spectators, guest)
			w.clients[guest.Username] = guest
			before := playbackState(t, room)
			for _, user := range []string{"watcher", "outsider", room.Players[0].Username} {
				if command(t, w, room, user, kind, map[string]any{"paused": false, "speed": 3}) == nil {
					t.Fatalf("%s could control playback", user)
				}
			}
			if !room.botPaused || room.botPlayback().Speed != 1 || !reflect.DeepEqual(before, playbackState(t, room)) {
				t.Fatal("rejected command changed playback or game")
			}
			// A stale client pointer to a different room never grants control.
			cmd := protocol.Command{RoomID: "other-room", Scope: protocol.ScopeRoom}
			if _, err := requireBotPlaybackHost(room.hostClient(), cmd); err == nil {
				t.Fatal("accepted control of a different room")
			}
		})
	}
}

func TestBotPlaybackRequiresCompleteBotGame(t *testing.T) {
	cases := map[string]func(*Room){
		"mixed":      func(r *Room) { r.Players[1].BotModel = "" },
		"empty seat": func(r *Room) { r.Players[1] = nil },
		"no seats":   func(r *Room) { r.Players = nil },
		"lobby":      func(r *Room) { r.InProgress = false },
		"no game":    func(r *Room) { r.Gamestate.TurnInfo = nil },
		"finished":   func(r *Room) { r.Gamestate.TurnInfo.Phase = gamestate.GameFinished },
		"display":    func(r *Room) { r.IsDisplayRoom = true },
	}
	for name, change := range cases {
		t.Run(name, func(t *testing.T) {
			w, room := playbackTestRoom(t)
			room.botPaused = true
			change(room)
			for _, kind := range []string{"setBotPlayback", "stepBot", "previousBotMove"} {
				if command(t, w, room, "host", kind, map[string]bool{"paused": false}) == nil {
					t.Fatalf("accepted %s in %s room", kind, name)
				}
			}
		})
	}
}

func TestBotPlaybackValidationAndDelay(t *testing.T) {
	w, room := playbackTestRoom(t)
	for _, data := range []any{nil, map[string]any{}, map[string]any{"paused": "true"}, map[string]any{"speed": "2"}, map[string]any{"paused": true, "speed": 0}, map[string]any{"paused": true, "speed": 1.5}} {
		if command(t, w, room, "host", "setBotPlayback", data) == nil {
			t.Fatalf("accepted malformed playback: %#v", data)
		}
		if room.botPaused || room.botSpeed != 0 {
			t.Fatal("malformed playback partially changed settings")
		}
	}
	for speed, delay := range map[float64]time.Duration{0.25: 1400 * time.Millisecond, 0.5: 700 * time.Millisecond, 1: 350 * time.Millisecond, 2: 175 * time.Millisecond, 3: 350 * time.Millisecond / 3} {
		if err := command(t, w, room, "host", "setBotPlayback", map[string]any{"speed": speed}); err != nil {
			t.Fatal(err)
		}
		if room.botPlayback().Speed != speed || w.botMoveDelay(room) != delay {
			t.Fatalf("speed %g delay %v, want %v", speed, w.botMoveDelay(room), delay)
		}
	}
	w.cfg.BotMoveDelay = 100 * time.Millisecond
	room.botSpeed = 3 // The preceding map iteration has no guaranteed final speed.
	if got := w.botMoveDelay(room); got != 100*time.Millisecond {
		t.Fatalf("speed bypassed minimum delay: %v", got)
	}
	room.Players[1].BotModel = ""
	if room.botPlayback() != nil || w.botMoveDelay(room) != w.cfg.BotMoveDelay {
		t.Fatal("playback affected a mixed room")
	}
}

func TestBotPlaybackPausedRunnerAndStepGuard(t *testing.T) {
	w, room := playbackTestRoom(t)
	if command(t, w, room, "host", "stepBot", map[string]any{}) == nil {
		t.Fatal("stepped while running")
	}
	if err := command(t, w, room, "host", "setBotPlayback", map[string]bool{"paused": true}); err != nil {
		t.Fatal(err)
	}
	before := playbackState(t, room)
	room.botNextMove = time.Now().Add(-time.Hour)
	for range 5 {
		if w.advanceBots(context.Background()) {
			t.Fatal("paused room kept polling for automatic moves")
		}
	}
	if !reflect.DeepEqual(before, playbackState(t, room)) {
		t.Fatal("automatic runner advanced a paused game")
	}
	room.botError = "invalid bot action"
	for _, kind := range []string{"stepBot", "setBotPlayback"} {
		if command(t, w, room, "host", kind, map[string]bool{"paused": false}) == nil {
			t.Fatalf("%s bypassed engine failure", kind)
		}
	}
}

func TestBotPlaybackSingleDecisionAndReplay(t *testing.T) {
	w, room := playbackRedisRoom(t)
	processPlaybackCommand(t, w, room, "pause", "setBotPlayback", map[string]any{"paused": true, "speed": 0.5})
	before := playbackState(t, room)
	beforeState, err := store.DecodeGameState([]byte(before))
	if err != nil {
		t.Fatal(err)
	}
	expectedRoom := &Room{Gamestate: *beforeState, Players: room.Players, InProgress: true}
	if err := expectedRoom.stepBot(w); err != nil {
		t.Fatal(err)
	}
	expected := playbackState(t, expectedRoom)
	processPlaybackCommand(t, w, room, "single-step", "stepBot", map[string]any{})
	if !reflect.DeepEqual(expected, playbackState(t, room)) || !room.botPaused || room.botDirty {
		t.Fatal("step did not commit exactly one decision and remain paused")
	}
	if w.advanceBots(context.Background()) || !reflect.DeepEqual(expected, playbackState(t, room)) {
		t.Fatal("paused game moved after step")
	}
	processPlaybackCommand(t, w, room, "single-step", "stepBot", map[string]any{})
	if !reflect.DeepEqual(expected, playbackState(t, room)) {
		t.Fatal("duplicate command applied a second step")
	}
	delete(rooms, room.ID)
	w.clients = make(map[string]*Client)
	restored, err := w.ensureRoomLoaded(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !restored.botPaused || restored.botPlayback().Speed != 0.5 || !restored.hasProcessedCommand("single-step") {
		t.Fatal("snapshot lost playback or command journal")
	}
	processPlaybackCommand(t, w, restored, "single-step", "stepBot", map[string]any{})
	if !reflect.DeepEqual(expected, playbackState(t, restored)) {
		t.Fatal("replayed step moved again after worker restart")
	}
	processPlaybackCommand(t, w, restored, "resume", "setBotPlayback", map[string]bool{"paused": false})
	if restored.botPaused || restored.botPlayback().Speed != 0.5 || time.Until(restored.botNextMove) < 500*time.Millisecond {
		t.Fatal("resume did not preserve speed and schedule a fresh interval")
	}
	if !w.advanceBots(context.Background()) || !reflect.DeepEqual(expected, playbackState(t, restored)) {
		t.Fatal("resume executed a hidden or overdue step")
	}
	restored.botNextMove = time.Now().Add(-time.Second)
	if !w.advanceBots(context.Background()) || reflect.DeepEqual(expected, playbackState(t, restored)) {
		t.Fatal("resumed runner did not advance when due")
	}
}

func TestBotPlaybackLegacySnapshotAndMetadata(t *testing.T) {
	w, room := playbackRedisRoom(t)
	ctx := context.Background()
	if err := w.persistRoom(ctx, room); err != nil {
		t.Fatal(err)
	}
	data, err := w.redis.Get(ctx, roomSnapshotKey(room.ID)).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var original map[string]json.RawMessage
	if err := json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	for _, playback := range []string{"", `{"paused":false,"speed":0}`, `{"paused":true,"speed":99}`} {
		if playback == "" {
			delete(original, "botPlayback")
		} else {
			original["botPlayback"] = json.RawMessage(playback)
		}
		data, _ := json.Marshal(original)
		if err := w.redis.Set(ctx, roomSnapshotKey(room.ID), data, 0).Err(); err != nil {
			t.Fatal(err)
		}
		delete(rooms, room.ID)
		loaded, err := w.ensureRoomLoaded(ctx, room.ID)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.botPlayback().Speed != 1 || loaded.botPaused != (playback == `{"paused":true,"speed":99}`) {
			t.Fatalf("legacy playback restored incorrectly: %+v", loaded.botPlayback())
		}
	}
	room = rooms[room.ID]
	room.hostClient().Connected = true
	room.botPaused, room.botSpeed = true, 2
	if err := w.syncLobbyMetadata(ctx); err != nil {
		t.Fatal(err)
	}
	raw, err := w.redis.HGet(ctx, protocol.RoomsHashKey, room.ID).Bytes()
	if err != nil {
		t.Fatal(err)
	}
	var meta protocol.RoomMetadata
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.BotPlayback == nil || !meta.BotPlayback.Paused || meta.BotPlayback.Speed != 2 || meta.Creator != room.HostUsername {
		t.Fatalf("room metadata lost host or playback: %+v", meta)
	}
	room.Players[1] = nil
	if err := w.syncLobbyMetadata(ctx); err != nil {
		t.Fatal(err)
	}
	raw, _ = w.redis.HGet(ctx, protocol.RoomsHashKey, room.ID).Bytes()
	var mixed map[string]json.RawMessage
	if err := json.Unmarshal(raw, &mixed); err != nil {
		t.Fatal(err)
	}
	if _, exists := mixed["botPlayback"]; exists {
		t.Fatal("incomplete room advertised playback controls")
	}
}

func TestBotPlaybackStepFallsBackForMissingModel(t *testing.T) {
	w, room := playbackTestRoom(t)
	room.botPaused = true
	room.Players[0].BotModel = "old-unavailable-model"
	if err := command(t, w, room, "host", "stepBot", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if room.Players[0].BotModel != "random" || !room.botDirty || !room.botPaused {
		t.Fatal("manual step lost missing-model fallback or pause")
	}
}

func TestBotPlaybackPreviousRestoresExactRandomMoves(t *testing.T) {
	w, room := playbackRedisRoom(t)
	for _, player := range room.Players {
		player.BotModel = "random"
	}
	processPlaybackCommand(t, w, room, "pause", "setBotPlayback", map[string]bool{"paused": true})
	states := []string{playbackState(t, room)}
	for i := range 8 {
		processPlaybackCommand(t, w, room, fmt.Sprintf("step-%d", i), "stepBot", map[string]any{})
		states = append(states, playbackState(t, room))
	}
	for i := 7; i >= 4; i-- {
		processPlaybackCommand(t, w, room, fmt.Sprintf("back-%d", i), "previousBotMove", map[string]any{})
		if got := playbackState(t, room); got != states[i] || !room.botPaused || !room.InProgress {
			t.Fatalf("back %d did not restore the exact prior state and pause", i)
		}
		if len(room.botHistory) != i || len(room.botFuture) != 8-i {
			t.Fatalf("back %d changed history incorrectly: %d / %d", i, len(room.botHistory), len(room.botFuture))
		}
		for index, player := range room.Gamestate.Players {
			if player.Name != room.Players[index].displayName() {
				t.Fatal("rewind lost bot display names")
			}
		}
	}
	delete(rooms, room.ID)
	w.clients = make(map[string]*Client)
	var err error
	room, err = w.ensureRoomLoaded(context.Background(), room.ID)
	if err != nil {
		t.Fatal(err)
	}
	processPlaybackCommand(t, w, room, "back-4", "previousBotMove", map[string]any{})
	if playbackState(t, room) != states[4] || len(room.botHistory) != 4 || len(room.botFuture) != 4 {
		t.Fatal("restart/replay lost timeline or applied Previous twice")
	}
	for i := 5; i <= 8; i++ {
		processPlaybackCommand(t, w, room, fmt.Sprintf("replay-%d", i), "stepBot", map[string]any{})
		if playbackState(t, room) != states[i] || !room.botPaused {
			t.Fatalf("forward replay %d did not reproduce exact random-policy state", i)
		}
	}
	if len(room.botFuture) != 0 {
		t.Fatal("replayed timeline retained queued future moves")
	}
	for i := 7; i >= 0; i-- {
		processPlaybackCommand(t, w, room, fmt.Sprintf("rewind-all-%d", i), "previousBotMove", map[string]any{})
		if playbackState(t, room) != states[i] {
			t.Fatalf("full rewind %d did not restore state", i)
		}
	}
	if room.botPlayback().CanGoBack || command(t, w, room, "host", "previousBotMove", map[string]any{}) == nil {
		t.Fatal("initial state advertised or accepted Previous")
	}
	processPlaybackCommand(t, w, room, "resume-replay", "setBotPlayback", map[string]bool{"paused": false})
	if command(t, w, room, "host", "previousBotMove", map[string]any{}) == nil {
		t.Fatal("running game accepted Previous")
	}
	room.botNextMove = time.Now().Add(-time.Second)
	if !w.advanceBots(context.Background()) || playbackState(t, room) != states[1] {
		t.Fatal("automatic runner did not replay the exact next random decision after rewind")
	}
}

func TestBotPlaybackFinishedRewindAndHistoryBound(t *testing.T) {
	w, room := playbackTestRoom(t)
	var beforeLast string
	decisions := 0
	for room.InProgress {
		if decisions > 4096 {
			t.Fatal("bot game did not finish")
		}
		beforeLast = playbackState(t, room)
		if err := room.applyBotMove(w); err != nil {
			t.Fatal(err)
		}
		decisions++
	}
	finished := playbackState(t, room)
	if !room.botPaused || !room.botPlayback().CanGoBack || len(room.botHistory) > botHistoryLimit {
		t.Fatalf("finished playback/history invalid: %+v / %d", room.botPlayback(), len(room.botHistory))
	}
	bytes := 0
	for _, state := range room.botHistory {
		bytes += len(state)
	}
	if bytes > botHistoryByteLimit {
		t.Fatalf("history exceeded byte bound: %d", bytes)
	}
	if err := command(t, w, room, "host", "previousBotMove", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if !room.InProgress || !room.botPaused || !room.botForceSave || playbackState(t, room) != beforeLast {
		t.Fatal("finished rewind did not restore paused game and request autosave")
	}
	if got := room.Gamestate.Messages; len(got) != 1 || got[0].Content != "Rewound one bot move." {
		t.Fatal("rewind did not explain the board reversal in the game log")
	}
	if err := command(t, w, room, "host", "stepBot", map[string]any{}); err != nil {
		t.Fatal(err)
	}
	if room.InProgress || !room.botPaused || playbackState(t, room) != finished {
		t.Fatal("forward replay did not reproduce exact final scores")
	}
}
