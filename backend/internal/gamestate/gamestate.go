package gamestate

import (
    "fmt"
    "math/rand"
    "time"
)

type GameState struct {
    Players            []*Player
    TribeList          []*TribeEntry
    TileList           map[string]*Tile
    TurnInfo           *TurnInfo
    RealTurninfo       *TurnInfo
    Messages           []Message
    ModifierPoints     map[string]func(int, *Player) int
    ModifierTurnsAfter []TurninfoEntry
    ModifierAfterPick  map[string]func(int, *TribeEntry)
    Powers             map[string]*Power
}

func New(playerNames []string, mapName string, raceKeys []string, traitKeys []string, powerKeys []string) (*GameState, error) {
    // Create a list of initialized players
    gs := &GameState{}
    gs.Players = make([]*Player, len(playerNames))
    for i, name := range playerNames {
        gs.Players[i] = &Player{
            Name:           name,
            Index:          i,
            ActiveTribe:    nil,
            PassiveTribes:  []*Tribe{}, // Initialize as empty slice
            CoinPile:       5,
            PieceStacks:    []PieceStack{},
            PointsEachTurn: []int{5},
        }
    }

    gs.TurnInfo = &TurnInfo{
        TurnIndex:   1,
        PlayerIndex: 0,
        Phase:       TribeChoice,
    }
    gs.RealTurninfo = nil
    gs.ModifierTurnsAfter = []TurninfoEntry{}
    gs.ModifierAfterPick = make(map[string]func(int, *TribeEntry))

    var err error
    gs.TribeList, err = createTribeList(raceKeys, traitKeys)
    if err != nil {
        return nil, err
    }

    gs.ModifierPoints = make(map[string]func(int, *Player) int)
    function, ok := MapRegistry[mapName]
    if !ok {
        return nil, fmt.Errorf("map not found")
    }

    gs.TileList = function(gs)

    // Testing Powers
    testingPowers := true
    if testingPowers {
        if len(playerNames)+2 < len(powerKeys) {
            rand.Seed(time.Now().UnixNano())
            rand.Shuffle(len(powerKeys), func(i, j int) {
                powerKeys[i], powerKeys[j] = powerKeys[j], powerKeys[i]
            })
            powerKeys = powerKeys[:len(playerNames)+2] // Adjust number of powers according to the number of players
        }
        gs.InitializePowers(powerKeys)
    }

    if err != nil {
        return nil, fmt.Errorf("failed to create list of tribe entries", err)
    }

    gs.Powers = make(map[string]*Power)

    return gs, nil
}

func (gs *GameState) HandleTribeChoice(chooserIndex int, entryIndex int) error {
    if gs.TurnInfo.PlayerIndex != chooserIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    if gs.TurnInfo.Phase != TribeChoice {
        return fmt.Errorf("The player is not supposed to pick a new tribe!")
    }

    if entryIndex > 5 || entryIndex < 0 {
        return fmt.Errorf("Invalid entry index")
    }

    if entryIndex > len(gs.TribeList)-1 {
        return fmt.Errorf("not enough tribe entries")
    }

    chooser := gs.Players[chooserIndex]

    if chooser.CoinPile < entryIndex {
        return fmt.Errorf("The player only has %d coins, but they need %d for picking this tribe", chooser.CoinPile, entryIndex)
    }

    entry := gs.TribeList[entryIndex]

    tempActiveTribe, err := CreateTribe(entry.Race, entry.Trait)
    if err != nil {
        return fmt.Errorf("Could not create tribe:", err)
    }
    chooser.ActiveTribe = tempActiveTribe
    chooser.ActiveTribe.Owner = chooser
    chooser.CoinPile += entry.CoinPile - entryIndex
    chooser.PointsEachTurn[len(chooser.PointsEachTurn)-1] += entry.CoinPile - entryIndex
    gs.TribeList = append(gs.TribeList[:entryIndex], gs.TribeList[entryIndex+1:]...)
    for _, tribeEntry := range gs.TribeList[:entryIndex] {
        tribeEntry.CoinPile += 1
    }

    chooser.PieceStacks = AddPieceStacks(chooser.PieceStacks, []PieceStack{{Type: string(entry.Race), Amount: entry.PiecePile}})
    chooser.PieceStacks = AddPieceStacks(chooser.PieceStacks, chooser.ActiveTribe.giveInitialStacks())
    gs.GetPieceStackForConquest(chooser)

    for _, f := range gs.ModifierAfterPick {
        f(entryIndex, entry)
    }

    gs.TurnInfo.Phase = TileAbandonment

    return nil
}

func (gs *GameState) HandleEntryAction(playerIndex int, entryIndex int, stackType string) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    player := gs.Players[playerIndex]

    if !DoesPlayerHaveStack(stackType, player) {
        return fmt.Errorf("The stack is invalid for this player!")
    }

    tribe, err := player.getTribe(stackType)
    if err != nil {
        return err
    }

    return tribe.handleEntryAction(entryIndex, stackType, gs)
}

func (gs *GameState) HandleAbandonment(playerIndex int, tileId string, stackType string) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    if gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice {
        return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
    }

    player := gs.Players[playerIndex]

    tribe, err := player.getTribe(stackType)
    if err != nil {
        return err
    }

    tile, ok := gs.TileList[tileId]
    if !ok {
        return fmt.Errorf("No tile with this id!")
    }

    if tile.CheckPresence() == None || !tile.OwningTribe.checkPresence(tile, tribe.Race) {
        return fmt.Errorf("This tile does not belong to the player!")
    }

    if !tile.OwningTribe.canTileBeAbandoned(tile) {
        return fmt.Errorf("Stack cannot be removed!")
    }

    tile.OwningTribe.handleAbandonment(tile, gs)

    gs.TurnInfo.Phase = TileAbandonment

    return nil

}

func (gs *GameState) HandleConquest(tileId string, attackerIndex int, attackingStackType string) error {
    if gs.TurnInfo.PlayerIndex != attackerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    if gs.TurnInfo.Phase != Conquest && gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice {
        return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
    }

    attacker := gs.Players[attackerIndex]

    if !DoesPlayerHaveStack(attackingStackType, attacker) {
        return fmt.Errorf("The stack is invalid for this player!")
    }

    tile, ok := gs.TileList[tileId]
    if !ok {
        return fmt.Errorf("No tile with this id!")
    }

    attackingTribe, err := attacker.getTribe(attackingStackType)
    if err != nil {
        for _, power := range gs.Powers {
            if power.Owner == attacker && power.HandleConquest != nil {
                ok, err := power.HandleConquest(gs, tile, attackingStackType)
                if ok {
                    return err
                }
            }
        }
        return err
    }

    if ok, err := tile.specialDefense(gs, attackingTribe, attackingStackType); ok {
        return err
    }

    if tile.CheckPresence() != None {
        ok, err := tile.OwningTribe.specialDefense(gs, tile, attackingTribe, attackingStackType)
        if ok {
            return err
        }
    }

    ok, err = attackingTribe.specialConquest(gs, tile, attackingStackType)
    if ok {
        return err
    }

    if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, attackingTribe.Race) {
        return fmt.Errorf("This tile already belongs to the tribe!")
    }

    if err := attackingTribe.checkZoneAccess(tile); err != nil {
        return err
    }
    if err := attackingTribe.checkAdjacency(tile, gs); err != nil {
        return err
    }

    tileCost, moneyGainDefender, moneyLossAttacker := 0, 0, 0
    if tile.CheckPresence() != None {
        tileCost, moneyGainDefender, moneyLossAttacker, err = tile.OwningTribe.countDefense(tile, attacker, gs)
    } else {
        tileCost, moneyGainDefender, moneyLossAttacker, err = tile.countDefense(gs)
    }

    if err != nil {
        return err
    }

    // counts the cost for the attacker
    attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill, err := attackingTribe.countAttack(tile, tileCost, attackingStackType)
    if err != nil {
        return err
    }

    newStacks, hasDiceBeenUsed, ok, err := attackingTribe.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
    if err != nil {
        return err
    }
    if !ok {
        return gs.HandleStartRedeployment(attackerIndex)
    }

    // Enact changes
    tile.handleAfterConquest(gs, attackingTribe)
    attackingTribe.postConquest(tile, gs)
    if tile.CheckPresence() != None {
        tile.OwningTribe.Owner.CoinPile += moneyGainDefender - moneyLossDefender
        tile.OwningTribe.handleReturn(tile, gs, pawnKill)
    }

    attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, newStacks)
    tile.PieceStacks = AddPieceStacks(tile.PieceStacks, attackingTribe.countNewTileStacks(newStacks, tile, gs))

    attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
    // attacker.PointsEachTurn[len(attacker.PointsEachTurn) - 1] += moneyGainDefender - moneyLossDefender
    tile.OwningTribe = attackingTribe

    if hasDiceBeenUsed || len(attacker.PieceStacks) == 0 {
        return gs.HandleStartRedeployment(attackerIndex)
    } else {
        gs.TurnInfo.Phase = Conquest
    }

    return nil
}

func (gs *GameState) HandleStartRedeployment(playerIndex int) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    if gs.TurnInfo.Phase != Conquest && gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice {
        return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
    }

    player := gs.Players[playerIndex]
    player.StartRedeployment(gs)

    gs.TurnInfo.Phase = Redeployment

    return nil

}

func (gs *GameState) HandleRedeploymentOut(playerIndex int, tileId string, stackType string) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    if gs.TurnInfo.Phase != Redeployment {
        return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
    }

    player := gs.Players[playerIndex]
    tribe, err := player.getTribe(stackType)
    if err != nil {
        return err
    }

    tile, ok := gs.TileList[tileId]
    if !ok {
        return fmt.Errorf("No tile with this id!")
    }

    if tile.CheckPresence() != None && !tile.OwningTribe.checkPresence(tile, tribe.Race) {
        return fmt.Errorf("This tile does not belong to the player!")
    }

    return tribe.handleDeploymentOut(tile, stackType, gs)
}

func (gs *GameState) HandleRedeploymentIn(playerIndex int, tileId string, stackType string, amount int) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    if gs.TurnInfo.Phase != Redeployment {
        return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
    }

    tile, ok := gs.TileList[tileId]
    if !ok {
        return fmt.Errorf("No tile with this id!")
    }

    player := gs.Players[playerIndex]
    tribe, err := player.getTribe(stackType)
    if err != nil {
        for _, power := range gs.Powers {
            if power.Owner == player && power.HandleRedeploymentIn != nil {
                err2 := power.HandleRedeploymentIn(tile, stackType, gs)
                if err2 == nil {
                    return nil
                }
            }
        }
        return err
    }

    return tribe.handleDeploymentIn(tile, stackType, amount, gs)

}

func (gs *GameState) HandleFinishTurn(playerIndex int) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    if gs.TurnInfo.Phase != Redeployment && gs.TurnInfo.Phase != Conquest && gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice {
        return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
    }

    if gs.TurnInfo.Phase != Redeployment {
        if err := gs.HandleStartRedeployment(playerIndex); err != nil {
            return fmt.Errorf("Error in redeployment phase:", err)
        }
    }

    player := gs.Players[playerIndex]

    if player.ActiveTribe != nil {
        if err := player.ActiveTribe.canEndTurn(gs); err != nil {
            return err
        }
    }

    gs.handleNextPlayerTurn()

    return nil
}

func (gs *GameState) HandleOpponentAction(playerIndex int, opponentIndex int, stackType string) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    player := gs.Players[playerIndex]
    opponent := gs.Players[opponentIndex]

    if !DoesPlayerHaveStack(stackType, player) {
        return fmt.Errorf("The stack is invalid for this player!")
    }

    playerTribe, err := player.getTribe(stackType)
    if err != nil {
        return err
    }
    return playerTribe.handleOpponentAction(stackType, opponent, gs)
}

func (gs *GameState) HandleMovement(playerIndex int, tileFromId string, tileToId string, stackType string) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    tileTo, ok := gs.TileList[tileToId]
    if !ok {
        return fmt.Errorf("No tile with this id!")
    }

    tileFrom, ok := gs.TileList[tileFromId]
    if !ok {
        return fmt.Errorf("No tile with this id!")
    }

    player := gs.Players[playerIndex]

    playerTribe, err := player.getTribe(stackType)
    if err != nil {
        for _, power := range gs.Powers {
            if power.Owner == player && power.HandleMovement != nil {
                err2 := power.HandleMovement(stackType, tileFrom, tileTo, gs)
                err = err2
                if err2 == nil {
                    return nil
                }
            }
        }
        return err
    }

    return playerTribe.handleMovement(stackType, tileFrom, tileTo, gs)
}

func (gs *GameState) HandleDecline(playerIndex int) error {
    if gs.TurnInfo.PlayerIndex != playerIndex {
        return fmt.Errorf("It is not this player's turn!")
    }

    player := gs.Players[playerIndex]

    if player.ActiveTribe == nil {
        return fmt.Errorf("The player does not have an active tribe!")
    }

    if err := player.ActiveTribe.canEndTurn(gs); err != nil {
        return err
    }

    if !player.ActiveTribe.canGoIntoDecline(gs) {
        return fmt.Errorf("The tribe cannot go in decline at this moment")
    }

    player.ActiveTribe.goIntoDecline(gs)

    gs.handleNextPlayerTurn()

    return nil
}

func (gs *GameState) handleNextPlayerTurn() {
    if gs.RealTurninfo != nil {
        gs.TurnInfo = gs.RealTurninfo
        gs.RealTurninfo = nil
    }

    for i := len(gs.ModifierTurnsAfter) - 1; i >= 0; i-- {
        if gs.ModifierTurnsAfter[i].player == gs.TurnInfo.PlayerIndex {
            if gs.ModifierTurnsAfter[i].actionBefore != nil {
                gs.ModifierTurnsAfter[i].actionBefore(gs)
            }
            if gs.ModifierTurnsAfter[i].TurnInfo != nil {
                gs.RealTurninfo = gs.TurnInfo
                gs.TurnInfo = gs.ModifierTurnsAfter[i].TurnInfo
                gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter[:i], gs.ModifierTurnsAfter[i+1:]...)
                return
            }
            gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter[:i], gs.ModifierTurnsAfter[i+1:]...)
        }
    }

    player := gs.Players[gs.TurnInfo.PlayerIndex]
    player.CoinPile += gs.countPoints(player)
    player.PointsEachTurn = append(player.PointsEachTurn, player.CoinPile)

    if player.ActiveTribe != nil {
        gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf(
            "%s made %d points this turn",
            player.Name,
            player.PointsEachTurn[len(player.PointsEachTurn)-1]-player.PointsEachTurn[len(player.PointsEachTurn)-2],
        )})
    } else {
        gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf(
            "%s went into decline and made %d points this turn",
            player.Name,
            player.PointsEachTurn[len(player.PointsEachTurn)-1]-player.PointsEachTurn[len(player.PointsEachTurn)-2],
        )})
    }

    if gs.TurnInfo.PlayerIndex == len(gs.Players)-1 {
        if gs.TurnInfo.TurnIndex == 9 {
            for _, player := range gs.Players {
                player.ActiveTribe.handleEndOfGame(gs)
                for _, tribe := range player.PassiveTribes {
                    tribe.handleEndOfGame(gs)
                }
            }
            gs.TurnInfo.Phase = GameFinished
        } else {
            gs.TurnInfo.TurnIndex++
            gs.TurnInfo.PlayerIndex = 0
            gs.ChoosePlayerStart()
        }
    } else {
        gs.TurnInfo.PlayerIndex++
        gs.ChoosePlayerStart()
    }
}

func (gs *GameState) ChoosePlayerStart() {
    if gs.Players[gs.TurnInfo.PlayerIndex].ActiveTribe != nil {
        gs.TurnInfo.Phase = DeclineChoice
        gs.GetPieceStackForConquest(gs.Players[gs.TurnInfo.PlayerIndex])
    } else {
        gs.TurnInfo.Phase = TribeChoice
    }
}
