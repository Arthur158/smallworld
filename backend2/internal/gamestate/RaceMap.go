package gamestate

import (
    "fmt"
    "log"
    "math/rand"
    "strings"
    "time"
)

type RaceValue struct {
    Transform func(*Tribe)
    Count     int
}

var RaceMap = map[Race]RaceValue{
    "Amazons": {Transform: func(t *Tribe) {
        t.canEndTurnMap["Amazons"] = func(gs *GameState) error {
            for _, stack := range t.Owner.PieceStacks {
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
        t.clearTileMap["Trolls"] = func(tile *Tile, gs *GameState, pk int) {
            for i, stack := range tile.PieceStacks {
                if stack.Type == "Lair" {
                    tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                    return // Exit after removal to avoid index shifting issues
                }
            }
        }
    }, Count: 5},
    "Wizards": {Transform: func(t *Tribe) {
        t.countPointsMap["Wizards"] = func(tile *Tile) int {
            count := 0
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
        t.countPointsMap["Khans"] = func(tile *Tile) int {
            if t.IsActive && (tile.Biome == Field || tile.Biome == Hill) {
                return 1
            } else if t.IsActive {
                return -1
            }
            return 0
        }
    }, Count: 5},
    "Humans": {Transform: func(t *Tribe) {
        t.countPointsMap["Humans"] = func(tile *Tile) int {
            if t.IsActive && tile.Biome == Field {
                return 1
            }
            return 0
        }
    }, Count: 5},
    "Dwarves": {Transform: func(t *Tribe) {
        t.countPointsMap["Dwarves"] = func(tile *Tile) int {
            count := 0
            for _, attr := range tile.Attributes {
                if attr == Mine {
                    count += 1
                }
            }
            return count
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
            val := t.State["holesleft"]
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
        t.countRemovablePiecesMap["Halflings"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
            for _, stack := range tile.PieceStacks {
                if stack.Type == "Hole" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.clearTileMap["Halflings"] = func(tile *Tile, gs *GameState, pk int) {
            for i, stack := range tile.PieceStacks {
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
            val := t.State["killcount"]
            var killcount int
            switch v := val.(type) {
            case float64:
                killcount = int(v)
            case int:
                killcount = v
            }
            if tile.CheckPresence() != None {
                t.State["killcount"] = killcount + 1
            }
        }
        t.startRedeploymentMap["Skeletons"] = func(gs *GameState) []PieceStack {
            val := t.State["killcount"]
            var killcount int
            switch v := val.(type) {
            case float64:
                killcount = int(v)
            case int:
                killcount = v
            }
            t.State["killcount"] = 0
            return []PieceStack{{Type: string(t.Race), Amount: killcount / 2}}
        }
    }, Count: 6},
    "Witch Doctors": {Transform: func(t *Tribe) {
        t.handleReturnMap["Witch Doctors"] = func(tile *Tile, gs *GameState, pk int) {
            if t.IsActive {
                diceThrow := RollDice()
                gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("Pygmies got back: %d Pawns!", diceThrow)})
                t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: max(0, diceThrow-(1-pk))}})
            }
        }
    }, Count: 6},
    "Elves": {Transform: func(t *Tribe) {
        t.handleReturnMap["Elves"] = func(tile *Tile, gs *GameState, pk int) {
            if t.IsActive {
                t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: pk}})
            }
        }
    }, Count: 6},
    "Pixies": {Transform: func(t *Tribe) {
        t.startRedeploymentMap["Pixies"] = func(gs *GameState) []PieceStack {
            for _, tile := range gs.TileList {
                if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
                    t.getStacksForConquest(tile, t.Owner)
                }
            }
            return []PieceStack{}
        }
        t.canBeRedeployedInMap["Pixies"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
            if old && stackType == string(t.Race) {
                return false
            }
            return old
        }
    }, Count: 11},
    "Barbarians": {Transform: func(t *Tribe) {
        t.canBeRedeployedInMap["Barbarians"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
            if old {
                return stackType != string(t.Race)
            }
            return false
        }
        t.canBeRedeployedOutMap["Barbarians"] = func(b bool, tile *Tile, s string) bool {
            if b && s == string(t.Race) {
                return false
            }
            return b
        }
    }, Count: 9},
    "Sorcerers": {Transform: func(t *Tribe) {
        t.countRemovablePiecesMap["Sorcerers"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
            for _, stack := range tile.PieceStacks {
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

            for _, stack := range tile.PieceStacks {
                if stack.Type == string(tile.OwningTribe.Race) && stack.Amount > 1 {
                    return true, fmt.Errorf("This tile contains more than one active pawn!")
                }
            }

            tile.handleAfterConquest(gs, t)
            tile.OwningTribe.handleReturn(tile, gs, 1)
            tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}})
            tile.OwningTribe = t
            t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Staff", Amount: 1}})
            return true, nil
        }
        t.countRemovableAttackingStacksMap["Sorcerers"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
                if stack.Type == "Staff" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.giveInitialStacksMap["Sorcerers"] = func() []PieceStack {
            return []PieceStack{{Type: "Staff", Amount: 1}}
        }
        t.getStacksForConquestTurnMap["Sorcerers"] = func(player *Player, gs *GameState) {
            if !t.IsActive {
                return
            }
            newstacks := []PieceStack{}
            for _, stack := range player.PieceStacks {
                if stack.Type != "Staff" {
                    newstacks = append(newstacks, stack)
                }
            }
            newstacks = append(newstacks, PieceStack{Type: "Staff", Amount: 1})
            player.PieceStacks = newstacks
        }
    }, Count: 5},
    "Wendigos": {Transform: func(t *Tribe) {
        t.getStacksForConquestTurnMap["Wendigos"] = func(player *Player, gs *GameState) {
            if !t.IsActive {
                return
            }
            newstacks := []PieceStack{}
            for _, stack := range player.PieceStacks {
                if stack.Type != "Power" {
                    newstacks = append(newstacks, stack)
                }
            }
            newstacks = append(newstacks, PieceStack{Type: "Power", Amount: 1})
            player.PieceStacks = newstacks
        }
        t.countRemovableAttackingStacksMap["Wendigos"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
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

            tile.handleAfterConquest(gs, nil)
            if tile.CheckPresence() != None {
                tile.OwningTribe.handleReturn(tile, gs, 1)
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
                    for _, id := range abandonedTileIds {
                        if id == tile.Id {
                            return fmt.Errorf("Abandoned zone cannot be reconquered")
                        }
                    }
                }
            }
            return old
        }
        t.countExtrapointsMap["Nomads"] = func(gs *GameState) int {
            abandonedTiles := t.State["abandonedTiles"].([]string)
            total := 0
            for _, tileId := range abandonedTiles {
                if tile, ok := gs.TileList[tileId]; ok {
                    tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Coin", Amount: 1}})
                    total += 1
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
        t.calculateRemainingAttackingStacksMap["Berserk"] = func(ps []PieceStack, diceUsed bool, ok bool, err error, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
            for _, stack := range ps {
                if stack.Type == string(t.Race) && stack.Amount < t.Minimum {
                    return ps, diceUsed, false, fmt.Errorf("Kobolds need at least 2 stacks!")
                }
            }
            return ps, diceUsed, ok, err
        }
    }, Count: 11},
    "Leprechauns": {Transform: func(t *Tribe) {
        t.giveInitialStacksMap["Leprechauns"] = func() []PieceStack {
            return []PieceStack{{Type: "Cauldron", Amount: 16}}
        }
        t.countRemovableAttackingStacksMap["Leprechauns"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
                if stack.Type == "Cauldron" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.IsStackValidMap["Leprechauns"] = func(stackType string) bool {
            return stackType == "Cauldron"
        }
        t.canBeRedeployedInMap["Leprechauns"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
            if stackType == "Cauldron" {
                for _, stack := range tile.PieceStacks {
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
            for _, stack := range tile.PieceStacks {
                if stack.Type == "Cauldron" {
                    loot = loot - 1
                }
            }
            return 0, 0, loot, nil
        }
        t.clearTileMap["Leprechauns"] = func(tile *Tile, gs *GameState, pk int) {
            for i, stack := range tile.PieceStacks {
                if stack.Type == "Cauldron" {
                    tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                    return // Exit after removal to avoid index shifting issues
                }
            }
        }
    }, Count: 6},
    "Drakons": {Transform: func(t *Tribe) {
        t.giveInitialStacksMap["Drakons"] = func() []PieceStack {
            return []PieceStack{{Type: "Drakon's Dragon", Amount: 3}}
        }
        t.countRemovableAttackingStacksMap["Drakons"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
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
            for _, stack := range ps {
                if stack.Type == "Drakon's Dragon" {
                    for i := range ps {
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
        t.startRedeploymentMap["Fauns"] = func(gs *GameState) []PieceStack {
            val := t.State["activecount"]
            var activecount int
            switch v := val.(type) {
            case float64:
                activecount = int(v)
            case int:
                activecount = v
            }
            t.State["activecount"] = 0
            return []PieceStack{{Type: string(t.Race), Amount: activecount / 2}}
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
        t.countRemovablePiecesMap["Ghouls"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
            newstacks := []PieceStack{}
            for _, stack := range oldStacks {
                if stack.Type != string(t.Race) {
                    stack.Tribe = t
                    newstacks = append(newstacks, stack)
                }
            }
            return newstacks
        }
        t.countRemovableAttackingStacksMap["Ghouls"] = func(oldStacks []PieceStack, player *Player) []PieceStack {
            newstacks := []PieceStack{}
            for _, stack := range oldStacks {
                if stack.Type != string(t.Race) {
                    stack.Tribe = t
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
            for _, stack := range tile.PieceStacks {
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
        t.goIntoDeclineMap["Ghouls"] = func(gs *GameState) {
            pawns, _ := t.State["deploy"].(map[string]int)
            for id, amount := range pawns {
                movingStack := []PieceStack{{Type: string(t.Race), Amount: amount}}
                tile := gs.TileList[id]
                tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)
                gs.Players[gs.TurnInfo.PlayerIndex].PieceStacks, _ = SubtractPieceStacks(gs.Players[gs.TurnInfo.PlayerIndex].PieceStacks, movingStack)
            }

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
            tile.handleAfterConquest(gs, t)
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
            for _, stack := range tile.PieceStacks {
                if stack.Tribe != nil && stack.Tribe.Race == race {
                    return true
                }
            }
            return false
        }
        t.canBeRedeployedInMap["Scavengers"] = func(old bool, tile *Tile, stackType string, gs *GameState) bool {
            if old {
                return old
            }
            for _, stack := range tile.PieceStacks {
                if stack.Tribe != nil && stack.Tribe.canBeRedeployedIn(tile, stackType, gs) {
                    return true
                }
            }
            return old
        }
        t.getStacksOutRedeploymentMap["Scavengers"] = func(tile *Tile, stackType string) ([]PieceStack, error) {
            for _, stack := range tile.PieceStacks {
                if stack.Tribe != nil {
                    newstacks, err2 := stack.Tribe.getStacksOutRedeployment(tile, stackType)
                    if err2 == nil {
                        return newstacks, err2
                    }
                }
            }
            return nil, fmt.Errorf("No One")
        }
        t.handleReturnMap["Scavengers"] = func(tile *Tile, gs *GameState, pk int) {
            for _, stack := range tile.PieceStacks {
                if stack.Tribe != nil {
                    stack.Tribe.handleReturn(tile, gs, pk)
                    return // Only one other tribe should be present
                }
            }
        }
        t.handleAbandonmentMap["Scavengers"] = func(tile *Tile, gs *GameState) {
            for _, stack := range tile.PieceStacks {
                if stack.Tribe != nil && stack.Tribe != t {
                    tile.OwningTribe = stack.Tribe
                }
            }
        }
        t.countRemovablePiecesMap["Scavengers"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
            for _, stack := range tile.PieceStacks {
                if stack.Tribe != nil && stack.Tribe != t {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.countDefenseMap["Scavengers"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
            a := 0
            b := 0
            c := 0
            minus, _, _, _ := tile.countDefense(gs)
            for _, stack := range tile.PieceStacks {
                if stack.Tribe != nil && stack.Tribe != t {
                    a2, b2, c2, err := stack.Tribe.countDefense(tile, p, gs)
                    if err != nil {
                        return a, b, c, err
                    }
                    a += a2 - minus
                    b += b2
                    c += c2
                }
            }
            return a, b, c, nil

        }
    }, Count: 6},
    "Priestesses": {Transform: func(t *Tribe) {
        t.IsStackValidMap["Priestesses"] = func(stackType string) bool {
            return stackType == "Decline"
        }
        t.goIntoDeclineMap["Priestesses"] = func(gs *GameState) {
            t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Decline", Amount: 1}})
            gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
                player: t.Owner.Index,
                TurnInfo: &TurnInfo{
                    TurnIndex:   gs.TurnInfo.TurnIndex,
                    PlayerIndex: gs.TurnInfo.PlayerIndex,
                    Phase:       Redeployment,
                },
                actionBefore: func(gs *GameState) {},
            })
        }
        t.handleDeploymentInMap["Priestesses"] = func(tile *Tile, stackType string, i int, gs *GameState) error {
            if t.IsActive || stackType != "Decline" {
                return fmt.Errorf("Not for the priestesses")
            }
            count := 0
            for _, otherTile := range gs.TileList {
                if tile.Id != otherTile.Id && otherTile.CheckPresence() != None && otherTile.OwningTribe.checkPresence(otherTile, t.Race) {
                    t.clearTile(otherTile, gs, 10000)
                    count += 1
                }
            }
            for i := range tile.PieceStacks {
                if tile.PieceStacks[i].Type == string(t.Race) {
                    tile.PieceStacks[i].Amount += count
                }
            }
            for i := range t.Owner.PieceStacks {
                if t.Owner.PieceStacks[i].Type == "Decline" {
                    t.Owner.PieceStacks = append(t.Owner.PieceStacks[:i], t.Owner.PieceStacks[i+1:]...)
                }
            }
            gs.handleNextPlayerTurn()
            return nil
        }
        t.countPointsMap["Priestesses"] = func(tile *Tile) int {
            if !t.IsActive {
                count := 0
                for _, stack := range tile.PieceStacks {
                    if stack.Type == string(t.Race) {
                        count += stack.Amount - 1
                    }
                }
                return count

            }
            return 0
        }
    }, Count: 4},
    "Ice Witches": {Transform: func(t *Tribe) {
        t.startRedeploymentMap["Ice Witches"] = func(gs *GameState) []PieceStack {
            count := 0
            for _, tile := range gs.TileList {
                if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
                    for _, attr := range tile.Attributes {
                        if attr == Magic {
                            count += 1
                        }
                    }
                }
            }
            return []PieceStack{{Type: "Winter", Amount: count}}
        }
        t.IsStackValidMap["Ice Witches"] = func(s string) bool {
            return s == "Winter"
        }
        t.handleDeploymentInMap["Ice Witches"] = func(tile *Tile, stackType string, i int, gs *GameState) error {
            if stackType == "Winter" {
                if tile.CheckPresence() == None || !tile.OwningTribe.checkPresence(tile, t.Race) {
                    return fmt.Errorf("This tile does not belong to you!")
                }
                for _, stack := range tile.PieceStacks {
                    if stack.Type == "Winter" {
                        return fmt.Errorf("Already a Winter there")
                    }
                }
                movingStack := []PieceStack{{Type: "Winter", Amount: 1}}
                newStacks, ok := SubtractPieceStacks(t.Owner.PieceStacks, movingStack)
                if !ok {
                    return fmt.Errorf("Cannot redeploy pieces you don't have")
                }
                t.Owner.PieceStacks = newStacks

                tile.PieceStacks = AddPieceStacks(tile.PieceStacks, movingStack)
                tile.ModifierPoints["Winter"] = TileModifierPoints["Winter"]
                tile.ModifierDefenses["Winter"] = TileModifierDefenses["Winter"]
                return nil
            }
            return fmt.Errorf("Not a Winter")
        }
        t.goIntoDeclineMap["Ice Witches"] = func(gs *GameState) {
            for _, tile := range gs.TileList {
                delete(tile.ModifierPoints, "Winter")
                for i, stack := range tile.PieceStacks {
                    if stack.Type == "Winter" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                    }
                }
            }
        }
        t.countRemovableAttackingStacksMap["Ice Witches"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
                if stack.Type == "Winter" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
    }, Count: 5},
    "Gnomes": {Transform: func(t *Tribe) {
        t.specialDefenseMap["Gnomes"] = func(gs *GameState, tile *Tile, attackingTribe *Tribe, attackingStackType string) (bool, error) {
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

            var err error
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
        t.State["pointsfromstart"] = 0
        t.countPointsMap["Escargots"] = func(tile *Tile) int {
            if t.IsActive {
                return -1
            }
            return 0
        }
        t.getStacksForConquestTurnMap["Escargots"] = func(p *Player, gs *GameState) {
            if t.IsActive {
                moneyCount := 0

                dummyTribe := CreateBaseTribe()
                dummyTribe.IsActive = t.IsActive
                dummyTribe.Race = t.Race
                dummyTribe.Trait = t.Trait

                for _, tile := range gs.TileList {
                    if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
                        moneyCount += dummyTribe.countPoints(tile)
                    }
                }

                gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The escargot just made %d Points for the start of their turn!", moneyCount)})
                t.State["pointsfromstart"] = moneyCount
            }
        }
        t.countExtrapointsMap["Escargots"] = func(gs *GameState) int {
            if !t.IsActive {
                return 0
            }
            points := t.State["pointsfromstart"].(int)
            t.State["pointsfromstart"] = 0

            return points
        }
    }, Count: 12},
    "Skags": {Transform: func(t *Tribe) {
        t.IsStackValidMap["Skags"] = func(s string) bool {
            return s == "Loot"
        }
        t.giveInitialStacksMap["Skags"] = func() []PieceStack {
            loots := []int{-1, -1, -1, 0, 0, 1, 1, 2, 2, 3}
            r := rand.New(rand.NewSource(time.Now().UnixNano()))
            r.Shuffle(len(loots), func(i, j int) { loots[i], loots[j] = loots[j], loots[i] })
            t.State["loots"] = loots
            return []PieceStack{}
        }
        t.countNewTileStacksMap["Skags"] = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
            for _, stack := range tile.PieceStacks {
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
        t.goIntoDeclineMap["Skags"] = func(gs *GameState) {
            for _, tile := range gs.TileList {
                delete(tile.ModifierAfterConquest, "Loot")
                delete(tile.ModifierSpecialDefenses, "Loot")
                for i, stack := range tile.PieceStacks {
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
        t.handleEndOfGameMap["Skags"] = func(gs *GameState) {
            for _, tile := range gs.TileList {
                delete(tile.ModifierPoints, "Loot")
                for i, stack := range tile.PieceStacks {
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
        t.handleDeploymentOutMap["Skags"] = func(tile *Tile, stackType string, gs *GameState) error {
            if stackType != "Loot" {
                return fmt.Errorf("Not for skags")
            }
            found := false
            for _, stack := range tile.PieceStacks {
                if stack.Type == "Loot" {
                    found = true
                }
            }
            if !found {
                return fmt.Errorf("not there")
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
    "Slingmen": {Transform: func(t *Tribe) {
        t.State["conqueredfromaway"] = false
        t.checkAdjacencyMap["Slingmen"] = func(tile *Tile, gs *GameState, err error) error {
            if err == nil {
                return nil
            }
            for _, tile2 := range tile.AdjacentTiles {
                for _, tile3 := range tile2.AdjacentTiles {
                    found := false
                    for _, tileComp := range tile.AdjacentTiles {
                        if tileComp == tile3 {
                            found = true
                        }
                    }
                    if !found && tile3.CheckPresence() != None && tile3.OwningTribe.checkPresence(tile3, t.Race) {
                        t.State["conqueredfromaway"] = true
                        return nil
                    }
                }
            }
            return err
        }
        t.postConquestMap["Slingmen"] = func(tile *Tile, gs *GameState) {
            if t.State["conqueredfromaway"] == true {
                t.State["conqueredfromaway"] = false
                t.Owner.CoinPile += 1
            }
        }
    },
        Count: 5},
    "Storm Giants": {Transform: func(t *Tribe) {
        t.IsStackValidMap["Storm Giants"] = func(s string) bool {
            return s == "Storm"
        }
        t.specialConquestMap["Storm Giants"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
            if stackType != "Storm" {
                return false, nil
            }

            if tile.Biome != Mountain {
                return true, fmt.Errorf("Storm can only be used on Mountain!")
            }

            _, mg, ld, _, _ := t.countAttack(tile, 0, string(t.Race))
            t.Owner.CoinPile += mg
            tile.handleAfterConquest(gs, t)
            if tile.CheckPresence() != None {
                _, gainDef, lossAttack, err := tile.OwningTribe.countDefense(tile, t.Owner, gs)
                tile.OwningTribe.Owner.CoinPile += gainDef - ld
                t.Owner.CoinPile += lossAttack
                if err != nil {
                    return true, err
                }
                tile.OwningTribe.handleReturn(tile, gs, 1)
            }
            tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}})
            tile.OwningTribe = t
            t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Storm", Amount: 1}, {Type: string(t.Race), Amount: 1}})
            return true, nil
        }
        t.checkAdjacencyMap["Storm Giants"] = func(t *Tile, gs *GameState, err error) error {
            if err == nil {
                return nil
            }
            if strings.HasSuffix(t.Id, "i") {
                return nil
            }
            return err
        }
        t.getStacksForConquestTurnMap["Storm Giants"] = func(player *Player, gs *GameState) {
            if !t.IsActive {
                return
            }
            newstacks := []PieceStack{}
            for _, stack := range player.PieceStacks {
                if stack.Type != "Power" {
                    newstacks = append(newstacks, stack)
                }
            }
            newstacks = append(newstacks, PieceStack{Type: "Storm", Amount: 2})
            player.PieceStacks = newstacks
        }
        t.countRemovableAttackingStacksMap["Storm Giants"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
                if stack.Type == "Storm" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
    },
        Count: 6},
    "Drows": {Transform: func(t *Tribe) {
        t.countPointsMap["Drows"] = func(tile *Tile) int {
            for _, neighbour := range tile.AdjacentTiles {
                if neighbour.CheckPresence() != None && neighbour.OwningTribe.checkPresence(neighbour, t.Race) {
                    return 0
                }
            }
            return 1
        }
    }, Count: 4},
    "Cultists": {Transform: func(t *Tribe) {
        t.State["hasmoved"] = false
        t.giveInitialStacksMap["Cultists"] = func() []PieceStack {
            return []PieceStack{{Type: "Great Ancient", Amount: 1}}
        }
        t.getStacksForConquestTurnMap["Cultists"] = func(p *Player, gs *GameState) {
            t.State["hasmoved"] = false
        }
        t.countDefenseMap["Cultists"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
            for _, stack := range tile.PieceStacks {
                if stack.Type == "Great Ancient" {
                    return 0, 0, 0, fmt.Errorf("A tile with the great ancient cannot be conquered!")
                }
            }
            return 0, 0, 0, nil
        }
        t.IsStackValidMap["Cultists"] = func(s string) bool {
            return s == "Great Ancient"
        }
        t.clearTileMap["Cultists"] = func(tile *Tile, gs *GameState, pk int) {
            for i, stack := range tile.PieceStacks {
                if stack.Type == "Great Ancient" {
                    tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                    return
                }
            }
        }
        t.calculateRemainingAttackingStacksMap["Cultists"] = func(ps []PieceStack, b1, b2 bool, err error, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
            if !b2 || err != nil {
                return ps, b1, b2, err
            }
            for _, stack := range t.Owner.PieceStacks {
                if stack.Type == "Great Ancient" {
                    ps = append(ps, stack)
                }
            }
            return ps, b1, b2, err
        }
        t.countRemovablePiecesMap["Cultists"] = func(oldStacks []PieceStack, tile *Tile) []PieceStack {
            for _, stack := range tile.PieceStacks {
                if stack.Type == "Great Ancient" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.countRemovableAttackingStacksMap["Cultists"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
                if stack.Type == "Great Ancient" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.computeDiscountMap["Cultists"] = func(tile *Tile) int {
            for _, neighbour := range tile.AdjacentTiles {
                for _, stack := range neighbour.PieceStacks {
                    if stack.Type == "Great Ancient" {
                        return 1
                    }
                }
            }
            return 0
        }
        t.specialConquestMap["Cultists"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
            if stackType != "Great Ancient" {
                return false, nil
            }

            if gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice {
                return true, fmt.Errorf("You are supposed to move the great ancient at the start of the turn!")
            }

            if !tile.OwningTribe.checkPresence(tile, t.Race) {
                return true, fmt.Errorf("Needs to be the tribe's own tile")
            }

            tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Great Ancient", Amount: 1}})
            t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Great Ancient", Amount: 1}})

            gs.TurnInfo.Phase = TileAbandonment
            t.State["hasmoved"] = true

            return true, nil
        }
        t.handleMovementMap["Cultists"] = func(stackType string, tileFrom, tileTo *Tile, gs *GameState) error {
            if gs.TurnInfo.Phase != TileAbandonment && gs.TurnInfo.Phase != DeclineChoice {
                return fmt.Errorf("You are supposed to move the great ancient at the start of the turn!")
            }
            if stackType != "Great Ancient" {
                return fmt.Errorf("not for you")
            }

            if tileFrom == tileTo {
                return fmt.Errorf("Cannot move great ancient on its own tile!")
            }

            if tileFrom.CheckPresence() == None || !tileFrom.OwningTribe.checkPresence(tileTo, t.Race) {
                return fmt.Errorf("Invalid starting tile")
            }
            if tileTo.CheckPresence() == None || !tileTo.OwningTribe.checkPresence(tileFrom, t.Race) {
                return fmt.Errorf("Invalid arriving tile")
            }

            found := false
            for _, stack := range tileFrom.PieceStacks {
                if stack.Type == stackType {
                    found = true
                    break
                }
            }
            if !found {
                return fmt.Errorf("No great ancient to move!")
            }

            hasPlayed := t.State["hasmoved"].(bool)
            if hasPlayed {
                return fmt.Errorf("the great ancient has already moved!")
            }

            tileFrom.PieceStacks, _ = SubtractPieceStacks(tileFrom.PieceStacks, []PieceStack{{Type: stackType, Amount: 1}})
            tileTo.PieceStacks = AddPieceStacks(tileTo.PieceStacks, []PieceStack{{Type: stackType, Amount: 1}})

            t.State["hasmoved"] = true

            return nil
        }
    }, Count: 5},
    "Flames": {Transform: func(t *Tribe) {
        t.giveInitialStacksMap["Flames"] = func() []PieceStack {
            return []PieceStack{{Type: "Volcano", Amount: 1}}
        }
        t.checkAdjacencyMap["Flames"] = func(tile *Tile, gs *GameState, err error) error {
            if gs.IsTribePresentOnTheBoard(t.Race) {
                return err
            } else {
                for _, stack := range t.Owner.PieceStacks {
                    if stack.Type == "Volcano" {
                        return fmt.Errorf("You first need to place the volcano")
                    }
                }
                for _, neighbour := range tile.AdjacentTiles {
                    for _, stack := range neighbour.PieceStacks {
                        if stack.Type == "Volcano" {
                            return nil
                        }
                    }
                }
                return fmt.Errorf("You need to enter the board next to the volcano")
            }
        }
        t.getStacksForConquestTurnMap["Flames"] = func(p *Player, gs *GameState) {
            for _, tile := range gs.TileList {
                for i, stack := range tile.PieceStacks {
                    if stack.Type == "Volcano" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        t.Owner.PieceStacks = append(t.Owner.PieceStacks, stack)
                    }
                }
            }
        }
        t.IsStackValidMap["Flames"] = func(s string) bool {
            return s == "Volcano"
        }
        t.countRemovableAttackingStacksMap["Flames"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
                if stack.Type == "Volcano" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.computeDiscountMap["Flames"] = func(tile *Tile) int {
            for _, neighbour := range tile.AdjacentTiles {
                if neighbour.Biome == AbyssalChasm {
                    for _, stack := range neighbour.PieceStacks {
                        if stack.Type == "Volcano" {
                            return 100000 // making the price tribe.minimum, maybe cleaner ways to do this
                        }
                    }
                }
            }
            return 0
        }
        t.specialConquestMap["Flames"] = func(gs *GameState, tile *Tile, stackType string) (bool, error) {
            if stackType != "Volcano" {
                return false, nil
            }

            if tile.Biome != AbyssalChasm {
                return true, fmt.Errorf("Needs to be an abysmal chasm")
            }

            tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Volcano", Amount: 1}})
            t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Volcano", Amount: 1}})

            return true, nil
        }
    }, Count: 4},
    "Iron Dwarves": {Transform: func(t *Tribe) {
        t.startRedeploymentMap["Iron Dwarves"] = func(gs *GameState) []PieceStack {
            count := 0
            for _, tile := range gs.TileList {
                if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, t.Race) {
                    for _, attr := range tile.Attributes {
                        if attr == Mine {
                            count += 1
                        }
                    }
                }
            }
            return []PieceStack{{Type: "Hammer", Amount: count}}
        }
        t.countAttackMap["Iron Dwarves"] = func(tile *Tile, cost int, stackType string) ([]PieceStack, int, int, int, error) {
            if stackType == "Hammer" {
                return []PieceStack{{Type: "Hammer", Amount: 1}, {Type: string(t.Race), Amount: max(t.Minimum, cost-1-t.computeDiscount(tile))}}, t.computeGainAttacker(tile), t.computeLossDefender(tile), t.computePawnKill(tile), nil
            }
            return []PieceStack{}, 0, 0, 0, fmt.Errorf("The piecestack was not recognized")
        }
        t.countNewTileStacksMap["Iron Dwarves"] = func(ps []PieceStack, tile *Tile, gs *GameState) []PieceStack {
            for i := range ps {
                if ps[i].Type == "Hammer" {
                    return append(ps[:i], ps[i+1:]...)
                }
            }
            return ps
        }
        t.countRemovableAttackingStacksMap["Iron Dwarves"] = func(oldStacks []PieceStack, p *Player) []PieceStack {
            for _, stack := range p.PieceStacks {
                if stack.Type == "Hammer" {
                    oldStacks = append(oldStacks, stack)
                }
            }
            return oldStacks
        }
        t.IsStackValidMap["Iron Dwarves"] = func(s string) bool {
            return s == "Hammer"
        }
    }, Count: 7},
    "Liches": {Transform: func(t *Tribe) {
        t.countDefenseMap["Liches"] = func(tile *Tile, p *Player, gs *GameState) (int, int, int, error) {
            g, l := 0, 0
            if !t.IsActive {
                g += 1
                l += 1
            }
            return 0, g, l, nil
        }
    }, Count: 4},
    "Lizardmen": {Transform: func(t *Tribe) {
        t.checkAdjacencyMap["Lizardmen"] = func(tile *Tile, gs *GameState, err error) error {
            if err == nil {
                return nil
            }
            toVisit := []*Tile{}
            visited := make(map[*Tile]bool)
            toVisit = append(toVisit, tile.AdjacentTiles...)

            for len(toVisit) > 0 {
                newTile := toVisit[0]
                toVisit = toVisit[1:]

                // Prevent revisiting
                if visited[newTile] {
                    continue
                }
                visited[newTile] = true

                for _, neighbour := range newTile.AdjacentTiles {
                    // Check if already visited
                    if visited[neighbour] {
                        continue
                    }

                    if neighbour.OwningTribe != nil && neighbour.OwningTribe.checkPresence(neighbour, t.Race) {
                        return nil
                    }
                    if neighbour.Biome == River && neighbour.CheckPresence() == None {
                        toVisit = append(toVisit, neighbour)
                    }
                }
            }
            return fmt.Errorf("The tile is not adjacent")
        }
    }, Count: 7},
    "Mudmen": {Transform: func(t *Tribe) {
        t.startRedeploymentTileMap["Mudmen"] = func(tile *Tile, gs *GameState) {
            if tile.Biome == Swamp {
                t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Mudmen", Amount: 1}})
            }
        }
    }, Count: 5},
    "Krakens": {Transform: func(t *Tribe) {
        t.startRedeploymentTileMap["Krakens"] = func(tile *Tile, gs *GameState) {
            if tile.Biome == River {
                tile.OwningTribe = t
                t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, t.countNewTileStacks([]PieceStack{{Type: string(t.Race), Amount: 1}}, tile, gs))
                tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}})
            }
        }
    }, Count: 5},
    "Mummies": {Transform: func(t *Tribe) {
        t.computeDiscountMap["Mummies"] = func(t *Tile) int {
            return -1
        }
    }, Count: 10},
    "Ogres": {Transform: func(t *Tribe) {
        t.computeDiscountMap["Ogres"] = func(tile *Tile) int {
            return 1
        }
    }, Count: 5},
    "Shrooms": {Transform: func(t *Tribe) {
        t.countPointsMap["Forest"] = func(tile *Tile) int {
            if t.IsActive && tile.Biome == Forest {
                return 1
            }
            return 0
        }
    }, Count: 5},
    "Spiderines": {Transform: func(t *Tribe) {
        t.checkAdjacencyMap["Spiderines"] = func(tile *Tile, gs *GameState, err error) error {
            if err == nil {
                return err
            }
            for _, neighbour := range tile.AdjacentTiles {
                if neighbour.Biome == AbyssalChasm {
                    return nil
                }
            }
            return err
        }
    }, Count: 7},
    "Will-o'-Wisps": {Transform: func(t *Tribe) {
        t.calculateRemainingAttackingStacksMap["Will-o'-Wisps"] = func(ps []PieceStack, b1, b2 bool, err error, tile *Tile, gs *GameState) ([]PieceStack, bool, bool, error) {
            found := false
            for _, attr := range tile.Attributes {
                if attr == Magic {
                    found = true
                }
            }
            if b1 || !b2 || err != nil || !found {
                return ps, b1, b2, err
            }
            diceThrow := RollDice()
            gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The %s threw the dice and got %d!", t.Race, diceThrow)})
            for i, stack := range ps {
                if stack.Type == string(t.Race) {
                    ps[i].Amount = max(t.Minimum, stack.Amount-diceThrow)
                }
            }
            return ps, b1, b2, err

        }
    }, Count: 6},
}

func InitRaceMap() {
    RaceMap["Shadow Mimes"] = RaceValue{Transform: func(t *Tribe) {
        t.handleEntryActionMap["Shadow Mime"] = func(i int, s string, gs *GameState) error {
            if s != "Mime" {
                return fmt.Errorf("Not a Mime")
            }
            trait := gs.TribeList[i].Trait
            gs.TribeList[i].Trait = t.Trait
            t.DeletePower(string(t.Trait), gs)
            t.GiveTrait(trait)
            for i, otherTrait := range t.AdditionalPowers {
                if otherTrait == trait {
                    t.AdditionalPowers = append(t.AdditionalPowers[:i], t.AdditionalPowers[i+1:]...)
                }
            }
            t.Trait = trait
            log.Println(t.Owner.PieceStacks)
            t.Owner.PieceStacks, _ = SubtractPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Mime", Amount: 1}})
            log.Println("aaaaa")
            gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The %s chose to mirror the trait %s!", t.Race, trait)})
            return nil
        }
        t.giveInitialStacksMap["Shadow Mimes"] = func() []PieceStack {
            return []PieceStack{{Type: "Mime", Amount: 1}}
        }
        t.countRemovableAttackingStacksMap["Shadow Mimes"] = func(ps []PieceStack, p *Player) []PieceStack {
            for _, stack := range t.Owner.PieceStacks {
                if stack.Type == "Mime" {
                    ps = append(ps, stack)
                }
            }
            return ps
        }
        t.IsStackValidMap["Shadow Mimes"] = func(s string) bool {
            return s == "Mime"
        }
    }, Count: 7}
}
