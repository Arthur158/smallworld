package gamestate

import (
    "fmt"
    "sort"
)

type Race string
type Trait string

type Tribe struct {
    Owner            *Player
    Race             Race
    Trait            Trait
    IsActive         bool
    State            map[string]interface{}
    AdditionalPowers []Trait
    Minimum          int

    giveInitialStacksMap map[string]func() []PieceStack
    checkPresenceMap     map[string]func(*Tile, Race) bool

    //abandonment
    canTileBeAbandonedMap map[string]func(*Tile) bool
    handleAbandonmentMap  map[string]func(*Tile, *GameState)

    // receive for conquest
    getStacksForConquestTurnMap map[string]func(*Player, *GameState)
    getStacksForConquestMap     map[string]func(*Tile, *Player)

    // conquest checks
    IsStackValidMap    map[string]func(string) bool
    checkZoneAccessMap map[string]func(*Tile, error) error
    checkAdjacencyMap  map[string]func(*Tile, *GameState, error) error

    // conquest for attacker
    countAttackMap                       map[string]func(*Tile, int, string) ([]PieceStack, int, int, int, error)
    computeDiscountMap                   map[string]func(*Tile) int
    computeGainAttackerMap               map[string]func(*Tile) int
    computeLossDefenderMap               map[string]func(*Tile) int
    computePawnKillMap                   map[string]func(*Tile) int
    countNewTileStacksMap                map[string]func([]PieceStack, *Tile, *GameState) []PieceStack
    calculateRemainingAttackingStacksMap map[string]func([]PieceStack, bool, bool, error, *Tile, *GameState) ([]PieceStack, bool, bool, error)
    postConquestMap                      map[string]func(*Tile, *GameState)
    specialConquestMap                   map[string]func(*GameState, *Tile, string) (bool, error)
    handleMovementMap                    map[string]func(string, *Tile, *Tile, *GameState) error

    //conquest for defender
    countDefenseMap   map[string]func(*Tile, *Player, *GameState) (int, int, int, error)
    handleReturnMap   map[string]func(*Tile, *GameState, int)
    clearTileMap      map[string]func(*Tile, *GameState, int)
    specialDefenseMap map[string]func(*GameState, *Tile, *Tribe, string) (bool, error)

    // redeployment
    startRedeploymentMap        map[string]func(*GameState) []PieceStack
    startRedeploymentTileMap    map[string]func(*Tile, *GameState)
    getStacksOutRedeploymentMap map[string]func(*Tile, string) ([]PieceStack, error)
    canBeRedeployedInMap        map[string]func(bool, *Tile, string, *GameState) bool
    canBeRedeployedOutMap       map[string]func(bool, *Tile, string) bool
    getRedeploymentStackMap     map[string]func(string, []PieceStack) []PieceStack
    handleDeploymentOutMap      map[string]func(*Tile, string, *GameState) error
    handleDeploymentInMap       map[string]func(*Tile, string, int, *GameState) error

    // misc
    handleOpponentActionMap map[string]func(string, *Player, *GameState) error
    handleEntryActionMap    map[string]func(int, string, *GameState) error

    // end of turn
    canEndTurnMap       map[string]func(*GameState) error
    countPointsMap      map[string]func(*Tile) int
    countExtrapointsMap map[string]func(*GameState) int

    // decline
    countRemovablePiecesMap          map[string]func([]PieceStack, *Tile) []PieceStack
    countRemovableAttackingStacksMap map[string]func([]PieceStack, *Player) []PieceStack
    canGoIntoDeclineMap              map[string]func(bool, *GameState) bool
    goIntoDeclineMap                 map[string]func(*GameState)
    prepareRemovalMap                map[string]func(*GameState) bool
    alternativeDeclineMap            map[string]func(*GameState) bool

    handleEndOfGameMap map[string]func(*GameState)
}

func (t *Tribe) giveInitialStacks() []PieceStack {
    stacks := []PieceStack{}
    for _, key := range sortedHookKeys(t.giveInitialStacksMap) {
        f := t.giveInitialStacksMap[key]
        if f == nil { continue }
        stacks = AddPieceStacks(stacks, f())
    }
    return stacks
}

func (t *Tribe) checkPresence(tile *Tile, race Race) bool {
    if t == nil || tile == nil || tile.OwningTribe == nil { return false }
    if tile.OwningTribe.Race == race {
        return true
    }
    for _, key := range sortedHookKeys(t.checkPresenceMap) {
        f := t.checkPresenceMap[key]
        if f == nil { continue }
        if f(tile, race) {
            return true
        }
    }
    return false
}

func (t *Tribe) canTileBeAbandoned(tile *Tile) bool {
    val := t.IsActive && tile.OwningTribe.checkPresence(tile, t.Race)
    // Here, not fully commutative, maybe add a system of priority for sorting making sure different tribes do not collide.
    for _, key := range sortedHookKeys(t.canTileBeAbandonedMap) {
        f := t.canTileBeAbandonedMap[key]
        if f == nil { continue }
        val = f(tile)
    }
    return val
}

func (t *Tribe) handleAbandonment(tile *Tile, gs *GameState) {
    tile.handleAfterConquest(gs, nil)
    t.clearTile(tile, gs, 0)

    for _, key := range sortedHookKeys(t.handleAbandonmentMap) {
        f := t.handleAbandonmentMap[key]
        if f == nil { continue }
        f(tile, gs)
    }
}

func (t *Tribe) getStacksForConquestTurn(player *Player, gs *GameState) {
    for _, key := range sortedHookKeys(t.getStacksForConquestTurnMap) {
        f := t.getStacksForConquestTurnMap[key]
        if f == nil { continue }
        f(player, gs)
    }
}

func (t *Tribe) getStacksForConquest(tile *Tile, player *Player) {
    for _, key := range sortedHookKeys(t.getStacksForConquestMap) {
        f := t.getStacksForConquestMap[key]
        if f == nil { continue }
        f(tile, player)
    }
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
}

func (t *Tribe) IsStackValid(s string) bool {
    if s == string(t.Race) && t.IsActive {
        return true
    }

    for _, key := range sortedHookKeys(t.IsStackValidMap) {
        f := t.IsStackValidMap[key]
        if f == nil { continue }
        if f(s) {
            return true
        }
    }
    return false
}

func (t *Tribe) checkZoneAccess(tile *Tile) error {
    var err error
    err = nil
    if tile.Biome == Water {
        err = fmt.Errorf("Cannot conquer water!")
    }
    if tile.Biome == AbyssalChasm {
        err = fmt.Errorf("Cannot conquer abysmal chasm!")
    }

    for _, key := range sortedHookKeys(t.checkZoneAccessMap) {
        f := t.checkZoneAccessMap[key]
        if f == nil { continue }
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

    for _, key := range sortedHookKeys(t.checkAdjacencyMap) {
        f := t.checkAdjacencyMap[key]
        if f == nil { continue }
        err = f(tile, gs, err)
    }
    return err
}

func (t *Tribe) countAttack(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
    if stackType == string(t.Race) {
        return []PieceStack{{Type: string(t.Race), Amount: max(t.Minimum, cost-t.computeDiscount(tile))}}, t.computeGainAttacker(tile), t.computeLossDefender(tile), t.computePawnKill(tile), nil
    }

    err := fmt.Errorf("The piecestack was not recognized")
    for _, key := range sortedHookKeys(t.countAttackMap) {
        f := t.countAttackMap[key]
        if f == nil { continue }
        stacks, a, b, c, err2 := f(tile, cost, stackType)
        err = err2
        if err == nil {
            return stacks, a, b, c, nil
        }
    }

    return []PieceStack{}, 0, 0, 0, err
}

func (t *Tribe) computeDiscount(tile *Tile) int {
    total := 0
    for _, key := range sortedHookKeys(t.computeDiscountMap) {
        f := t.computeDiscountMap[key]
        if f == nil { continue }
        total += f(tile)
    }
    return total
}

func (t *Tribe) computeGainAttacker(tile *Tile) int {
    total := 0
    for _, key := range sortedHookKeys(t.computeGainAttackerMap) {
        f := t.computeGainAttackerMap[key]
        if f == nil { continue }
        total += f(tile)
    }
    return total
}

func (t *Tribe) computeLossDefender(tile *Tile) int {
    total := 0
    for _, key := range sortedHookKeys(t.computeLossDefenderMap) {
        f := t.computeLossDefenderMap[key]
        if f == nil { continue }
        total += f(tile)
    }
    return total
}

func (t *Tribe) computePawnKill(tile *Tile) int {
    total := 1
    for _, key := range sortedHookKeys(t.computePawnKillMap) {
        f := t.computePawnKillMap[key]
        if f == nil { continue }
        total += f(tile)
    }
    return total
}

func (t *Tribe) countNewTileStacks(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
    for _, key := range sortedHookKeys(t.countNewTileStacksMap) {
        f := t.countNewTileStacksMap[key]
        if f == nil { continue }
        ps = f(ps, tile, gs)
    }
    return ps
}

func (t *Tribe) postConquest(tile *Tile, gs *GameState) {
    for _, key := range sortedHookKeys(t.postConquestMap) {
        f := t.postConquestMap[key]
        if f == nil { continue }
        f(tile, gs)
    }
}
func (t *Tribe) calculateRemainingAttackingStacks(expanses []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
    result := []PieceStack{} // Start with an empty list
    hasDiceBeenUsed := false
    ok := true
    var err error
    err = nil

    for _, expanse := range expanses {
        found := false
        // Search for a matching type in expanses
        for _, reserve := range t.Owner.PieceStacks {
            if expanse.Type == reserve.Type {
                found = true
                if reserve.Amount < expanse.Amount && expanse.Type != string(t.Race) {
                    err = fmt.Errorf("Player did not have enough of %s", expanse.Type)
                } else if reserve.Amount+3 < expanse.Amount {
                    err = fmt.Errorf("Even the dice can't help")
                } else if reserve.Amount < expanse.Amount {
                    diceThrow := gs.rollDice()
                    hasDiceBeenUsed = true
                    if reserve.Amount+diceThrow >= expanse.Amount {
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
    for _, key := range sortedHookKeys(t.calculateRemainingAttackingStacksMap) {
        f := t.calculateRemainingAttackingStacksMap[key]
        if f == nil { continue }
        result, hasDiceBeenUsed, ok, err = f(result, hasDiceBeenUsed, ok, err, tile, gs)
    }
    return result, hasDiceBeenUsed, ok, err
}

func (t *Tribe) specialConquest(gs *GameState, tile *Tile, s string) (bool, error) {
    for _, key := range sortedHookKeys(t.specialConquestMap) {
        f := t.specialConquestMap[key]
        if f == nil { continue }
        ok, err := f(gs, tile, s)
        if ok {
            return ok, err
        }
    }
    return false, nil
}

func (t *Tribe) handleMovement(s string, t1, t2 *Tile, gs *GameState) error {
    for _, key := range sortedHookKeys(t.handleMovementMap) {
        f := t.handleMovementMap[key]
        if f == nil { continue }
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
    for _, key := range sortedHookKeys(t.countDefenseMap) {
        f := t.countDefenseMap[key]
        if f == nil { continue }
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

func (t *Tribe) handleReturn(tile *Tile, gs *GameState, cost int) {
    t.clearTile(tile, gs, cost)
    for _, key := range sortedHookKeys(t.handleReturnMap) {
        f := t.handleReturnMap[key]
        if f == nil { continue }
        f(tile, gs, cost)
    }
}

func (t *Tribe) clearTile(tile *Tile, gs *GameState, cost int) {
    for i := len(tile.PieceStacks)-1; i >= 0; i-- {
 stack := tile.PieceStacks[i]
        if stack.Type == string(t.Race) {
            tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
            stack.Amount = max(0, stack.Amount-cost)
            if tile.CheckPresence() != None {
                t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{stack})
            }
            if tile.OwningTribe == t {
                tile.OwningTribe = nil
            }
        }
    }
    for _, key := range sortedHookKeys(t.clearTileMap) {
        f := t.clearTileMap[key]
        if f == nil { continue }
        f(tile, gs, cost)
    }
}

func (t *Tribe) specialDefense(gs *GameState, tile *Tile, attackingTribe *Tribe, s string) (bool, error) {
    for _, key := range sortedHookKeys(t.specialDefenseMap) {
        f := t.specialDefenseMap[key]
        if f == nil { continue }
        ok, err := f(gs, tile, attackingTribe, s)
        if ok {
            return ok, err
        }
    }
    return false, nil
}

func (t *Tribe) startRedeployment(gs *GameState) []PieceStack {
    stacks := []PieceStack{}
    for _, key := range sortedHookKeys(t.startRedeploymentMap) {
        f := t.startRedeploymentMap[key]
        if f == nil { continue }
        stacks = AddPieceStacks(stacks, f(gs))
    }

    for _, tile := range gs.TileList {
        if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
            t.startRedeploymentTile(tile, gs)
        }
    }
    return stacks
}

func (t *Tribe) startRedeploymentTile(tile *Tile, gs *GameState) {
    if tile.Biome == River {
        t.clearTile(tile, gs, 0)
    }

    for _, key := range sortedHookKeys(t.startRedeploymentTileMap) {
        f := t.startRedeploymentTileMap[key]
        if f == nil { continue }
        f(tile, gs)
    }
}

func (t *Tribe) getStacksOutRedeployment(tile *Tile, stackType string) ([]PieceStack, error) {

    if !t.canBeRedeployedOut(tile, stackType) {
        return nil, fmt.Errorf("cannot redeploy out")
    }

    err := fmt.Errorf("There is no such stack")
    ps := []PieceStack{}
    if stackType == string(t.Race) {
        for _, stack := range tile.PieceStacks {
            if stack.Type == stackType {
                if stack.Amount == t.Minimum {
                    err = fmt.Errorf("cannot take off single tribe")
                } else {
                    ps = []PieceStack{{Type: stackType, Amount: 1}}
                    err = nil
                }
            }
        }
    }
    for _, key := range sortedHookKeys(t.getStacksOutRedeploymentMap) {
        f := t.getStacksOutRedeploymentMap[key]
        if f == nil { continue }
        ps, err := f(tile, stackType)
        if err == nil {
            return ps, nil
        }
    }

    return ps, err
}

func (t *Tribe) canBeRedeployedOut(tile *Tile, s string) bool {
    res := s == string(t.Race)
    for _, key := range sortedHookKeys(t.canBeRedeployedOutMap) {
        f := t.canBeRedeployedOutMap[key]
        if f == nil { continue }
        res = f(res, tile, s)
    }
    return res
}

func (t *Tribe) canBeRedeployedIn(tile *Tile, s string, gs *GameState) bool {
    res := s == string(t.Race) && tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race)
    for _, key := range sortedHookKeys(t.canBeRedeployedInMap) {
        f := t.canBeRedeployedInMap[key]
        if f == nil { continue }
        res = f(res, tile, s, gs)
    }
    return res
}

func (t *Tribe) handleDeploymentOut(tile *Tile, stackType string, gs *GameState) error {

    for _, key := range sortedHookKeys(t.handleDeploymentOutMap) {
        f := t.handleDeploymentOutMap[key]
        if f == nil { continue }
        err := f(tile, stackType, gs)
        if err == nil {
            return nil
        }
    }

    if tile.CheckPresence() == None {
        return fmt.Errorf("This tile does not contain any tribe!")
    }
    stacks, err := tile.OwningTribe.getStacksOutRedeployment(tile, stackType)
    if err != nil {
        return err
    }

    player := t.Owner

    ok := false
    tile.PieceStacks, ok = SubtractPieceStacks(tile.PieceStacks, stacks)
    if !ok {
        return fmt.Errorf("Could not substract the stacks")
    }

    player.PieceStacks = AddPieceStacks(player.PieceStacks, stacks)

    return nil
}

func (t *Tribe) handleDeploymentIn(tile *Tile, stackType string, i int, gs *GameState) error {
    for _, key := range sortedHookKeys(t.handleDeploymentInMap) {
        f := t.handleDeploymentInMap[key]
        if f == nil { continue }
        err := f(tile, stackType, i, gs)
        if err == nil {
            return nil
        }
    }

    if tile.CheckPresence() == None || !tile.OwningTribe.checkPresence(tile, t.Race) {
        return fmt.Errorf("This tile does not belong to you")
    }

    if !t.canBeRedeployedIn(tile, stackType, gs) {
        return fmt.Errorf("Cannot redeploy here")
    }

    player := t.Owner

    movingStack := t.getRedeploymentStack(stackType, player.PieceStacks)

    newStacks, ok := SubtractPieceStacks(player.PieceStacks, movingStack)
    if !ok {
        return fmt.Errorf("Cannot redeploy pieces you don't have")
    }
    player.PieceStacks = newStacks

    tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)

    return nil
}

func (t *Tribe) getRedeploymentStack(s string, ps []PieceStack) []PieceStack {
    for _, key := range sortedHookKeys(t.getRedeploymentStackMap) {
        f := t.getRedeploymentStackMap[key]
        if f == nil { continue }
        stacks := f(s, ps)
        if len(stacks) > 0 {
            return stacks
        }
    }
    return []PieceStack{{Type: s, Amount: 1, Tribe: t}}
}

func (t *Tribe) handleOpponentAction(s string, p *Player, gs *GameState) error {
    for _, key := range sortedHookKeys(t.handleOpponentActionMap) {
        f := t.handleOpponentActionMap[key]
        if f == nil { continue }
        err := f(s, p, gs)
        if err == nil {
            return nil
        }
    }
    return fmt.Errorf("Invalid opponent action!")
}

func (t *Tribe) handleEntryAction(i int, s string, gs *GameState) error {
    for _, key := range sortedHookKeys(t.handleEntryActionMap) {
        f := t.handleEntryActionMap[key]
        if f == nil { continue }
        err := f(i, s, gs)
        if err == nil {
            return nil
        }
    }
    return fmt.Errorf("Invalid opponent action!")
}

func (t *Tribe) canEndTurn(gs *GameState) error {
    for _, key := range sortedHookKeys(t.canEndTurnMap) {
        f := t.canEndTurnMap[key]
        if f == nil { continue }
        err := f(gs)
        if err != nil {
            return err
        }
    }
    return nil
}
func (t *Tribe) countPoints(tile *Tile) int {
    total := tile.countPoints()
    for _, key := range sortedHookKeys(t.countPointsMap) {
        f := t.countPointsMap[key]
        if f == nil { continue }
        total += f(tile)
    }
    return max(0, total)
}

func (t *Tribe) countExtrapoints(gs *GameState) int {
    count := 0
    for _, key := range sortedHookKeys(t.countExtrapointsMap) {
        f := t.countExtrapointsMap[key]
        if f == nil { continue }
        count += f(gs)
    }
    return count
}

func (t *Tribe) countRemovablePieces(tile *Tile) []PieceStack {
    amount := 1
    for _, stack := range tile.PieceStacks {
        if stack.Type == string(t.Race) {
            amount = stack.Amount
        }
    }
    stacks := []PieceStack{{Type: string(tile.OwningTribe.Race), Amount: amount - 1}}

    for _, key := range sortedHookKeys(t.countRemovablePiecesMap) {
        f := t.countRemovablePiecesMap[key]
        if f == nil { continue }
        stacks = f(stacks, tile)
    }

    return stacks
}

func (t *Tribe) countRemovableAttackingStacks(player *Player) []PieceStack {
    newstacks := []PieceStack{}
    for _, stack := range player.PieceStacks {
        if stack.Type == string(t.Race) {
            newstacks = append(newstacks, stack)
        }
    }

    for _, key := range sortedHookKeys(t.countRemovableAttackingStacksMap) {
        f := t.countRemovableAttackingStacksMap[key]
        if f == nil { continue }
        newstacks = f(newstacks, player)
    }
    return newstacks
}

func (t *Tribe) canGoIntoDecline(gs *GameState) bool {
    res := gs.TurnInfo.Phase == DeclineChoice
    for _, key := range sortedHookKeys(t.canGoIntoDeclineMap) {
        f := t.canGoIntoDeclineMap[key]
        if f == nil { continue }
        res = f(res, gs)
    }
    return res
}

func (t *Tribe) goIntoDecline(gs *GameState) {
    for _, key := range sortedHookKeys(t.alternativeDeclineMap) {
        f := t.alternativeDeclineMap[key]
        if f == nil { continue }
        ok := f(gs)
        if ok {
            return
        }
    }
    player := t.Owner

    for i := len(player.PassiveTribes)-1; i >= 0; i-- {
 tribe := player.PassiveTribes[i]
        if tribe.prepareRemoval(gs) {
            player.PassiveTribes = append(player.PassiveTribes[:i], player.PassiveTribes[i+1:]...)
            player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, tribe.countRemovableAttackingStacks(player))
        }
    }

    for _, tile := range gs.TileList {
        if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
            tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, tile.OwningTribe.countRemovablePieces(tile))
        }
    }

    player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, player.ActiveTribe.countRemovableAttackingStacks(player))
    t.IsActive = false

    player.PassiveTribes = append(player.PassiveTribes, player.ActiveTribe)
    player.ActiveTribe = nil

    for _, key := range sortedHookKeys(t.goIntoDeclineMap) {
        f := t.goIntoDeclineMap[key]
        if f == nil { continue }
        f(gs)
    }
}

func (t *Tribe) prepareRemoval(gs *GameState) bool {
    for _, key := range sortedHookKeys(t.prepareRemovalMap) {
        f := t.prepareRemovalMap[key]
        if f == nil { continue }
        return f(gs)
    }
    for _, tile := range gs.TileList {
        if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
            tile.handleAfterConquest(gs, nil)
            t.clearTile(tile, gs, 0)
        }
    }
    player := t.Owner
    player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, t.countRemovableAttackingStacks(player))
    return true
}

func (t *Tribe) handleEndOfGame(gs *GameState) {
    for _, key := range sortedHookKeys(t.handleEndOfGameMap) {
        f := t.handleEndOfGameMap[key]
        if f == nil { continue }
        f(gs)
    }
}

// Hook composition must be stable when race and trait both define a rule.
func sortedHookKeys[V any](m map[string]V) []string {
 keys:=make([]string,0,len(m));for key:=range m{keys=append(keys,key)};sort.Strings(keys);return keys
}
