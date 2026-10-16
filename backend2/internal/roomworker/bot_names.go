package roomworker

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
)

// Keep this vocabulary synchronized with frontend/src/data/roomNameAdjectives.ts.
// Embedding it avoids a runtime dependency on frontend assets.
//
//go:embed bot_name_adjectives.json
var botNameAdjectivesJSON []byte

var botNameAdjectives = func() []string {
	var adjectives []string
	if err := json.Unmarshal(botNameAdjectivesJSON, &adjectives); err != nil {
		panic(fmt.Sprintf("invalid embedded bot name adjectives: %v", err))
	}
	return adjectives
}()

func botModelDisplayName(w *Worker, modelID string) (string, error) {
	if w != nil {
		for _, model := range w.bots.Catalog() {
			if model.ID == modelID {
				return model.Name, nil
			}
		}
	}
	return "", fmt.Errorf("unknown bot model %q", modelID)
}

func (room *Room) newBotName(w *Worker, modelID string, replacing *Client) (string, error) {
	modelName, err := botModelDisplayName(w, modelID)
	if err != nil {
		return "", err
	}
	used := make(map[string]bool)
	for _, occupants := range [][]*Client{room.Players, room.Spectators} {
		for _, occupant := range occupants {
			if occupant != nil && occupant != replacing {
				used[occupant.displayName()] = true
			}
		}
	}
	if replacing != nil {
		if oldModelName, err := botModelDisplayName(w, replacing.BotModel); err == nil {
			adjective, found := strings.CutSuffix(replacing.BotName, " "+oldModelName)
			name := adjective + " " + modelName
			if found && slices.Contains(botNameAdjectives, adjective) && !used[name] {
				return name, nil
			}
		}
	}

	// Cosmetic randomness must not consume the game's seeded random stream.
	start := rand.IntN(len(botNameAdjectives))
	for offset := range len(botNameAdjectives) {
		name := botNameAdjectives[(start+offset)%len(botNameAdjectives)] + " " + modelName
		if !used[name] {
			return name, nil
		}
	}
	base := botNameAdjectives[start] + " " + modelName
	for suffix := 2; ; suffix++ {
		name := fmt.Sprintf("%s %d", base, suffix)
		if !used[name] {
			return name, nil
		}
	}
}
