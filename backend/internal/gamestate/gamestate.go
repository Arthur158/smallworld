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
}

func New(playerCount int, mapName string) (*GameState, error) {
	// Create a list of initialized players
	players := make([]*Player, playerCount)
	for i := 0; i < playerCount; i++ {
		players[i] = &Player{
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

	log.Println(mapName)
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
	tempActiveTribe, err := createTribe(entry.Race, entry.Trait)
	if err != nil {
		return fmt.Errorf("Could not create tribe:", err)
	}
	chooser.ActiveTribe = tempActiveTribe
	chooser.HasActiveTribe = true
	chooser.CoinPile += entry.CoinPile - entryIndex 
	chooser.PointsEachTurn[len(chooser.PointsEachTurn) - 1] += entry.CoinPile - entryIndex
	gs.TribeList = append(gs.TribeList[:entryIndex], gs.TribeList[entryIndex+1:]...)
	for _, tribeEntry := range gs.TribeList[:entryIndex] {
		tribeEntry.CoinPile += 1
	}

	chooser.PieceStacks = AddPieceStacks(chooser.PieceStacks, []PieceStack{{Type: string(entry.Race), Amount: entry.PiecePile}})
	chooser.PieceStacks = AddPieceStacks(chooser.PieceStacks, chooser.ActiveTribe.giveInitialStacks())

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

	stacks := tile.OwningTribe.receiveAbandonment(tile)

	tile.OwningPlayer.PieceStacks = AddPieceStacks(tile.OwningPlayer.PieceStacks, stacks)
	tile.OwningTribe = nil
	tile.OwningPlayer = nil
	tile.PieceStacks = []PieceStack{}
	tile.Presence = None
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

	defendingTribe := tile.OwningTribe

	attackingTribe, err := GetPlayerTribe(attackingStackType, attacker)
	if err != nil {
		return fmt.Errorf("Could not create retrieve attacker's tribe", err)
	}

	if tile.OwningTribe == attackingTribe {
		return fmt.Errorf("tribe cannot attack itself")
	}

	if err := attackingTribe.checkZoneAccess(tile); err != nil {
		return fmt.Errorf("cannot access zone", err)
	}
	if err := attackingTribe.checkAdjacency(tile, gs); err != nil {
		return fmt.Errorf("cannot reach zone", err)
	}

	var tileCost, moneyGainDefender, moneyLossAttacker int
	if tile.Presence == Passive || tile.Presence == Active {
		tileCost, moneyGainDefender, moneyLossAttacker, err = defendingTribe.countDefense(tile)
		if err != nil {
			return fmt.Errorf("Impossible to attack", err)
		}
	} else {
		tileCost = CountDefense(tile)
	}

	attackCostStacks, moneyGainAttacker, moneyLossDefender := attackingTribe.countAttack(tile, tileCost, attackingStackType)
	newTileStacks := attackingTribe.countNewTileStacks(attackCostStacks, tile)
	newStacks, stacksToRemove, hasDiceBeenUsed, err := attackingTribe.calculateRemainingAttackingStacks(attacker.PieceStacks, attackCostStacks)
	if err != nil && hasDiceBeenUsed {
		return gs.HandleStartRedeployment(attackerIndex)
	} else if err != nil {
		return fmt.Errorf("Failure", err)
	}

	defenderRemainingStacks	:= []PieceStack{}
	// Enact changes
	if tile.Presence != None {
		defenderReturningStacks, tempDefenderRemainingStacks := defendingTribe.countReturningStacks(tile)
		tile.OwningPlayer.PieceStacks = AddPieceStacks(tile.OwningPlayer.PieceStacks, defenderReturningStacks)
		defenderRemainingStacks = tempDefenderRemainingStacks
		tile.OwningPlayer.CoinPile += moneyGainDefender - moneyLossDefender
		// tile.OwningPlayer.PointsEachTurn[len(tile.OwningPlayer.PointsEachTurn) - 1] += moneyGainDefender - moneyLossDefender
	}
	tile.PieceStacks, _ = SubtractPieceStacks(AddPieceStacks(newTileStacks, defenderRemainingStacks), stacksToRemove)
	attacker.PieceStacks = newStacks
	attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
	// attacker.PointsEachTurn[len(attacker.PointsEachTurn) - 1] += moneyGainDefender - moneyLossDefender
	tile.OwningTribe = attackingTribe
	tile.OwningPlayer = attacker

	if tile.OwningTribe != nil && tile.OwningTribe.IsActive {
		tile.Presence = Active
	} else if tile.OwningTribe != nil {
		tile.Presence = Passive
	} else {
		tile.Presence = None
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

	tile, ok := gs.TileList[tileId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	if tile.OwningPlayer != player {
		return fmt.Errorf("This tile does not belong to the player!")
	}

	stacks, err := tile.OwningTribe.getStacksOutRedeployment(tile, stackType)
	if err != nil {
		return fmt.Errorf("Unable to redeploy", err)
	}

	tile.PieceStacks, ok = SubtractPieceStacks(tile.PieceStacks, stacks)
	if !ok {
		return fmt.Errorf("Could not substract the stacks")
	}
	
	player.PieceStacks = AddPieceStacks(player.PieceStacks, stacks)

	return nil
}

func (gs *GameState) HandleRedeploymentIn(playerIndex int, tileId string, stackType string) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != Redeployment {
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	player := gs.Players[playerIndex]

	tile, ok := gs.TileList[tileId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	if tile.OwningPlayer != player {
		return fmt.Errorf("This tile does not belong to the player!")
	}

	if !tile.OwningTribe.canBeRedeployedIn(tile, stackType) {
		return fmt.Errorf("Cannot redeploy here")
	}

	movingStack := []PieceStack{{Type: stackType, Amount: 1}}

	// Subtract  from player stacks
	newStacks, ok := SubtractPieceStacks(player.PieceStacks, movingStack)
	if !ok {
		return fmt.Errorf("Cannot redeploy pieces you don't have")
	}
	player.PieceStacks = newStacks

	// Add to tile stack
	tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)

	return nil
}

func (gs *GameState) HandleFinishTurn(playerIndex int) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != Redeployment && gs.TurnInfo.Phase != Conquest && gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice{
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	player := gs.Players[playerIndex]

	player.CoinPile += gs.countPoints(player)
	player.PointsEachTurn = append(player.PointsEachTurn, player.CoinPile)

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


	log.Println(player.PassiveTribes)
	for i, tribe := range player.PassiveTribes {
		if (tribe.prepareRemoval(gs)) {
			player.PassiveTribes = append(player.PassiveTribes[:i], player.PassiveTribes[i+1:]...)
		}
	}

        if !player.ActiveTribe.canGoIntoDecline(gs) {
            return fmt.Errorf("The tribe cannot go in decline at this moment")
        }

	player.ActiveTribe.goIntoDecline(gs)

	player.PieceStacks = player.ActiveTribe.countRemainingAttackingStacks(player)

	// iterate over the tiles and remove pieces accordingly if the tribe is the one going into decline
	for _, tile := range gs.TileList {
            if tile.Presence != None && tile.OwningTribe.Race == player.ActiveTribe.Race {
                tile.PieceStacks = tile.OwningTribe.countPiecesRemaining(tile)
                tile.Presence = Passive
            }
        }


	player.PassiveTribes = append(player.PassiveTribes, player.ActiveTribe)
	player.ActiveTribe = nil
	player.HasActiveTribe = false

	player.CoinPile += gs.countPoints(player)
	player.PointsEachTurn = append(player.PointsEachTurn, player.CoinPile)

	gs.handleNextPlayerTurn()

	return nil
}

func (gs *GameState) countPoints(player *Player) int {
	total := 0
	for _, tile := range gs.TileList {
		if tile.OwningPlayer == player {
			total += tile.OwningTribe.countPoints(tile)
		}
	}
	if player.HasActiveTribe {
		total += player.ActiveTribe.countExtrapoints()
	}
	for _, passiveTribe := range player.PassiveTribes {
		total += passiveTribe.countExtrapoints()
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

