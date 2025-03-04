package gamestate

import (
	"fmt"
)

type RaceValue struct {
    Transform func(*Tribe)
    Count     int
}

var RaceMap = map[Race]RaceValue {
	"Amazons": {Transform: func(t *Tribe) {
		oldCanEndTurn := t.canEndTurn
		t.canEndTurn = func(gs *GameState) error {
			err := oldCanEndTurn(gs)
			if err != nil {
				return err
			}
			for _, stack := range(t.Owner.PieceStacks) {
				if stack.Type == string(t.Race) && stack.Amount >= 4 {
					return nil
				}
			}
			return fmt.Errorf("You cannot end your turn with less than 4 amazons in your hand!")
		}
		}, Count: 10},
	"Trolls": {Transform: func(t *Tribe) {
		// Make a newly conquered region contain a lair
		oldCountNewTileStacks := t.countNewTileStacks
		t.countNewTileStacks = func(stacks []PieceStack, tile *Tile) []PieceStack {
			oldstacks := oldCountNewTileStacks(stacks, tile)
			return AddPieceStacks(oldstacks, []PieceStack{{Type: "Lair", Amount: 1}})
		}

		// Make the defense of the tile + 1
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p)
			if err != nil {
				return old, g, l, err
			}
			return old+1, g, l, nil
		}
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Lair"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
		}
		}, Count: 5},
	"Wizards": {Transform: func(t *Tribe) {
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
	"Khans": {Transform: func(t *Tribe) {
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
	"Humans": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && tile.Biome == Field {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 5},
	"Halflings": {Transform: func(t *Tribe) {
		t.State["holesleft"] = 2
		t.State["startedalready"] = false
		oldCheckAdjacency := t.checkAdjacency
		t.checkAdjacency = func(tile *Tile, gs *GameState) error {
			err := oldCheckAdjacency(tile, gs)
			if err != nil && !gs.IsTribePresentOnTheBoard(t.Race) && t.State["startedalready"] == false {
				t.State["startedalready"] = true
				return nil
			} else {
				t.State["startedalready"] = true
				return err
			}
		}

		oldCountNewTileStacks := t.countNewTileStacks
		t.countNewTileStacks = func(ps []PieceStack, tile *Tile) []PieceStack {
			stacks := oldCountNewTileStacks(ps, tile)
			if value, ok := t.State["holesleft"].(int); ok && value > 0 {
				t.State["holesleft"] = value - 1
				stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Hole", Amount: 1}})
			}
			return stacks
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			count, g, l, err := oldCountDefense(tile, p)
			if err != nil {
				return count, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Hole" {
					return 1000, g, l, fmt.Errorf("A hole in the ground cannot be conquered!")
				}
			}
			return count, g, l, err
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Hole" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldclearTile := t.clearTile
		t.clearTile  = func(tile *Tile, gs *GameState, pk int) {
			oldclearTile(tile, gs, pk)
			for i, stack := range(tile.PieceStacks) {
				if stack.Type == "Hole" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					break
				}
			}
		}
		}, Count: 6},
	"White Ladies": {Transform: func(t *Tribe) {
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			count, g, l, err := oldCountDefense(tile, p)
			if err != nil {
				return count, g, l, err
			}
			if !t.IsActive {
				return 1000, g, l, fmt.Errorf("Cannot conquer white ladies when they are in decline") 
			}
			return count, g, l, err
		}
		}, Count: 2},
	"Tritons": {Transform: func(t *Tribe) {
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.Biome == Water {
					return oldcomputeDiscount(stackType, tile) + 1
				}
			}
			return oldcomputeDiscount(stackType, tile)
		}
		}, Count: 6},
	"Shrubmen": {Transform: func(t *Tribe) {
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			count, g, l, err := oldCountDefense(tile, p)
			if err != nil {
				return count, g, l, err
			}
			if tile.Biome == Forest {
				return 1000, g, l, fmt.Errorf("cannot conquer shrubman when they are in a forest") 
			}
			return count, g, l, err
		}
		}, Count: 6},
	"Ratmen": {Transform: func(t *Tribe) {
		}, Count: 8},
	"Goblins": {Transform: func(t *Tribe) {
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			if tile.Presence == Passive {
				return oldcomputeDiscount(stackType, tile) + 1
			}
			return oldcomputeDiscount(stackType, tile)
		}
		}, Count: 6},
	"Giants": {Transform: func(t *Tribe) {
		oldcomputeDiscount := t.computeDiscount
		t.computeDiscount = func(stackType string, tile *Tile) int {
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.Biome == Mountain && neighbour.Presence != None && neighbour.OwningTribe.checkPresence(neighbour, t.Race) {
					return oldcomputeDiscount(stackType, tile) + 1
				}
			}
			return oldcomputeDiscount(stackType, tile)
		}
		}, Count: 6},
	"Orcs": {Transform: func(t *Tribe) {
		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			old, g, l, k := oldCountAttack(tile, cost, stackType)
			if tile.Presence != None {
				g += 1
			}
			return old, g , l, k
		}
		}, Count: 5},
	"Skeletons": {Transform: func(t *Tribe) {
		t.State["killcount"] = 0
		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			stacks, diceUsed, ok, err := oldCalculateRemainingAttackingStacks(ps, tile, gs)
			if err != nil || !ok {
				return stacks, diceUsed, ok, err
			}
			if tile.Presence != None {
				if killCount, ok := t.State["killcount"].(int); ok {
					t.State["killcount"] = killCount + 1
				}
			}
			return stacks, diceUsed, ok, err
		}
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			if killCount, ok := t.State["killcount"].(int); ok {
				t.State["killcount"] = 0
				stacks = AddPieceStacks(stacks, []PieceStack{{Type: string(t.Race), Amount: killCount / 2}})
			}
			return stacks
		}
		}, Count: 6},
	"Witch Doctors": {Transform: func(t *Tribe) {
		oldHandleReturn := t.handleReturn
		t.handleReturn = func(tile *Tile, gs *GameState, pk int) {
			oldHandleReturn(tile, gs, pk)
			if t.IsActive {
				diceThrow := RollDice()
				gs.Messages = append(gs.Messages, fmt.Sprintf("Pygmies got back: %d Pawns!", diceThrow))
				t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: max(0, diceThrow - (1 - pk))}})
			}
		}
		}, Count: 6},
	"Elves": {Transform: func(t *Tribe) {
		oldHandleReturn := t.handleReturn
		t.handleReturn = func(tile *Tile, gs *GameState, pk int) {
			if t.IsActive {
				oldHandleReturn(tile, gs, 0)
			} else {
				oldHandleReturn(tile, gs, pk)
			}
		}
		}, Count: 6},
	"Pixies": {Transform: func(t *Tribe) {
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			for _, tile := range gs.TileList {
				if tile.Presence != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					t.getStacksForConquest(tile, t.Owner)
				}
			}
			return stacks
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
				return stackType != string(t.Race)
			}
			return false
		}
		}, Count: 11},
	"Barbarians": {Transform: func(t *Tribe) {
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
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
		}, Count: 9},
	"Sorcerers": {Transform: func(t *Tribe) {
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Staff" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Staff"
		}
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string, attacker *Player, attackerIndex int) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType, attacker, attackerIndex)
			if ok {
				return ok, err
			}
			if stackType != "Staff" {
				return false, nil
			}

			if tile.Presence != Active {
				return true, fmt.Errorf("tribe needs to be active")
			}

			if err := t.checkZoneAccess(tile); err != nil {
				return true, err
			}
			if err := t.checkAdjacency(tile, gs); err != nil {
				return true, err
			}

			if !gs.IsTribePresentOnTheBoard(t.Race) {
				return true, fmt.Errorf("you need to be already present on the board")
			}

			if tile.Presence == Active {
				// Maybe do something with those, in case this is actually considered a conquest
				_, _, _, err := tile.OwningTribe.countDefense(tile, attacker)
				if err != nil {
					return true, err
				}
			} else {
				return true, fmt.Errorf("This tile does not contain an active tribe")
			}

			for _, stack := range(tile.PieceStacks) {
				if stack.Type == string(tile.OwningTribe.Race) && stack.Amount > 1 {
					return true, fmt.Errorf("This tile contains more than one active pawn!")
				}
			}

			tile.OwningTribe.clearTile(tile, gs, 1)
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}})
			tile.OwningTribe = t
			attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, []PieceStack{{Type: "Staff", Amount: 1}})
			return true, nil
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Staff" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		}, Count: 5},
	"Wendigos": {Transform: func(t *Tribe) {
		oldgetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(player *Player, gs *GameState) {
			oldgetStacksForConquestTurn(player, gs)
			newstacks := []PieceStack{}
			for _, stack := range(player.PieceStacks) {
				if stack.Type != "Power" {
					newstacks = append(newstacks, stack)
				}
			}
			newstacks = append(newstacks, PieceStack{Type: "Power", Amount: 1})
			player.PieceStacks = newstacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Power" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Power"
		}
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string, attacker *Player, attackerIndex int) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType, attacker, attackerIndex)
			if ok {
				return ok, err
			}
			if stackType != "Power" {
				return false, nil
			}

			if tile.Biome != Forest {
				return true, fmt.Errorf("This is not a forest!")
			}

			if tile.Presence != None {
				// Maybe do something with those, in case this is actually considered a conquest
				_, _, _, err := tile.OwningTribe.countDefense(tile, attacker)
				if err != nil {
					return true, fmt.Errorf("Impossible to attack", err)
				}
			} 

			if tile.Presence != None {
				tile.OwningTribe.clearTile(tile, gs, 1)
			}
			tile.OwningTribe = nil
			tile.Presence = None
			attacker.PieceStacks = AddPieceStacks(attacker.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}})
			attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, []PieceStack{{Type: "Power", Amount: 1}})
			gs.TurnInfo.Phase = Conquest
			return true, nil
		}
		}, Count: 6},
	"Nomads": {Transform: func(t *Tribe) {
		t.State["abandonedTiles"] = []string{}
		oldhandleAbandonment := t.handleAbandonment
		t.handleAbandonment = func(tile *Tile, gs *GameState) {
			oldhandleAbandonment(tile, gs)
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Coin", Amount: 1, Tribe: t}})
			if abandonedTiles, ok := t.State["abandonedTiles"].([]string); ok {
				t.State["abandonedTiles"] = append(abandonedTiles, tile.Id)
			} else {
				// If it's not a slice of strings, reinitialize it
				t.State["abandonedTiles"] = []string{tile.Id}
			}
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
	"Scarecrows": {Transform: func(t *Tribe) {
		oldCountDefense:= t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			a, b, c, err := oldCountDefense(tile, p)
			return a, b, c-1, err
		}
		}, Count: 12},
	"Kobolds": {Transform: func(t *Tribe) {
		t.Minimum = 2
		}, Count: 11},
	"Leprechauns": {Transform: func(t *Tribe) {
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Cauldron", Amount: 16}})
			return stacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Cauldron" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(stackType string) bool {
			return stackType == "Cauldron" || oldIsStackValid(stackType)
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
				return true
			}
			if stackType == "Cauldron" {
				for _, stack := range(tile.PieceStacks) {
					if stack.Type == "Cauldron" {
						return false
					}
				}
				return true
			}
			return false
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Cauldron" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.CoinPile += 1
				}
			}
		}
		oldCountDefense:= t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			a, b, c, err := oldCountDefense(tile, p)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Cauldron" {
					c = c - 1
				}
			}
			return a, b, c, err
		}
		oldClearTile := t.clearTile
		t.clearTile = func(tile *Tile, gs *GameState, pk int) {
			oldClearTile(tile, gs, pk)
			for i, stack := range tile.PieceStacks {
			    if stack.Type == "Cauldron"{
				tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
				return // Exit after removal to avoid index shifting issues
			    }
			}
		}
		oldgetRedeploymentStack := t.getRedeploymentStack
		t.getRedeploymentStack = func(s string, ps []PieceStack) []PieceStack {
			if s == "Cauldron" {
				return []PieceStack{{Type: s, Amount: 1}}
			}
			return oldgetRedeploymentStack(s, ps)
		}
		}, Count: 6},
	"Drakons": {Transform: func(t *Tribe) {
		oldgiveInitialStacks := t.giveInitialStacks
		t.giveInitialStacks = func() []PieceStack {
			stacks := oldgiveInitialStacks()
			stacks = AddPieceStacks(stacks, []PieceStack{{Type: "Drakon's Dragon", Amount: 3}})
			return stacks
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			for _, stack := range(p.PieceStacks) {
				if stack.Type == "Drakon's Dragon" {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			oldGetStacksForConquest(tile, p)
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Drakon's Dragon" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
				}
			}
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Drakon's Dragon" {
					return 1000, g, l, fmt.Errorf("A tile with a drakon's dragon cannot be conquered")
				}
			}
			return old, g, l, nil
		}

		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(stackType string) bool {
			return stackType == "Drakon's Dragon" || oldIsStackValid(stackType)
		}


		oldCountAttack := t.countAttack
		t.countAttack = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int) {
			stacks, g, l, k := oldCountAttack(tile, cost, stackType)
			if stackType == "Drakon's Dragon" {
				return []PieceStack{{Type: string(t.Race), Amount: 2}, {Type: "Drakon's Dragon", Amount: 1}}, g, l, k
			}
			return stacks, g, l, k
		}
		oldcountNewTileStacks := t.countNewTileStacks
		t.countNewTileStacks = func(ps []PieceStack, tile *Tile) []PieceStack {
			stacks := oldcountNewTileStacks(ps, tile)
			for _, stack := range(stacks) {
				if stack.Type == "Drakon's Dragon" {
					for i := range(stacks) {
						if stacks[i].Type == string(t.Race) {
							stacks[i].Amount -= 1
						}
					}
				}
			}
			return stacks
		}
		}, Count: 6},
	"Fauns": {Transform: func(t *Tribe) {
		t.State["activecount"] = 0
		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			stacks, diceUsed, ok, err := oldCalculateRemainingAttackingStacks(ps, tile, gs)
			if err != nil || !ok {
				return stacks, diceUsed, ok, err
			}
			if tile.Presence == Active {
				if activeCount, ok := t.State["activecount"].(int); ok {
					t.State["activecount"] = activeCount + 1
				}
			}
			return stacks, diceUsed, ok, err
		}
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			if activecount, ok := t.State["activecount"].(int); ok {
				t.State["activecount"] = 0
				stacks = AddPieceStacks(stacks, []PieceStack{{Type: string(t.Race), Amount: activecount}})
			}
			return stacks
		}
		}, Count: 5},
	"Ghouls": {Transform: func(t *Tribe) {
		t.State["hasThrownDice"] = false
		oldCalculateRemainingAttackingStacks := t.calculateRemainingAttackingStacks
		t.calculateRemainingAttackingStacks = func(ps []PieceStack, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			hasThrownDice := t.State["hasThrownDice"].(bool)
			if !t.IsActive && hasThrownDice {
				return nil, true, false, fmt.Errorf("You already threw the dice for the ghouls")
			}
			stacks, diceUsed, ok, err := oldCalculateRemainingAttackingStacks(ps, tile, gs)
			if !t.IsActive && diceUsed {
				t.State["hasThrownDice"] = true
				if !ok {
					return nil, true, false, fmt.Errorf("The dice was not enough for your zombies")
				}
				return stacks, false, ok, err
			}
			return stacks, diceUsed, ok, err
		}
		t.State["deploy"] = make(map[string]int)
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
		oldgetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, player *Player) {
			oldgetStacksForConquest(tile, player)
			if !t.IsActive {
			    for _, stack := range tile.PieceStacks {
				if stack.Type == string(t.Race) {
				    // Making sure the action is atomic
				    movingStack := []PieceStack{{Type: stack.Type, Amount: stack.Amount - t.Minimum}}
				    tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, movingStack)
				    player.PieceStacks = AddPieceStacks(player.PieceStacks, movingStack)
				}
			    }
			    t.State["hasThrownDice"] = false
			}
		}
		oldcountRemovableAttackingStacks := t.countRemovableAttackingStacks
		t.countRemovableAttackingStacks = func(p *Player) []PieceStack {
			oldStacks := oldcountRemovableAttackingStacks(p)
			newstacks := []PieceStack{}
			for _, stack := range(oldStacks) {
				if stack.Type != string(t.Race) {
					stack.Tribe = t
					newstacks = append(newstacks, stack)
				}
			}
			return newstacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || (s == string(t.Race) && !t.IsActive)
		}
		oldcanTileBeAbandoned := t.canTileBeAbandoned
		t.canTileBeAbandoned = func(tile *Tile) bool {
			return oldcanTileBeAbandoned(tile) || (tile.OwningTribe.Race == t.Race)
		}
		oldgetStacksForConquestTurn := t.getStacksForConquestTurn
		t.getStacksForConquestTurn = func(player *Player, gs *GameState) {
			oldgetStacksForConquestTurn(player, gs)
			t.State["deploy"] = make(map[string]int)
		}
		oldGetStacksForConquest := t.getStacksForConquest
		t.getStacksForConquest = func(tile *Tile, p *Player) {
			pawns, _ := t.State["deploy"].(map[string]int)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == string(t.Race) {
					pawns[tile.Id] = stack.Amount - 1
				}
			}
			oldGetStacksForConquest(tile, p)

		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			pawns, _ := t.State["deploy"].(map[string]int)
			for id, amount := range(pawns) {
				movingStack := []PieceStack{{Type: string(t.Race), Amount: amount}}
				tile, _ := gs.TileList[id]
				tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)
				gs.Players[gs.TurnInfo.PlayerIndex].PieceStacks, _ = SubtractPieceStacks(gs.Players[gs.TurnInfo.PlayerIndex].PieceStacks, movingStack)
			}
			// gs.ModifierTurnsBefore = append(gs.ModifierTurnsBefore, TurninfoEntry{
			// 	TurnInfo: TurnInfo{
			// 		TurnIndex: gs.TurnInfo.TurnIndex + 1,
			// 		PlayerIndex: t.Owner.Index,
			// 		Phase: TileAbandonment,
			// 	},
			// 	player: t.Owner.Index,
			// 	actionBefore: func(gs *GameState) {
			// 		gs.GetPieceStackForConquest(t)
			// 	},
			// })
			oldgoIntoDecline(gs)

		}
		}, Count: 5},
	"Scavengers": {Transform: func(t *Tribe) {
		oldSpecialConquest := t.specialConquest
		t.specialConquest = func(gs *GameState, tile *Tile, stackType string, attacker *Player, attackerIndex int) (bool, error) {
			ok, err := oldSpecialConquest(gs, tile, stackType, attacker, attackerIndex)
			if ok {
				return ok, err
			}
			if tile.Presence != Passive {
				return false, nil
			}

			defendingTribe := tile.OwningTribe


			if tile.Presence != None && tile.OwningTribe.checkPresence(tile, t.Race) {
				return true, fmt.Errorf("tribe cannot attack itself")
			}

			if err := t.checkZoneAccess(tile); err != nil {
				return true, fmt.Errorf("cannot access zone", err)
			}
			if err := t.checkAdjacency(tile, gs); err != nil {
				return true, fmt.Errorf("cannot reach zone", err)
			}

			tileCost, moneyGainDefender, moneyLossAttacker, err := defendingTribe.countDefense(tile, attacker)
			if err != nil {
				return true, fmt.Errorf("Impossible to attack", err)
			}

			attackCostStacks, moneyGainAttacker, moneyLossDefender, _ := t.countAttack(tile, tileCost, stackType)
			stacks, hasDiceBeenUsed, ok, err := t.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
			newTileStacks := t.countNewTileStacks(stacks, tile)
			if err != nil {
				return true, err
			}
			if !ok {
				return true, nil
			}
			tile.OwningTribe.Owner.CoinPile += moneyGainDefender - moneyLossDefender
			for i := range tile.PieceStacks {
			    tile.PieceStacks[i].Tribe = tile.OwningTribe
			}
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, newTileStacks)
			attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, stacks)
			attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
			tile.OwningTribe = t

			if tile.OwningTribe != nil && tile.OwningTribe.IsActive {
				tile.Presence = Active
			} else if tile.OwningTribe != nil {
				tile.Presence = Passive
			} else {
				tile.Presence = None
			}
			if hasDiceBeenUsed {
				return true, gs.HandleStartRedeployment(attackerIndex)
			} else {
				gs.TurnInfo.Phase = Conquest
			}

			return true, nil
		}
		oldcheckPresence := t.checkPresence
		t.checkPresence = func(tile *Tile, race Race) bool {
			old := oldcheckPresence(tile, race)
			for _, stack := range(tile.PieceStacks) {
				if stack.Tribe != nil && stack.Tribe.Race == race {
					return true
				}
			}
			return old
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			old := oldCanBeRedeployedIn(tile, stackType, gs)
			if old {
				return old
			}
			for _, stack := range(tile.PieceStacks) {
				if stack.Tribe != nil && stack.Tribe.canBeRedeployedIn(tile, stackType, gs) {
					return true
				}
			}
			return old
		}
		oldGetStacksOutRedeployment := t.getStacksOutRedeployment
		t.getStacksOutRedeployment = func(tile *Tile, stackType string) ([]PieceStack, error) {
			stacks, err := oldGetStacksOutRedeployment(tile, stackType)
			if err != nil {
				for _, stack := range(tile.PieceStacks) {
					if stack.Tribe != nil {
						newstacks, err2 := stack.Tribe.getStacksOutRedeployment(tile, stackType)
						if err2 == nil {
							return newstacks, err2
						}
					}
				}
			}
			return stacks, err
		}
		oldHandleReturn := t.handleReturn
		t.handleReturn = func(tile *Tile, gs *GameState, pk int) {
			oldHandleReturn(tile, gs, pk)
			for _, stack := range tile.PieceStacks {
			    if stack.Tribe != nil {
				stack.Tribe.handleReturn(tile, gs, pk)
				return // Only one other tribe should be present
			    }
			}
		}
		oldhandleAbandonment := t.handleAbandonment
		t.handleAbandonment = func(tile *Tile, gs *GameState) {
			oldhandleAbandonment(tile, gs)
			for _, stack := range(tile.PieceStacks) {
				if stack.Tribe != nil && stack.Tribe != t {
					tile.OwningTribe = stack.Tribe
					tile.Presence = Passive // Scavengers only accept passive tribes
				}
			}
		}
		oldcountRemovablePieces := t.countRemovablePieces
		t.countRemovablePieces = func(tile *Tile) []PieceStack {
			oldStacks := oldcountRemovablePieces(tile)
			for _, stack := range(tile.PieceStacks) {
				if stack.Tribe != nil && stack.Tribe != t {
					oldStacks = append(oldStacks, stack)
				}
			}
			return oldStacks 
		}
		}, Count: 6},
	"Priestesses": {Transform: func(t *Tribe) {
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(stackType string) bool {
			return stackType == "Decline" || oldIsStackValid(stackType)
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			oldgoIntoDecline(gs)
			t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Decline", Amount: 1}})
			gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
				player: t.Owner.Index,
				TurnInfo: TurnInfo{
					TurnIndex: gs.TurnInfo.TurnIndex,
					PlayerIndex: gs.TurnInfo.PlayerIndex,
					Phase: Redeployment,
				},
				actionBefore: func(gs *GameState) {},
			})
			return
		}
		oldhandleDeploymentIn := t.handleDeploymentIn
		t.handleDeploymentIn = func (tile *Tile, stackType string, i int, gs *GameState) error {
			if t.IsActive || stackType != "Decline" {
				return oldhandleDeploymentIn(tile, stackType, i, gs)
			}
			count := 0
			for _, otherTile := range(gs.TileList) {
				if tile.Id != otherTile.Id && otherTile.Presence != None && otherTile.OwningTribe.checkPresence(otherTile, t.Race) {
					t.clearTile(otherTile, gs, 10000)
					count += 1
				}
			}
			for i := range(tile.PieceStacks) {
				if tile.PieceStacks[i].Type == string(t.Race) {
					tile.PieceStacks[i].Amount += count
				}
			}
			for i := range(t.Owner.PieceStacks) {
				if t.Owner.PieceStacks[i].Type == "Decline" {
					t.Owner.PieceStacks = append(t.Owner.PieceStacks[:i], t.Owner.PieceStacks[i+1:]...)
				}
			}
			gs.handleNextPlayerTurn()
			return nil
		}
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if !t.IsActive {
				for _, stack := range(tile.PieceStacks) {
					if stack.Type == string(t.Race) {
						count += stack.Amount - 1
					}
				}
				
			}
			return count
		}
		}, Count: 4},
	"Ice Witches": {Transform: func(t *Tribe) {
		oldStartRedeployment := t.startRedeployment
		t.startRedeployment = func(gs *GameState) []PieceStack {
			stacks := oldStartRedeployment(gs)
			count := 0
			for _, tile := range(gs.TileList) {
				if tile.Presence != None && tile.OwningTribe.checkPresence(tile, t.Race) {
					for _, attr := range(tile.Attributes) {
						if attr == Magic {
							count += 1
						}
					}
				}
			}
			stacks = append(stacks, PieceStack{Type:"Winter", Amount: count})
			return stacks
		}
		oldIsStackValid := t.IsStackValid
		t.IsStackValid = func(s string) bool {
			return oldIsStackValid(s) || s == "Winter"
		}
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile, p *Player) (int, int, int, error) {
			old, g, l, err := oldCountDefense(tile, p)
			if err != nil {
				return old, g, l, err
			}
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Winter" {
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
			if stackType == "Winter" {
				for _, stack := range tile.PieceStacks {
					if stack.Type == "Winter" {
						return false
					}
				}
				return true
			}
			return false
		}
		oldhandleDeploymentIn := t.handleDeploymentIn
		t.handleDeploymentIn = func (tile *Tile, stackType string, i int, gs *GameState) error {
			err := oldhandleDeploymentIn(tile, stackType, i, gs)
			if err != nil {
				return err
			}
			if stackType == "Winter" {
				tile.ModifierPoints["Winter"] = TileModifierPoints["Winter"]
			}
			return err
		}
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Winter" {
					return count + 1
				}
			}
			return count
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			oldgoIntoDecline(gs)
			for _, tile := range(gs.TileList) {
				delete(tile.ModifierPoints, "Winter")
				for i, stack := range(tile.PieceStacks) {
					if stack.Type == "Winter" {
						tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					}
				}
			}
		}
		}, Count: 5},
}
