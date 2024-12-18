package gamestate

import (
	"fmt"
)

type GameState struct {
	Actions []string
	Players []Player
	TribeList []TribeEntry
	TileList map[string]Tile
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
			HasActiveTribe: false,
		}
	}

	turnInfo := TurnInfo{
		TurnIndex: 0,
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
		Actions:	[]string{},
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

func (gs *GameState) GetPlayers() []Player {
	return gs.Players
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

	// Enact changes
	chooser.ActiveTribe = gs.TribeList[entryIndex].Tribe
	chooser.CoinPile += gs.TribeList[entryIndex].CoinPile - entryIndex 
	gs.TribeList = append(gs.TribeList[:entryIndex], gs.TribeList[entryIndex+1])

	// Error handling too small a list to do here.

	return nil;
}

func (gs *GameState) HandleAbandonment(playerIndex int, tileId string) error {
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

	defendingTribe := tile.OwningTribe
	attackingTribe, err := GetPlayerTribe(attackingStackType, attacker)

	if err != nil {
		return fmt.Errorf("Could not create retrieve attacker's tribe", err)
	}

	if defendingTribe.IsTileUnTakeable(tile) {
		return fmt.Errorf("The attacked tile is untakeable")
	}

	tileCost := defendingTribe.countDefense(tile)
	attackCostStacks := attackingTribe.countAttack(tile, tileCost, attackingStackType)
		
	defenderReturningStacks := defendingTribe.countReturningStacks(tile)
	newTileStacks := attackingTribe.countNewTileStacks(attackCostStacks)
	newStacks, ok := SubtractPieceStacks(attacker.PieceStacks, attackCostStacks)
	if !ok {
		return fmt.Errorf("The player does not have enough pieces")
	}

	// Enact changes
	attacker.PieceStacks = newStacks
	tile.OwningPlayer.addReserves(defenderReturningStacks)
	tile.PieceStacks = newTileStacks
	tile.OwningTribe = attackingTribe
	tile.OwningPlayer = attacker
	gs.TurnInfo.Phase = Conquest

	return nil
}

func (gs *GameState) HandleStartRedeployment(playerIndex int) error {
	return nil
}

func (gs *GameState) HandleRedeploymentIn(playerIndex int, tileIndex string, stack PieceStack) error {
	return nil
}

func (gs *GameState) HandleRedeploymentOut(playerIndex int, tileIndex string, stack PieceStack) error {
	return nil
}

func (gs *GameState) HandleDecline(playerIndex int) error {
	return nil
}

func (gs *GameState) ApplyAction(action string) {
	gs.Actions = append(gs.Actions, action)
}

func (gs *GameState) GetActions() []string {
	copyActions := make([]string, len(gs.Actions))
	copy(copyActions, gs.Actions)
	return copyActions
}
