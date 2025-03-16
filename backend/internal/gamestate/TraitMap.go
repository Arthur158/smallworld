package gamestate

import (
	"fmt"
	"log"
)



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
		t.countExtrapoints = func(gs *GameState) int {
			count := oldCountExtraPoints(gs)
			if hasReceivedAlready, ok := t.State["hasreceivedalready"].(bool); ok && !hasReceivedAlready {
				count += 7
				t.State["hasreceivedalready"] = true
			}
			return count
		}
		}, Count: 4},
	"Alchemist": {Transform: func(t *Tribe) {
		oldCountExtraPoints := t.countExtrapoints
		t.countExtrapoints = func(gs *GameState) int {
			count := oldCountExtraPoints(gs)
			if t.IsActive {
				count += 2
			}
			return count
		}
		}, Count: 4},
	"Commando": {Transform: func(t *Tribe) {
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			return oldcomputeDiscount(stackType, tile) + 1
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
		oldCheckZoneAccess := t.checkZoneAccess
		t.checkZoneAccess = func(tile *Tile) error {
			old := oldCheckZoneAccess(tile)
			if old != nil && tile.Biome == Water {
				return nil
			}
			return old
		}
		}, Count: 5},
	"Mounted": {Transform: func(t *Tribe) {
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			if (tile.Biome == Field || tile.Biome == Hill) && t.IsActive {
				return oldcomputeDiscount(stackType, tile) + 1
			}
			return oldcomputeDiscount(stackType, tile)
		}
		}, Count: 5},
	"Underworld": {Transform: func(t *Tribe) {
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			discount := 0
			for _, attr := range(tile.Attributes) {
				if attr == Cave {
					discount += 1
				}
			}
			return oldcomputeDiscount(stackType, tile) + discount
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
						if otherTile.CheckPresence() != None && otherTile.OwningTribe.Race == t.Race {
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
	"Berserk": {Transform: func(t *Tribe) {
		t.State["diceroll"] = RollDice()
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			amount, _ := t.State["diceroll"].(int)
			return oldcomputeDiscount(stackType, tile) + amount
		}
		oldGetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(p *Player, gs *GameState) {
			oldGetStacksForConquestTurn(p, gs)
			val := RollDice()
			t.State["diceroll"] = val
			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("New throw of dice for berserk tribe: %d", val)})
		}
		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			stacks, diceUsed, ok, err := oldCalculateRemainingAttackingStacks(ps, tile, gs)
			if err != nil {
				return stacks, diceUsed, ok, err
			}
			if diceUsed {
				return nil, true, false, fmt.Errorf("Berserk tribe cannot use the dice twice!")
			}
			val := RollDice()
			t.State["diceroll"] = val
			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("New throw of dice for berserk tribe: %d", val)})
			return stacks, false, true, nil
		}
		}, Count: 4},
	"Fortified": {Transform: func(t *Tribe) {
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			stacks = append(stacks, PieceStack{Type:"Fortress", Amount: 1})
			return stacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Fortress"
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Fortress" {
					old += stack.Amount
				}
			}
			return old, g, l, nil
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
				return true
			}
			if stackType == "Fortress" {
				for _, stack := range tile.PieceStacks {
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
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Fortress"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
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
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
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
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
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
					p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
				}
			}
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Encampment" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
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
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Encampment"
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Encampment" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Encampment"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				t.Owner.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{stack})
				return // Exit after removal to avoid index shifting issues
			    }
			}
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
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
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
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
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
					p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
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
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Hero"
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Hero" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Hero" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Hero"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
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
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
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
			return (stackType == "Dragon" && t.IsActive) || oldIsStackValid(stackType)
		}


		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			stacks, g, l, k := oldCountAttack(tile, cost, stackType)
			if stackType == "Dragon" {
				return []PieceStack{{Type: string(t.Race), Amount: 1}, {Type: "Dragon", Amount: 1}}, g, l, k
			}
			return stacks, g, l, k
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Dragon" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Dragon" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		}, Count: 5},
	"Corrupt": {Transform: func(t *Tribe) {
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
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
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			old, g, l, k := oldCountAttack(tile, cost, stackType)
			if tile.CheckPresence() == Active {
				g += 1
				l += 1
			}
			return old, g , l, k
		}
		}, Count: 4},
	"Pillaging": {Transform: func(t *Tribe) {
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			old, g, l, k := oldCountAttack(tile, cost, stackType)
			if tile.CheckPresence() != None && t.IsActive {
				g += 1
			}
			return old, g , l, k
		}
		}, Count: 5},
	"Barricade": {Transform: func(t *Tribe) {
		oldCountExtraPoints := t.countExtrapoints
		t.countExtrapoints = func(gs *GameState) int {
			old := oldCountExtraPoints(gs)
			if t.IsActive {
				count := 0
				for _, tile := range(gs.TileList) {
					if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
						count += 1
					}
				}
				if count <= 4 {
					old += 3
				}
			}
			return old
		}
		}, Count: 4},
	"Behemoth": {Transform: func(t *Tribe) {
		addBehemoth := func(gs *GameState) {
			maleFound, femaleFound := false, false
			for _, tile := range(gs.TileList) {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					for _, stack := range(tile.PieceStacks) {
						if stack.Type == "Male Behemoth" {
							tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Male Behemoth", Amount: 1}})
							maleFound = true
						}
						if stack.Type == "Female Behemoth" {
							tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Female Behemoth", Amount: 1}})
							femaleFound = true
						}
					}
				}
			}
			if !maleFound {
				t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Male Behemoth", Amount: 1}})
			}
			if !femaleFound {
				t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Female Behemoth", Amount: 1}})
			}
		}
		deleteBehemoth := func(gs *GameState) {
			maleFound, femaleFound := false, false
			for _, tile := range(gs.TileList) {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					for _, stack := range(tile.PieceStacks) {
						if stack.Type == "Male Behemoth" {
							tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Male Behemoth", Amount: 1}})
							maleFound = true
						}
						if stack.Type == "Female Behemoth" {
							tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Female Behemoth", Amount: 1}})
							femaleFound = true
						}
					}
				}
			}
			if !maleFound {
				t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Male Behemoth", Amount: 1}})
			}
			if !femaleFound {
				t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Female Behemoth", Amount: 1}})
			}
		}
		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			stacks, diceUsed, ok, err := oldCalculateRemainingAttackingStacks(ps, tile, gs)
			if err != nil || !ok {
				return stacks, diceUsed, ok, err
			}
			if tile.Biome == Swamp {
				addBehemoth(gs)
				for i := range(stacks) {
					if stacks[i].Type == "Male Behemoth" || stacks[i].Type == "Female Behemoth" {
						stacks[i].Amount += 1
					}
				}
			}
			return stacks, diceUsed, ok, err
		}
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i := len(tile.PieceStacks) - 1; i >= 0; i-- { // Loop backward
			    stack := tile.PieceStacks[i]
			    if stack.Type == "Male Behemoth" || stack.Type == "Female Behemoth" {
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{stack})
			    }
			}
			if tile.Biome == Swamp {
				deleteBehemoth(gs)
			}
		}
		oldgetRedeploymentStack := t.getRedeploymentStack
		t.getRedeploymentStack = func(s string, ps []PieceStack) []PieceStack {
			if s == "Female Behemoth" || s == "Male Behemoth" {
				amount := 0
				for _, stack := range(ps) {
					if stack.Type == s {
						amount = stack.Amount
					}
				}
				return []PieceStack{{Type: s, Amount: amount}}
			}
			return oldgetRedeploymentStack(s, ps)
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Male Behemoth" || stack.Type == "Female Behemoth" {
					old += stack.Amount
				}
			}
			return old, g, l, nil
		}
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			stacks, a, b, c := oldCountAttack(tile, cost, stackType)
			if stackType == "Male Behemoth" || stackType == "Female Behemoth" {
				for _, stack := range(t.Owner.PieceStacks) {
					if stack.Type == stackType {
						return []PieceStack{stack, {Type: string(t.Race), Amount: max(t.Minimum, cost - stack.Amount)}}, a, b, c
					}
				}
			}
			return stacks, a, b, c
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(stackType string) bool {
			return (stackType == "Female Behemoth" && t.IsActive) || (stackType == "Male Behemoth" && t.IsActive) || oldIsStackValid(stackType)
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
				return true
			}
			return stackType == "Female Behemoth" || stackType == "Male Behemoth"
		}
		oldGetStacksOutRedeployment := t.getStacksOutRedeployment
		t.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
			stacks, err := oldGetStacksOutRedeployment(tile, stackType)
			if err != nil {
				if stackType == "Female Behemoth" || stackType == "Male Behemoth" {
					for _, stack := range tile.PieceStacks {
						if stack.Type == stackType {
							return []PieceStack{{Type: stackType, Amount: stack.Amount}}, nil
						}

					}
				}
			}
			return stacks, err
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
		    oldGetStacksForConquest(tile, p)

		    for i := len(tile.PieceStacks) - 1; i >= 0; i-- { // Loop backward
			stack := tile.PieceStacks[i]
			if stack.Type == "Female Behemoth" || stack.Type == "Male Behemoth" {
			    // Remove stack safely
			    tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
			    p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
			}
		    }
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Female Behemoth" || stack.Type == "Male Behemoth" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Female Behemoth" || stack.Type == "Male Behemoth" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}

		}, Count: 4},
	"Catapult": {Transform: func(t *Tribe) {
		t.State["justPlaced"] = false
		t.State["justUsed"] = false
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Catapult", Amount: 1}})
			return stacks
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Catapult" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = append(p.PieceStacks, stack)
				}
			}
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Catapult" {
					return 1000, g, l, fmt.Errorf("A tile with a catapult cannot be conquered")
				}
			}
			return old, g, l, nil
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(stackType string) bool {
			return (stackType == "Catapult" && t.IsActive) || oldIsStackValid(stackType)
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Catapult" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Catapult" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType)
			if ok {
				return ok, err
			}
			if stackType != "Catapult" {
				return false, nil
			}

			if !tile.OwningTribe.checkPresence(tile, t.Race)  {
				return true, fmt.Errorf("Needs to be the tribe's own tile")
			}

			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Catapult", Amount: 1}})
			t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Catapult", Amount: 1}})

			t.State["justPlaced"] = true
			gs.Messages = append(gs.Messages,Message{Content:  "The catapult was just placed!"})

			return true, nil
		}
		oldcheckAdjacency := t.checkAdjacency
		t.checkAdjacency = func(tile *Tile, gs *GameState) error {
			err := oldcheckAdjacency(tile, gs)
			if b, ok := t.State["justPlaced"].(bool); ok && !b {
				return err
			}
			t.State["justPlaced"] = false
			for _, tile2 := range(gs.TileList) {
				if tile2.CheckPresence() != None && tile2.OwningTribe.checkPresence(tile2, t.Race) {
					for _, stack := range(tile2.PieceStacks) {
						if stack.Type == "Catapult" && gs.CheckJump(tile, tile2) {
							t.State["justUsed"] = true
							return nil
						}
					}
				}
			}
			return err
		}
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			if b, ok := t.State["justUsed"].(bool); ok && b {
				t.State["justUsed"] = false
				return oldcomputeDiscount(stackType, tile) + 1
			}
			return oldcomputeDiscount(stackType, tile)
		}
		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			stacks, a, b, err := oldCalculateRemainingAttackingStacks(ps, tile, gs)
			t.State["justUsed"], t.State["justPlaced"] = false, false
			return stacks, a, b, err
		}
		}, Count: 4},
	"Stout": {Transform: func(t *Tribe) {
		t.State["extrapoints"] = 0
		t.canGoIntoDecline = func(gs *GameState) bool {
			return true
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			pluspoints := gs.countPoints(t.Owner)
			gs.GetPieceStackForConquest(t.Owner)
			oldgoIntoDecline(gs)
			minuspoints := gs.countPoints(t.Owner)
			t.State["extrapoints"] = pluspoints - minuspoints
		}
		oldCountExtraPoints := t.countExtrapoints
		t.countExtrapoints = func(gs *GameState) int {
			count := oldCountExtraPoints(gs)
			extraPoints := t.State["extrapoints"].(int)
			t.State["extrapoints"] = 0
			return count + extraPoints
		}
		}, Count: 4},
	"Fireball": {Transform: func(t *Tribe) {
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			count := 0
			for _, tile := range(gs.TileList) {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					for _, attr := range(tile.Attributes) {
						if attr == Magic {
							count += 1
						}
					}
				}
			}
			stacks = append(stacks, PieceStack{Type:"Fireball", Amount: count})
			return stacks
		}
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			stacks, a, b, c := oldCountAttack(tile, cost, stackType)
			if stackType == "Fireball" {
				for _, stack := range(t.Owner.PieceStacks) {
					if stack.Type == stackType {
						return []PieceStack{{Type: "Fireball", Amount: 1}, {Type: string(t.Race), Amount: max(t.Minimum, cost - 2)}}, a, b, c
					}
				}
			}
			return stacks, a, b, c
		}
		oldcountNewTileStacks := t.countNewTileStacks
		t.countNewTileStacks = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
			stacks := oldcountNewTileStacks(ps, tile, gs)
			for i := range(stacks) {
				if stacks[i].Type == "Fireball" {
					return append(stacks[:i], stacks[i+1:]...)
				}
			}
			return stacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Fireball" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Fireball"
		}
		}, Count: 5},
	"Imperial": {Transform: func(t *Tribe) {
		oldCountExtraPoints := t.countExtrapoints
		t.countExtrapoints = func(gs *GameState) int {
			count := oldCountExtraPoints(gs)
			amount := 0
			for _, tile := range(gs.TileList) {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					amount += 1
				}
			}
			return count + max(0, amount - 3)
		}
		}, Count: 4},
	"Mercenary": {Transform: func(t *Tribe) {
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Mercenary", Amount: 1}})
			return stacks
		}
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			stacks, moneyGainAttacker, b, c := oldCountAttack(tile, cost, stackType)
			if stackType == "Mercenary" {
				return []PieceStack{{Type: string(t.Race), Amount: max(t.Minimum, cost - 2 - t.computeDiscount(stackType, tile))}}, moneyGainAttacker - 1, b, c
			}
			return stacks, moneyGainAttacker, b, c
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Mercenary" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Mercenary"
		}
		}, Count: 4},
	"Peace-loving": {Transform: func(t *Tribe) {
		t.State["hasattacked"] = false
		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			stacks, diceUsed, ok, err := oldCalculateRemainingAttackingStacks(ps, tile, gs)
			if err != nil  || !ok{
				return stacks, diceUsed, ok, err
			}
			if tile.CheckPresence() == Active {
				t.State["hasattacked"] = true
			}
			return stacks, diceUsed, true, nil
		}
		oldCountExtraPoints := t.countExtrapoints
		t.countExtrapoints = func(gs *GameState) int {
			if hasAttacked, ok := t.State["hasattacked"].(bool); ok && !hasAttacked {
				return oldCountExtraPoints(gs) + 3
			}
			t.State["hasattacked"] = false
			return oldCountExtraPoints(gs)
		}
		}, Count: 4},
	"Spirit": {Transform: func(t *Tribe) {
		t.prepareRemoval = func(gs *GameState) bool {
			return false
		}
		}, Count: 4},
	"Lava": {Transform: func(t *Tribe) {
		t.State["mountains"] = []string{}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Lava"
		}
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			count := 0
			mountains, _ := t.State["mountains"].([]string)
			for _, tile := range(gs.TileList) {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) && tile.Biome == Mountain {
					mountains = append(mountains, tile.Id)
					count += 1
				}
			}
			t.State["mountains"] = mountains
			stacks = append(stacks, PieceStack{Type:"Lava", Amount: count})
			return stacks
		}
		oldhandleDeploymentIn := t.handleDeploymentIn
		t.handleDeploymentIn = func(tile *Tile, stackType string, i int, gs *GameState) error {
			err := oldhandleDeploymentIn(tile, stackType, i, gs)
			if err == nil {
				return nil
			}
			if stackType != "Lava" {
				return err
			}

			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Lava" {
					return fmt.Errorf("There is already lava here!")
				}
			}

			found := false
			mountains, _ := t.State["mountains"].([]string)
			for _, neighbor := range(tile.AdjacentTiles) {
				if neighbor.Biome == Mountain {
					for i, id := range(mountains) {
						if neighbor.Id == id {
							found = true
							t.State["mountains"] = append(mountains[:i], mountains[i+1:]...)
						}
					}
				}
			}
			if !found {
				return fmt.Errorf("No adjacent mountain!")
			}

			player := t.Owner

			movingStack := t.getRedeploymentStack(stackType, player.PieceStacks)

			newStacks, ok := SubtractPieceStacks(player.PieceStacks, movingStack)
			if !ok {
				return fmt.Errorf("Cannot redeploy pieces you don't have")
			}
			player.PieceStacks = newStacks

			if tile.CheckPresence() != None {
				tile.OwningTribe.handleReturn(tile, gs, 0)
			}

			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)
			tile.ModifierDefenses["Lava"] = TileModifierDefenses["Lava"]
			return nil
		}
		oldgetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(p *Player, gs *GameState) {
			for _, tile := range(gs.TileList) {
				for i := range(tile.PieceStacks) {
					if tile.PieceStacks[i].Type == "Lava" {
						tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
						delete(tile.ModifierDefenses, "Lava")
					}
				}
			}
			for i := range(p.PieceStacks) {
				if p.PieceStacks[i].Type == "Lava" {
					p.PieceStacks = append(p.PieceStacks[:i], p.PieceStacks[i+1:]...)
					break;
				}
			}
			t.State["mountains"] = []string{}
			oldgetStacksForConquestTurn(p, gs)
		}
		}, Count: 5},
	"Diplomat": {Transform: func(t *Tribe) {
		t.State["playersAttacked"] = []int{}
		oldHandleopponentAction := t.handleOpponentAction
		t.handleOpponentAction = func(stackType string, opponent *Player, gs *GameState) error {
			err := oldHandleopponentAction(stackType, opponent, gs)
			if err == nil {
				return nil
			}
			if stackType == "Diplomat" {
				if gs.TurnInfo.Phase != Redeployment {
					return fmt.Errorf("You must be in Redeployment to establish a peace treaty!")
				}

				playersAttacked, _ := t.State["playersAttacked"].([]int)
				for _, index := range(playersAttacked) {
					if index == opponent.Index {
						return fmt.Errorf("You have attacked this player during your turn!")
					}
				}

				opponent.PieceStacks = AddPieceStacks(opponent.PieceStacks, []PieceStack{{Type: "Diplomat", Amount: 1}})
				t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Diplomat", Amount: 1}})
				return nil
			}
			return err
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Diplomat"
		}
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Diplomat", Amount: 1}})
			return stacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range p.PieceStacks {
				if stack.Type == "Diplomat" {
					return old, g, l, fmt.Errorf("You have a Peace pact with this player!")
				}
			}
			return old, g, l, nil
		}
		oldcalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			stacks, b1, b2, err := oldcalculateRemainingAttackingStacks(ps, tile, gs)
			if err != nil {
				return stacks, b1, b2, err
			}
			if tile.CheckPresence() == Active && tile.OwningTribe != nil {
				playersAttacked, _ := t.State["playersAttacked"].([]int)
				playersAttacked = append(playersAttacked, tile.OwningTribe.Owner.Index)
				t.State["playersAttacked"] = playersAttacked
			}
			return stacks, b1, b2, err
		}
		oldGetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(p *Player, gs *GameState) {
			oldGetStacksForConquestTurn(p, gs)
			t.State["playersAttacked"] = []int{}
			for _, player := range(gs.Players) {
				for i := range(player.PieceStacks) {
					if player.PieceStacks[i].Type == "Diplomat" && player != p {
						t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Diplomat", Amount: 1}})
						player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, []PieceStack{{Type: "Diplomat", Amount: 1}})
						return

					}
				}
			}
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Diplomat" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		}, Count: 5},
	"Haggling": {Transform: func(t *Tribe) {
		oldHandleopponentAction := t.handleOpponentAction
		t.handleOpponentAction = func(stackType string, opponent *Player, gs *GameState) error {
			err := oldHandleopponentAction(stackType, opponent, gs)
			if err == nil {
				return nil
			}
			if stackType == "Treaty" {
				if gs.TurnInfo.Phase != Redeployment {
					return fmt.Errorf("You must be in Redeployment to establish a peace treaty!")
				}

				opponent.PieceStacks = AddPieceStacks(opponent.PieceStacks, []PieceStack{{Type: "Treaty", Amount: 1}})
				t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Treaty", Amount: 1}})
				return nil
			}
			return err
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Treaty"
		}
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Treaty", Amount: 5}})
			return stacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range p.PieceStacks {
				if stack.Type == "Treaty" {
					l += stack.Amount
					g += stack.Amount
					p.PieceStacks, _ = SubtractPieceStacks(p.PieceStacks, []PieceStack{stack})
					t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{stack})
				}
			}
			return old, g, l, nil
		}
		oldGetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(p *Player, gs *GameState) {
			oldGetStacksForConquestTurn(p, gs)
			for _, player := range(gs.Players) {
				for i := range(player.PieceStacks) {
					if player.PieceStacks[i].Type == "Treaty" && player != p {
						t.Owner.CoinPile -= player.PieceStacks[i].Amount
						player.CoinPile += player.PieceStacks[i].Amount
						t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Treaty", Amount: player.PieceStacks[i].Amount}})
						player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, []PieceStack{{Type: "Treaty", Amount: player.PieceStacks[i].Amount}})
						break

					}
				}
			}
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Treaty" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		}, Count: 5},
	"Zeppelined": {Transform: func(t *Tribe) {
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Zeppelin"
		}
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Zeppelin", Amount: 5}})
			return stacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Zeppelin" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType)
			if ok {
				return ok, err
			}
			if stackType != "Zeppelin" {
				return false, nil
			}

			if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
				return true, fmt.Errorf("This tile already belongs to the tribe!")
			}

			if err := t.checkZoneAccess(tile); err != nil {
				return true, err
			}

			found := false
			for _, stack := range(t.Owner.PieceStacks) {
				if stack.Type == string(t.Race) {
					found = true
				}
			}

			if !found {
				return true, fmt.Errorf("You need at least one Race token to launch a zeppelin attack!")
			}

			tileCost, moneyGainDefender, moneyLossAttacker := 0, 0, 0
			if tile.CheckPresence() != None {
				tileCost, moneyGainDefender, moneyLossAttacker, err = tile.OwningTribe.countDefense(tile, t.Owner, gs)
			} else {
				tileCost, moneyGainDefender, moneyLossAttacker, err = tile.countDefense(gs)
			}
			
			if err != nil {
				return true, err
			}

			diceThrow := RollDice()

			if diceThrow == 0 {
				gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("Failure: The throw of dice for zeppelined tribe was: %d", diceThrow)})
				t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}, {Type: "Zeppelin", Amount: 1}})
				if tile.CheckPresence() != None {
					tile.OwningTribe.handleReturn(tile, gs, 1)
				}
				tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Burning Zeppelin", Amount: 1}})
				tile.ModifierDefenses["Burning Zeppelin"] = TileModifierDefenses["Burning Zeppelin"]
				return true, nil
			}

			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("Success: The throw of dice for zeppelined tribe was: %d", diceThrow)})
			// counts the cost for the attacker
			attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill := t.countAttack(tile, tileCost - diceThrow, string(t.Race))
			attackCostStacks = append(attackCostStacks, PieceStack{Type: "Zeppelin", Amount: 1})
			newStacks, hasDiceBeenUsed, _, err := t.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
			if err != nil {
				return true, err
			}

			if hasDiceBeenUsed {
				return true, fmt.Errorf("The dice cannot be used twice on a Zeppelin attack!")
			}

			// Enact changes
			if tile.CheckPresence() != None {
				tile.OwningTribe.Owner.CoinPile += moneyGainDefender - moneyLossDefender
				tile.OwningTribe.handleReturn(tile, gs, pawnKill)
			}

			newTileStacks := t.countNewTileStacks(newStacks, tile, gs)
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, newTileStacks)
			tile.handleAfterConquest(gs)

			t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, newStacks)
			t.Owner.CoinPile += moneyGainAttacker - moneyLossAttacker
			tile.OwningTribe = t

			if hasDiceBeenUsed {
				return true, gs.HandleStartRedeployment(t.Owner.Index)
			} else {
				gs.TurnInfo.Phase = Conquest
			}

			return true, nil
		}
		oldgetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(p *Player, gs *GameState) {
			for _, tile := range(gs.TileList) {
				for i := range(tile.PieceStacks) {
					if tile.PieceStacks[i].Type == "Burning Zeppelin" || tile.PieceStacks[i].Type == "Zeppelin" {
						tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
						delete(tile.ModifierDefenses, "Burning Zeppelin")
						t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Zeppelin", Amount: 1}})
					}
				}
			}
			oldgetStacksForConquestTurn(p, gs)
		}
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Zeppelin" {
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				t.Owner.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{stack})
				return // Exit after removal to avoid index shifting issues
			    }
			}
		}
		}, Count: 5},
	"Gunner": {Transform: func(t *Tribe) {
		t.State["leftcannonplayed"] = false
		t.State["rightcannonplayed"] = false
		oldHandleMovement := t.handleMovement
		t.handleMovement = func(stackType string, tileFrom, tileTo *Tile, gs *GameState) error {
			err := oldHandleMovement(stackType, tileFrom, tileTo, gs)
			if err == nil {
				return nil
			}
			if stackType != "Left Cannon" && stackType != "Right Cannon" {
				return err
			}

			if tileFrom == tileTo {
				return fmt.Errorf("Cannot move cannon on its own tile!")
			}

			if tileFrom.CheckPresence() == None || !tileFrom.OwningTribe.checkPresence(tileTo, t.Race) {
				return fmt.Errorf("Invalid starting tile")
			}
			if tileTo.CheckPresence() == None || !tileTo.OwningTribe.checkPresence(tileFrom, t.Race) {
				return fmt.Errorf("Invalid arriving tile")
			}

			found := false
			for _, stack := range(tileFrom.PieceStacks) {
				if stack.Type == stackType {
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("No cannon to move!")
			}

			hasPlayed := false
			if stackType == "Left Cannon" {
				hasPlayed = t.State["leftcannonplayed"].(bool)
			} else {
				hasPlayed = t.State["rightcannonplayed"].(bool)
			}
			if hasPlayed {
				return fmt.Errorf("This cannon has already moved!")
			}

			tileFrom.PieceStacks, _ = SubtractPieceStacks(tileFrom.PieceStacks, []PieceStack{{Type: stackType, Amount: 1}})
			tileTo.PieceStacks = AddPieceStacks(tileTo.PieceStacks, []PieceStack{{Type: stackType, Amount: 1}})

			if stackType == "Left Cannon" {
				t.State["leftcannonplayed"] = true
			} else {
				t.State["rightcannonplayed"] = true
			}

			return nil
		}
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Cannon Trigger", Amount: 1}, {Type: "Left Cannon", Amount: 1}, {Type: "Right Cannon", Amount: 1}})
			return stacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Cannon Trigger" || s == "Left Cannon" || s == "Right Cannon"
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			return oldCanBeRedeployedIn(tile, stackType, gs) || stackType == "Left Cannon" || stackType == "Right Cannon"
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Left Cannon" || stack.Type == "Right Cannon" || stack.Type == "Firing Left Cannon" || stack.Type == "Firing Right Cannon"{
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Left Cannon" || stack.Type == "Right Cannon" || stack.Type == "Firing Left Cannon" || stack.Type == "Firing Right Cannon" {
					return 1000, g, l, fmt.Errorf("A tile with a cannon cannot be conquered")
				}
			}
			return old, g, l, nil
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Left Cannon" || stack.Type == "Right Cannon" || stack.Type == "Cannon Trigger" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType)
			if ok {
				return ok, err
			}
			if stackType != "Cannon Trigger" {
				return false, nil
			}

			if !tile.OwningTribe.checkPresence(tile, t.Race)  {
				return true, fmt.Errorf("Needs to be the tribe's own tile")
			}

			found := false
			cannonType := ""
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Left Cannon" || stack.Type == "Right Cannon" {
					found = true
					cannonType = stack.Type
					break
				}
			}
			if !found {
				return true, fmt.Errorf("No cannon to trigger!")
			}

			hasPlayed := false
			if cannonType == "Left Cannon" {
				hasPlayed = t.State["leftcannonplayed"].(bool)
			} else {
				hasPlayed = t.State["rightcannonplayed"].(bool)
			}
			if hasPlayed {
				return true, fmt.Errorf("This cannon has already moved!")
			}
			
			if cannonType == "Left Cannon" {
				tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Firing Left Cannon", Amount: 1}})
				tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Left Cannon", Amount: 1}})
			} else {
				tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Firing Right Cannon", Amount: 1}})
				tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Right Cannon", Amount: 1}})
			}

			if stackType == "Left Cannon" {
				t.State["leftcannonplayed"] = true
			} else {
				t.State["rightcannonplayed"] = true
			}

			gs.Messages = append(gs.Messages, Message{Content: "A cannon was just triggered!"})

			return true, nil
		}
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			discount := oldcomputeDiscount(stackType, tile)
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.CheckPresence() != None && neighbour.OwningTribe.checkPresence(neighbour, t.Race) {
					for _, stack := range(neighbour.PieceStacks) {
						if stack.Type == "Firing Left Cannon" || stack.Type == "Firing Right Cannon" {
							discount += 2
						}
					}
				}
			}
			return discount
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Firing Left Cannon" {
					tile.PieceStacks[i].Type = "Left Cannon"
				}
				if stack.Type == "Firing Right Cannon" {
					tile.PieceStacks[i].Type = "Right Cannon"
				}
			}
			t.State["leftcannonplayed"] = false
			t.State["rightcannonplayed"] = false
		}
		}, Count: 3},
	"Tomb": {Transform: func(t *Tribe) {
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			newstacks := []PieceStack{}
			for _, stack := range(oldStacks) {
				if stack.Type != string(t.Race) {
					stack.Tribe  = t
					newstacks = append(newstacks, stack)
				}
			}
			return newstacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || (s == string(t.Race) && !t.IsActive)
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			oldgoIntoDecline(gs)
			log.Println("we here")
			gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
				player: gs.TurnInfo.PlayerIndex,
				TurnInfo: &TurnInfo{
					TurnIndex: gs.TurnInfo.TurnIndex,
					PlayerIndex: gs.TurnInfo.PlayerIndex,
					Phase: Redeployment,
				},
			})

		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for i, stack := range(oldStacks) {
				if stack.Type == string(t.Race) {
					oldStacks = append(oldStacks[:i], oldStacks[i+1:]...)
					break
				}
			}
			return oldStacks
		}
		oldHandleReturn := t.handleReturn
		t.handleReturn = func(tile *Tile, gs *GameState, pk int) {
			oldHandleReturn(tile, gs, pk)
			if !t.IsActive {
				for _, stack := range(t.Owner.PieceStacks) {
					if stack.Type == string(t.Race) {
						found := false
						for _, entry := range(gs.ModifierTurnsAfter) {
							if entry.player == t.Owner.Index && entry.TurnInfo.TurnIndex == gs.TurnInfo.TurnIndex && entry.TurnInfo.Phase == Redeployment {
								found = true
							}
						}
						if !found {
							gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
								player: gs.TurnInfo.PlayerIndex,
								TurnInfo: &TurnInfo{
									TurnIndex: gs.TurnInfo.TurnIndex,
									PlayerIndex: t.Owner.Index,
									Phase: Redeployment,
								},
								actionBefore: func(gs *GameState) {},
							})

						}
					}
				}
			}
		}
		}, Count: 5},
	"Royal": {Transform: func(t *Tribe) {
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Queen", Amount: 1}})
			return stacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p, gs)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Queen" {
					return 1000, g, l, fmt.Errorf("A tile with the queen cannot be conquered!")
				}
			}
			return old, g, l, nil
		}

		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
				return true
			}
			return stackType == "Queen"
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			if !t.IsActive {
				return
			}
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Queen" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
				}
			}
		}
		oldGetStacksOutRedeployment := t.getStacksOutRedeployment
		t.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
			stacks, err := oldGetStacksOutRedeployment(tile, stackType)
			if err != nil && t.IsActive {
				if stackType == "Queen" {
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
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Queen"
		}
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Queen"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			oldgoIntoDecline(gs)
			gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
				player: t.Owner.Index,
				TurnInfo: &TurnInfo{
					TurnIndex: gs.TurnInfo.TurnIndex,
					PlayerIndex: gs.TurnInfo.PlayerIndex,
					Phase: Redeployment,
				},
				actionBefore: func(gs *GameState) {},
			})

		}
		}, Count: 5},
	"Immortal": {Transform: func(t *Tribe) {
		oldHandleReturn := t.handleReturn
		t.handleReturn = func(tile *Tile, gs *GameState, pk int) {
			if t.IsActive {
				oldHandleReturn(tile, gs, 0)
			} else {
				oldHandleReturn(tile, gs, pk)
			}
		}
		}, Count: 5},
}
