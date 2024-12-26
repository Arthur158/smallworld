package gamestate

import "fmt"

type TraitValue struct {
    Transform func(*Tribe)
    Count     int
}


var TraitMap = map[Trait]TraitValue {
	"Hill": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && tile.Biome == Hill {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 4},
	"Merchant": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 2},
	"Forest": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && tile.Biome == Forest {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 4},
	"Goldsmith": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			containsMine := false
			for _, attr := range tile.Attributes {
				if attr == Mine {
					containsMine = true
				}
			}
			if t.IsActive && containsMine {
				count += 2
			} else if t.IsActive {
				count -= 1
			}
			return max(0, count)
		}
		}, Count: 4},
	"Aquatic": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			isNextToWater := false
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.Biome == Water {
					isNextToWater = true
				}
			}
			if t.IsActive && isNextToWater {
				count += 1
			} else if t.IsActive {
				count -= 1
			}
			return max(0, count)
		}
		}, Count: 4},
	"Swamp": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && tile.Biome == Swamp {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 4},
	"Wealthy": {Transform: func(t *Tribe) {
		t.State["hasreceivedalready"] = false
		oldCountExtraPoints := t.countExtrapoints
		t.countExtrapoints = func() int {
			count := oldCountExtraPoints()
			if hasReceivedAlready, ok := t.State["hasreceivedalready"].(bool); ok && !hasReceivedAlready {
				count += 7
				t.State["hasreceivedalready"] = true
			}
			return count
		}
		}, Count: 4},
	"Alchemist": {Transform: func(t *Tribe) {
		oldCountExtraPoints := t.countExtrapoints
		t.countExtrapoints = func() int {
			count := oldCountExtraPoints()
			if t.IsActive {
				count += 2
			}
			return count
		}
		}, Count: 4},
	"Commando": {Transform: func(t *Tribe) {
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
			stacks, g, l := oldCountAttack(tile, cost, stackType)
			for i, stack := range stacks {
				if stack.Type == string(t.Race) {
					stacks[i].Amount -= 1
				}
			}
			return stacks, g, l
		}
		}, Count: 4},
	"Flying": {Transform: func(t *Tribe) {
		t.checkAdjacency = func(t *Tile, gs *GameState) error {
			return nil
		}
		}, Count: 4},
	"Hordes of": {Transform: func(t *Tribe) {
		}, Count: 7},
	"Seafaring": {Transform: func(t *Tribe) {
		t.checkZoneAccess = func(t *Tile) error {
			return nil
		}
		}, Count: 4},
	"Mounted": {Transform: func(t *Tribe) {
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
			stacks, g, l := oldCountAttack(tile, cost, stackType)
			if tile.Biome == Hill || tile.Biome == Field {
				for i, stack := range stacks {
					if stack.Type == string(t.Race) {
						stacks[i].Amount -= 1
					}
				}
			}
			return stacks, g, l
		}
		}, Count: 5},
	"Underworld": {Transform: func(t *Tribe) {
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
			stacks, g, l := oldCountAttack(tile, cost, stackType)
			for _, attr := range tile.Attributes {
				if attr == Cave {
					for i, stack := range stacks {
						if stack.Type == string(t.Race) {
							tempAmount := stacks[i].Amount - 1
							stacks[i].Amount = max(0, tempAmount)
						}
					}
				}
			}
			return stacks, g, l
		}
		oldCheckAdjacency := t.checkAdjacency
		// For the future, if this is too inefficient, we can try the semaphore approach
		t.checkAdjacency = func(tile *Tile, gs *GameState) error {
			err := oldCheckAdjacency(tile, gs)
			if err != nil {
				hasCave := false
				for _, attr := range tile.Attributes {
					if attr == Cave {
						hasCave = true
					}
				}
				if hasCave {
					for _, otherTile := range gs.TileList {
						if otherTile.Presence != None && otherTile.OwningTribe.Race == t.Race {
							for _, attr := range otherTile.Attributes {
								if attr == Cave {
									return nil
								}
							}
						}
					}
				}

			}
			return err
		}
		}, Count: 5},
			// if hasReceivedAlready, ok := t.State["hasreceivedalready"].(bool); ok && !hasReceivedAlready {
	"Berserk": {Transform: func(t *Tribe) {
		t.State["diceroll"] = RollDice()
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
			amount, _ := t.State["diceroll"].(int)
			return oldCountAttack(tile, cost - amount, stackType)
		}

		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps1, ps2 []PieceStack) ([]PieceStack, []PieceStack, bool, error) {
			stacks, stacksToRemove, diceUsed, err := oldCalculateRemainingAttackingStacks(ps1, ps2)
			if diceUsed {
				return nil, nil ,false, fmt.Errorf("Berserk tribe cannot use the dice twice!")
			}
			if err == nil {
				t.State["diceroll"] = RollDice()
			}
			return stacks, stacksToRemove, false, err
		}
		}, Count: 4},
	"Fortified": {Transform: func(t *Tribe) {
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			stacks = append(stacks, PieceStack{Type:"Fortress", Amount: 1})
			return stacks
		}
		oldCountPiecesRemaining := t.countPiecesRemaining
		t.countPiecesRemaining = func(tile *Tile) []PieceStack {
			stacks := oldCountPiecesRemaining(tile)
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Fortress" {
					stacks = append(stacks, stack)
				}
			}
			return stacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Encampment" {
					old += stack.Amount
				}
			}
			return old, g, l, nil
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
			if oldCanBeRedeployedIn(tile, stackType) {
				return true
			}
			if stackType == "Fortress" {
				for _, stack := range tile.PieceStacks {
					println(stack.Type)
					if stack.Type == "Fortress" {
						return false
					}
				}
				return true
			}
			return false
		}
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive {
				for _, stack := range tile.PieceStacks {
					if stack.Type == "Fortress" {
						count += 1
					}
				}
			}
			return count
		}
		}, Count: 3},
	"Bivouacking": {Transform: func(t *Tribe) {
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Encampment", Amount: 5}})
			return stacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Encampment" {
					old += stack.Amount
				}
			}
			return old, g, l, nil
		}

		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
			if oldCanBeRedeployedIn(tile, stackType) {
				return true
			}
			return stackType == "Encampment"
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Encampment" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = append(p.PieceStacks, stack)
				}
			}
		}
		oldGetStacksOutRedeployment := t.getStacksOutRedeployment
		t.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
			stacks, err := oldGetStacksOutRedeployment(tile, stackType)
			if err != nil {
				if stackType == "Encampment" {
					for _, stack := range tile.PieceStacks {
						if stack.Type == stackType {
							stack.Amount -= 1
							return []PieceStack{{Type: "Encampment", Amount: 1}}, nil
						}

					}
				}
			}
			return stacks, err
		}
		}, Count: 5},
	"Heroic": {Transform: func(t *Tribe) {
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Hero", Amount: 2}})
			return stacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Hero" {
					return 1000, g, l, fmt.Errorf("A tile with a hero cannot be conquered")
				}
			}
			return old, g, l, nil
		}

		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string) bool {
			if oldCanBeRedeployedIn(tile, stackType) {
				return true
			}
			return stackType == "Hero"
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Hero" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = append(p.PieceStacks, stack)
				}
			}
		}
		oldGetStacksOutRedeployment := t.getStacksOutRedeployment
		t.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
			stacks, err := oldGetStacksOutRedeployment(tile, stackType)
			if err != nil {
				if stackType == "Hero" {
					for _, stack := range tile.PieceStacks {
						if stack.Type == stackType {
							stack.Amount -= 1
							return []PieceStack{{Type: "Hero", Amount: 1}}, nil
						}

					}
				}
			}
			return stacks, err
		}
		}, Count: 5},
	"Dragon Master": {Transform: func(t *Tribe) {
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Dragon", Amount: 1}})
			return stacks
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Dragon" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = append(p.PieceStacks, stack)
				}
			}
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Dragon" {
					return 1000, g, l, fmt.Errorf("A tile with a dragon cannot be conquered")
				}
			}
			return old, g, l, nil
		}

		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(stackType string) bool {
			return stackType == "Dragon" || oldIsStackValid(stackType)
		}


		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
			old, g, l := oldCountAttack(tile, cost, stackType)
			if stackType == "Dragon" {
				return []PieceStack{{Type: string(t.Race), Amount: 1}, {Type: "Dragon", Amount: 1}}, g, l
			}
			return old, g, l
		}
		}, Count: 5},
	"Corrupt": {Transform: func(t *Tribe) {
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile)
			if err != nil {
				return old, g, l, err
			}
			if t.IsActive {
				g += 1
				l += 1
			}
			return old, g, l, err
		}
		}, Count: 4},
	"Ransacking": {Transform: func(t *Tribe) {
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
			old, g, l := oldCountAttack(tile, cost, stackType)
			if tile.Presence == Active {
				g += 1
				l += 1
			}
			return old, g , l
		}
		}, Count: 4},
	"Pillaging": {Transform: func(t *Tribe) {
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int) {
			old, g, l := oldCountAttack(tile, cost, stackType)
			if tile.Presence != None {
				g += 1
			}
			return old, g , l
		}
		}, Count: 5},
	// "Barricade": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Behemoth": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Catapult": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Fireball": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Imperial": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Mercenary": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Peace-loving": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Spirit": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
	// "Stout": {Transform: func(t *Tribe) {
	// 	}, Count: 4},
}
