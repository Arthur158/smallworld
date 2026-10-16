package roomworker

import (
	"backend/internal/bot"
	"backend/internal/gamestate"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func botNameTestRegistry(t *testing.T) *bot.Registry {
	t.Helper()
	dir := t.TempDir()
	for _, id := range []string{"training-a", "training-b"} {
		model := bot.Model{
			Info:           bot.Info{ID: id, Name: "Training 6"},
			FormatVersion:  1,
			Schema:         gamestate.BotSchema,
			StateFeatures:  gamestate.BotStateFeatures,
			ActionFeatures: gamestate.BotActionFeatures,
			Layers: []bot.Layer{
				{Weights: [][]float64{make([]float64, gamestate.BotStateFeatures+gamestate.BotActionFeatures)}, Bias: []float64{0}},
				{Weights: [][]float64{{0}}, Bias: []float64{0}},
				{Weights: [][]float64{{0}}, Bias: []float64{0}},
			},
		}
		data, err := json.Marshal(model)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, id+".json"), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	registry, err := bot.NewRegistry(dir)
	if err != nil {
		t.Fatal(err)
	}
	return registry
}

func TestBotNamesUseDisplayNamesAndAvoidOccupiedNames(t *testing.T) {
	w, room := botTestRoom(t)
	w.bots = botNameTestRegistry(t)
	room.Players[1] = nil
	// Leave only two adjective combinations available. This deterministically
	// exercises collisions with a seated human and with spectators.
	room.Players[0].Username = botNameAdjectives[0] + " Training 6"
	for _, adjective := range botNameAdjectives[1 : len(botNameAdjectives)-2] {
		room.Spectators = append(room.Spectators, &Client{Username: adjective + " Training 6"})
	}
	for _, modelID := range []string{"training-a", "training-b"} {
		if err := room.addBot(w, modelID); err != nil {
			t.Fatal(err)
		}
	}
	first, second := room.Players[1], room.Players[2]
	if first.BotName == second.BotName {
		t.Fatalf("different model IDs with identical display names produced duplicate %q", first.BotName)
	}
	for _, player := range []*Client{first, second} {
		adjective, found := strings.CutSuffix(player.BotName, " Training 6")
		if !found || !slices.Contains(botNameAdjectives[len(botNameAdjectives)-2:], adjective) {
			t.Fatalf("expected an unoccupied adjective and actual model display name, got %q", player.BotName)
		}
	}
	if err := room.setBotModel(w, second.Username, "training-a"); err != nil {
		t.Fatal(err)
	}
	if first.BotName == second.BotName {
		t.Fatal("same-model bots share a display name")
	}
}

func TestBotNameSuffixFallbackAvoidsOccupiedNames(t *testing.T) {
	w, room := botTestRoom(t)
	w.bots = botNameTestRegistry(t)
	for _, adjective := range botNameAdjectives {
		room.Spectators = append(room.Spectators,
			&Client{Username: adjective + " Training 6"},
			&Client{Username: adjective + " Training 6 2"},
		)
	}
	if err := room.addBot(w, "training-a"); err != nil {
		t.Fatal(err)
	}
	name := room.Players[2].BotName
	adjective, found := strings.CutSuffix(name, " Training 6 3")
	if !found || !slices.Contains(botNameAdjectives, adjective) {
		t.Fatalf("expected unique numbered fallback after exhausting adjectives, got %q", name)
	}
}

func TestBotModelRenameRetainsAdjectiveAndSkipsSelf(t *testing.T) {
	w, room := botTestRoom(t)
	w.bots = botNameTestRegistry(t)
	gs, err := gamestate.NewSeeded([]string{"host", "guest", "bot"}, "map3players", gamestate.BotRaces(), gamestate.BotTraits(), nil, 19)
	if err != nil {
		t.Fatal(err)
	}
	room.Gamestate = *gs
	rngBefore := room.Gamestate.RNGState
	if err := room.addBot(w, "training-a"); err != nil {
		t.Fatal(err)
	}
	p := room.Players[2]
	original := p.BotName
	for _, modelID := range []string{"training-a", "training-b"} {
		if err := room.setBotModel(w, p.Username, modelID); err != nil {
			t.Fatal(err)
		}
		if p.BotName != original {
			t.Fatalf("same display name changed the bot's name: %q -> %q", original, p.BotName)
		}
	}
	adjective := strings.TrimSuffix(original, " Training 6")
	if err := room.setBotModel(w, p.Username, "greedy"); err != nil {
		t.Fatal(err)
	}
	if p.BotName != adjective+" Greedy bot" {
		t.Fatalf("model change did not retain the adjective and update the model name: %q", p.BotName)
	}
	room.Players[1].Username = original
	if err := room.setBotModel(w, p.Username, "training-a"); err != nil {
		t.Fatal(err)
	}
	if p.BotName == original || !strings.HasSuffix(p.BotName, " Training 6") {
		t.Fatalf("model change did not avoid a human's name: %q", p.BotName)
	}
	if room.Gamestate.RNGState != rngBefore {
		t.Fatal("adding and renaming bots consumed gameplay randomness")
	}
}

func TestLegacyBotNameStaysUntilModelChanges(t *testing.T) {
	w, room := botTestRoom(t)
	if err := room.addBot(w, "random"); err != nil {
		t.Fatal(err)
	}
	p := room.Players[2]
	p.BotName = "Bot 3"
	if err := room.setBotModel(w, p.Username, "random"); err != nil {
		t.Fatal(err)
	}
	if p.BotName != "Bot 3" {
		t.Fatal("same-model request renamed an existing bot")
	}
	if err := room.setBotModel(w, p.Username, "greedy"); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(p.BotName, " Greedy bot") {
		t.Fatalf("new model retained a stale legacy name: %q", p.BotName)
	}
}

func TestGeneratedBotNamesSurviveSnapshotRoundTrip(t *testing.T) {
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
	room.ID = "bot-names-snapshot-" + time.Now().Format("150405.000000000")
	rooms[room.ID] = room
	t.Cleanup(func() { w.redis.Del(context.Background(), roomSnapshotKey(room.ID)) })
	room.Players[1] = nil
	for range 2 {
		if err := room.addBot(w, "random"); err != nil {
			t.Fatal(err)
		}
	}
	names := []string{room.Players[0].displayName(), room.Players[1].BotName, room.Players[2].BotName}
	gs, err := gamestate.NewSeeded(names, "map3players", gamestate.BotRaces(), gamestate.BotTraits(), nil, 19)
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
	for i := 1; i < 3; i++ {
		if loaded.Players[i].BotName != names[i] || loaded.Gamestate.Players[i].Name != names[i] {
			t.Fatalf("snapshot changed bot %d's name", i)
		}
	}
	if loaded.Gamestate.RNGState != gs.RNGState {
		t.Fatal("naming changed the game's seeded random stream")
	}
}
