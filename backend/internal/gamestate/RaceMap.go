package gamestate

import (
	"fmt"
)

type RaceValue struct {
    Transform func(*Tribe)
    Count     int
}

var RaceMap = map[Race]RaceValue {
	"Troll": {Transform: func(t *Tribe) {
		// Make a newly conquered region contain a lair
		oldCountNewTileStacks := t.countNewTileStacks
		t.countNewTileStacks = func(stacks []PieceStack, tile *Tile) []PieceStack {
			oldstacks := oldCountNewTileStacks(stacks, tile)
			return AddPieceStacks(oldstacks, []PieceStack{{Type: "Lair", Amount: 1}})
		}

		// Make the defense of the tile + 1
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile)
			if err != nil {
				return old, g, l, err
			}
			return old+1, g, l, nil
		}

		oldCountPiecesRemaining := t.countPiecesRemaining
		t.countPiecesRemaining = func(tile *Tile) []PieceStack {
			oldstacks := oldCountPiecesRemaining(tile)
			return AddPieceStacks(oldstacks, []PieceStack{{Type: "Lair", Amount: 1}})
		}
		}, Count: 5},
	"Wizard": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive {
				for _, attr := range tile.Attributes {
					if attr == Magic {
						count += 1
					}
				}
			}
			return count
		}
		}, Count: 5},
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
	"Pygmy": {Transform: func(t *Tribe) {
		oldCountReturningStacks := t.countReturningStacks
		t.countReturningStacks = func(tile *Tile, gs *GameState) ([]PieceStack, []PieceStack) {
			a, b := oldCountReturningStacks(tile, gs)
			diceThrow := RollDice()
			gs.Messages = append(gs.Messages, fmt.Sprintf("Pygmies got back: %d Pawns!", diceThrow))
			
			a = AddPieceStacks(a, []PieceStack{{Type: string(t.Race), Amount: diceThrow}})
			return a, b
		}
		}, Count: 6},
	"Elf": {Transform: func(t *Tribe) {
		oldCountReturningStacks := t.countReturningStacks
		t.countReturningStacks = func(tile *Tile, gs *GameState) ([]PieceStack, []PieceStack) {
			a, b := oldCountReturningStacks(tile, gs)
			a = AddPieceStacks(a, []PieceStack{{Type: string(t.Race), Amount: 1}})
			return a, b
		}
		}, Count: 6},
	"Pixy": {Transform: func(t *Tribe) {
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			for _, tile := range gs.TileList {
				if tile.Presence != None && tile.OwningTribe.Race == t.Race {
					t.getStacksForConquest(tile, tile.OwningPlayer)
				}
			}
			return stacks
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
			if oldCanBeRedeployedIn(tile, stackType) {
				return stackType != string(t.Race)
			}
			return false
		}
		}, Count: 11},
	"Barbarian": {Transform: func(t *Tribe) {
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
			if oldCanBeRedeployedIn(tile, stackType) {
				return stackType != string(t.Race)
			}
			return false
		}
		oldGetStacksOutRedeployment := t.getStacksOutRedeployment
		t.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
			stacks, err := oldGetStacksOutRedeployment(tile, stackType)
			if err != nil {
				if stackType == string(t.Race) {
					return nil, fmt.Errorf("Barbarians cannot redeploy!")
				}
			}
			return stacks, err
		}
		}, Count: 8},
	// "Wendigo": {Transform: func(t *Tribe) {
	// 	}, Count: 5},
	// "Sorcerer": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Scavenger": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Scarecrow": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Priestess": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Leprechaun": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Kobold": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Gypsy": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Faun": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
	// "Ghoul": {Transform: func(t *Tribe) {
	// 	}, Count: 6},
}
