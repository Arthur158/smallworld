package gamestate

import (
	"fmt"
	"math/rand"
	"time"
        "log"
)

func createTribe(race Race, trait Trait) (*Tribe, error) {
    tribe := createBaseTribe()
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

func createBaseTribe() *Tribe {
    tribe := Tribe{
        Race: "Unknown",
        Trait: "Unknown",
        IsActive: true,
        State: make(map[string]interface{}),
    }

    tribe.IsStackValid = func(s string) bool {
        return  s == string(tribe.Race) && tribe.IsActive
    }

    tribe.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
        if stackType == string(tribe.Race) {
            return []PieceStack{{Type: string(tribe.Race), Amount: max(1, cost)}}, 0, 0
        } else {
            return []PieceStack{{Type: string(tribe.Race), Amount: 1000 + cost}}, 0, 0
        }
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

    tribe.countReturningStacks = func(tile *Tile) ([]PieceStack, []PieceStack) {
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                return []PieceStack{{Type: stack.Type, Amount: stack.Amount - 1}}, nil
            }
        }
        return nil, nil
    }

    tribe.countNewTileStacks = func(ps []PieceStack, tile *Tile) []PieceStack {
        return ps
    }

    tribe.canTileBeAbandoned = func(tile *Tile) bool {
        return tribe.IsActive
    }

    tribe.receiveAbandonment = func(tile *Tile) []PieceStack {
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                return []PieceStack{{Type: stack.Type, Amount: stack.Amount }}
            }
        }
        return []PieceStack{}
    }

    tribe.startRedeployment = func(gs *GameState) []PieceStack {
        return []PieceStack{}
    }

    tribe.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
        if stackType == string(tribe.Race) {
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
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                // Making sure the action is atomic
                movingStack := []PieceStack{{Type: stack.Type, Amount: stack.Amount - 1}}
                tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, movingStack)
                player.PieceStacks = AddPieceStacks(player.PieceStacks, movingStack)
            }
        }
    }

    tribe.countPoints = func(tile *Tile) int {
        return 1
    }

    tribe.countRemainingAttackingStacks = func(player *Player) []PieceStack {
        return []PieceStack{}
    }

    tribe.countPiecesRemaining = func(tile *Tile) []PieceStack {
        return []PieceStack{{Type: string(tile.OwningTribe.Race), Amount: 1}}
    }
    

    tribe.prepareRemoval = func(gs *GameState) bool {
        for _, tile := range gs.TileList {
            if tile.Presence != None && tile.OwningTribe.Race == tribe.Race {
                tile.PieceStacks = []PieceStack{}
                tile.Presence = None
            }
        }
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

    tribe.countExtrapoints = func() int {
        return 0
    }

    // probs the ugliest piece of code of this project
    tribe.calculateRemainingAttackingStacks = func(reserves []PieceStack, expanses []PieceStack) ([]PieceStack, []PieceStack, bool, error) {
        result := []PieceStack{} // Start with an empty list
        stacksToRemove := []PieceStack{}
        hasDiceBeenUsed := false

	for _, stack1 := range reserves {
		subtracted := false

		// Search for a matching type in expanses
		for _, stack2 := range expanses {
			if stack1.Type == stack2.Type {
				if stack1.Amount + 3 < stack2.Amount {
					// Not enough quantity to subtract
					return nil, nil, false, fmt.Errorf("Even the dice can't help")
				} else if stack1.Amount < stack2.Amount {
                                        diceThrow := RollDice()
                                        println(diceThrow)
                                        if stack1.Amount + diceThrow >= stack2.Amount {
                                            hasDiceBeenUsed = true
                                            stacksToRemove = append(stacksToRemove, PieceStack{Type: stack1.Type, Amount: stack2.Amount - stack1.Amount})
                                            log.Println(stacksToRemove)
                                        } else {
                                            return nil, nil, true, fmt.Errorf("The dice was not enough")
                                        }
                                        
                                }
				// Subtract the amount
				remainingAmount := stack1.Amount - stack2.Amount
				if remainingAmount > 0 {
					// Only add to result if the remaining amount is greater than 0
					result = append(result, PieceStack{Type: stack1.Type, Amount: remainingAmount})
				}
				subtracted = true
				break
			}
		}

		// If no match was found in expanses, add the stack1 element unchanged
		if !subtracted {
			result = append(result, stack1)
		}
	}

	// Verify no types in expanses are missing from reserves
	for _, stack2 := range expanses {
		found := false
		for _, stack1 := range reserves {
			if stack1.Type == stack2.Type {
				found = true
				break
			}
		}
		if !found {
			return nil, nil, false, fmt.Errorf("A stack is missing")
		}
	}

	return result, stacksToRemove, hasDiceBeenUsed, nil

    }

    tribe.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
        return stackType == string(tribe.Race)
    }

    return &tribe
}

