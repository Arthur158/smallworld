package gamestate

import (
	"fmt"
)

type GameState struct {
	Players []*Player
	TribeList []*TribeEntry
	TileList map[string]*Tile
	TurnInfo *TurnInfo
	Messages []string
}

func New(playerNames []string, mapName string) (*GameState, error) {
	// Create a list of initialized players
	players := make([]*Player, len(playerNames))
	for i, name := range(playerNames) {
		players[i] = &Player{
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

	turnInfo := &TurnInfo{
		TurnIndex: 1,
		PlayerIndex: 0,
		Phase: TribeChoice,
	}

	tribelist, err := createTribeList()

	tilelist := MapRegistry[mapName]()

        if err != nil {
            return nil, fmt.Errorf("failed to create list of tribe entries", err)
        }

	return &GameState{
		Players:	players,
		TribeList:	tribelist,
		TileList:	tilelist,
		TurnInfo:	turnInfo,
	}, nil
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
	gs.GetPieceStackForConquest(gs.Players[gs.TurnInfo.PlayerIndex])

	gs.TurnInfo.Phase = Conquest

	return nil;
}

func (gs *GameState) HandleAbandonment(playerIndex int, tileId string) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != TileAbandonment  && gs.TurnInfo.Phase != DeclineChoice {
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	player := gs.Players[playerIndex]

	tile, ok := gs.TileList[tileId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	if player != tile.OwningPlayer {
		return fmt.Errorf("Player can only abandon their own region!")
	}

	// maybe do the necessary in cantilebeabandoned for now, meaning checking if the zone only contains the piecestack with 1, this function is necessary to keep the functionality for zombies anyways, so we might as well use it for checking if there is only 1 stack on there.
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


	attackingTribe, err := GetPlayerTribe(attackingStackType, attacker)
	if err != nil {
		return err
	}

	ok, err = attackingTribe.specialConquest(gs, tile, attackingStackType, attacker, attackerIndex)
	if ok {
		return err
	}

	defendingTribe := tile.OwningTribe


	if tile.Presence != None && tile.OwningTribe.checkPresence(tile, attackingTribe.Race) {
		return err
	}

	if err := attackingTribe.checkZoneAccess(tile); err != nil {
		return err
	}
	if err := attackingTribe.checkAdjacency(tile, gs); err != nil {
		return err
	}

	tileCost, moneyGainDefender, moneyLossAttacker := 0, 0, 0
	if tile.Presence == Passive || tile.Presence == Active {
		tileCost, moneyGainDefender, moneyLossAttacker, err = defendingTribe.countDefense(tile)
		if err != nil {
			return err
		}
	} else {
		tileCost, err = tile.countDefense()
		if err != nil {
			return err
		}
	}

	// counts the cost for the attacker
	attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill := attackingTribe.countAttack(tile, tileCost, attackingStackType)
	newStacks, hasDiceBeenUsed, ok, err := attackingTribe.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
	newTileStacks := attackingTribe.countNewTileStacks(newStacks, tile)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}

	// Enact changes
	if tile.Presence != None {
		tile.OwningPlayer.CoinPile += moneyGainDefender - moneyLossDefender
		defendingTribe.clearTile(tile, gs, pawnKill)
		// tile.OwningPlayer.PointsEachTurn[len(tile.OwningPlayer.PointsEachTurn) - 1] += moneyGainDefender - moneyLossDefender
	}
	tile.PieceStacks = AddPieceStacks(tile.PieceStacks, newTileStacks)
	attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, attackCostStacks)
	attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
	// attacker.PointsEachTurn[len(attacker.PointsEachTurn) - 1] += moneyGainDefender - moneyLossDefender
	tile.OwningTribe = attackingTribe
	tile.OwningPlayer = attacker

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
	tribe, err := GetPlayerTribe(stackType, player)
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
	tribe, err := GetPlayerTribe(stackType, player)
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

	player.CoinPile += gs.countPoints(player)
	player.PointsEachTurn = append(player.PointsEachTurn, player.CoinPile)

	gs.Messages = append(gs.Messages, fmt.Sprintf(
			"%s made %d points this turn",
			player.Name,
			player.PointsEachTurn[len(player.PointsEachTurn) - 1]-player.PointsEachTurn[len(player.PointsEachTurn) - 2],
		    ),
)

	gs.handleNextPlayerTurn()

	return nil
}

func (gs *GameState) HandleDecline(playerIndex int) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	player := gs.Players[playerIndex];

	if !player.HasActiveTribe {
		return fmt.Errorf("The player does not have an active tribe!")
	}

        if !player.ActiveTribe.canGoIntoDecline(gs) {
            return fmt.Errorf("The tribe cannot go in decline at this moment")
        }


	points := player.ActiveTribe.goIntoDecline(gs)
	player.CoinPile += points
	player.PointsEachTurn = append(player.PointsEachTurn, player.CoinPile)

	gs.Messages = append(gs.Messages, fmt.Sprintf(
			"%s went into decline and made %d points this turn",
			player.Name,
			player.PointsEachTurn[len(player.PointsEachTurn) - 1]-player.PointsEachTurn[len(player.PointsEachTurn) - 2],
		    ))

	gs.handleNextPlayerTurn()

	return nil
}

func (gs *GameState) countPoints(player *Player) int {
	total := 0
	for _, tile := range gs.TileList {
		if tile.Presence != None {
			if player.HasActiveTribe && tile.OwningTribe.checkPresence(tile, player.ActiveTribe.Race) {
				total += player.ActiveTribe.countPoints(tile)
			}
			for _, tribe := range(player.PassiveTribes) {
			    if tile.OwningTribe.checkPresence(tile, tribe.Race) {
				total += tribe.countPoints(tile)
			    }
			}
		}
	}
	if player.HasActiveTribe {
		total += player.ActiveTribe.countExtrapoints(gs)
	}
	for _, passiveTribe := range player.PassiveTribes {
		total += passiveTribe.countExtrapoints(gs)
	}

	return total
}


func (gs *GameState) handleNextPlayerTurn() {
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

