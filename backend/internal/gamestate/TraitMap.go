package gamestate

import (
	"fmt"
)



type TraitValue struct {
    Transform func(*Tribe)
    Count     int
}


var TraitMap = map[Trait]TraitValue {
	"Hill": {Transform: func(t *Tribe) {
		t.countPointsMap["Hill"] = func(tile *Tile) int {
			if t.IsActive && tile.Biome == Hill {
				return 1
			} 
			return 0
		}
		}, Count: 4},
	"Merchant": {Transform: func(t *Tribe) {
		t.countPointsMap["Merchant"] = func(tile *Tile) int {
			if t.IsActive {
				return 1
			} 
			return 0
		}
		}, Count: 2},
	"Forest": {Transform: func(t *Tribe) {
		t.countPointsMap["Forest"] = func(tile *Tile) int {
			if t.IsActive && tile.Biome == Forest {
				return 1
			} 
			return 0
		}
		}, Count: 4},
	"Goldsmith": {Transform: func(t *Tribe) {
		t.countPointsMap["Goldsmith"] = func(tile *Tile) int {
			if !t.IsActive {
				return 0
			}
			containsMine := false
			for _, attr := range tile.Attributes {
				if attr == Mine {
					containsMine = true
				}
			}
			if containsMine {
				return 2
			} 
			return -1
		}
		}, Count: 4},
	"Aquatic": {Transform: func(t *Tribe) {
		t.countPointsMap["Aquatic"] = func(tile *Tile) int {
			if !t.IsActive {
				return 0
			}
			isNextToWater := false
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.Biome == Water {
					isNextToWater = true
				}
			}
			if isNextToWater {
				return 1
			} 
			return -1
		}
		}, Count: 3},
	"Swamp": {Transform: func(t *Tribe) {
		t.countPointsMap["Swamp"] = func(tile *Tile) int {
			if t.IsActive && tile.Biome == Swamp {
				return 1
			} 
			return 0
		}
		}, Count: 4},
	"Wealthy": {Transform: func(t *Tribe) {
		_, ok := t.State["hasreceivedalready"].(bool)
		if !ok {
			t.State["hasreceivedalready"] = false
		}
		t.countExtrapointsMap["Wealthy"] = func(gs *GameState) int {
			if hasReceivedAlready, ok := t.State["hasreceivedalready"].(bool); ok && !hasReceivedAlready {
				t.State["hasreceivedalready"] = true
				return 7
			}
			return 0
		}
		}, Count: 4},
	"Alchemist": {Transform: func(t *Tribe) {
		t.countExtrapointsMap["Alchemist"] = func(gs *GameState) int {
			if t.IsActive {
				return 2
			}
			return 0
		}
		}, Count: 4},
	"Commando": {Transform: func(t *Tribe) {
		t.computeDiscountMap["Commando"] = func(tile *Tile) int {
			return 1
		}
		}, Count: 4},
	"Flying": {Transform: func(t *Tribe) {
		t.checkAdjacencyMap["Flying"] = func(t *Tile, gs *GameState, err error) error {
			return nil
		}
		}, Count: 4},
	"Hordes of": {Transform: func(t *Tribe) {
		}, Count: 7},
	"Seafaring": {Transform: func(t *Tribe) {
		t.checkZoneAccessMap["Seafaring"] = func(tile *Tile, old error) error {
			if old != nil && tile.Biome == Water {
				return nil
			}
			return old
		}
		}, Count: 5},
	"Mounted": {Transform: func(t *Tribe) {
		t.computeDiscountMap["Mounted"] = func(tile *Tile) int {
			if (tile.Biome == Field || tile.Biome == Hill) && t.IsActive {
				return 1
			}
			return 0
		}
		}, Count: 5},
	"Underworld": {Transform: func(t *Tribe) {
		t.computeDiscountMap["Underworld"] = func(tile *Tile) int {
			discount := 0
			for _, attr := range(tile.Attributes) {
				if attr == Cave {
					discount += 1
				}
			}
			return discount
		}
		t.checkAdjacencyMap["Underworld"] = func(tile *Tile, gs *GameState, err error) error {
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
		//here
		t.State["diceroll"] = RollDice()
		t.computeDiscountMap["Berserk"] = func(tile *Tile) int {
			if !t.IsActive {
				return 0
			}
			val := t.State["diceroll"]
			var amount int
			switch v := val.(type) {
			case float64:
			    amount = int(v)
			case int:
			    amount = v
			}
			return amount
		}
		t.getStacksForConquestTurnMap["Berserk"] = func(p *Player, gs *GameState) {
			if !t.IsActive {
				return
			}
			val := RollDice()
			t.State["diceroll"] = val
			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("New throw of dice for berserk tribe: %d", val)})
		}
		t.calculateRemainingAttackingStacksMap["Berserk"] = func(ps []PieceStack, diceUsed bool, ok bool, err error, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			if !t.IsActive {
				return ps, diceUsed, ok, err
			}
			if diceUsed {
				return nil, true, false, fmt.Errorf("Berserk tribe cannot use the dice twice!")
			}
			val := RollDice()
			t.State["diceroll"] = val
			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("New throw of dice for berserk tribe: %d", val)})
			return ps, false, true, nil
		}
		}, Count: 4},
	"Fortified": {Transform: func(t *Tribe) {
		t.startRedeploymentMap["Fortified"] = func(gs *GameState) []PieceStack {
			return []PieceStack{{Type:"Fortress", Amount: 1}}
		}
		t.IsStackValidMap["Fortified"] = func(s string) bool {
			return s == "Fortress"
		}
		t.countDefenseMap["Fortified"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			def := 0
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Fortress" {
					def += stack.Amount
				}
			}
			return def, 0, 0, nil
		}
		t.canBeRedeployedInMap["Fortified"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
			if old {
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
		t.countPointsMap["Fortified"] = func(tile *Tile) int {
			if t.IsActive {
				for _, stack := range tile.PieceStacks {
					if stack.Type == "Fortress" {
						return 1
					}
				}
			}
			return 0
		}
		t.clearTileMap["Fortified"] = func(tile *Tile, gs *GameState, pk int) {
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Fortress"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
		}
		}, Count: 3},
	"Bivouacking": {Transform: func(t *Tribe) {
		t.giveInitialStacksMap["Bivouacking"] = func() []PieceStack {
			return[]PieceStack{{Type: "Encampment", Amount: 5}} 
		}
		t.countDefenseMap["Bivouacking"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			def := 0
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Encampment" {
					def += stack.Amount
				}
			}
			return def, 0, 0, nil
		}

		t.canBeRedeployedInMap["Bivouacking"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
			if old {
				return true
			}
			return stackType == "Encampment"
		}
		t.getStacksForConquestMap["Bivouacking"] = func(tile *Tile, p *Player) {
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Encampment" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
				}
			}
		}
		t.countRemovableAttackingStacksMap["Bivouacking"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Encampment" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.getStacksOutRedeploymentMap["Bivouacking"] = func(tile *Tile, stackType string) ([]PieceStack, error) {
			if stackType == "Encampment" {
				for _, stack := range tile.PieceStacks {
					if stack.Type == stackType {
						stack.Amount -= 1
						return []PieceStack{{Type: "Encampment", Amount: 1}}, nil
					}

				}
			}
			return nil, fmt.Errorf("")
		}
		t.IsStackValidMap["Bivouacking"] = func(s string) bool {
			return s == "Encampment"
		}
		t.countRemovablePiecesMap["Bivouacking"] = func(ps []PieceStack, tile *Tile) []PieceStack {
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Encampment" {
					ps = append(ps, stack)
				}
			}
			return ps
		}
		t.clearTileMap["Bivouacking"] = func(tile *Tile, gs *GameState, pk int) {
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
		t.giveInitialStacksMap["Heroic"] = func() []PieceStack {
			return []PieceStack{{Type: "Hero", Amount: 2}} 
		}
		t.countDefenseMap["Heroic"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Hero" {
					return 0, 0, 0, fmt.Errorf("A tile with a hero cannot be conquered")
				}
			}
			return 0, 0, 0, nil
		}

		t.canBeRedeployedInMap["Heroic"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
			if old {
				return true
			}
			return stackType == "Hero"
		}
		t.getStacksForConquestMap["Heroic"] = func(tile *Tile, p *Player) {
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Hero" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
				}
			}
		}
		t.getStacksOutRedeploymentMap["Heroic"] = func(tile *Tile, stackType string) ([]PieceStack, error) {
			if stackType == "Hero" {
				for _, stack := range tile.PieceStacks {
					if stack.Type == stackType {
						stack.Amount -= 1
						return []PieceStack{{Type: "Hero", Amount: 1}}, nil
					}

				}
			}
			return nil, fmt.Errorf("No hero")
		}
		t.IsStackValidMap["Heroic"] = func(s string) bool {
			return s == "Hero"
		}
		t.countRemovablePiecesMap["Heroic"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Hero" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.countRemovableAttackingStacksMap["Heroic"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Hero" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.clearTileMap["Heroic"] = func(tile *Tile, gs *GameState, pk int) {
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Hero"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
		}
		}, Count: 5},
	"Dragon Master": {Transform: func(t *Tribe) {
		t.giveInitialStacksMap["Dragon Master"] = func() []PieceStack {
			return []PieceStack{{Type: "Dragon", Amount: 1}} 
		}
		t.getStacksForConquestMap["Dragon Master"] = func(tile *Tile, p *Player) {
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Dragon" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = append(p.PieceStacks, stack)
				}
			}
		}
		t.countDefenseMap["Dragon Master"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Dragon" {
					return 0, 0, 0, fmt.Errorf("A tile with a dragon cannot be conquered")
				}
			}
			return 0, 0, 0, nil
		}

		t.IsStackValidMap["Dragon Master"] = func(stackType string) bool {
			return (stackType == "Dragon" && t.IsActive)
		}


		t.countAttackMap["Dragon Master"] = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
			if stackType == "Dragon" {
				return []PieceStack{{Type: string(t.Race), Amount: 1}, {Type: "Dragon", Amount: 1}}, t.computeGainAttacker(tile), t.computeLossDefender(tile), t.computePawnKill(tile), nil
			}
			return []PieceStack{}, 0, 0, 0, fmt.Errorf("The piecestack was not recognized")
		}
		t.countRemovablePiecesMap["Dragon Master"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Dragon" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.countRemovableAttackingStacksMap["Dragon Master"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Dragon" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		}, Count: 5},
	"Corrupt": {Transform: func(t *Tribe) {
		t.countDefenseMap["Corrupt"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			g, l := 0, 0
			if t.IsActive {
				g += 1
				l += 1
			}
			return 0, g, l, nil
		}
		}, Count: 4},
	"Ransacking": {Transform: func(t *Tribe) {
		t.computeGainAttackerMap["Ransacking"] = func(tile *Tile) int {
			if tile.CheckPresence() == Active {
				return 1
			}
			return 0
		}
		t.computeLossDefenderMap["Ransacking"] = func(tile *Tile) int {
			if tile.CheckPresence() == Active {
				return 1
			}
			return 0
		}
		}, Count: 3},
	"Pillaging": {Transform: func(t *Tribe) {
		t.computeGainAttackerMap["Pillaging"] = func(tile *Tile) int {
			if tile.CheckPresence() != None {
				return 1
			}
			return 0
		}
		}, Count: 5},
	"Barricade": {Transform: func(t *Tribe) {
		t.countExtrapointsMap["Barricade"] = func(gs *GameState) int {
			if t.IsActive {
				count := 0
				for _, tile := range(gs.TileList) {
					if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
						count += 1
					}
				}
				if count <= 4 {
					return 3
				}
			}
			return 0
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
		t.calculateRemainingAttackingStacksMap["Behemoth"] = func(ps []PieceStack, diceUsed bool, ok bool, err error, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			if tile.Biome == Swamp {
				addBehemoth(gs)
				for i := range(ps) {
					if ps[i].Type == "Male Behemoth" || ps[i].Type == "Female Behemoth" {
						ps[i].Amount += 1
					}
				}
			}
			return ps, diceUsed, ok, err
		}
		t.clearTileMap["Behemoth"] = func(tile *Tile, gs *GameState, pk int) {
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
		t.getRedeploymentStackMap["Behemoth"] = func(s string, ps []PieceStack) []PieceStack {
			if s == "Female Behemoth" || s == "Male Behemoth" {
				amount := 0
				for _, stack := range(ps) {
					if stack.Type == s {
						amount = stack.Amount
					}
				}
				return []PieceStack{{Type: s, Amount: amount}}
			}
			return []PieceStack{}
		}
		t.countDefenseMap["Behemoth"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			def := 0
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Male Behemoth" || stack.Type == "Female Behemoth" {
					def += stack.Amount
				}
			}
			return def, 0, 0, nil
		}
		t.countAttackMap["Behemoth"] = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
			if stackType == "Male Behemoth" || stackType == "Female Behemoth" {
				for _, stack := range(t.Owner.PieceStacks) {
					if stack.Type == stackType {
						return []PieceStack{stack, {Type: string(t.Race), Amount: max(t.Minimum, cost - stack.Amount - t.computeDiscount(tile))}}, t.computeGainAttacker(tile), t.computeLossDefender(tile), t.computePawnKill(tile), nil
					}
				}
			}
			return []PieceStack{}, 0, 0, 0, fmt.Errorf("The piecestack was not recognized")
		}
		t.IsStackValidMap["Behemoth"] = func(stackType string) bool {
			return (stackType == "Female Behemoth" && t.IsActive) || (stackType == "Male Behemoth" && t.IsActive)
		}
		t.canBeRedeployedInMap["Behemoth"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
			if old {
				return true
			}
			return stackType == "Female Behemoth" || stackType == "Male Behemoth"
		}
		t.getStacksOutRedeploymentMap["Behemoth"] = func(tile *Tile, stackType string) ([]PieceStack, error) {
			if stackType == "Female Behemoth" || stackType == "Male Behemoth" {
				for _, stack := range tile.PieceStacks {
					if stack.Type == stackType {
						return []PieceStack{{Type: stackType, Amount: stack.Amount}}, nil
					}

				}
			}
			return nil, fmt.Errorf("No behemoth")
		}
		t.getStacksForConquestMap["Behemoth"] = func(tile *Tile, p *Player) {
		    for i := len(tile.PieceStacks) - 1; i >= 0; i-- { // Loop backward
			stack := tile.PieceStacks[i]
			if stack.Type == "Female Behemoth" || stack.Type == "Male Behemoth" {
			    // Remove stack safely
			    tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
			    p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
			}
		    }
		}
		t.countRemovableAttackingStacksMap["Behemoth"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Female Behemoth" || stack.Type == "Male Behemoth" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.countRemovablePiecesMap["Behemoth"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
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
		t.giveInitialStacksMap["Catapult"] = func() []PieceStack {
			return []PieceStack{{Type: "Catapult", Amount: 1}} 
		}
		t.getStacksForConquestMap["Catapult"] = func(tile *Tile, p *Player) {
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Catapult" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = append(p.PieceStacks, stack)
				}
			}
		}
		t.countDefenseMap["Catapult"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Catapult" {
					return 0, 0, 0, fmt.Errorf("A tile with a catapult cannot be conquered")
				}
			}
			return 0, 0, 0, nil
		}
		t.IsStackValidMap["Catapult"] = func(stackType string) bool {
			return (stackType == "Catapult" && t.IsActive)
		}
		t.countRemovablePiecesMap["Catapult"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Catapult" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.countRemovableAttackingStacksMap["Catapult"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Catapult" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.specialConquestMap["Catapult"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
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
		t.checkAdjacencyMap["Catapult"] = func(tile *Tile, gs *GameState, err error) error {
			if b, ok := t.State["justPlaced"].(bool); ok && !b {
				return err
			}
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
		t.computeDiscountMap["Catapult"] = func(tile *Tile) int {
			if b, ok := t.State["justUsed"].(bool); ok && b {
				t.State["justUsed"] = false
				return 1
			}
			return 0
		}
		t.postConquestMap["Catapult"] = func(tile *Tile, gs *GameState) {
			t.State["justUsed"], t.State["justPlaced"] = false, false
		}
		}, Count: 4},
	"Stout": {Transform: func(t *Tribe) {
		t.State["extrapoints"] = 0
		t.State["pluspoints"] = 0
		t.canGoIntoDeclineMap["Stout"] = func(b bool, gs *GameState) bool {
			t.State["pluspoints"] = gs.countPoints(t.Owner)
			return b || gs.TurnInfo.Phase == Redeployment
		}
		t.goIntoDeclineMap["Stout"] = func(gs *GameState) {
			gs.GetPieceStackForConquest(t.Owner)
			minuspoints := gs.countPoints(t.Owner)
			val := t.State["pluspoints"]
			var pluspoints int
			switch v := val.(type) {
			case float64:
			    pluspoints = int(v)
			case int:
			    pluspoints = v
			}
			t.State["extrapoints"] = pluspoints - minuspoints
		}
		t.countExtrapointsMap["Stout"] = func(gs *GameState) int {
			val := t.State["extrapoints"]
			var extraPoints int
			switch v := val.(type) {
			case float64:
			    extraPoints = int(v)
			case int:
			    extraPoints = v
			}
			t.State["extrapoints"] = 0
			return extraPoints
		}
		}, Count: 4},
	"Fireball": {Transform: func(t *Tribe) {
		t.startRedeploymentMap["Fireball"] = func(gs *GameState) []PieceStack {
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
			return []PieceStack{{Type: "Fireball", Amount: count}}
		}
		t.countAttackMap["Fireball"] = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
			if stackType == "Fireball" {
				return []PieceStack{{Type: "Fireball", Amount: 1}, {Type: string(t.Race), Amount: max(t.Minimum, cost - 2 - t.computeDiscount(tile))}}, t.computeGainAttacker(tile), t.computeLossDefender(tile), t.computePawnKill(tile), nil
			}
			return []PieceStack{}, 0, 0, 0, fmt.Errorf("The piecestack was not recognized")
		}
		t.countNewTileStacksMap["Fireball"] = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
			for i := range(ps) {
				if ps[i].Type == "Fireball" {
					return append(ps[:i], ps[i+1:]...)
				}
			}
			return ps
		}
		t.countRemovableAttackingStacksMap["Fireball"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Fireball" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.IsStackValidMap["Fireball"] = func(s string) bool {
			return s == "Fireball"
		}
		}, Count: 5},
	"Imperial": {Transform: func(t *Tribe) {
		t.countExtrapointsMap["Imperial"] = func(gs *GameState) int {
			if !t.IsActive {
				return 0
			}
			amount := 0
			for _, tile := range(gs.TileList) {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					amount += 1
				}
			}
			return max(0, amount - 3)
		}
		}, Count: 4},
	"Mercenary": {Transform: func(t *Tribe) {
		t.giveInitialStacksMap["Mercenary"] = func() []PieceStack {
			return []PieceStack{{Type: "Mercenary", Amount: 1}} 
		}
		t.countAttackMap["Mercenary"] = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
			if stackType == "Mercenary" {
				if t.Owner.CoinPile == 0 {
					return []PieceStack{}, 0, 0, 0, fmt.Errorf("Player does not have any coins")

				}
				return []PieceStack{{Type: string(t.Race), Amount: max(t.Minimum, cost - 2 - t.computeDiscount(tile))}}, t.computeGainAttacker(tile) - 1, t.computeLossDefender(tile), t.computePawnKill(tile), nil
			}
			return []PieceStack{}, 0, 0, 0, fmt.Errorf("The piecestack was not recognized")
		}
		t.countRemovableAttackingStacksMap["Mercenary"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Mercenary" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.IsStackValidMap["Mercenary"] = func(s string) bool {
			return s == "Mercenary"
		}
		}, Count: 4},
	"Peace-loving": {Transform: func(t *Tribe) {
		t.State["hasattacked"] = false
		t.postConquestMap["Peace-loving"] = func(tile *Tile, gs *GameState) {
			if tile.CheckPresence() == Active {
				t.State["hasattacked"] = true
			}
		}
		t.countExtrapointsMap["Peace-loving"] = func(gs *GameState) int {
			if !t.IsActive {
				return 0
			}
			if hasAttacked, ok := t.State["hasattacked"].(bool); ok && !hasAttacked {
				return 3
			}
			t.State["hasattacked"] = false
			return 0
		}
		}, Count: 4},
	"Spirit": {Transform: func(t *Tribe) {
		t.prepareRemovalMap["Spirit"] = func(gs *GameState) bool {
			return false
		}
		t.alternativeDeclineMap["Spirit"] = func(gs *GameState) bool {
			player := t.Owner

			for _, tile := range gs.TileList {
				if tile.CheckPresence() != None && tile.OwningTribe.Race == player.ActiveTribe.Race {
				    tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, tile.OwningTribe.countRemovablePieces(tile))
				}
			}

			player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, player.ActiveTribe.countRemovableAttackingStacks(player))
			t.IsActive = false

			player.PassiveTribes = append(player.PassiveTribes, player.ActiveTribe)
			player.ActiveTribe = nil

			for _, f := range(t.goIntoDeclineMap) {
				f(gs)
			}
			return true

		}
		}, Count: 4},
	"Lava": {Transform: func(t *Tribe) {
		t.State["mountains"] = []string{}
		t.IsStackValidMap["Lava"] = func(s string) bool {
			return s == "Lava"
		}
		t.startRedeploymentMap["Lava"] = func(gs *GameState) []PieceStack {
			count := 0
			mountains, _ := t.State["mountains"].([]string)
			for _, tile := range(gs.TileList) {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) && tile.Biome == Mountain {
					mountains = append(mountains, tile.Id)
					count += 1
				}
			}
			t.State["mountains"] = mountains
			return []PieceStack{{Type:"Lava", Amount: count}}
		}
		t.handleDeploymentInMap["Lava"] = func(tile *Tile, stackType string, i int, gs *GameState) error {
			if stackType != "Lava" {
				return fmt.Errorf("Not Lava")
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
				tile.handleAfterConquest(gs, nil)
				tile.OwningTribe.handleReturn(tile, gs, 0)
			}

			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)
			tile.ModifierDefenses["Lava"] = TileModifierDefenses["Lava"]
			return nil
		}
		t.getStacksForConquestTurnMap["Lava"] = func(p *Player, gs *GameState) {
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
		}
		}, Count: 5},
	"Diplomat": {Transform: func(t *Tribe) {
		t.State["playersAttacked"] = []int{}
		t.handleOpponentActionMap["Diplomat"] = func(stackType string, opponent *Player, gs *GameState) error {
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
			return fmt.Errorf("Not for us")
		}
		t.IsStackValidMap["Diplomat"] = func(s string) bool {
			return s == "Diplomat"
		}
		t.giveInitialStacksMap["Diplomat"] = func() []PieceStack {
			return []PieceStack{{Type: "Diplomat", Amount: 1}} 
		}
		t.countDefenseMap["Diploomat"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range p.PieceStacks {
				if stack.Type == "Diplomat" {
					return 0, 0, 0, fmt.Errorf("You have a Peace pact with this player!")
				}
			}
			return 0, 0, 0, nil
		}
		t.postConquestMap["Diplomat"] = func( tile *Tile, gs *GameState) {
			if tile.CheckPresence() == Active && tile.OwningTribe != nil {
				playersAttacked, _ := t.State["playersAttacked"].([]int)
				playersAttacked = append(playersAttacked, tile.OwningTribe.Owner.Index)
				t.State["playersAttacked"] = playersAttacked
			}
		}
		t.getStacksForConquestTurnMap["Diplomat"] = func(p *Player, gs *GameState) {
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
		t.countRemovableAttackingStacksMap["Diplomat"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Diplomat" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		}, Count: 5},
	"Haggling": {Transform: func(t *Tribe) {
		t.handleOpponentActionMap["Haggling"] = func(stackType string, opponent *Player, gs *GameState) error {
			if stackType == "Treaty" {
				if gs.TurnInfo.Phase != Redeployment {
					return fmt.Errorf("You must be in Redeployment to establish a peace treaty!")
				}

				opponent.PieceStacks = AddPieceStacks(opponent.PieceStacks, []PieceStack{{Type: "Treaty", Amount: 1}})
				t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Treaty", Amount: 1}})
				return nil
			}
			return fmt.Errorf("Not for us")
		}
		t.IsStackValidMap["Treaty"] = func(s string) bool {
			return s == "Treaty"
		}
		t.giveInitialStacksMap["Haggling"] = func() []PieceStack {
			return []PieceStack{{Type: "Treaty", Amount: 5}} 
		}
		t.countDefenseMap["Haggling"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			l, g := 0, 0
			for _, stack := range p.PieceStacks {
				if stack.Type == "Treaty" {
					l += stack.Amount
					g += stack.Amount
					p.PieceStacks, _ = SubtractPieceStacks(p.PieceStacks, []PieceStack{stack})
					t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{stack})
				}
			}
			return 0, g, l, nil
		}
		t.getStacksForConquestTurnMap["Haggling"] = func(p *Player, gs *GameState) {
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
		t.countRemovableAttackingStacksMap["Haggling"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Treaty" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		}, Count: 5},
	"Zeppelined": {Transform: func(t *Tribe) {
		t.IsStackValidMap["Zeppelined"] = func(s string) bool {
			return s == "Zeppelin"
		}
		t.giveInitialStacksMap["Zeppelined"] = func() []PieceStack {
			return []PieceStack{{Type: "Zeppelin", Amount: 5}} 
		}
		t.countRemovableAttackingStacksMap["Zeppelined"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Zeppelin" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.specialConquestMap["Zeppelined"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
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
			var err error
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
			attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill, err := t.countAttack(tile, tileCost - diceThrow, string(t.Race))
			if err != nil {
				return true, err
			}
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
			tile.handleAfterConquest(gs, t)

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
		t.getStacksForConquestTurnMap["Zeppelined"] = func(p *Player, gs *GameState) {
			for _, tile := range(gs.TileList) {
				for i := range(tile.PieceStacks) {
					if tile.PieceStacks[i].Type == "Burning Zeppelin" || tile.PieceStacks[i].Type == "Zeppelin" {
						tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
						delete(tile.ModifierDefenses, "Burning Zeppelin")
						t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Zeppelin", Amount: 1}})
					}
				}
			}
		}
		t.clearTileMap["Zeppelined"] = func(tile *Tile, gs *GameState, pk int) {
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
		t.handleMovementMap["Gunner"] = func(stackType string, tileFrom, tileTo *Tile, gs *GameState) error {
			if stackType != "Left Cannon" && stackType != "Right Cannon" {
				return fmt.Errorf("")
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
		t.giveInitialStacksMap["Gunner"] = func() []PieceStack {
			return[]PieceStack{{Type: "Cannon Trigger", Amount: 1}, {Type: "Left Cannon", Amount: 1}, {Type: "Right Cannon", Amount: 1}} 
		}
		t.IsStackValidMap["Gunner"] = func(s string) bool {
			return s == "Cannon Trigger" || s == "Left Cannon" || s == "Right Cannon"
		}
		t.canBeRedeployedInMap["Gunner"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
			return old || stackType == "Left Cannon" || stackType == "Right Cannon"
		}
		t.countRemovablePiecesMap["Gunner"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Left Cannon" || stack.Type == "Right Cannon" || stack.Type == "Firing Left Cannon" || stack.Type == "Firing Right Cannon"{
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.countDefenseMap["Gunner"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Left Cannon" || stack.Type == "Right Cannon" || stack.Type == "Firing Left Cannon" || stack.Type == "Firing Right Cannon" {
					return 0, 0, 0, fmt.Errorf("A tile with a cannon cannot be conquered")
				}
			}
			return 0, 0, 0, nil
		}
		t.countRemovableAttackingStacksMap["Gunner"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Left Cannon" || stack.Type == "Right Cannon" || stack.Type == "Cannon Trigger" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		t.specialConquestMap["Gunner"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
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
		t.computeDiscountMap["Gunner"] = func(tile *Tile) int {
			discount := 0
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
		t.getStacksForConquestMap["Gunner"] = func(tile *Tile, p *Player) {
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
		t.countRemovablePiecesMap["Tomb"] = func(ps []PieceStack, tile *Tile) []PieceStack {
			newstacks := []PieceStack{}
			for _, stack := range(ps) {
				if stack.Type != string(t.Race) {
					stack.Tribe  = t
					newstacks = append(newstacks, stack)
				}
			}
			return newstacks
		}
		t.IsStackValidMap["Tomb"] = func(s string) bool {
			return (s == string(t.Race) && !t.IsActive)
		}
		t.goIntoDeclineMap["Tomb"] = func(gs *GameState) {
			gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
				player: gs.TurnInfo.PlayerIndex,
				TurnInfo: &TurnInfo{
					TurnIndex: gs.TurnInfo.TurnIndex,
					PlayerIndex: gs.TurnInfo.PlayerIndex,
					Phase: Redeployment,
				},
			})

		}
		t.countRemovableAttackingStacksMap["Tomb"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for i, stack := range(oldStacks) {
				if stack.Type == string(t.Race) {
					oldStacks = append(oldStacks[:i], oldStacks[i+1:]...)
					break
				}
			}
			return oldStacks
		}
		t.handleReturnMap["Tomb"] = func(tile *Tile, gs *GameState, pk int) {
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
		t.giveInitialStacksMap["Royal"] = func() []PieceStack {
			return []PieceStack{{Type: "Queen", Amount: 1}} 
		}
		t.countDefenseMap["Royal"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Queen" {
					return 0, 0, 0, fmt.Errorf("A tile with the queen cannot be conquered!")
				}
			}
			return 0, 0, 0, nil
		}

		t.canBeRedeployedInMap["Royal"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
			if old {
				return true
			}
			return stackType == "Queen"
		}
		t.getStacksForConquestMap["Royal"] = func(tile *Tile, p *Player) {
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
		t.getStacksOutRedeploymentMap["Royal"] = func(tile *Tile, stackType string) ([]PieceStack, error) {
			if t.IsActive && stackType == "Queen" {
				for _, stack := range tile.PieceStacks {
					if stack.Type == stackType {
						stack.Amount -= 1
						return []PieceStack{{Type: "Queen", Amount: 1}}, nil
					}

				}
			}
			return nil, fmt.Errorf("No queen")
		}
		t.IsStackValidMap["Royal"] = func(s string) bool {
			return s == "Queen"
		}
		t.clearTileMap["Royal"] = func(tile *Tile, gs *GameState, pk int) {
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Queen"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
		}
		t.goIntoDeclineMap["Royal"] = func(gs *GameState) {
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
		t.handleReturnMap["Immortal"] = func(tile *Tile, gs *GameState, pk int) {
			if t.IsActive {
				t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: pk}})
			}
		}
		}, Count: 5},
	"Great Brass Pipe": {Transform: func(t *Tribe) {
		t.checkAdjacencyMap["Great Brass Pipe"] = func(tile *Tile, gs *GameState, err error) error {
                if err == nil {
                    return nil
                }
		var biome Biome
		OuterLoop:
		for _, tile2 := range(gs.TileList) {
			if tile2.CheckPresence() != None && tile2.OwningTribe.checkPresence(tile2, t.Race) {
				for _, stack := range(tile2.PieceStacks) {
					if stack.Type == "Great Brass Pipe" {
						biome = tile2.Biome
						break OuterLoop
					}	
				}
			}
		}
                if tile.Biome == biome {
                    return nil
                }
                return err
		}
	}, Count: 0},
	"Racketeering": {Transform: func(t *Tribe) {
		t.getStacksForConquestTurnMap["Racketeering"] = func(p *Player, gs *GameState) {
			if !t.IsActive {
				return
			}
			gs.ModifierAfterPick["Racketeering"] = func(i int, te *TribeEntry) {
				if gs.TurnInfo.PlayerIndex != t.Owner.Index {
					t.Owner.CoinPile += i
				}
			}
		}
		t.goIntoDeclineMap["Racketeering"] = func(gs *GameState) {
			gs.ModifierAfterPick["Racketeering"] = func(i int, te *TribeEntry) {
				if gs.TurnInfo.PlayerIndex == t.Owner.Index {
					t.Owner.CoinPile += i
				}
			}
		}
		}, Count: 5},
}

func InitTraitMap() {
	TraitMap["Copycat"] = TraitValue{Transform: func(t *Tribe) {
		t.State["additionalpower"] = ""
		t.handleEntryActionMap["Copycat"] = func(i int, s string, gs *GameState) error {
			if s != "Mirror" {
				return fmt.Errorf("Not a mirror")
			}
			trait := gs.TribeList[i].Trait
			t.State["additionalpower"] = string(trait)
			t.GiveTrait(trait)
			t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Mirror", Amount: 1}})
			gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The copycat %s chose to mirror the trait %s!", t.Race, trait)})
			gs.ModifierAfterPick["Copycat"] = func(i int, te *TribeEntry) {
				additionalPower := t.State["additionalpower"].(string)
				if string(te.Trait) == additionalPower {
					t.State["additionalpower"] = ""
					t.DeletePower(additionalPower, gs)
					t.Owner.PieceStacks = append(t.Owner.PieceStacks, PieceStack{Type: "Mirror", Amount: 1})
					gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The copycat %s had to stop mirroring the trait %s!", t.Race, trait)})
				}
			}
			return nil
		}
		t.IsStackValidMap["Copycat"] = func(s string) bool {
			return s == "Mirror"
		}
		t.giveInitialStacksMap["Copycat"] = func() []PieceStack {
			return []PieceStack{{Type: "Mirror", Amount: 1}} 
		}
		t.getStacksForConquestTurnMap["Copycat"] = func(p *Player, gs *GameState) {
			additionalPower := t.State["additionalpower"].(string)
			if additionalPower != "" {
				t.State["additionalpower"] = ""
				t.DeletePower(additionalPower, gs)
				t.Owner.PieceStacks = append(t.Owner.PieceStacks, PieceStack{Type: "Mirror", Amount: 1})
			}
		}
		t.countRemovableAttackingStacksMap["Copycat"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Mirror" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
	}, Count: 3}
	TraitMap["Soul-touch"] = TraitValue{Transform: func(t *Tribe) {
		t.alternativeDeclineMap["Soul-touch"] = func(gs *GameState) bool {
			player := t.Owner


			for _, tile := range gs.TileList {
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, tile.OwningTribe.countRemovablePieces(tile))
				}
			}

			player.PieceStacks, _ = SubtractPieceStacks(player.PieceStacks, player.ActiveTribe.countRemovableAttackingStacks(player))
			t.IsActive = false

			if (len(player.PassiveTribes) > 0) {
				player.PassiveTribes = append(player.PassiveTribes, player.ActiveTribe)
				player.ActiveTribe = player.PassiveTribes[len(player.PassiveTribes) - 2]
				player.PassiveTribes = append(player.PassiveTribes[:len(player.PassiveTribes) - 2], player.PassiveTribes[len(player.PassiveTribes) - 1])
				player.ActiveTribe.IsActive = true
				tribe := player.ActiveTribe
				total := 0
				for _, tile := range gs.TileList {
					if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, tribe.Race) {
						for _, stack := range tile.PieceStacks {
							if stack.Type == string(tribe.Race) {
								total += stack.Amount
							}
						}
					}
				}
				race := RaceMap[tribe.Race]
				trait := TraitMap[tribe.Trait]
				tribe.Owner.PieceStacks = AddPieceStacks(tribe.Owner.PieceStacks, []PieceStack{{Type: string(tribe.Race), Amount: race.Count + trait.Count - total}})
				tribe.Owner.PieceStacks = AddPieceStacks(tribe.Owner.PieceStacks, tribe.Owner.ActiveTribe.giveInitialStacks())
				gs.GetPieceStackForConquest(tribe.Owner)
				gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
					player: gs.TurnInfo.PlayerIndex,
					TurnInfo: &TurnInfo{
						TurnIndex: gs.TurnInfo.TurnIndex,
						PlayerIndex: t.Owner.Index,
						Phase: TileAbandonment,
					},
					actionBefore: func(gs *GameState) {},
				})
			} else {
				player.PassiveTribes = append(player.PassiveTribes, player.ActiveTribe)
				player.ActiveTribe = nil
			}


			for _, f := range(t.goIntoDeclineMap) {
				f(gs)
			}
			return true
		}
	}, Count: 4}
}
