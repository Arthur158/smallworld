package gamestate

import (
	"fmt"
	"math/rand"
	"time"
)

func CreateTribe(race Race, trait Trait) (*Tribe, error) {
    tribe := CreateBaseTribe()
    tribe.Race = race
    tribe.Trait = trait
    pieceCount := 0

    raceVal, raceExists := RaceMap[race]
    if !raceExists {
        return &Tribe{}, fmt.Errorf("race '%s' not found in RaceMap", race)
    }
    raceVal.Transform(tribe)
    pieceCount += raceVal.Count

    traitVal, traitExists := TraitMap[trait]
    if !traitExists {
        return &Tribe{}, fmt.Errorf("trait '%s' not found in TraitMap", trait)
    }
    traitVal.Transform(tribe)
    pieceCount += traitVal.Count

    return tribe, nil
}

func createTribeList(raceKeys []string, traitKeys []string) ([]*TribeEntry, error) {

    r := rand.New(rand.NewSource(time.Now().UnixNano()))

    r.Shuffle(len(raceKeys), func(i, j int) { raceKeys[i], raceKeys[j] = raceKeys[j], raceKeys[i] })
    r.Shuffle(len(traitKeys), func(i, j int) { traitKeys[i], traitKeys[j] = traitKeys[j], traitKeys[i] })

    tribeEntries := []*TribeEntry{}
    pairCount := min(len(raceKeys), len(traitKeys)) 

    for i := 0; i < pairCount; i++ {
        tribeEntries = append(tribeEntries, &TribeEntry{
            Race: Race(raceKeys[i]),
            Trait: Trait(traitKeys[i]),
            CoinPile: 0,
            PiecePile: RaceMap[Race(raceKeys[i])].Count + TraitMap[Trait(traitKeys[i])].Count,
        })
    }

    return tribeEntries, nil
}

func CreateBaseTribe() *Tribe {
    tribe := Tribe{
        Race: "Unknown",
        Trait: "Unknown",
        IsActive: true,
        Minimum: 1,
        State: make(map[string]interface{}),
    }

    tribe.checkPresenceMap = make(map[string]func(*Tile, Race) bool)

    tribe.IsStackValidMap = make(map[string]func(string) bool)

    tribe.countAttackMap = make(map[string]func(*Tile, int, string) ([]PieceStack, int, int, int, error))
    tribe.computeDiscountMap = make(map[string]func(*Tile) int)
    tribe.computeGainAttackerMap = make(map[string]func(*Tile) int)
    tribe.computeLossDefenderMap = make(map[string]func(*Tile) int)
    tribe.computePawnKillMap = make(map[string]func(*Tile) int)

    tribe.countDefenseMap = make(map[string]func(*Tile, *Player, *GameState) (int, int, int, error))

    tribe.handleAbandonmentMap = make(map[string]func(*Tile, *GameState))
    tribe.handleReturnMap = make(map[string]func(*Tile, *GameState, int))

    tribe.clearTileMap = make(map[string]func(*Tile, *GameState, int))
    tribe.countNewTileStacksMap = make(map[string]func([]PieceStack, *Tile, *GameState) []PieceStack)

    tribe.canTileBeAbandonedMap = make(map[string]func(*Tile) bool)
    tribe.startRedeploymentMap = make(map[string]func(*GameState) []PieceStack)

    tribe.getStacksOutRedeploymentMap = make(map[string]func(*Tile, string) ([]PieceStack, error))
    tribe.handleDeploymentOutMap = make(map[string]func(*Tile, string, *GameState) error)
    tribe.handleDeploymentInMap = make(map[string]func(*Tile, string, int, *GameState) error)
    tribe.checkZoneAccessMap = make(map[string]func(*Tile, error) error)

    tribe.checkAdjacencyMap = make(map[string]func(*Tile, *GameState, error) error)
    tribe.getStacksForConquestMap = make(map[string]func(*Tile, *Player))
    tribe.countPointsMap = make(map[string]func(*Tile) int)

    tribe.countRemovableAttackingStacksMap = make(map[string]func([]PieceStack, *Player) []PieceStack)
    tribe.countRemovablePiecesMap = make(map[string]func([]PieceStack, *Tile) []PieceStack)

    tribe.specialConquestMap = make(map[string]func(*GameState, *Tile, string) (bool, error))
    
    tribe.specialDefenseMap = make(map[string]func(*GameState, *Tile, *Tribe, string) (bool, error))

    tribe.getStacksForConquestTurnMap = make(map[string]func(*Player, *GameState))

    tribe.prepareRemovalMap = make(map[string]func(*GameState) bool)
    tribe.alternativeDeclineMap = make(map[string]func(*GameState) bool)

    tribe.canGoIntoDeclineMap = make(map[string]func(bool, *GameState) bool)
    tribe.goIntoDeclineMap = make(map[string]func(*GameState))
    tribe.giveInitialStacksMap = make(map[string]func() []PieceStack)
    tribe.countExtrapointsMap = make(map[string]func(*GameState) int)
    tribe.calculateRemainingAttackingStacksMap = make(map[string]func([]PieceStack, bool, bool, error, *Tile, *GameState) ([]PieceStack, bool, bool, error))
    tribe.postConquestMap = make(map[string]func(*Tile, *GameState))
    tribe.canBeRedeployedInMap = make(map[string]func(bool, *Tile, string, *GameState) bool)    
    tribe.canBeRedeployedOutMap = make(map[string]func(bool, *Tile, string) bool)
    tribe.getRedeploymentStackMap = make(map[string]func(string, []PieceStack) []PieceStack)
    tribe.canEndTurnMap = make(map[string]func(*GameState) error)
    tribe.handleOpponentActionMap = make(map[string]func(string, *Player, *GameState) error)
    tribe.handleMovementMap = make(map[string]func(string, *Tile, *Tile, *GameState) error)
    tribe.handleEndOfGameMap = make(map[string]func(*GameState))

    return &tribe
}

