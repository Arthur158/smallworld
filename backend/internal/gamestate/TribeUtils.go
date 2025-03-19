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
    tribe.handleReturn = func(tile *Tile, gs *GameState, cost int) {
        tribe.clearTile(tile, gs , cost)
    }

    tribe.clearTile = func(tile *Tile, gs *GameState, pawnKill int) {
        for i, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                stack.Amount = max(0, stack.Amount - pawnKill)
                if tile.CheckPresence() != None {
                    tribe.Owner.PieceStacks = AddPieceStacks(tribe.Owner.PieceStacks, []PieceStack{stack})
                }
                if tile.OwningTribe == &tribe {
                    tile.OwningTribe = nil
                }
                return // Exit after removal to avoid index shifting issues
            }
        }
    }

    tribe.countNewTileStacksMap = make(map[string]func([]PieceStack, *Tile, *GameState) []PieceStack)

    tribe.canTileBeAbandonedMap = make(map[string]func(*Tile) bool)
    tribe.startRedeployment = func(gs *GameState) []PieceStack {
        return []PieceStack{}
    }

    tribe.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
        if stackType == string(tribe.Race) {
            for _, stack := range tile.PieceStacks {
                if stack.Type == stackType {
                    if stack.Amount == tribe.Minimum {
                        return nil, fmt.Errorf("cannot take off single tribe")
                    } else {
                        return []PieceStack{{Type: stackType, Amount: 1}}, nil
                    }
                }
            }
        }
        return nil, fmt.Errorf("There is no such stack")
    }

    tribe.handleDeploymentOut = func(tile *Tile, stackType string, i int, gs *GameState) error {
        if tile.CheckPresence() == None {
            return fmt.Errorf("This tile does not contain any tribe!")
        }
	stacks, err := tile.OwningTribe.getStacksOutRedeployment(tile, stackType)
	if err != nil {
		return fmt.Errorf("Unable to redeploy", err)
	}

        player := tribe.Owner

        ok := false
        tile.PieceStacks, ok = SubtractPieceStacks(tile.PieceStacks, stacks)
	if !ok {
		return fmt.Errorf("Could not substract the stacks")
	}
	
	player.PieceStacks = AddPieceStacks(player.PieceStacks, stacks)

	return nil
    }

    tribe.handleDeploymentIn = func(tile *Tile, stackType string, i int, gs *GameState) error {
	if !tribe.canBeRedeployedIn(tile, stackType, gs) {
		return fmt.Errorf("Cannot redeploy here")
	}

        player := tribe.Owner

	movingStack := tribe.getRedeploymentStack(stackType, player.PieceStacks)

	newStacks, ok := SubtractPieceStacks(player.PieceStacks, movingStack)
	if !ok {
		return fmt.Errorf("Cannot redeploy pieces you don't have")
	}
	player.PieceStacks = newStacks

	tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)

	return nil
    }

    tribe.checkZoneAccessMap = make(map[string]func(*Tile, error) error)

    tribe.checkAdjacencyMap = make(map[string]func(*Tile, *GameState, error) error)
    tribe.getStacksForConquestMap = make(map[string]func(*Tile, *Player))
    tribe.countPoints = func(tile *Tile) int {
        return tile.countPoints()
    }

    tribe.countRemovableAttackingStacks = func(player *Player) []PieceStack {
        newstacks := []PieceStack{}
        for _, stack := range(player.PieceStacks) {
            if stack.Type == string(tribe.Race) {
                newstacks = append(newstacks, stack)
            }
        }
        return newstacks
    }

    tribe.countRemovablePieces = func(tile *Tile) []PieceStack {
        amount := 1
        for _, stack := range(tile.PieceStacks) {
            if stack.Type == string(tribe.Race) {
                amount = stack.Amount
            }
        }
        return []PieceStack{{Type: string(tile.OwningTribe.Race), Amount: amount - 1}}
    }

    tribe.specialConquestMap = make(map[string]func(*GameState, *Tile, string) (bool, error))
    
    tribe.specialDefense = func(gs *GameState, t1 *Tile, t2 *Tribe, s string) (bool, error) {
        return false, nil
    }

    tribe.getStacksForConquestTurnMap = make(map[string]func(*Player, *GameState))

    tribe.prepareRemoval = func(gs *GameState) bool {
        for _, tile := range gs.TileList {
            if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, tribe.Race){
                tribe.clearTile(tile, gs, 0)
            }
        }
        player := tribe.Owner
        player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, tribe.countRemovableAttackingStacks(player))

        return true
    }

    tribe.canGoIntoDecline = func(gs *GameState) bool {
        return gs.TurnInfo.Phase == DeclineChoice
    }

    // This function should be used to undo tribe advantages in certain tribe, although in an ideal world this would also deactivate any illegal actions, but should not be possible in the first place.
    tribe.goIntoDecline = func(gs *GameState) {
        player := tribe.Owner

	for i, tribe := range player.PassiveTribes {
		if (tribe.prepareRemoval(gs)) {
			player.PassiveTribes = append(player.PassiveTribes[:i], player.PassiveTribes[i+1:]...)
                        player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, tribe.countRemovableAttackingStacks(player))
		}
	}

	for _, tile := range gs.TileList {
            if tile.CheckPresence() != None && tile.OwningTribe.Race == player.ActiveTribe.Race {
                tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, tile.OwningTribe.countRemovablePieces(tile))
            }
        }

	player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, player.ActiveTribe.countRemovableAttackingStacks(player))
        tribe.IsActive = false

	player.PassiveTribes = append(player.PassiveTribes, player.ActiveTribe)
	player.ActiveTribe = nil
    }

    tribe.giveInitialStacksMap = make(map[string]func() []PieceStack)

    tribe.countExtrapoints = func(gs *GameState) int {
        return 0
    }

    // probs the ugliest piece of code of this project
    tribe.calculateRemainingAttackingStacksMap = make(map[string]func([]PieceStack, bool, bool, error, *Tile, *GameState) ([]PieceStack, bool, bool, error))
    tribe.postConquestMap = make(map[string]func(*Tile, *GameState))

    tribe.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
        return stackType == string(tribe.Race) && tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, tribe.Race)
    }
    
    tribe.getRedeploymentStack = func(s string, ps []PieceStack) []PieceStack {
        return []PieceStack{{Type: s, Amount: 1, Tribe: &tribe}}
    }

    tribe.canEndTurn = func(gs *GameState) error {
        return nil
    }

    tribe.handleOpponentAction = func(s string, p *Player, gs *GameState) error {
        return fmt.Errorf("Invalid opponent action!")
    }

    tribe.handleMovementMap = make(map[string]func(string, *Tile, *Tile, *GameState) error)

    tribe.handleEndOfGame = func(gs *GameState) {}

    return &tribe
}

