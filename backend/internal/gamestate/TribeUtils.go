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
        price, error := tile.countDefense()
        if error != nil {
            return price, 0, 0, error
        }
        for _, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                price += stack.Amount
            }
        }
        return price, 0, 0, nil
    }

    tribe.handleAbandonment = func(tile *Tile, gs *GameState) {
        tribe.clearTile(tile, gs, 0)
    }

    tribe.handleReturn = func(tile *Tile, gs *GameState, cost int) {
        tribe.clearTile(tile, gs , cost)
    }

    tribe.clearTile = func(tile *Tile, gs *GameState, pawnKill int) {
        for i, stack := range tile.PieceStacks {
            if stack.Type == string(tribe.Race) {
                tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                stack.Amount = max(0, stack.Amount - pawnKill)
                if tile.Presence != None {
                    tribe.Owner.PieceStacks = AddPieceStacks(tribe.Owner.PieceStacks, []PieceStack{stack})
                }
                if tile.OwningTribe == &tribe {
                    tile.Presence = None
                    tile.OwningTribe = nil
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

    tribe.specialConquest = func(gs *GameState, tile *Tile, s string, attacker *Player, attackerIndex int) (bool, error) {
        return false, nil
    }

    tribe.getStacksForConquestTurn = func(*Player, *GameState) {}

    tribe.prepareRemoval = func(gs *GameState) bool {
        for _, tile := range gs.TileList {
            if tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race){
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
    tribe.goIntoDecline = func(gs *GameState) int {
        player := tribe.Owner

	for i, tribe := range player.PassiveTribes {
		if (tribe.prepareRemoval(gs)) {
			player.PassiveTribes = append(player.PassiveTribes[:i], player.PassiveTribes[i+1:]...)
                        player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, tribe.countRemovableAttackingStacks(player))
		}
	}

	for _, tile := range gs.TileList {
            if tile.Presence != None && tile.OwningTribe.Race == player.ActiveTribe.Race {
                tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, tile.OwningTribe.countRemovablePieces(tile))
                tile.Presence = Passive
            }
        }

	player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, player.ActiveTribe.countRemovableAttackingStacks(player))
        tribe.IsActive = false

	player.PassiveTribes = append(player.PassiveTribes, player.ActiveTribe)
	player.ActiveTribe = nil
	player.HasActiveTribe = false

	return gs.countPoints(player)
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

    tribe.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
        return stackType == string(tribe.Race) && tile.Presence != None && tile.OwningTribe.checkPresence(tile, tribe.Race)
    }
    
    tribe.getRedeploymentStack = func(s string, ps []PieceStack) []PieceStack {
        return []PieceStack{{Type: s, Amount: 1, Tribe: &tribe}}
    }

    tribe.canEndTurn = func(gs *GameState) error {
        return nil
    }

    return &tribe
}

