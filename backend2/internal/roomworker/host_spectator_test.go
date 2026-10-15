package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"backend/internal/store"
	"context"
	"encoding/json"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func spectatorTestRoom(t *testing.T) (*Worker, *Room) {
	t.Helper()
	addr := os.Getenv("SMALLWORLD_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SMALLWORLD_TEST_REDIS_ADDR to an isolated Redis")
	}
	w, room := botTestRoom(t)
	w.redis = redis.NewClient(&redis.Options{Addr: addr})
	if err := w.redis.Ping(context.Background()).Err(); err != nil {
		t.Fatal(err)
	}
	delete(rooms, room.ID)
	room.ID = "host-spectator-" + uuid.NewString()
	rooms[room.ID] = room
	names := []string{}
	for _, p := range room.Players {
		if p == nil {
			continue
		}
		delete(w.clients, p.Username)
		p.Username += "-" + room.ID
		w.clients[p.Username] = p
		names = append(names, p.Username)
	}
	room.HostUsername = room.Players[0].Username
	t.Cleanup(func() {
		w.redis.HDel(context.Background(), protocol.UserRoomsHashKey, names...)
		w.redis.Del(context.Background(), roomSnapshotKey(room.ID))
		w.redis.Close()
	})
	return w, room
}

func TestHostSpectatorAuthorization(t *testing.T) {
	w, room := botTestRoom(t)
	for _, user := range []string{"guest", "outsider"} {
		if command(t, w, room, user, "setHostSpectator", map[string]bool{"spectator": true}) == nil {
			t.Fatalf("%s changed the host role", user)
		}
	}
	for _, data := range []any{nil, map[string]any{}, map[string]string{"spectator": "true"}} {
		if command(t, w, room, "host", "setHostSpectator", data) == nil {
			t.Fatalf("accepted malformed role: %#v", data)
		}
	}
	room.InProgress = true
	if command(t, w, room, "host", "setHostSpectator", map[string]bool{"spectator": true}) == nil {
		t.Fatal("accepted a role switch during a game")
	}
	if room.Players[0].Username != "host" || len(room.Spectators) != 0 {
		t.Fatal("rejected command changed roster")
	}
}

func TestHostSpectatorSwitchAndFullRoom(t *testing.T) {
	w, room := spectatorTestRoom(t)
	host := room.Players[0]
	for range 2 {
		if err := command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": true}); err != nil {
			t.Fatal(err)
		}
	}
	if room.Players[0] != nil || len(room.Spectators) != 1 || room.Spectators[0] != host || !host.IsSpectator || room.hostClient() != host {
		t.Fatal("watching host did not keep ownership while freeing seat")
	}
	assertMembership := func(spectator bool) {
		t.Helper()
		data, err := w.redis.HGet(context.Background(), protocol.UserRoomsHashKey, host.Username).Bytes()
		if err != nil {
			t.Fatal(err)
		}
		var m protocol.RoomMembership
		if err := json.Unmarshal(data, &m); err != nil {
			t.Fatal(err)
		}
		if m.RoomID != room.ID || m.IsSpectator != spectator {
			t.Fatalf("incorrect membership: %+v", m)
		}
	}
	assertMembership(true)
	for _, model := range []string{"random", "greedy"} {
		if err := command(t, w, room, host.Username, "addBot", map[string]string{"modelId": model}); err != nil {
			t.Fatal(err)
		}
	}
	if command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": false}) == nil {
		t.Fatal("host displaced a player from a full room")
	}
	assertMembership(true)
	if err := command(t, w, room, host.Username, "kickPlayer", map[string]string{"username": room.Players[0].Username}); err != nil {
		t.Fatal(err)
	}
	if err := command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": false}); err != nil {
		t.Fatal(err)
	}
	if room.Players[0] != host || len(room.Spectators) != 0 || host.IsSpectator {
		t.Fatal("host did not return to the free seat")
	}
	assertMembership(false)
}

func TestHostSpectatorMembershipFailure(t *testing.T) {
	w, room := spectatorTestRoom(t)
	host := room.Players[0]
	w.redis.Close()
	if command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": true}) == nil {
		t.Fatal("accepted role change without persisted membership")
	}
	if room.Players[0] != host || len(room.Spectators) != 0 || host.IsSpectator {
		t.Fatal("failed membership write changed the room")
	}
}

func TestHostSpectatorRoomLifecycle(t *testing.T) {
	w, room := spectatorTestRoom(t)
	host, guest := room.Players[0], room.Players[1]
	if err := command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": true}); err != nil {
		t.Fatal(err)
	}
	room.removePlayer(guest.Username)
	if rooms[room.ID] != room || room.HostUsername != host.Username {
		t.Fatal("last seated human leaving removed spectator-hosted room")
	}
	if err := room.addBot(w, "random"); err != nil {
		t.Fatal(err)
	}
	room.removePlayer(room.Players[0].Username)
	if rooms[room.ID] != room || room.hostClient() != host {
		t.Fatal("removing last bot removed watching host")
	}
	if err := room.addBot(w, "greedy"); err != nil {
		t.Fatal(err)
	}
	bot := room.Players[0]
	room.removeSpectator(host.Username)
	if rooms[room.ID] != nil || host.Room != nil || bot.Room != nil {
		t.Fatal("last human leaving did not clean up bot-only room")
	}
}

func TestHostMigratesBetweenPlayerAndSpectator(t *testing.T) {
	w, room := spectatorTestRoom(t)
	host, guest := room.Players[0], room.Players[1]
	watcher := &Client{Username: "watcher-" + room.ID, Room: room, IsSpectator: true, worker: w}
	room.Spectators = append(room.Spectators, watcher)
	t.Cleanup(func() { w.redis.HDel(context.Background(), protocol.UserRoomsHashKey, watcher.Username) })
	if err := command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": true}); err != nil {
		t.Fatal(err)
	}
	room.removeSpectator(host.Username)
	if room.HostUsername != guest.Username {
		t.Fatal("departing watching host did not transfer to seated human")
	}
	room.removePlayer(guest.Username)
	if room.HostUsername != watcher.Username || rooms[room.ID] != room {
		t.Fatal("host did not transfer to remaining spectator")
	}
	room.removeSpectator(watcher.Username)
	if rooms[room.ID] != nil {
		t.Fatal("room survived last human")
	}
}

func TestHostSpectatorSnapshotAndLobbyReconnect(t *testing.T) {
	w, room := spectatorTestRoom(t)
	host := room.Players[0]
	if err := command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": true}); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.persistRoom(ctx, room); err != nil {
		t.Fatal(err)
	}
	delete(rooms, room.ID)
	w.clients = make(map[string]*Client)
	loaded, err := w.ensureRoomLoaded(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	host = loaded.hostClient()
	if host == nil || !host.IsSpectator || roomPlayerIndex(loaded, host.Username) >= 0 || loaded.HostUsername != room.HostUsername {
		t.Fatal("snapshot lost spectator host")
	}
	gatewayID := "gateway-" + room.ID
	sub := w.redis.Subscribe(ctx, protocol.GatewayEventsChannel(gatewayID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	if err := w.HandleCommand(ctx, protocol.Command{ID: "reconnect", Username: host.Username, RoomID: room.ID, Scope: protocol.ScopeRoom, GatewayID: gatewayID, ConnectionID: "connection", Message: messages.Message{Type: "reconnect"}}); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	// A waiting host gets the usual lobby and host settings, never the spectate
	// message that navigates the frontend into an actual game.
	for range 5 {
		msg, err := sub.ReceiveMessage(ctx)
		if err != nil {
			t.Fatal(err)
		}
		var event protocol.DirectEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			t.Fatal(err)
		}
		for _, m := range event.Messages {
			seen[m.Type] = true
		}
	}
	for _, kind := range []string{"roomid", "smallmapupdate", "playerStatuses", "mapChoices", "choices"} {
		if !seen[kind] {
			t.Fatalf("missing %s after reconnect: %v", kind, seen)
		}
	}
	if seen["spectate"] || !host.Connected || !host.IsSpectator {
		t.Fatal("lobby reconnect entered game or lost host role")
	}
}

func TestBotOnlyRoomStartsAndCompletes(t *testing.T) {
	w, room := spectatorTestRoom(t)
	host, guest := room.Players[0], room.Players[1]
	if err := command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": true}); err != nil {
		t.Fatal(err)
	}
	room.removePlayer(guest.Username)
	for range len(room.Players) {
		if err := command(t, w, room, host.Username, "addBot", map[string]string{"modelId": "greedy"}); err != nil {
			t.Fatal(err)
		}
	}
	if err := command(t, w, room, host.Username, "startGame", nil); err != nil {
		t.Fatal(err)
	}
	if !room.InProgress || !room.hasConnectedUsers() || room.Gamestate.TurnInfo == nil {
		t.Fatal("watching host could not start game")
	}
	if got := room.saveOwners(); len(got) != 1 || got[0] != host {
		t.Fatalf("save does not belong to watching host: %v", got)
	}
	before, err := store.EncodeGameState(&room.Gamestate)
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []string{"tribepick", "finishturn", "decline"} {
		if err := command(t, w, room, host.Username, kind, map[string]int{"pickIndex": 0}); err != nil {
			t.Fatal(err)
		}
	}
	for _, kind := range []string{"savegame", "rollback"} {
		if command(t, w, room, host.Username, kind, nil) == nil {
			t.Fatalf("spectator could invoke %s", kind)
		}
	}
	after, err := store.EncodeGameState(&room.Gamestate)
	if err != nil {
		t.Fatal(err)
	}
	beforeState, err := store.DecodeGameState(before)
	if err != nil {
		t.Fatal(err)
	}
	afterState, err := store.DecodeGameState(after)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeState, afterState) {
		t.Fatal("spectator commands changed game")
	}
	for decisions := 0; room.InProgress; decisions++ {
		if decisions >= 4096 {
			t.Fatal("bot-only match did not finish")
		}
		if err := room.stepBot(w); err != nil {
			t.Fatal(err)
		}
	}
	if room.Gamestate.TurnInfo.Phase != gamestate.GameFinished || room.hostClient() != host {
		t.Fatal("game did not finish with host intact")
	}
}

func TestFinishedBotRoomReconnectRestoresSpectatorAndScores(t *testing.T) {
	w, room := spectatorTestRoom(t)
	host := room.Players[0]
	if err := command(t, w, room, host.Username, "setHostSpectator", map[string]bool{"spectator": true}); err != nil {
		t.Fatal(err)
	}
	room.removePlayer(room.Players[1].Username)
	for range len(room.Players) {
		if err := room.addBot(w, "greedy"); err != nil {
			t.Fatal(err)
		}
	}
	gs, err := gamestate.NewSeeded([]string{"Bot 1", "Bot 2", "Bot 3"}, room.Map.Name, gamestate.BotRaces(), gamestate.BotTraits(), nil, 123)
	if err != nil {
		t.Fatal(err)
	}
	room.Gamestate = *gs
	room.Gamestate.TurnInfo.Phase = gamestate.GameFinished
	wantScores := []int{37, 42, 51}
	for i, score := range wantScores {
		room.Gamestate.Players[i].CoinPile = score
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	host.GatewayID, host.ConnectionID = "gateway-"+room.ID, "connection"
	sub := w.redis.Subscribe(ctx, protocol.GatewayEventsChannel(host.GatewayID), protocol.RoomEventsChannel(room.ID))
	defer sub.Close()
	if _, err := sub.Receive(ctx); err != nil {
		t.Fatal(err)
	}
	currentWorker = w
	if err := w.handleReconnect(ctx, host, protocol.Command{Username: host.Username, RoomID: room.ID}); err != nil {
		t.Fatal(err)
	}
	seen := map[string]json.RawMessage{}
	for range 8 {
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
	for _, kind := range []string{"spectate", "megaUpdate", "gamefinished"} {
		if _, ok := seen[kind]; !ok {
			t.Fatalf("missing %s in finished reconnect: %v", kind, seen)
		}
	}
	var scores []int
	if err := json.Unmarshal(seen["gamefinished"], &scores); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(scores, wantScores) {
		t.Fatalf("restored scores %v, want %v", scores, wantScores)
	}
}
