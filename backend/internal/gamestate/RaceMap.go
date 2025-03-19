package gamestate

import (
	"fmt"
	"math/rand"
	"time"
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
		t.countNewTileStacksMap["Trolls"] = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
			return AddPieceStacks(ps, []PieceStack{{Type: "Lair", Amount: 1}})
		}

		t.countDefenseMap["Trolls"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			return 1, 0, 0, nil
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
	"Dwarves": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			for _, attr := range(tile.Attributes) {
				if attr == Mine {
					count += 1
				} 
			}
			return max(0, count)
		}
		}, Count: 4},
	"Halflings": {Transform: func(t *Tribe) {
		t.State["holesleft"] = 2
		t.State["startedalready"] = false
		t.checkAdjacencyMap["Halflings"] = func(tile *Tile, gs *GameState, err error) error {
			if err != nil && !gs.IsTribePresentOnTheBoard(t.Race) && t.State["startedalready"] == false {
				t.State["startedalready"] = true
				return nil
			} else {
				t.State["startedalready"] = true
				return err
			}
		}

		t.countNewTileStacksMap["Halflings"] = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
			val := tile.State["holesleft"]
			var holesleft int
			switch v := val.(type) {
			case float64:
			    holesleft = int(v)
			case int:
			    holesleft = v
			}
			if holesleft > 0 {
				ps = AddPieceStacks(ps, []PieceStack{{Type: "Hole", Amount: 1}})
			}
			t.State["holesleft"] = holesleft - 1
			return ps
		}
		t.countDefenseMap["Halflings"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Hole" {
					return 0, 0, 0, fmt.Errorf("A hole in the ground cannot be conquered!")
				}
			}
			return 0, 0, 0, nil
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
		t.countDefenseMap["White Ladies"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			if !t.IsActive {
				return 0, 0, 0, fmt.Errorf("Cannot conquer white ladies when they are in decline") 
			}
			return 0, 0, 0, nil
		}
		}, Count: 2},
	"Tritons": {Transform: func(t *Tribe) {
		t.computeDiscountMap["Triton"] = func(tile *Tile) int {
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.Biome == Water {
					return 1
				}
			}
			return 0
		}
		}, Count: 6},
	"Shrubmen": {Transform: func(t *Tribe) {
		t.countDefenseMap["Shrubmen"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			if tile.Biome == Forest {
				return 0, 0, 0, fmt.Errorf("cannot conquer shrubman when they are in a forest") 
			}
			return 0, 0, 0, nil
		}
		}, Count: 6},
	"Ratmen": {Transform: func(t *Tribe) {
		}, Count: 8},
	"Goblins": {Transform: func(t *Tribe) {
		t.computeDiscountMap["Goblins"] = func(tile *Tile) int {
			if tile.CheckPresence() == Passive {
				return 1
			}
			return 0
		}
		}, Count: 6},
	"Giants": {Transform: func(t *Tribe) {
		t.computeDiscountMap["Giants"] = func(tile *Tile) int {
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.Biome == Mountain && neighbour.CheckPresence() != None && neighbour.OwningTribe.checkPresence(neighbour, t.Race) {
					return 1
				}
			}
			return 0
		}
		}, Count: 6},
	"Orcs": {Transform: func(t *Tribe) {
		t.computeGainAttackerMap["Orcs"] = func(tile *Tile) int {
			if tile.CheckPresence() != None {
				return 1
			}
			return 0
		}
		}, Count: 5},
	"Skeletons": {Transform: func(t *Tribe) {
		t.State["killcount"] = 0
		t.postConquestMap["Skeletons"] = func(tile *Tile, gs *GameState) {
			if tile.CheckPresence() != None {
				if killCount, ok := t.State["killcount"].(int); ok {
					t.State["killcount"] = killCount + 1
				}
			}
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
				gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("Pygmies got back: %d Pawns!", diceThrow)})
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
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
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
			if err == nil && stackType == string(t.Race) {
				return nil, fmt.Errorf("Barbarians cannot redeploy!")
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
		t.IsStackValidMap["Sorcerers"] = func(s string) bool {
			return s == "Staff"
		}
		t.specialConquestMap["Sorcerers"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
			if stackType != "Staff" {
				return false, nil
			}

			if tile.CheckPresence() != Active {
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

			if tile.CheckPresence() == Active {
				// Maybe do something with those, in case this is actually considered a conquest
				_, _, _, err := tile.OwningTribe.countDefense(tile, t.Owner, gs)
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
			tile.handleAfterConquest(gs)
			tile.OwningTribe = t
			t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Staff", Amount: 1}})
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
		t.getStacksForConquestTurnMap["Wendigos"] = func(player *Player, gs *GameState) {
			oldgetStacksForConquestTurn(player, gs)
			if !t.IsActive {
				return
			}
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
		t.IsStackValidMap["Wendigos"] = func(s string) bool {
			return s == "Power"
		}
		t.specialConquestMap["Wendigos"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
			attacker := t.Owner
			if stackType != "Power" {
				return false, nil
			}

			if tile.Biome != Forest {
				return true, fmt.Errorf("This is not a forest!")
			}

			if tile.CheckPresence() != None {
				// Maybe do something with those, in case this is actually considered a conquest
				_, _, _, err := tile.OwningTribe.countDefense(tile, attacker, gs)
				if err != nil {
					return true, err
				}
			} 

			if tile.CheckPresence() != None {
				tile.OwningTribe.clearTile(tile, gs, 1)
			}
			tile.OwningTribe = nil
			attacker.PieceStacks = AddPieceStacks(attacker.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}})
			attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, []PieceStack{{Type: "Power", Amount: 1}})
			gs.TurnInfo.Phase = Conquest
			return true, nil
		}
		}, Count: 6},
	"Nomads": {Transform: func(t *Tribe) {
		t.State["abandonedTiles"] = []string{}
		t.handleAbandonmentMap["Nomads"] = func(tile *Tile, gs *GameState) {
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Coin", Amount: 1, Tribe: t}})
			if abandonedTiles, ok := t.State["abandonedTiles"].([]string); ok {
				t.State["abandonedTiles"] = append(abandonedTiles, tile.Id)
			} else {
				// If it's not a slice of strings, reinitialize it
				t.State["abandonedTiles"] = []string{tile.Id}
			}
		}
		t.checkZoneAccessMap["Nomads"] = func(tile *Tile, old error) error {
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
		t.countDefenseMap["Scarecrows"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			return 0, 0, -1, nil
		}
		}, Count: 12},
	"Kobolds": {Transform: func(t *Tribe) {
		t.Minimum = 2
		}, Count: 11},
	"Leprechauns": {Transform: func(t *Tribe) {
		t.giveInitialStacksMap["Leprechauns"] = func() []PieceStack {
			return []PieceStack{{Type: "Cauldron", Amount: 16}}
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
		t.IsStackValidMap["Leprechauns"] = func(stackType string) bool {
			return stackType == "Cauldron"
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
		t.getStacksForConquestMap["Leprechauns"] = func(tile *Tile, p *Player) {
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Cauldron" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.CoinPile += 1
				}
			}
		}
		t.countDefenseMap["Leprechauns"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			loot := 0
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Cauldron" {
					loot = loot - 1
				}
			}
			return 0, 0, loot, nil
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
		t.giveInitialStacksMap["Drakons"] = func() []PieceStack {
			return []PieceStack{{Type: "Drakon's Dragon", Amount: 3}} 
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
		t.getStacksForConquestMap["Drakons"] = func(tile *Tile, p *Player) {
			for i, stack := range tile.PieceStacks {
				if stack.Type == "Drakon's Dragon" {
					tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					p.PieceStacks = AddPieceStacks(p.PieceStacks, []PieceStack{stack})
				}
			}
		}
		t.countDefenseMap["Drakons"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Drakon's Dragon" {
					return 0, 0, 0, fmt.Errorf("A tile with a drakon's dragon cannot be conquered")
				}
			}
			return 0, 0, 0, nil
		}

		t.IsStackValidMap["Drakons"] = func(stackType string) bool {
			return stackType == "Drakon's Dragon"
		}


		t.countAttackMap["Drakons"] = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
			if stackType == "Drakon's Dragon" {
				return []PieceStack{{Type: string(t.Race), Amount: 2}, {Type: "Drakon's Dragon", Amount: 1}}, t.computeGainAttacker(tile), t.computeLossDefender(tile), t.computePawnKill(tile), nil
			}
			return []PieceStack{}, 0, 0, 0, fmt.Errorf("The piecestack was not recognized")
		}
		t.countNewTileStacksMap["Drakons"] = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
			for _, stack := range(ps) {
				if stack.Type == "Drakon's Dragon" {
					for i := range(ps) {
						if ps[i].Type == string(t.Race) {
							ps[i].Amount -= 1
						}
					}
				}
			}
			return ps
		}
		}, Count: 6},
	"Fauns": {Transform: func(t *Tribe) {
		t.State["activecount"] = 0
		t.postConquestMap["Fauns"] = func(tile *Tile, gs *GameState) {
			if tile.CheckPresence() == Active {
				if activeCount, ok := t.State["activecount"].(int); ok {
					t.State["activecount"] = activeCount + 1
				}
			}
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
		t.computePawnKillMap["Fauns"] = func(tile *Tile) int {
			if tile.CheckPresence() == Active {
				return -1
			}
			return 0
		}
		}, Count: 5},
	"Ghouls": {Transform: func(t *Tribe) {
		t.State["hasThrownDice"] = false
		t.calculateRemainingAttackingStacksMap["Ghouls"] = func(ps []PieceStack, diceUsed bool, ok bool, err error, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
			hasThrownDice := t.State["hasThrownDice"].(bool)
			if !t.IsActive && hasThrownDice {
				return nil, true, false, fmt.Errorf("You already threw the dice for the ghouls")
			}
			if !t.IsActive && diceUsed {
				t.State["hasThrownDice"] = true
				if !ok {
					return nil, true, false, fmt.Errorf("The dice was not enough for your zombies")
				}
				return ps, false, ok, err
			}
			return ps, diceUsed, ok, err
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
		t.IsStackValidMap["Ghouls"] = func(s string) bool {
			return (s == string(t.Race) && !t.IsActive)
		}
		t.canTileBeAbandonedMap["Ghouls"] = func(tile *Tile) bool {
			return tile.OwningTribe.Race == t.Race
		}
		t.getStacksForConquestTurnMap["Ghouls"] = func(player *Player, gs *GameState) {
			t.State["deploy"] = make(map[string]int)
			t.State["hasThrownDice"] = false
		}
		t.getStacksForConquestMap["Ghouls"] = func(tile *Tile, player *Player) {
			pawns, _ := t.State["deploy"].(map[string]int)
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == string(t.Race) {
					pawns[tile.Id] = stack.Amount - 1
				}
			}
			t.State["deploy"] = pawns
			if !t.IsActive {
			    for _, stack := range tile.PieceStacks {
				if stack.Type == string(t.Race) {
				    // Making sure the action is atomic
				    movingStack := []PieceStack{{Type: stack.Type, Amount: stack.Amount - t.Minimum}}
				    tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, movingStack)
				    player.PieceStacks = AddPieceStacks(player.PieceStacks, movingStack)
				}
			    }
			}
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			pawns, _ := t.State["deploy"].(map[string]int)
			for id, amount := range(pawns) {
				movingStack := []PieceStack{{Type: string(t.Race), Amount: amount}}
				tile := gs.TileList[id]
				tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)
				gs.Players[gs.TurnInfo.PlayerIndex].PieceStacks, _ = SubtractPieceStacks(gs.Players[gs.TurnInfo.PlayerIndex].PieceStacks, movingStack)
			}
			oldgoIntoDecline(gs)

		}
		}, Count: 5},
	"Scavengers": {Transform: func(t *Tribe) {
		t.specialConquestMap["Scavengers"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
			if tile.CheckPresence() != Passive {
				return false, nil
			}

			defendingTribe := tile.OwningTribe
			attacker := t.Owner


			if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
				return true, fmt.Errorf("tribe cannot attack itself")
			}

			if err := t.checkZoneAccess(tile); err != nil {
				return true, err
			}
			if err := t.checkAdjacency(tile, gs); err != nil {
				return true, err
			}

			tileCost, moneyGainDefender, moneyLossAttacker, err := defendingTribe.countDefense(tile, attacker, gs)
			if err != nil {
				return true, err
			}

			attackCostStacks, moneyGainAttacker, moneyLossDefender, _, err := t.countAttack(tile, tileCost, stackType)
			if err != nil {
				return true, err
			}
			stacks, hasDiceBeenUsed, ok, err := t.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
			newTileStacks := t.countNewTileStacks(stacks, tile, gs)
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

			if hasDiceBeenUsed {
				return true, gs.HandleStartRedeployment(attacker.Index)
			} else {
				gs.TurnInfo.Phase = Conquest
			}

			return true, nil
		}
		t.checkPresenceMap["Scavengers"] = func(tile *Tile, race Race) bool {
			for _, stack := range(tile.PieceStacks) {
				if stack.Tribe != nil && stack.Tribe.Race == race {
					return true
				}
			}
			return false
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
		t.handleAbandonmentMap["Scavengers"] = func(tile *Tile, gs *GameState) {
			for _, stack := range(tile.PieceStacks) {
				if stack.Tribe != nil && stack.Tribe != t {
					tile.OwningTribe = stack.Tribe
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
		t.IsStackValidMap["Priestesses"] = func(stackType string) bool {
			return stackType == "Decline"
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			oldgoIntoDecline(gs)
			t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Decline", Amount: 1}})
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
		oldhandleDeploymentIn := t.handleDeploymentIn
		t.handleDeploymentIn = func (tile *Tile, stackType string, i int, gs *GameState) error {
			if t.IsActive || stackType != "Decline" {
				return oldhandleDeploymentIn(tile, stackType, i, gs)
			}
			count := 0
			for _, otherTile := range(gs.TileList) {
				if tile.Id != otherTile.Id && otherTile.CheckPresence() != None && otherTile.OwningTribe.checkPresence(otherTile, t.Race) {
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
				if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
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
		t.IsStackValidMap["Ice Witches"] = func(s string) bool {
			return s == "Winter"
		}
		t.countDefenseMap["Ice Witches"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
			def := 0
			for _, stack := range tile.PieceStacks {
				if stack.Type == "Winter" {
					def += stack.Amount
				}
			}
			return def, 0, 0, nil
		}
		oldCanBeRedeployedIn := t.canBeRedeployedIn
		t.canBeRedeployedIn = func(tile *Tile, stackType string, gs *GameState) bool {
			if oldCanBeRedeployedIn(tile, stackType, gs) {
				return true
			}
			if stackType == "Winter" {
				if tile.CheckPresence() == None || !tile.OwningTribe.checkPresence(tile, t.Race) {
					return false
				}
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
	"Gnomes": {Transform: func(t *Tribe) {
		oldSpecialDefense := t.specialDefense
		t.specialDefense = func(gs *GameState, tile *Tile, attackingTribe *Tribe, attackingStackType string) (bool, error) {
			ok, err := oldSpecialDefense(gs, tile, attackingTribe, attackingStackType)
			if ok {
				return ok, err
			}

			attacker := attackingTribe.Owner
			attackerIndex := attacker.Index

			if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, attackingTribe.Race) {
				return true, fmt.Errorf("This tile already belongs to the tribe!")
			}

			if err := attackingTribe.checkZoneAccess(tile); err != nil {
				return true, err
			}
			if err := attackingTribe.checkAdjacency(tile, gs); err != nil {
				return true, err
			}

			tileCost, moneyGainDefender, moneyLossAttacker := 0, 0, 0
			if tile.CheckPresence() != None {
				tileCost, moneyGainDefender, moneyLossAttacker, err = tile.OwningTribe.countDefense(tile, attacker, gs)
			} else {
				tileCost, moneyGainDefender, moneyLossAttacker, err = tile.countDefense(gs)
			}
			
			if err != nil {
				return true, err
			}

			// Here the gnome magic happens
			dummyTribe := CreateBaseTribe()
			dummyTribe.IsActive = attackingTribe.IsActive
			dummyTribe.Race = attackingTribe.Race
			dummyTribe.Trait = attackingTribe.Trait

			attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill, err := dummyTribe.countAttack(tile, tileCost, attackingStackType)
			if err != nil {
				return true, err
			}
			newStacks, hasDiceBeenUsed, ok, err := attackingTribe.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
			if err != nil {
				return true, fmt.Errorf("Your power has no effect against the gnomes!")
			}
			if !ok {
				return true, gs.HandleStartRedeployment(attackerIndex)
			}

			// Enact changes
			if tile.CheckPresence() != None {
				tile.OwningTribe.Owner.CoinPile += moneyGainDefender - moneyLossDefender
				tile.OwningTribe.handleReturn(tile, gs, pawnKill)
			}

			attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, newStacks)
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, attackingTribe.countNewTileStacks(newStacks, tile, gs))

			attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
			// attacker.PointsEachTurn[len(attacker.PointsEachTurn) - 1] += moneyGainDefender - moneyLossDefender
			tile.OwningTribe = attackingTribe

			if hasDiceBeenUsed {
				return true, gs.HandleStartRedeployment(attackerIndex)
			} else {
				gs.TurnInfo.Phase = Conquest
			}

			return true, nil
		}
		}, Count: 6},
	"Escargots": {Transform: func(t *Tribe) {
		oldCountPoints := t.countPoints
		t.countPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive {
				count -= 1
			} 
			return max(0, count)
		}
		t.getStacksForConquestTurnMap["Escargots"] = func(p *Player, gs *GameState) {
			if t.IsActive {
				moneyCount := 0

				dummyTribe := CreateBaseTribe()
				dummyTribe.IsActive = t.IsActive
				dummyTribe.Race = t.Race
				dummyTribe.Trait = t.Trait

				for _, tile := range(gs.TileList) {
					if tile.CheckPresence() != None  && tile.OwningTribe.checkPresence(tile, t.Race) {
						moneyCount += dummyTribe.countPoints(tile)
					}
				}

				gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The escargot just made %d Points for the start of their turn!", moneyCount)})
				p.CoinPile += moneyCount
			}
		}
		}, Count: 12},
	"Skags": {Transform: func(t *Tribe) {
		t.IsStackValidMap["Skags"] = func(s string) bool {
			return s == "Loot"
		}
		t.giveInitialStacksMap["Skags"] = func() []PieceStack {
			loots := []int{-1,-1,-1,0,0,1,1,2,2,3}
			r := rand.New(rand.NewSource(time.Now().UnixNano()))
			r.Shuffle(len(loots), func(i, j int) { loots[i], loots[j] = loots[j], loots[i] })
			t.State["loots"] = loots
			return []PieceStack{}
		}
		t.countNewTileStacksMap["Skags"] = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Loot" {
					return ps
				}
			}
			raw := t.State["loots"]

			var loots []int
			ifaceSlice, ok := raw.([]interface{})
			if !ok {
			    if directSlice, ok2 := raw.([]int); ok2 {
				loots = directSlice
			    }
			} else {
			    loots = make([]int, len(ifaceSlice))
			    for i, val := range ifaceSlice {
				switch v := val.(type) {
				case float64:
				    loots[i] = int(v)
				case int:
				    loots[i] = v
				}
			    }
			}
			
			if len(loots) > 0 {
				tile.State["loot"] = loots[0]
				tile.ModifierAfterConquest["Loot"] = TileModifierAfterConquests["Loot"]
				tile.ModifierSpecialDefenses["Loot"] = TileModifierSpecialDefenses["Loot"]
				ps = append(ps, PieceStack{Type: "Loot", Amount: 1})

				if len(loots) > 1 {
					t.State["loots"] = loots[1:]
				} else {
					t.State["loots"] = []int{} 
				}

				if loots[0] == -1 {
					gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The %s now contains a Skag Attack!", tile.Biome), Receivers: []int{t.Owner.Index}})
				} else {
					gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The %s was attributed a loot of %d!", tile.Biome, loots[0]), Receivers: []int{t.Owner.Index}})
				}

			}

			return ps
		}
		oldgoIntoDecline := t.goIntoDecline
		t.goIntoDecline = func(gs *GameState) {
			oldgoIntoDecline(gs)
			for _, tile := range(gs.TileList) {
				delete(tile.ModifierAfterConquest, "Loot")
				delete(tile.ModifierSpecialDefenses, "Loot")
				for i, stack := range(tile.PieceStacks) {
					if stack.Type == "Loot" {
						tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					}
				}
				val := tile.State["loot"]
				var loot int
				switch v := val.(type) {
				case float64:
				    loot = int(v)
				case int:
				    loot = v
				}
				if loot != -1 && loot != 0 {
					t.Owner.CoinPile += loot
					gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The loot was: %d", loot)})
				}
			}
		}
		oldhandleEndOfGame := t.handleEndOfGame
		t.handleEndOfGame = func(gs *GameState) {
			oldhandleEndOfGame(gs)
			for _, tile := range(gs.TileList) {
				delete(tile.ModifierPoints, "Loot")
				for i, stack := range(tile.PieceStacks) {
					if stack.Type == "Loot" {
						tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
					}
				}
				val := tile.State["loot"]
				var loot int
				switch v := val.(type) {
				case float64:
				    loot = int(v)
				case int:
				    loot = v
				}
				if loot != -1 {
					t.Owner.CoinPile += loot
				}
			}
		}
		oldhandleDeploymentOut := t.handleDeploymentOut
		t.handleDeploymentOut = func (tile *Tile, stackType string, i int, gs *GameState) error {
			err := oldhandleDeploymentOut(tile, stackType, i, gs) 
			if err == nil {
				return nil
			}
			if stackType != "Loot" {
				return err
			}
			found := false
			for _, stack := range(tile.PieceStacks) {
				if stack.Type == "Loot" {
					found = true
				}
			}
			if !found {
				return err
			}
			val := tile.State["loot"]
			var loot int
			switch v := val.(type) {
			case float64:
			    loot = int(v)
			case int:
			    loot = v
			}

			if loot == -1 {
				gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The %s contains a Skag Attack!", tile.Biome), Receivers: []int{t.Owner.Index}})
			} else {
				gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The %s has a loot of %d!", tile.Biome, loot), Receivers: []int{t.Owner.Index}})
			}
			return nil
		}
		}, Count: 6},
}
