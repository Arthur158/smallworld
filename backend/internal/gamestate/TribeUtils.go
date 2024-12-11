package gamestate

import (
	"fmt"
	"math/rand"
	"time"
)

func createTribe(race Race, trait Trait) (Tribe, error) {
    tribe := Tribe{
        Race:  race,
        Trait: trait,
    }

    raceFunc, raceExists := RaceMap[race]
    if !raceExists {
        return Tribe{}, fmt.Errorf("race '%s' not found in RaceMap", race)
    }
    tribe = raceFunc(tribe)

    traitFunc, traitExists := TraitMap[trait]
    if !traitExists {
        return Tribe{}, fmt.Errorf("trait '%s' not found in TraitMap", trait)
    }
    tribe = traitFunc(tribe)

    return tribe, nil
}

func createTribeList() ([]TribeEntry, error) {
    raceKeys := make([]Race, 0, len(RaceMap))
    for race := range RaceMap {
        raceKeys = append(raceKeys, race)
    }

    traitKeys := make([]Trait, 0, len(TraitMap))
    for trait := range TraitMap {
        traitKeys = append(traitKeys, trait)
    }

    r := rand.New(rand.NewSource(time.Now().UnixNano()))

    r.Shuffle(len(raceKeys), func(i, j int) { raceKeys[i], raceKeys[j] = raceKeys[j], raceKeys[i] })
    r.Shuffle(len(traitKeys), func(i, j int) { traitKeys[i], traitKeys[j] = traitKeys[j], traitKeys[i] })

    tribeEntries := []TribeEntry{}
    pairCount := min(len(raceKeys), len(traitKeys)) 

    for i := 0; i < pairCount; i++ {
        tribe, err := createTribe(raceKeys[i], traitKeys[i])
        if err != nil {
            return nil, fmt.Errorf("failed to create tribe with race '%s' and trait '%s': %w", raceKeys[i], traitKeys[i], err)
        }

        tribeEntries = append(tribeEntries, TribeEntry{
            Tribe:     tribe,
            CoinsPile: 0,
        })
    }

    return tribeEntries, nil
}

