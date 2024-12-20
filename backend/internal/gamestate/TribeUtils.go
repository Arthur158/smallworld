package gamestate

import (
	"fmt"
	"math/rand"
	"time"
)

func createTribe(race Race, trait Trait) (TribeEntry, error) {
    tribe := createBaseTribe()
    tribe.Race = race
    tribe.Trait = trait
    pieceCount := 0

    raceVal, raceExists := RaceMap[race]
    if !raceExists {
        return TribeEntry{}, fmt.Errorf("race '%s' not found in RaceMap", race)
    }
    raceVal.Transform(tribe)
    pieceCount += raceVal.Count

    traitVal, traitExists := TraitMap[trait]
    if !traitExists {
        return TribeEntry{}, fmt.Errorf("trait '%s' not found in TraitMap", trait)
    }
    traitVal.Transform(tribe)
    pieceCount += traitVal.Count

    return TribeEntry{
        Tribe: tribe,
        CoinPile: 0,
        PiecePile: pieceCount,
    }, nil
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
        tribeEntry, err := createTribe(raceKeys[i], traitKeys[i])
        if err != nil {
            return nil, fmt.Errorf("failed to create tribe with race '%s' and trait '%s': %w", raceKeys[i], traitKeys[i], err)
        }

        tribeEntries = append(tribeEntries, tribeEntry)
    }

    return tribeEntries, nil
}

func createBaseTribe() *Tribe {
    tribe := Tribe{
        Race: "Unknown",
        Trait: "Unknown",
    }

    tribe.IsStackValid = func(s string) bool {
        return  s == string(tribe.Race)
    }

    tribe.countAttack = func(tile *Tile, cost int, stackType string) []PieceStack {
        if stackType == string(tribe.Race) {
            return []PieceStack{{Type: string(tribe.Race), Amount: max(1, cost)}}
        } else {
            return []PieceStack{{Type: string(tribe.Race), Amount: 1000 + cost}}
        }
    }

    tribe.countDefense = func(tile *Tile) (int, error) {
        price := CountDefense(tile)
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                price += stack.Amount
            }
        }
        return price, nil
    }

    tribe.countReturningStacks = func(tile *Tile) []PieceStack {
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                return []PieceStack{{Type: stack.Type, Amount: stack.Amount - 1}}
            }
        }
        return nil
    }

    tribe.countNewTileStacks = func(ps []PieceStack) []PieceStack {
        return ps
    }

    tribe.CanTileBeAbandoned = func(tile *Tile) bool {
        return true
    }

    tribe.ReceiveAbandonment = func(tile *Tile) []PieceStack {
        return []PieceStack{{Type: string(tribe.Race), Amount: 1}}
    }

    tribe.startRedeployment = func() []PieceStack {
        return []PieceStack{}
    }

    tribe.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
        for _, stack := range tile.PieceStacks {
            if stack.Type == stackType {
                if stack.Amount == 1 {
                    return nil, fmt.Errorf("cannot take off single tribe")
                } else {
                    stack.Amount -= 1
                    return []PieceStack{{Type: stackType, Amount: 1}}, nil
                }
            }
        }
        return nil, fmt.Errorf("There is no such stack")
    }

    tribe.checkZoneAccess = func(t *Tile) error {
        if t.Biome == Water {
            return fmt.Errorf("Cannot conquer water!")
        }
        return nil
    }

    tribe.checkAdjacency = func(t *Tile, gs *GameState) error {
        if gs.IsTribePresentOnTheBoard(tribe.Race) {
            for _, neighbour := range t.AdjacentTiles {
                if neighbour.OwningTribe.Race == tribe.Race {
                    return nil
                }
            }
            return fmt.Errorf("The tile is not adjacent to current territory")
        } else {
            if !t.IsEdge {
                return fmt.Errorf("The tile is not an edge!")
            }
            return nil
        }
    }

    tribe.GetStacksForConquest = func(tile *Tile) []PieceStack {
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                movingStack := []PieceStack{{Type: stack.Type, Amount: stack.Amount - 1}}
                tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, movingStack)
                return movingStack
            }
        }
        return nil
    }

    tribe.CountPoints = func(tile *Tile) int {
        return 1
    }

    return &tribe
}

