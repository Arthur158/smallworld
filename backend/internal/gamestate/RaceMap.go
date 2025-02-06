package gamestate

import (
	"fmt"
	"log"
)

type RaceValue struct {
    Transform func(*Tribe)
    Count     int
}

var RaceMap = map[Race]RaceValue {
	// "Troll": {Transform: func(t *Tribe) {
	// 	// Make a newly conquered region contain a lair
	// 	oldCountNewTileStacks := t.countNewTileStacks
	// 	t.countNewTileStacks = func(stacks []PieceStack, tile *Tile) []PieceStack {
	// 		oldstacks := oldCountNewTileStacks(stacks, tile)
	// 		return AddPieceStacks(oldstacks, []PieceStack{{Type: "Lair", Amount: 1}})
	// 	}
	//
	// 	// Make the defense of the tile + 1
	// 	oldCountDefense := t.countDefense
	// 	t.countDefense = func(tile *Tile) (int, int, int, error) {
	// 		old, g, l, err := oldCountDefense(tile)
	// 		if err != nil {
	// 			return old, g, l, err
	// 		}
	// 		return old+1, g, l, nil
	// 	}
	//
	// 	oldCountPiecesRemaining := t.countPiecesRemaining
	// 	t.countPiecesRemaining = func(tile *Tile) []PieceStack {
	// 		oldstacks := oldCountPiecesRemaining(tile)
	// 		return AddPieceStacks(oldstacks, []PieceStack{{Type: "Lair", Amount: 1}})
	// 	}
	// 	}, Count: 5},
	// "Wizard": {Transform: func(t *Tribe) {
	// 	oldCountPoints := t.countPoints
	// 	t.countPoints = func(tile *Tile) int {
	// 		count := oldCountPoints(tile)
	// 		if t.IsActive {
	// 			for _, attr := range tile.Attributes {
	// 				if attr == Magic {
	// 					count += 1
	// 				}
	// 			}
	// 		}
	// 		return count
	// 	}
	// 	}, Count: 5},
	"Khan": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && (tile.Biome == Field || tile.Biome == Hill) {
				count += 1
			} else if t.IsActive {
				count -= 1
			}
			return max(0, count)
		}
		}, Count: 5},
	// "Human": {Transform: func(t *Tribe) {
	// 	oldCountPoints := t.countPoints
	// 	t.countPoints = func(tile *Tile) int {
	// 		count := oldCountPoints(tile)
	// 		if t.IsActive && tile.Biome == Field {
	// 			count += 1
	// 		} 
	// 		return max(0, count)
	// 	}
	// 	}, Count: 5},
	// "Halfling": {Transform: func(t *Tribe) {
	// 	t.State["holesleft"] = 2
	// 	t.State["startedalready"] = false
	// 	oldCheckAdjacency := t.checkAdjacency
	// 	t.checkAdjacency = func(tile *Tile, gs *GameState) error {
	// 		err := oldCheckAdjacency(tile, gs)
	// 		if err != nil && !gs.IsTribePresentOnTheBoard(t.Race) && t.State["startedalready"] == false {
	// 			t.State["startedalready"] = true
	// 			return nil
	// 		} else {
	// 			t.State["startedalready"] = true
	// 			return err
	// 		}
	// 	}
	//
	// 	oldCountNewTileStacks := t.countNewTileStacks
	// 	t.countNewTileStacks = func(ps []PieceStack, tile *Tile) []PieceStack {
	// 		stacks := oldCountNewTileStacks(ps, tile)
	// 		if value, ok := t.State["holesleft"].(int); ok && value > 0 {
	// 			t.State["holesleft"] = value - 1
	// 			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Hole", Amount: 1}})
	// 		}
	// 		return stacks
	// 	}
	// 	oldCountDefense := t.countDefense
	// 	t.countDefense = func(tile *Tile) (int, int, int, error) {
	// 		count, g, l, err := oldCountDefense(tile)
	// 		if err != nil {
	// 			return count, g, l, err
	// 		}
	// 		for _, stack := range tile.PieceStacks {
	// 			if stack.Type == "Hole" {
	// 				return 1000, g, l, fmt.Errorf("A hole in the ground cannot be conquered!")
	// 			}
	// 		}
	// 		return count, g, l, err
	// 	}
	// 	}, Count: 6},
	// "White Lady": {Transform: func(t *Tribe) {
	// 	oldCountDefense := t.countDefense
	// 	t.countDefense = func(tile *Tile) (int, int, int, error) {
	// 		count, g, l, err := oldCountDefense(tile)
	// 		if err != nil {
	// 			return count, g, l, err
	// 		}
	// 		if !t.IsActive {
	// 			return 1000, g, l, fmt.Errorf("Cannot conquer white ladies when they are in decline") 
	// 		}
	// 		return count, g, l, err
	// 	}
	// 	}, Count: 2},
	// "Triton": {Transform: func(t *Tribe) {
	// 	oldCountAttack := t.countAttack
	// 	t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
	// 		stacks, g, l := oldCountAttack(tile, cost, stackType)
	// 		isNextToWater := false
	// 		for _, neighbour := range tile.AdjacentTiles {
	// 			if neighbour.Biome == Water {
	// 				isNextToWater = true
	// 			}
	// 		}
	// 		if isNextToWater {
	// 			for i, stack := range stacks {
	// 				if stack.Type == string(t.Race) {
	// 					stacks[i].Amount -= 1
	// 				}
	// 			}
	// 		}
	// 		return stacks, g, l
	// 	}
	// 	}, Count: 6},
	// "Shrubman": {Transform: func(t *Tribe) {
	// 	oldCountDefense := t.countDefense
	// 	t.countDefense = func(tile *Tile) (int, int, int, error) {
	// 		count, g, l, err := oldCountDefense(tile)
	// 		if err != nil {
	// 			return count, g, l, err
	// 		}
	// 		if tile.Biome == Forest {
	// 			return 1000, g, l, fmt.Errorf("cannot conquer shrubman when they are in a forest") 
	// 		}
	// 		return count, g, l, err
	// 	}
	// 	}, Count: 6},
	// "Ratman": {Transform: func(t *Tribe) {
	// 	}, Count: 8},
	// "Goblin": {Transform: func(t *Tribe) {
	// 	oldCountAttack := t.countAttack
	// 	t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
	// 		stacks, g, l := oldCountAttack(tile, cost, stackType)
	// 		if tile.Presence == Passive {
	// 			for i, stack := range stacks {
	// 				if stack.Type == string(t.Race) {
	// 					stacks[i].Amount -= 1
	// 				}
	// 			}
	// 		}
	// 		return stacks, g, l
	// 	}
	// 	}, Count: 6},
	// "Giant": {Transform: func(t *Tribe) {
	// 	oldCountAttack := t.countAttack
	// 	t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
	// 		stacks, g, l := oldCountAttack(tile, cost, stackType)
	// 		possessAdjacentMountain := false
	// 		for _, neighbour := range tile.AdjacentTiles {
	// 			if neighbour.Biome == Mountain && neighbour.Presence != None && neighbour.OwningTribe.Race == t.Race {
	// 				possessAdjacentMountain = true
	// 			}
	// 		}
	// 		if possessAdjacentMountain {
	// 			for i, stack := range stacks {
	// 				if stack.Type == string(t.Race) {
	// 					stacks[i].Amount -= 1
	// 				}
	// 			}
	// 		}
	// 		return stacks, g, l
	// 	}
	// 	}, Count: 6},
	// "Orc": {Transform: func(t *Tribe) {
	// 	oldCountAttack := t.countAttack
	// 	t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
	// 		old, g, l := oldCountAttack(tile, cost, stackType)
	// 		if tile.Presence != None {
	// 			g += 1
	// 		}
	// 		return old, g , l
	// 	}
	// 	}, Count: 5},
	// "Skeletton": {Transform: func(t *Tribe) {
	// 	t.State["killcount"] = 0
	// 	oldCountAttack := t.countAttack
	// 	t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
	// 		println("are we here")
	// 		old, g, l := oldCountAttack(tile, cost, stackType)
	// 		if tile.Presence != None {
	// 			if killCount, ok := t.State["killcount"].(int); ok {
	// 				println("here right now")
	// 				t.State["killcount"] = killCount + 1
	// 				println(t.State["killcount"])
	// 			}
	// 		}
	// 		return old, g , l
	// 	}
	// 	oldStartRedeployment := t.startRedeployment
	// 	t.startRedeployment = func(gs *GameState) []PieceStack {
	// 		stacks := oldStartRedeployment(gs)
	// 		if killCount, ok := t.State["killcount"].(int); ok {
	// 			t.State["killcount"] = 0
	// 			stacks = AddPieceStacks(stacks, []PieceStack{{Type: string(t.Race), Amount: killCount / 2}})
	// 		}
	// 		return stacks
	// 	}
	// 	}, Count: 6},
	// "Pygmy": {Transform: func(t *Tribe) {
	// 	oldCountReturningStacks := t.countReturningStacks
	// 	t.countReturningStacks = func(tile *Tile, gs *GameState) ([]PieceStack, []PieceStack) {
	// 		a, b := oldCountReturningStacks(tile, gs)
	// 		diceThrow := RollDice()
	// 		gs.Messages = append(gs.Messages, fmt.Sprintf("Pygmies got back: %d Pawns!", diceThrow))
	// 		
	// 		a = AddPieceStacks(a, []PieceStack{{Type: string(t.Race), Amount: diceThrow}})
	// 		return a, b
	// 	}
	// 	}, Count: 6},
	// "Elf": {Transform: func(t *Tribe) {
	// 	oldCountReturningStacks := t.countReturningStacks
	// 	t.countReturningStacks = func(tile *Tile, gs *GameState) ([]PieceStack, []PieceStack) {
	// 		a, b := oldCountReturningStacks(tile, gs)
	// 		a = AddPieceStacks(a, []PieceStack{{Type: string(t.Race), Amount: 1}})
	// 		return a, b
	// 	}
	// 	}, Count: 6},
	// "Pixy": {Transform: func(t *Tribe) {
	// 	oldStartRedeployment := t.startRedeployment
	// 	t.startRedeployment = func(gs *GameState) []PieceStack {
	// 		stacks := oldStartRedeployment(gs)
	// 		for _, tile := range gs.TileList {
	// 			if tile.Presence != None && tile.OwningTribe.Race == t.Race {
	// 				t.getStacksForConquest(tile, tile.OwningPlayer)
	// 			}
	// 		}
	// 		return stacks
	// 	}
	// 	oldCanBeRedeployedIn := t.canBeRedeployedIn
	// 	t.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
	// 		if oldCanBeRedeployedIn(tile, stackType) {
	// 			return stackType != string(t.Race)
	// 		}
	// 		return false
	// 	}
	// 	}, Count: 11},
	// "Barbarian": {Transform: func(t *Tribe) {
	// 	oldCanBeRedeployedIn := t.canBeRedeployedIn
	// 	t.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
	// 		if oldCanBeRedeployedIn(tile, stackType) {
	// 			return stackType != string(t.Race)
	// 		}
	// 		return false
	// 	}
	// 	oldGetStacksOutRedeployment := t.getStacksOutRedeployment
	// 	t.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
	// 		stacks, err := oldGetStacksOutRedeployment(tile, stackType)
	// 		if err != nil {
	// 			if stackType == string(t.Race) {
	// 				return nil, fmt.Errorf("Barbarians cannot redeploy!")
	// 			}
	// 		}
	// 		return stacks, err
	// 	}
	// 	}, Count: 9},
	"Sorcerer": {Transform: func(t *Tribe) {
		oldgetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(player *Player) {
			oldgetStacksForConquestTurn(player)
			newstacks := []PieceStack{}
			for _, stack := range(player.PieceStacks) {
				if stack.Type != "Staff" {
					newstacks = append(newstacks, stack)
				}
			}
			newstacks = append(newstacks, PieceStack{Type: "Staff", Amount: 1})
			player.PieceStacks = newstacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Staff"
		}
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string, attacker *Player) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType, attacker)
			if ok {
				return ok, err
			}
			if stackType != "Staff" {
				return false, nil
			}

			if err := t.checkZoneAccess(tile); err != nil {
				return true, fmt.Errorf("cannot access zone", err)
			}
			if err := t.checkAdjacency(tile, gs); err != nil {
				return true, fmt.Errorf("cannot reach zone", err)
			}

			if !gs.IsTribePresentOnTheBoard(t.Race) {
				return true, fmt.Errorf("you need to be already present on the board")
			}


			// var tileCost, moneyGainDefender, moneyLossAttacker int
			if tile.Presence == Active {
				// Maybe do something with those, in case this is actually considered a conquest
				_, _, _, err := tile.OwningTribe.countDefense(tile)
				if err != nil {
					return true, fmt.Errorf("Impossible to attack", err)
				}
			} else {
				return true, fmt.Errorf("This tile does not contain an active tribe")
			}

			for _, stack := range(tile.PieceStacks) {
				if stack.Type == string(tile.OwningTribe.Race) && stack.Amount > 1 {
					return true, fmt.Errorf("This tile contains more than one active pawn!")
				}
			}

			returnStacks, stayStacks := tile.OwningTribe.countReturningStacks(tile, gs)
			tile.OwningPlayer.PieceStacks = AddPieceStacks(tile.OwningPlayer.PieceStacks, returnStacks)
			tile.PieceStacks = AddPieceStacks([]PieceStack{{Type: string(t.Race), Amount: 1}}, stayStacks)
			tile.OwningPlayer = attacker
			tile.OwningTribe = t
			attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, []PieceStack{{Type: "Staff", Amount: 1}})
			return true, nil
		}
		}, Count: 5},
	"Wendigo": {Transform: func(t *Tribe) {
		oldgetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(player *Player) {
			oldgetStacksForConquestTurn(player)
			newstacks := []PieceStack{}
			for _, stack := range(player.PieceStacks) {
				if stack.Type != "Power" {
					newstacks = append(newstacks, stack)
				}
			}
			newstacks = append(newstacks, PieceStack{Type: "Power", Amount: 1})
			player.PieceStacks = newstacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Power"
		}
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string, attacker *Player) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType, attacker)
			if ok {
				return ok, err
			}
			if stackType != "Power" {
				return false, nil
			}

			if tile.Biome != Forest {
				return true, fmt.Errorf("This is not a forest!")
			}

			// var tileCost, moneyGainDefender, moneyLossAttacker int
			if tile.Presence == Active {
				// Maybe do something with those, in case this is actually considered a conquest
				_, _, _, err := tile.OwningTribe.countDefense(tile)
				if err != nil {
					return true, fmt.Errorf("Impossible to attack", err)
				}
			} 

			if tile.Presence != None {
				returnStacks, stayStacks := tile.OwningTribe.countReturningStacks(tile, gs)
				tile.OwningPlayer.PieceStacks = AddPieceStacks(tile.OwningPlayer.PieceStacks, returnStacks)
				tile.PieceStacks = stayStacks
			}
			tile.OwningPlayer = nil
			tile.OwningTribe = nil
			tile.Presence = None
			attacker.PieceStacks = AddPieceStacks(attacker.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}})
			attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, []PieceStack{{Type: "Power", Amount: 1}})
			return true, nil
		}
		}, Count: 6},
	"Gypsy": {Transform: func(t *Tribe) {
		t.State["abandonedTiles"] = []string{}
		oldReceiveAbandonment := t.receiveAbandonment
		t.receiveAbandonment = func(tile *Tile) []PieceStack {
			oldStacks := oldReceiveAbandonment(tile)
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Coin", Amount: 1}})
			log.Println(tile.PieceStacks)
			if abandonedTiles, ok := t.State["abandonedTiles"].([]string); ok {
				t.State["abandonedTiles"] = append(abandonedTiles, tile.Id)
			} else {
				// If it's not a slice of strings, reinitialize it
				t.State["abandonedTiles"] = []string{tile.Id}
			}
			return oldStacks
		}
		oldCheckZoneAccess := t.checkZoneAccess
		t.checkZoneAccess = func(tile *Tile) error {
			old := oldCheckZoneAccess(tile)
			if old == nil {
				if abandonedTileIds, ok := t.State["abandonedTiles"].([]string); ok {
					for _, id := range(abandonedTileIds) {
						if id == tile.Id {
							return fmt.Errorf("Abandoned zone cannot be reconquered")
						}
					}
				}
			}
			return old
		}
		oldCountExtrapoints := t.countExtrapoints
		t.countExtrapoints = func(gs *GameState) int {
			total := oldCountExtrapoints(gs)
			if abandonedTiles, ok := t.State["abandonedTiles"].([]string); ok {
				for _, tileId := range(abandonedTiles) {
					if tile, ok := gs.TileList[tileId]; ok {
						tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type:"Coin", Amount:1}})
						total += 1
					}
				}
			} 
			t.State["abandonedTiles"] = []string{}
			return total
		}
		}, Count: 6},
	"Scarecrow": {Transform: func(t *Tribe) {
		oldCountDefense:= t.countDefense
		t.countDefense = func(tile *Tile) (int, int, int, error) {
			a, b, c, err := oldCountDefense(tile)
			return a, b, c-1, err
		}
		}, Count: 11},
	"Kobold": {Transform: func(t *Tribe) {
		t.Minimum = 2
		}, Count: 11},
	// "Scavenger": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Priestess": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Leprechaun": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Faun": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Ghoul": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
}
