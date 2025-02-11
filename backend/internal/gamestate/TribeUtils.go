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

func createTribeList() ([]*TribeEntry, error) {
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

    tribeEntries := []*TribeEntry{}
    pairCount := min(len(raceKeys), len(traitKeys)) 

    for i := 0; i < pairCount; i++ {
        tribeEntries = append(tribeEntries, &TribeEntry{
            Race: raceKeys[i],
            Trait: traitKeys[i],
            CoinPile: 0,
            PiecePile: RaceMap[raceKeys[i]].Count + TraitMap[traitKeys[i]].Count,
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

    tribe.checkPresence = func(tile *Tile, race Race) bool {
        return tile.OwningTribe.Race == race
    }

    tribe.IsStackValid = func(s string) bool {
        return  s == string(tribe.Race) && tribe.IsActive
    }

    tribe.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
        if stackType == string(tribe.Race) {
            return []PieceStack{{Type: string(tribe.Race), Amount: max(tribe.Minimum, cost - tribe.computeDiscount(stackType, tile))}}, 0, 0, 1
        } else {
            return []PieceStack{{Type: string(tribe.Race), Amount: 1000 + cost}}, 0, 0, 1
        }
    }

    tribe.computeDiscount = func(s string, tile *Tile) int {
        return 0
    }

    tribe.countDefense = func(tile *Tile) (int, int, int, error) {
        price := CountDefense(tile)
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                price += stack.Amount
            }
        }
        return price, 0, 0, nil
    }

    tribe.clearTile = func(tile *Tile, gs *GameState, pawnKill int) {
        for i, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                stack.Amount = max(0, stack.Amount - pawnKill)
                tile.OwningPlayer.PieceStacks = AddPieceStacks(tile.OwningPlayer.PieceStacks, []PieceStack{stack})
                if tile.OwningTribe == &tribe {
                    tile.Presence = None
                    tile.OwningTribe = nil
                    tile.OwningPlayer = nil
                }
                return // Exit after removal to avoid index shifting issues
            }
        }
    }

    tribe.countNewTileStacks = func(ps []PieceStack, tile *Tile) []PieceStack {
        return ps
    }

    tribe.canTileBeAbandoned = func(tile *Tile) bool {
        return tribe.IsActive && tile.OwningTribe.checkPresence(tile, tribe.Race)
    }

    tribe.handleAbandonment = func(tile *Tile, gs *GameState) {
        tribe.clearTile(tile, gs, 0)
        // tile.OwningPlayer.PieceStacks = AddPieceStacks(tile.OwningPlayer.PieceStacks, []PieceStack{{Type: string(tribe.Race), Amount: tribe.Minimum}})
        // for i, stack := range(tile.PieceStacks) {
        //     if stack.Type == string(tribe.Race) {
        //         tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
        //         break
        //     }
        // }
        // tile.Presence = None
    }

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

    tribe.checkZoneAccess = func(t *Tile) error {
        if t.Biome == Water {
            return fmt.Errorf("Cannot conquer water!")
        }
        return nil
    }

    tribe.checkAdjacency = func(t *Tile, gs *GameState) error {
        if gs.IsTribePresentOnTheBoard(tribe.Race) {
            for _, neighbour := range t.AdjacentTiles {
                if neighbour.Presence != None && neighbour.OwningTribe.Race == tribe.Race {
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

    // the dilemma here is that we could make it return the piecestack, but then the action would not be atomic anymore since the piecestack would be removed from the tile and then returned and the stack would be given to the player later.
    tribe.getStacksForConquest = func(tile *Tile, player *Player) {
        if tribe.IsActive {
            for _, stack := range tile.PieceStacks {
                if stack.Type == string(tribe.Race) {
                    // Making sure the action is atomic
                    movingStack := []PieceStack{{Type: stack.Type, Amount: stack.Amount - tribe.Minimum}}
                    tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, movingStack)
                    player.PieceStacks = AddPieceStacks(player.PieceStacks, movingStack)
                }
            }
        }
    }

    tribe.countPoints = func(tile *Tile) int {
        return 1
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

    tribe.specialConquest = func(gs *GameState, tile *Tile, s string, attacker *Player, attackerIndex int) (bool, error) {
        return false, nil
    }

    tribe.getStacksForConquestTurn = func(*Player) {
        return
    }
    

    tribe.prepareRemoval = func(gs *GameState) bool {
        for _, tile := range gs.TileList {
            if tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race){
                tribe.clearTile(tile, gs, 0)
            }
        }
        newstacks := []PieceStack{}
        for _, stack := range(tribe.Owner.PieceStacks) {
            if !tribe.IsStackValid(stack.Type) {
                newstacks = append(newstacks, stack)
            }
        }
        tribe.Owner.PieceStacks = newstacks

        return true
    }

    tribe.canGoIntoDecline = func(gs *GameState) bool {
        return gs.TurnInfo.Phase == DeclineChoice
    }

    // This function should be used to undo tribe advantages in certain tribe, although in an ideal world this would also deactivate any illegal actions, but should not be possible in the first place.
    tribe.goIntoDecline = func(gs *GameState) {
        tribe.IsActive = false
    }

    tribe.giveInitialStacks = func() []PieceStack {
        return nil
    }

    tribe.countExtrapoints = func(gs *GameState) int {
        return 0
    }

    // probs the ugliest piece of code of this project
    tribe.calculateRemainingAttackingStacks = func(expanses []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
        result := []PieceStack{} // Start with an empty list
        hasDiceBeenUsed := false

	for _, expanse := range expanses {
                found := false
		// Search for a matching type in expanses
		for _, reserve := range tribe.Owner.PieceStacks {
			if expanse.Type == reserve.Type {
                                found = true
				if reserve.Amount + 3 < expanse.Amount {
					// Not enough quantity to subtract
					return nil, false, false, fmt.Errorf("Even the dice can't help")
				} else if reserve.Amount < expanse.Amount {
                                        diceThrow := RollDice()
                                        if reserve.Amount + diceThrow >= expanse.Amount {
                                            gs.Messages = append(gs.Messages, fmt.Sprintf("Success: the result of the dice throw was: %d", diceThrow))
                                            hasDiceBeenUsed = true
                                            result = append(result, reserve)
                                        } else {
                                            gs.Messages = append(gs.Messages, fmt.Sprintf("Failure: the result of the dice throw was: %d", diceThrow))
                                            return nil, true, false, nil
                                        }        
                                } else {
                                    result = append(result, expanse)
                                }
                                break
			}
		}
                if !found {
                return nil, false, false, fmt.Errorf("Player did not have stack %s", expanse.Type)
                }
	}
	return result, hasDiceBeenUsed, true, nil
    }

    tribe.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
        return stackType == string(tribe.Race)
    }
    
    tribe.getRedeploymentStack = func(s string, ps []PieceStack) []PieceStack {
        if s == string(tribe.Race) {
            return []PieceStack{{Type: string(tribe.Race), Amount: 1}}
        }
        return []PieceStack{}
    }

    return &tribe
}

