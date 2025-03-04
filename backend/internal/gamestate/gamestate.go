package gamestate

import (
	"fmt"
	"log"
)

type GameState struct {
	Players []*Player
	TribeList []*TribeEntry
	TileList map[string]*Tile
	TurnInfo *TurnInfo
	RealTurninfo *TurnInfo
	Messages []string
	ModifierPoints map[string]func(int, *Player) int;
	ModifierTurnsAfter []TurninfoEntry
}

func New(playerNames []string, mapName string, raceKeys []string, traitKeys []string) (*GameState, error) {
	// Create a list of initialized players
	gs := &GameState{}
	gs.Players = make([]*Player, len(playerNames))
	for i, name := range(playerNames) {
		gs.Players[i] = &Player{
			Name: name,
			Index: i,
			ActiveTribe:    nil,
			PassiveTribes:  []*Tribe{}, // Initialize as empty slice
			CoinPile: 5,
			PieceStacks: []PieceStack{},
			HasActiveTribe: false,
			PointsEachTurn: []int{5},
		}
	}

	gs.TurnInfo = &TurnInfo{
		TurnIndex: 1,
		PlayerIndex: 0,
		Phase: TribeChoice,
	}

	gs.RealTurninfo = nil

	var err error;
	gs.TribeList, err = createTribeList(raceKeys, traitKeys)

	gs.ModifierPoints = make(map[string]func(int, *Player) int)
	function, ok := MapRegistry[mapName]
	if !ok {
		return nil, fmt.Errorf("map not found")
	}
	gs.TileList = function(gs)

        if err != nil {
            return nil, fmt.Errorf("failed to create list of tribe entries", err)
        }

	gs.ModifierTurnsAfter = []TurninfoEntry{}

	return gs, nil
}

func (gs *GameState) HandleTribeChoice(chooserIndex int, entryIndex int) error {
	if gs.TurnInfo.PlayerIndex != chooserIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	chooser := gs.Players[chooserIndex]

	if gs.TurnInfo.Phase != TribeChoice {
		return fmt.Errorf("The player is not supposed to pick a new tribe!")
	}

	if entryIndex > 5 || entryIndex < 0 {
		return fmt.Errorf("Invalid entry index")
	}

	if entryIndex > len(gs.TribeList) - 1 {
		return fmt.Errorf("not enough tribe entries")
	}

	if chooser.CoinPile < entryIndex {
		return fmt.Errorf("The player only has %d coins, but they need %d for picking this tribe", chooser.CoinPile, entryIndex)
	}

	entry := gs.TribeList[entryIndex]
	// Enact changes
	tempActiveTribe, err := CreateTribe(entry.Race, entry.Trait)
	if err != nil {
		return fmt.Errorf("Could not create tribe:", err)
	}
	chooser.ActiveTribe = tempActiveTribe
	chooser.ActiveTribe.Owner = chooser
	chooser.HasActiveTribe = true
	chooser.CoinPile += entry.CoinPile - entryIndex 
	chooser.PointsEachTurn[len(chooser.PointsEachTurn) - 1] += entry.CoinPile - entryIndex
	gs.TribeList = append(gs.TribeList[:entryIndex], gs.TribeList[entryIndex+1:]...)
	for _, tribeEntry := range gs.TribeList[:entryIndex] {
		tribeEntry.CoinPile += 1
	}

	chooser.PieceStacks = AddPieceStacks(chooser.PieceStacks, []PieceStack{{Type: string(entry.Race), Amount: entry.PiecePile}})
	chooser.PieceStacks = AddPieceStacks(chooser.PieceStacks, chooser.ActiveTribe.giveInitialStacks())
	log.Println(chooser.PieceStacks)
	gs.GetPieceStackForConquest(chooser)

	gs.TurnInfo.Phase = TileAbandonment

	return nil;
}

func (gs *GameState) HandleAbandonment(playerIndex int, tileId string, stackType string) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != TileAbandonment  && gs.TurnInfo.Phase != DeclineChoice {
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

	if tile.Presence == None || !tile.OwningTribe.checkPresence(tile, tribe.Race) {
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
		return err
	}

	if tile.Presence != None {
	    ok, err := tile.OwningTribe.specialDefense(gs, tile, attackingTribe, attackingStackType)
	    if ok {
		return err
	    }
	}

	ok, err = attackingTribe.specialConquest(gs, tile, attackingStackType)
	if ok {
		return err
	}

	if tile.Presence != None && tile.OwningTribe.checkPresence(tile, attackingTribe.Race) {
		return fmt.Errorf("This tile already belongs to the tribe!")
	}

	if err := attackingTribe.checkZoneAccess(tile); err != nil {
		return err
	}
	if err := attackingTribe.checkAdjacency(tile, gs); err != nil {
		return err
	}

	tileCost, moneyGainDefender, moneyLossAttacker := 0, 0, 0
	if tile.Presence != None {
		tileCost, moneyGainDefender, moneyLossAttacker, err = tile.OwningTribe.countDefense(tile, attacker)
	} else {
		tileCost, err = tile.countDefense()
	}
	
	if err != nil {
		return err
	}

	// counts the cost for the attacker
	attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill := attackingTribe.countAttack(tile, tileCost, attackingStackType)
	newStacks, hasDiceBeenUsed, ok, err := attackingTribe.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
	if err != nil {
		return err
	}
	if !ok {
		return gs.HandleStartRedeployment(attackerIndex)
	}

	// Enact changes
	if tile.Presence != None {
		tile.OwningTribe.Owner.CoinPile += moneyGainDefender - moneyLossDefender
		tile.OwningTribe.handleReturn(tile, gs, pawnKill)
	}

	attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, newStacks)
	tile.PieceStacks = AddPieceStacks(tile.PieceStacks, attackingTribe.countNewTileStacks(newStacks, tile))

	attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
	// attacker.PointsEachTurn[len(attacker.PointsEachTurn) - 1] += moneyGainDefender - moneyLossDefender
	tile.OwningTribe = attackingTribe

	if tile.OwningTribe.IsActive {
		tile.Presence = Active
	} else {
		tile.Presence = Passive
	} 	
	if hasDiceBeenUsed {
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
	newStacks := player.ActiveTribe.startRedeployment(gs)
	player.PieceStacks = AddPieceStacks(player.PieceStacks, newStacks)

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

	if tile.Presence != None && !tile.OwningTribe.checkPresence(tile, tribe.Race) {
		return fmt.Errorf("This tile does not belong to the player!")
	}

	return tribe.handleDeploymentOut(tile, stackType, 1, gs)
}

func (gs *GameState) HandleRedeploymentIn(playerIndex int, tileId string, stackType string, amount int) error {
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

	
	return tribe.handleDeploymentIn(tile, stackType, amount, gs)

}

func (gs *GameState) HandleFinishTurn(playerIndex int) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != Redeployment && gs.TurnInfo.Phase != Conquest && gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice{
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	if gs.TurnInfo.Phase != Redeployment {
		if err := gs.HandleStartRedeployment(playerIndex); err != nil {
			return fmt.Errorf("Error in redeployment phase:", err)
		}
	}

	player := gs.Players[playerIndex]

	if player.HasActiveTribe {
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

	player := gs.Players[playerIndex]

	playerTribe, err := player.getTribe(stackType)
	if err != nil {
		return err
	}

	tileTo, ok := gs.TileList[tileToId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	tileFrom, ok := gs.TileList[tileFromId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	return playerTribe.handleMovement(stackType, tileFrom, tileTo, gs)
}


func (gs *GameState) HandleDecline(playerIndex int) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	player := gs.Players[playerIndex];

	if !player.HasActiveTribe {
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

	for i := range(gs.ModifierTurnsAfter) {
		if (gs.ModifierTurnsAfter[i].player == gs.TurnInfo.PlayerIndex) {
			gs.RealTurninfo = gs.TurnInfo
			gs.TurnInfo = &gs.ModifierTurnsAfter[i].TurnInfo
			gs.ModifierTurnsAfter[i].actionBefore(gs)
			gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter[:i], gs.ModifierTurnsAfter[i+1:]...)
			return
		}
	}

	player := gs.Players[gs.TurnInfo.PlayerIndex]
	player.CoinPile += gs.countPoints(player)
	player.PointsEachTurn = append(player.PointsEachTurn, player.CoinPile)

	if player.HasActiveTribe {
		gs.Messages = append(gs.Messages, fmt.Sprintf(
			"%s made %d points this turn",
			player.Name,
			player.PointsEachTurn[len(player.PointsEachTurn) - 1]-player.PointsEachTurn[len(player.PointsEachTurn) - 2],
		))
	} else {
		gs.Messages = append(gs.Messages, fmt.Sprintf(
			"%s went into decline and made %d points this turn",
			player.Name,
			player.PointsEachTurn[len(player.PointsEachTurn) - 1]-player.PointsEachTurn[len(player.PointsEachTurn) - 2],
		))
	}

	if gs.TurnInfo.PlayerIndex == len(gs.Players) - 1 {
		if gs.TurnInfo.TurnIndex == 10 {
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
	if gs.Players[gs.TurnInfo.PlayerIndex].HasActiveTribe {
		gs.TurnInfo.Phase = DeclineChoice
		gs.GetPieceStackForConquest(gs.Players[gs.TurnInfo.PlayerIndex])
	} else {
		gs.TurnInfo.Phase = TribeChoice
	}
}

