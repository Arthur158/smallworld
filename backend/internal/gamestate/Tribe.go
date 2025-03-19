package gamestate

import (
	"fmt"
)

type Race string;
type Trait string;

type Tribe struct {
	Owner *Player
	Race Race;
	Trait Trait;
	IsActive bool;
	State map[string]interface{};
	Minimum int;

	giveInitialStacksMap map[string]func() []PieceStack;
	checkPresenceMap map[string]func(*Tile, Race) bool;

	//abandonment
	canTileBeAbandonedMap map[string]func(*Tile) bool;
	handleAbandonmentMap map[string]func(*Tile, *GameState);

	// receive for conquest
	getStacksForConquestTurnMap map[string]func(*Player, *GameState);
	getStacksForConquestMap map[string]func(*Tile, *Player);

	// conquest checks
	IsStackValidMap map[string]func(string) bool;
	checkZoneAccessMap map[string]func(*Tile, error) error;
	checkAdjacencyMap map[string]func(*Tile, *GameState, error) error;

	// conquest for attacker
	countAttackMap map[string]func(*Tile, int, string) ([]PieceStack, int, int, int, error);
	computeDiscountMap map[string]func(*Tile) int;
	computeGainAttackerMap map[string]func(*Tile) int;
	computeLossDefenderMap map[string]func(*Tile) int;
	computePawnKillMap map[string]func(*Tile) int;
	countNewTileStacksMap map[string]func([]PieceStack, *Tile, *GameState) []PieceStack;
	calculateRemainingAttackingStacksMap map[string]func([]PieceStack, bool, bool, error, *Tile, *GameState) ([]PieceStack, bool, bool, error)
	postConquestMap map[string]func(*Tile, *GameState)
	specialConquestMap map[string]func(*GameState, *Tile, string) (bool, error);
	handleMovementMap map[string]func(string, *Tile, *Tile, *GameState) error

	//conquest for defender
	countDefenseMap map[string]func(*Tile, *Player, *GameState) (int, int, int, error);
	handleReturn func(*Tile, *GameState, int)
	clearTile func(*Tile, *GameState, int);
	specialDefense func(*GameState, *Tile, *Tribe, string) (bool, error);

	// redeployment
	startRedeployment func(*GameState) []PieceStack;
	getStacksOutRedeployment func(*Tile, string) ([]PieceStack, error);
	canBeRedeployedIn func(*Tile, string, *GameState) bool;
	getRedeploymentStack func(string, []PieceStack) []PieceStack
	handleDeploymentOut func(*Tile, string, int, *GameState) error;
	handleDeploymentIn func(*Tile, string, int, *GameState) error;

	// misc
	handleOpponentAction func(string, *Player, *GameState) error;

	// end of turn
	canEndTurn func(*GameState) error;
	countPoints func(*Tile) int;
	countExtrapoints func(*GameState) int;

	// decline
	countRemovablePieces func(*Tile) []PieceStack;
	countRemovableAttackingStacks func(*Player) []PieceStack;
	canGoIntoDecline func(*GameState) bool;
	goIntoDecline func(*GameState);
	prepareRemoval func(*GameState) bool;

	handleEndOfGame func(*GameState);
}

func (t *Tribe) giveInitialStacks() []PieceStack {
    stacks := []PieceStack{}
    for _, f := range(t.giveInitialStacksMap) {
        stacks = AddPieceStacks(stacks, f())
    }
    return stacks
}

func (t *Tribe) checkPresence(tile *Tile, race Race) bool {
    if tile.OwningTribe.Race == race {
	return true
    }
    for _, f := range(t.checkPresenceMap) {
	if f(tile, race) {
	    return true
	}
    }
    return false
}

func (t *Tribe) canTileBeAbandoned(tile *Tile) bool {
    val := t.IsActive && tile.OwningTribe.checkPresence(tile, t.Race)
    // Here, not fully commutative, maybe add a system of priority for sorting making sure different tribes do not collide.
    for _, f := range(t.canTileBeAbandonedMap) {
	val = f(tile)
    }
    return val
}

func (t *Tribe) handleAbandonment(tile *Tile, gs *GameState) {
    t.clearTile(tile, gs, 0)

    for _, f := range(t.handleAbandonmentMap) {
	f(tile, gs)
    }
}

func (t *Tribe) getStacksForConquestTurn(player *Player, gs *GameState) {
    for _, f := range(t.getStacksForConquestTurnMap) {
	f(player, gs)
    }
}

func (t *Tribe) getStacksForConquest(tile *Tile, player *Player) {
    if t.IsActive {
	for _, stack := range tile.PieceStacks {
	    if stack.Type == string(t.Race) {
		// Making sure the action is atomic
		movingStack := []PieceStack{{Type: stack.Type, Amount: stack.Amount - t.Minimum}}
		tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, movingStack)
		player.PieceStacks = AddPieceStacks(player.PieceStacks, movingStack)
	    }
	}
    }
    for _, f := range(t.getStacksForConquestMap) {
	f(tile, player)
    }
}

func (t *Tribe) IsStackValid(s string) bool {
    if s == string(t.Race) && t.IsActive {
	return true
    }

    for _, f := range(t.IsStackValidMap) {
	if f(s) {
	    return true
	}
    }
    return false
}

func (t *Tribe) checkZoneAccess(tile *Tile) error {
    var err error;
    err = nil
    if tile.Biome == Water {
	err = fmt.Errorf("Cannot conquer water!")
    }

    for _, f := range(t.checkZoneAccessMap) {
	err = f(tile, err)
    }
    return err
}

func (t *Tribe) checkAdjacency(tile *Tile, gs *GameState) error {
    err := fmt.Errorf("The tile is not adjacent to current territory")
    if gs.IsTribePresentOnTheBoard(t.Race) {
	for _, neighbour := range tile.AdjacentTiles {
	    if neighbour.CheckPresence() != None && neighbour.OwningTribe.Race == t.Race {
		err = nil
	    }
	}
    } else if !tile.IsEdge {
	err = fmt.Errorf("The tile is not an edge!")
    } else {
	err = nil
    }

    for _, f := range(t.checkAdjacencyMap) {
	err = f(tile, gs, err)
    }
    return err
}

func (t *Tribe) countAttack(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
    if stackType == string(t.Race) {
	return []PieceStack{{Type: string(t.Race), Amount: max(t.Minimum, cost - t.computeDiscount( tile))}}, t.computeGainAttacker(tile), t.computeLossDefender(tile), t.computePawnKill(tile), nil
    }

    for _, f := range(t.countAttackMap) {
	stacks, a, b, c, err := f(tile, cost, stackType)
	if err == nil {
	    return stacks, a, b, c, nil
	}
    }

    return []PieceStack{}, 0, 0, 0, fmt.Errorf("The piecestack was not recognized")
}

func (t *Tribe) computeDiscount(tile *Tile) int {
    total := 0
    for _, f := range(t.computeDiscountMap) {
	total += f(tile)
    }
    return total
}

func (t *Tribe) computeGainAttacker(tile *Tile) int {
    total := 0
    for _, f := range(t.computeGainAttackerMap) {
	total += f(tile)
    }
    return total
}

func (t *Tribe) computeLossDefender(tile *Tile) int {
    total := 0
    for _, f := range(t.computeLossDefenderMap) {
	total += f(tile)
    }
    return total
}

func (t *Tribe) computePawnKill(tile *Tile) int {
    total := 1
    for _, f := range(t.computePawnKillMap) {
	total += f(tile)
    }
    return total
}

func (t *Tribe) countNewTileStacks(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
    for _, f := range(t.countNewTileStacksMap) {
	ps = f(ps, tile, gs)
    }
    return ps
}

func (t *Tribe) postConquest(tile *Tile, gs *GameState) {
    for _, f := range(t.postConquestMap) {
	f(tile, gs)
    }
}
func (t *Tribe) calculateRemainingAttackingStacks(expanses []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
    result := []PieceStack{} // Start with an empty list
    hasDiceBeenUsed := false
    ok := true
    var err error;
    err = nil

    for _, expanse := range expanses {
	found := false
	// Search for a matching type in expanses
	for _, reserve := range t.Owner.PieceStacks {
	    if expanse.Type == reserve.Type {
		found = true
		if reserve.Amount < expanse.Amount && expanse.Type != string(t.Race) {
		    err = fmt.Errorf("Player did not have enough of %s", expanse.Type)
		} else if reserve.Amount + 3 < expanse.Amount {
		    err = fmt.Errorf("Even the dice can't help")
		} else if reserve.Amount < expanse.Amount {
		    diceThrow := RollDice()
		    hasDiceBeenUsed = true
		    if reserve.Amount + diceThrow >= expanse.Amount {
			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("Success: the result of the dice throw was: %d", diceThrow)})
			result = append(result, reserve)
		    } else {
			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("Failure: the result of the dice throw was: %d", diceThrow)})
			ok = false
		    }        
		} else {
		    result = append(result, expanse)
		}
		break
	    }
	}
	if !found {
	    err = fmt.Errorf("Player did not have stack %s", expanse.Type)
	}
	}
	for _, f := range(t.calculateRemainingAttackingStacksMap) {
	    result, hasDiceBeenUsed, ok, err = f(result, hasDiceBeenUsed, ok, err, tile, gs)
	}
	return result, hasDiceBeenUsed, ok, err
}

func (t *Tribe) specialConquest(gs *GameState, tile *Tile, s string) (bool, error) {
    for _, f := range(t.specialConquestMap) {
	ok, err := f(gs, tile, s)
	if ok {
	    return ok, err
	}
    }
    return false, nil
}

func (t *Tribe) handleMovement(s string, t1, t2 *Tile, gs *GameState) error {
    for _, f := range(t.handleMovementMap) {
	err := f(s, t1, t2, gs)
	if err == nil {
	    return nil
	}
    }
    return fmt.Errorf("Invalid Opponent Action!")
}

func (t *Tribe) countDefense(tile *Tile, player *Player, gs *GameState) (int, int, int, error) {
    price, b, c, error := tile.countDefense(gs)
    if error != nil {
	return price, b, c, error
    }
    for _, stack := range tile.PieceStacks {
	if stack.Type == string(t.Race) {
	    price += stack.Amount
	}
    }
    for _, f := range(t.countDefenseMap) {
	newprice, newb, newc, err := f(tile, player, gs)
	if err != nil {
	    return 0, 0, 0, err
	}

	price += newprice
	b += newb
	c += newc
    }
    return price, b, c, nil
}
