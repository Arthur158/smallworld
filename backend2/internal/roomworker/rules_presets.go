package roomworker

import (
	"encoding/json"
	"fmt"
)

// rulesPresetSettings builds independent choices so validation cannot partially
// change a lobby or share checkbox slices between rooms.
func rulesPresetSettings(preset string) (Map, []Extension, error) {
	mapName := "generatedmap"
	switch preset {
	case "standard":
	case "underground":
		mapName = "underground2players"
	default:
		return Map{}, nil, fmt.Errorf("unknown rules preset %q", preset)
	}
	gameMap, ok := mapMap[mapName]
	if !ok {
		return Map{}, nil, fmt.Errorf("preset map %q is unavailable", mapName)
	}
	choices := make([]Extension, len(extensions))
	for i, extension := range extensions {
		underground := extension.Name == "Underground" || extension.Name == "Powers"
		checked := underground == (preset == "underground")
		choice := Extension{
			ExtensionName: extension.Name,
			IsChecked:     checked,
			RaceChoices:   make([]ChoiceEntry, len(extension.Races)),
			TraitChoices:  make([]ChoiceEntry, len(extension.Traits)),
		}
		for j, race := range extension.Races {
			choice.RaceChoices[j] = ChoiceEntry{Choice: race, IsChecked: checked}
		}
		for j, trait := range extension.Traits {
			choice.TraitChoices[j] = ChoiceEntry{Choice: trait, IsChecked: checked}
		}
		choices[i] = choice
	}
	return gameMap, choices, nil
}

func (room *Room) setRulesPreset(preset string) error {
	gameMap, choices, err := rulesPresetSettings(preset)
	if err != nil {
		return err
	}

	roomsMu.Lock()
	if room.InProgress || room.IsDisplayRoom {
		roomsMu.Unlock()
		return fmt.Errorf("rules presets can only be changed in a waiting game room")
	}
	players := make([]*Client, 0, gameMap.Capacity)
	for _, player := range room.Players {
		if player != nil {
			players = append(players, player)
		}
	}
	if len(players) > gameMap.Capacity {
		roomsMu.Unlock()
		return fmt.Errorf("the %s preset supports at most %d seated players; remove players or bots first", preset, gameMap.Capacity)
	}
	for len(players) < gameMap.Capacity {
		players = append(players, nil)
	}

	// Commit the map, roster and choices together only after every check passes.
	// The command consumer persists this state and command ID before ACKing.
	room.Map = gameMap
	room.Players = players
	room.ExtensionChoices = choices
	room.GlobalToggle = false
	room.saveId = -1
	room.playerStatuses = []string{}
	for i, player := range room.Players {
		if player != nil {
			player.Index = i
		}
	}
	roomsMu.Unlock()

	sendRoomsUpdateToAll()
	room.sendSmallMapUpdate()
	room.sendChoices()
	room.sendPlayerStatuses()
	if host := room.hostClient(); host != nil {
		host.sendMessage("saveSelection", json.RawMessage(`{"index": -1}`))
	}
	return nil
}
