package gamestate

// DefaultRaceEnabled controls new-game defaults, not compatibility. Explicit
// lobby selections and old snapshots may still contain these races.
func DefaultRaceEnabled(race string) bool {
	switch race {
	case "Storm Giants", "Scarecrows", "Escargots":
		return false
	default:
		return true
	}
}

// BotTrainingRaces filters the training deck without changing feature indices
// or the vocabulary understood by existing checkpoints and playable bots.
func BotTrainingRaces() []string {
	races := make([]string, 0, len(botV2Races))
	for _, race := range botV2Races {
		if DefaultRaceEnabled(race) {
			races = append(races, race)
		}
	}
	return races
}

// DefaultTraitEnabled controls the default deck without banning custom games.
func DefaultTraitEnabled(trait string) bool { return trait != "Racketeering" }

func BotTrainingTraits() []string {
	traits := make([]string, 0, len(botV2Traits))
	for _, trait := range botV2Traits {
		if DefaultTraitEnabled(trait) {
			traits = append(traits, trait)
		}
	}
	return traits
}
