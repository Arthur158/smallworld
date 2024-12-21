package gamestate

import (
	"fmt"
)

type GameState struct {
	Players []Player
	TribeList []TribeEntry
	TileList map[string]*Tile
	TurnInfo TurnInfo
}

func New(playerCount int) (*GameState, error) {
	// Create a list of initialized players
	players := make([]Player, playerCount)
	for i := 0; i < playerCount; i++ {
		players[i] = Player{
			ActiveTribe:    nil,
			PassiveTribes:  []*Tribe{}, // Initialize as empty slice
			CoinPile: 5,
			PieceStacks: []PieceStack{},
			HasActiveTribe: false,
		}
	}

	turnInfo := TurnInfo{
		TurnIndex: 1,
		PlayerIndex: 0,
		Phase: TribeChoice,
		ConqueredPassive: 0,
		ConqueredActive: 0,
	}

	tribelist, err := createTribeList()

	tilelist := Map1()

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

func (gs *GameState) GetTribeEntries() []TribeEntry {
	return gs.TribeList[:5]
}

func (gs *GameState) GetTileStacks(id string) []PieceStack {
	// Careful error management here
	return gs.TileList[id].PieceStacks
}

func (gs *GameState) HandleTribeChoice(chooserIndex int, entryIndex int) error {
	if gs.TurnInfo.PlayerIndex != chooserIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	chooser := &gs.Players[chooserIndex]

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
	chooser.ActiveTribe = entry.Tribe
	chooser.CoinPile += entry.CoinPile - entryIndex 
	gs.TribeList = append(gs.TribeList[:entryIndex], gs.TribeList[entryIndex+1:]...)
	chooser.addReserves([]PieceStack{{Type: string(entry.Tribe.Race), Amount: entry.PiecePile}})
	gs.TurnInfo.Phase = Conquest

	// Error handling too small a list to do here.

	return nil;
}

func (gs *GameState) HandleAbandonment(playerIndex int, tileId string) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != TileAbandonment  && gs.TurnInfo.Phase != DeclineChoice {
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	player := &gs.Players[playerIndex]

	tile, ok := gs.TileList[tileId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	if player != tile.OwningPlayer {
		return fmt.Errorf("Player can only abandon their own region!")
	}

	// maybe do the necessary in cantilebeabandoned for now, meaning checking if the zone only contains the piecestack with 1, this function is necessary to keep the functionality for zombies anyways, so we might as well use it for checking if there is only 1 stack on there.
	if !tile.OwningTribe.CanTileBeAbandoned(tile) {
		return fmt.Errorf("Stack cannot be removed!")
	}

	stacks := tile.OwningTribe.ReceiveAbandonment(tile)

	tile.OwningPlayer.addReserves(stacks)
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

	if gs.TurnInfo.Phase != Conquest && gs.TurnInfo.Phase != TileAbandonment {
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	attacker := &gs.Players[attackerIndex]

	if !DoesPlayerHaveStack(attackingStackType, attacker) {
		return fmt.Errorf("The stack is invalid for this player!")
	}


	tile, ok := gs.TileList[tileId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	if tile.OwningPlayer == attacker {
		return fmt.Errorf("player cannot attack themselves")
	}

	defendingTribe := tile.OwningTribe

	attackingTribe, err := GetPlayerTribe(attackingStackType, attacker)
	if err != nil {
		return fmt.Errorf("Could not create retrieve attacker's tribe", err)
	}

	if err := attackingTribe.checkZoneAccess(tile); err != nil {
		return fmt.Errorf("cannot access zone", err)
	}
	if err := attackingTribe.checkAdjacency(tile, gs); err != nil {
		return fmt.Errorf("cannot reach zone", err)
	}

	var tileCost int
	if tile.Presence == Passive || tile.Presence == Active {
		tileCost, err = defendingTribe.countDefense(tile)
		if err != nil {
			fmt.Errorf("Impossible to attack", err)
		}
	} else {
		tileCost = CountDefense(tile)
	}

	attackCostStacks := attackingTribe.countAttack(tile, tileCost, attackingStackType)
	newTileStacks := attackingTribe.countNewTileStacks(attackCostStacks)
	newStacks, ok := SubtractPieceStacks(attacker.PieceStacks, attackCostStacks)
	if !ok {
		return fmt.Errorf("The player does not have enough pieces")
	}

	// Enact changes
	if tile.Presence == Passive || tile.Presence == Active {
		defenderReturningStacks := defendingTribe.countReturningStacks(tile)
		tile.OwningPlayer.addReserves(defenderReturningStacks)
	}
	attacker.PieceStacks = newStacks
	tile.PieceStacks = newTileStacks
	tile.OwningTribe = attackingTribe
	tile.OwningPlayer = attacker
	tile.Presence = Active; // fucking zombies here
	gs.TurnInfo.Phase = Conquest

	return nil
}

func (gs *GameState) HandleStartRedeployment(playerIndex int) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != Conquest {
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	player := &gs.Players[playerIndex]
	newStacks := player.ActiveTribe.startRedeployment()
	player.addReserves(newStacks)

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

	player := &gs.Players[playerIndex]

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

	player.addReserves(stacks)

	return nil
}

func (gs *GameState) HandleRedeploymentIn(playerIndex int, tileId string, stackType string) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	if gs.TurnInfo.Phase != Redeployment {
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	player := &gs.Players[playerIndex]

	tile, ok := gs.TileList[tileId]
	if !ok {
		return fmt.Errorf("No tile with this id!")
	}

	if tile.OwningPlayer != player {
		return fmt.Errorf("This tile does not belong to the player!")
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

	if gs.TurnInfo.Phase != Redeployment {
		return fmt.Errorf("The player is in the %s phase!", gs.TurnInfo.Phase)
	}

	player := &gs.Players[playerIndex]

	CoinsMade := gs.CountPoints(player)

	player.CoinPile += CoinsMade

	gs.handleNextPlayerTurn()

	return nil
}

func (gs *GameState) HandleDecline(playerIndex int) error {
	if gs.TurnInfo.PlayerIndex != playerIndex {
		return fmt.Errorf("It is not this player's turn!")
	}

	player := gs.Players[playerIndex];

	if player.ActiveTribe.CanGoIntoDecline(gs){
		return fmt.Errorf("You cannot go into decline now!")
	}

	for i, tribe := range player.PassiveTribes {
		if (tribe.prepareRemoval(gs)) {
			player.PassiveTribes = append(player.PassiveTribes[:i], player.PassiveTribes[i+1:]...)
		}
	}

	player.ActiveTribe.prepareDecline(gs)
	player.ActiveTribe = nil

	gs.handleNextPlayerTurn()

	return nil
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
		gs.TurnInfo.Phase = TileAbandonment
		gs.GetPieceStackForConquest(&gs.Players[gs.TurnInfo.PlayerIndex])
	} else {
		gs.TurnInfo.Phase = TribeChoice
	}
}

