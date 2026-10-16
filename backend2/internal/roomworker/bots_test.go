package roomworker

import (
	"backend/internal/bot"
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func botTestRoom(t *testing.T) (*Worker, *Room) {
	t.Helper()
	oldRooms, oldWorker := rooms, currentWorker
	rooms, currentWorker = make(map[string]*Room), nil
	t.Cleanup(func() { rooms, currentWorker = oldRooms, oldWorker })
	r, _ := bot.NewRegistry("")
	w := &Worker{bots: r, clients: make(map[string]*Client)}
	room := &Room{ID: "bot-test", Name: "Test", HostUsername: "host", Players: make([]*Client, 3), Map: mapMap["map3players"], saveId: -1}
	for _, extension := range extensions {
		e := Extension{ExtensionName: extension.Name, IsChecked: true}
		for _, race := range extension.Races {
			e.RaceChoices = append(e.RaceChoices, ChoiceEntry{Choice: race, IsChecked: true})
		}
		for _, trait := range extension.Traits {
			e.TraitChoices = append(e.TraitChoices, ChoiceEntry{Choice: trait, IsChecked: true})
		}
		room.ExtensionChoices = append(room.ExtensionChoices, e)
	}
	host := &Client{Username: "host", Room: room, worker: w, Connected: true}
	guest := &Client{Username: "guest", Room: room, worker: w}
	room.Players[0], room.Players[1] = host, guest
	w.clients[host.Username], w.clients[guest.Username] = host, guest
	rooms[room.ID] = room
	return w, room
}

func command(t *testing.T, w *Worker, room *Room, user, kind string, data any) error {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatal(err)
	}
	return w.HandleCommand(context.Background(), protocol.Command{ID: "test", Username: user, RoomID: room.ID, Scope: protocol.ScopeRoom, Message: messages.Message{Type: kind, Data: raw}})
}

func TestBotHostAuthorizationAndSeats(t *testing.T) {
	w, room := botTestRoom(t)
	for _, user := range []string{"guest", "outsider"} {
		if command(t, w, room, user, "addBot", map[string]string{"modelId": "random"}) == nil {
			t.Fatalf("%s added a bot", user)
		}
	}
	if command(t, w, room, "host", "addBot", map[string]string{"modelId": "../external"}) == nil {
		t.Fatal("accepted an unknown model")
	}
	if err := command(t, w, room, "host", "addBot", map[string]string{"modelId": "random"}); err != nil {
		t.Fatal(err)
	}
	p := room.Players[2]
	if p == nil || p.BotModel != "random" || p.Connected || p.GatewayID != "" {
		t.Fatal("invalid logical bot client")
	}
	if _, exists := w.clients[p.Username]; exists {
		t.Fatal("bot entered authenticated client registry")
	}
	if command(t, w, room, p.Username, "finishturn", nil) == nil {
		t.Fatal("bot identity could authenticate")
	}
	if command(t, w, room, "host", "addBot", map[string]string{"modelId": "random"}) == nil {
		t.Fatal("overfilled room")
	}
	if command(t, w, room, "guest", "setBotModel", map[string]string{"username": p.Username, "modelId": "greedy"}) == nil {
		t.Fatal("guest changed bot")
	}
	if err := command(t, w, room, "host", "setBotModel", map[string]string{"username": p.Username, "modelId": "greedy"}); err != nil {
		t.Fatal(err)
	}
	if p.BotModel != "greedy" {
		t.Fatal("model unchanged")
	}
	room.InProgress = true
	for _, kind := range []string{"addBot", "setBotModel", "kickPlayer", "useBotRules", "setRulesPreset"} {
		if command(t, w, room, "host", kind, map[string]string{"username": p.Username, "modelId": "random", "preset": "standard"}) == nil {
			t.Fatalf("accepted %s mid-game", kind)
		}
	}
	room.InProgress = false
	if err := command(t, w, room, "host", "kickPlayer", map[string]string{"username": p.Username}); err != nil {
		t.Fatal(err)
	}
	if room.Players[2] != nil || p.Room != nil {
		t.Fatal("bot was not removed")
	}
}

func TestBotPresetAndCompleteRoomTurns(t *testing.T) {
	w, room := botTestRoom(t)
	if err := room.addBot(w, "greedy"); err != nil {
		t.Fatal(err)
	}
	races, traits, powers := []string{}, []string{}, []string{}
	for _, e := range room.ExtensionChoices {
		for _, r := range e.RaceChoices {
			if r.IsChecked {
				if e.ExtensionName == "Powers" {
					powers = append(powers, r.Choice)
				} else {
					races = append(races, r.Choice)
				}
			}
		}
		for _, r := range e.TraitChoices {
			if r.IsChecked {
				if e.ExtensionName == "Powers" {
					powers = append(powers, r.Choice)
				} else {
					traits = append(traits, r.Choice)
				}
			}
		}
	}
	if len(races) != 12 || len(traits) != 12 {
		t.Fatalf("preset missing choices: %v / %v", races, traits)
	}
	if err := gamestate.ValidateBotChoices(races, traits, powers); err != nil {
		t.Fatal(err)
	}
	gs, err := gamestate.NewSeeded([]string{"Host", "Guest", "Bot"}, "map3players", races, traits, nil, 123)
	if err != nil {
		t.Fatal(err)
	}
	room.Gamestate, room.InProgress = *gs, true
	for _, p := range room.Players {
		p.BotModel = "greedy"
	}
	for i := 0; room.InProgress; i++ {
		if i > 4096 {
			t.Fatal("room game stuck")
		}
		if err := room.stepBot(w); err != nil {
			t.Fatal(err)
		}
	}
	if room.Gamestate.TurnInfo.Phase != gamestate.GameFinished {
		t.Fatal("room did not finish")
	}
}

func TestHostMigrationAndLastHumanCleanup(t *testing.T) {
	addr := os.Getenv("SMALLWORLD_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SMALLWORLD_TEST_REDIS_ADDR to an isolated Redis")
	}
	w, room := botTestRoom(t)
	w.redis = redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { w.redis.Close() })
	suffix := time.Now().Format("150405.000000000")
	host, guest := "bot-host-"+suffix, "bot-guest-"+suffix
	room.Players[0].Username, room.Players[1].Username, room.HostUsername = host, guest, host
	if err := room.addBot(w, "random"); err != nil {
		t.Fatal(err)
	}
	room.removePlayer(host)
	if room.HostUsername != guest {
		t.Fatal("host did not migrate to remaining human")
	}
	if err := room.addBot(w, "greedy"); err != nil {
		t.Fatal(err)
	}
	room.removePlayer(guest)
	if rooms[room.ID] != nil {
		t.Fatal("bots kept a room alive without human players")
	}
}

// Opt-in Redis test uses only its unique room key and user field, never FLUSHDB.
func TestBotSnapshotRoundTrip(t *testing.T) {
	addr := os.Getenv("SMALLWORLD_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("set SMALLWORLD_TEST_REDIS_ADDR to an isolated Redis")
	}
	w, room := botTestRoom(t)
	w.redis = redis.NewClient(&redis.Options{Addr: addr})
	t.Cleanup(func() { w.redis.Close() })
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := w.redis.Ping(ctx).Err(); err != nil {
		t.Fatal(err)
	}
	room.ID = "bot-snapshot-" + time.Now().Format("150405.000000000")
	rooms[room.ID] = room
	t.Cleanup(func() { w.redis.Del(context.Background(), roomSnapshotKey(room.ID)) })
	if err := room.addBot(w, "greedy"); err != nil {
		t.Fatal(err)
	}
	botID := room.Players[2].Username
	// Existing snapshots retain their legacy names without automatic renaming.
	room.Players[2].BotName = "Bot 3"
	gs, err := gamestate.NewSeeded([]string{"host", "guest", "Bot 3"}, "map3players", gamestate.BotRaces(), gamestate.BotTraits(), nil, 19)
	if err != nil {
		t.Fatal(err)
	}
	room.Gamestate, room.InProgress = *gs, true
	if err := w.persistRoom(ctx, room); err != nil {
		t.Fatal(err)
	}
	delete(rooms, room.ID)
	w.clients = make(map[string]*Client)
	loaded, err := w.ensureRoomLoaded(ctx, room.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Players[2].BotModel != "greedy" || loaded.Players[2].Username != botID || loaded.Players[2].BotName != "Bot 3" {
		t.Fatal("lost bot identity")
	}
	if loaded.Players[2].Connected || w.clients[botID] != nil {
		t.Fatal("restored bot as an authenticated human")
	}
	if loaded.Gamestate.RNGState != gs.RNGState {
		t.Fatal("lost random stream")
	}
	if loaded.Gamestate.Players[0].Name != "host" {
		t.Fatal("lost restored display names")
	}
	if _, err := loaded.Gamestate.BotDecision(); err != nil {
		t.Fatal(err)
	}
}
