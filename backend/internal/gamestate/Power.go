package gamestate
import "math/rand"
import "time"
import "fmt"

type Power struct {
    Name string
    State map[string]interface{};
    Owner *Player
    Tile *Tile
    Spawn func(*Tile, *Tribe, *GameState)
    CountPoints func(*GameState) int;
    GetStacksForConquest func(gs *GameState)
    StartRedeployment func(*GameState) []PieceStack
    HandleRedeploymentIn func(*Tile, string, *GameState) error
    HandleMovement func(string, *Tile, *Tile, *GameState) error
    HandleConquest func(gs *GameState, tile *Tile, s string) (bool, error)
}

var PowerMap = map[string]func()*Power {
    "Scepter of Avarice": func() *Power {
        power := &Power{
                Name: "Scepter of Avarice",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            gs.Powers["Scepter of Avarice"] = power
            tribe.Owner.PieceStacks = append(tribe.Owner.PieceStacks, PieceStack{Type: "Scepter of Avarice", Amount: 1})
            power.Owner = tribe.Owner
            delete(t.ModifierAfterConquest, "Scepter of Avarice Spawn")
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for _, tile := range(gs.TileList) {
                for i, stack := range(tile.PieceStacks) {
                    if stack.Type == "Scepter of Avarice" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        power.Owner.PieceStacks = AddPieceStacks(power.Owner.PieceStacks, []PieceStack{stack})
                        delete(tile.ModifierAfterConquest, "Scepter of Avarice")
                    }
                }
            }
        }
        power.HandleRedeploymentIn = func(tile *Tile, s string, gs *GameState) error {
            if tile.OwningTribe.Owner != power.Owner {
                return fmt.Errorf("This tile does not belong to you!")
            }
            tile.PieceStacks = append(tile.PieceStacks, PieceStack{Type: "Scepter of Avarice", Amount: 1})
            tile.ModifierAfterConquest["Scepter of Avarice"] = TileModifierAfterConquests["Scepter of Avarice"]
            power.Tile = tile
            power.Owner.PieceStacks, _ = SubtractPieceStacks(power.Owner.PieceStacks, []PieceStack{{Type: "Scepter of Avarice", Amount: 1}})
            power.Owner.CoinPile += tile.OwningTribe.countPoints(tile)
            return nil
        }
        return power
    },
    "Froggy's Ring": func() *Power {
        power := &Power{
                Name: "Froggy's Ring",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Froggy's Ring Spawn")
            gs.Powers["Froggy's Ring"] = power
            tribe.Owner.PieceStacks = append(tribe.Owner.PieceStacks, PieceStack{Type: "Froggy's Ring", Amount: 1})
            power.Owner = tribe.Owner
            delete(t.ModifierAfterConquest, "Froggy's Ring")
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for _, tile := range(gs.TileList) {
                for i, stack := range(tile.PieceStacks) {
                    if stack.Type == "Froggy's Ring" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        power.Owner.PieceStacks = AddPieceStacks(power.Owner.PieceStacks, []PieceStack{stack})
                        delete(tile.ModifierAfterConquest, "Froggy's Ring")
                    }
                }
            }
        }
        power.HandleRedeploymentIn = func(tile *Tile, s string, gs *GameState) error {
            if tile.OwningTribe.Owner != power.Owner {
                return fmt.Errorf("This tile does not belong to you!")
            }
            tile.PieceStacks = append(tile.PieceStacks, PieceStack{Type: "Froggy's Ring", Amount: 1})
            tile.ModifierAfterConquest["Froggy's Ring"] = TileModifierAfterConquests["Froggy's Ring"]
            power.Tile = tile
            power.Owner.PieceStacks, _ = SubtractPieceStacks(power.Owner.PieceStacks, []PieceStack{{Type: "Froggy's Ring", Amount: 1}})
            indices := []int{}
            for _, tile2 := range(tile.AdjacentTiles) {
                if tile2.CheckPresence() != Active {
                    continue
                }
                index := tile2.OwningTribe.Owner.Index
                found := false
                for _, i := range(indices) {
                    if index == i {
                        found = true
                    }
                }
                if !found {
                    indices = append(indices, index)
                    tile2.OwningTribe.Owner.CoinPile -= 1
                    power.Owner.CoinPile += 1
                }
            }
            return nil
        }
        return power
    },
    "Shiny Orb": func() *Power {
        power := &Power{
                Name: "Shiny Orb",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Shiny Orb Spawn")
            gs.Powers["Shiny Orb"] = power
            tribe.Owner.PieceStacks = append(tribe.Owner.PieceStacks, PieceStack{Type: "Shiny Orb", Amount: 1})
            power.Owner = tribe.Owner
            delete(t.ModifierAfterConquest, "Shiny Orb")
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for _, tile := range(gs.TileList) {
                for i, stack := range(tile.PieceStacks) {
                    if stack.Type == "Shiny Orb" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        power.Owner.PieceStacks = AddPieceStacks(power.Owner.PieceStacks, []PieceStack{stack})
                    }
                }
            }
        }
        power.HandleConquest = func(gs *GameState, tile *Tile, attackingStackType string) (bool, error) {
			if attackingStackType != "Shiny Orb" {
				return false, nil
			}
                        if power.Owner.ActiveTribe == nil {
                            return true, fmt.Errorf("The player does not have an active tribe")
                        }
                        attacker := power.Owner
                        t := attacker.ActiveTribe

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

			tile.OwningTribe.handleReturn(tile, gs, 1)
			tile.PieceStacks = AddPieceStacks(tile.PieceStacks, []PieceStack{{Type: string(t.Race), Amount: 1}, {Type: "Shiny Orb", Amount: 1}})
			tile.handleAfterConquest(gs, t)
                        tile.ModifierAfterConquest["Shiny Orb"] = TileModifierAfterConquests["Shiny Orb"]
                        attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, []PieceStack{{Type: "Shiny Orb", Amount: 1}})
			tile.OwningTribe = t
			return true, nil
        }
        return power
    },
    "Sword of the Killer Rabbit": func() *Power {
        power := &Power{
                Name: "Sword of the Killer Rabbit",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Sword of the Killer Rabbit Spawn")
            gs.Powers["Sword of the Killer Rabbit"] = power
            tribe.Owner.PieceStacks = append(tribe.Owner.PieceStacks, PieceStack{Type: "Sword of the Killer Rabbit", Amount: 1})
            power.Owner = tribe.Owner
            delete(t.ModifierAfterConquest, "Sword of the Killer Rabbit")
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for _, tile := range(gs.TileList) {
                for i, stack := range(tile.PieceStacks) {
                    if stack.Type == "Sword of the Killer Rabbit" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        power.Owner.PieceStacks = AddPieceStacks(power.Owner.PieceStacks, []PieceStack{stack})
                    }
                }
            }
        }
        power.HandleConquest = func(gs *GameState, tile *Tile, attackingStackType string) (bool, error) {
            if attackingStackType != "Sword of the Killer Rabbit" {
                return false, nil
            }
            if power.Owner.ActiveTribe == nil {
                return true, fmt.Errorf("The player does not have an active tribe")
            }
            attacker := power.Owner
            attackingTribe := attacker.ActiveTribe
            
            if ok, err := tile.specialDefense(gs, attackingTribe, attackingStackType); ok {
                    return true, err
            }

            if tile.CheckPresence() != None {
                ok, err := tile.OwningTribe.specialDefense(gs, tile, attackingTribe, attackingStackType)
                if ok {
                    return true, err
                }
            }

            ok, err := attackingTribe.specialConquest(gs, tile, attackingStackType)
            if ok {
                    return true, err
            }

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

            // counts the cost for the attacker
            attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill, err := attackingTribe.countAttack(tile, tileCost, string(attackingTribe.Race))
            if err != nil {
                    return true, err
            }
            attackCostStacks = AddPieceStacks(attackCostStacks, []PieceStack{{Type: "Sword of the Killer Rabbit", Amount: 1}})
            for i, stack := range(attackCostStacks) {
                if stack.Type == string(attackingTribe.Race) {
                    attackCostStacks[i].Amount = max(1, stack.Amount - 2)
                }
            }

            newStacks, hasDiceBeenUsed, ok, err := attackingTribe.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
            if err != nil {
                    return true, err
            }
            if !ok {
                    return true, gs.HandleStartRedeployment(attacker.Index)
            }

            // Enact changes
            tile.handleAfterConquest(gs, attackingTribe)
            attackingTribe.postConquest(tile, gs)
            power.Tile = tile
            tile.ModifierAfterConquest["Sword of the Killer Rabbit"] = TileModifierAfterConquests["Sword of the Killer Rabbit"]
            if tile.CheckPresence() != None {
                    tile.OwningTribe.Owner.CoinPile += moneyGainDefender - moneyLossDefender
                    tile.OwningTribe.handleReturn(tile, gs, pawnKill)
            }

            attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, newStacks)
            tile.PieceStacks = AddPieceStacks(tile.PieceStacks, attackingTribe.countNewTileStacks(newStacks, tile, gs))

            attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
            tile.OwningTribe = attackingTribe

            if hasDiceBeenUsed {
                    return true, gs.HandleStartRedeployment(attacker.Index)
            } else {
                    gs.TurnInfo.Phase = Conquest
            }

            return true, nil
        }
        return power
    },
    "Stinky Troll's Socks": func() *Power {
        power := &Power{
                Name: "Stinky Troll's Socks",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Stinky Troll's Socks Spawn")
            gs.Powers["Stinky Troll's Socks"] = power
            tribe.Owner.PieceStacks = append(tribe.Owner.PieceStacks, PieceStack{Type: "Stinky Troll's Socks", Amount: 1})
            power.Owner = tribe.Owner
            delete(t.ModifierAfterConquest, "Stinky Troll's Socks")
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for _, tile := range(gs.TileList) {
                for i, stack := range(tile.PieceStacks) {
                    if stack.Type == "Stinky Troll's Socks" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        power.Owner.PieceStacks = AddPieceStacks(power.Owner.PieceStacks, []PieceStack{stack})
                    }
                }
            }
        }
        power.HandleConquest = func(gs *GameState, tile *Tile, attackingStackType string) (bool, error) {
            if attackingStackType != "Stinky Troll's Socks" {
                return false, nil
            }
            if power.Owner.ActiveTribe == nil {
                return true, fmt.Errorf("The player does not have an active tribe")
            }
            attacker := power.Owner
            attackingTribe := attacker.ActiveTribe
            
            if ok, err := tile.specialDefense(gs, attackingTribe, attackingStackType); ok {
                    return true, err
            }

            if tile.CheckPresence() != None {
                ok, err := tile.OwningTribe.specialDefense(gs, tile, attackingTribe, attackingStackType)
                if ok {
                    return true, err
                }
            }

            ok, err := attackingTribe.specialConquest(gs, tile, attackingStackType)
            if ok {
                    return true, err
            }

            if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, attackingTribe.Race) {
                    return true, fmt.Errorf("This tile already belongs to the tribe!")
            }

            if err := attackingTribe.checkZoneAccess(tile); err != nil {
                    return true, err
            }
            if err := attackingTribe.checkAdjacency(tile, gs); err != nil {
                    return true, err
            }

            // stinky socks magic, i dont know if the modifier defenses should come into play on that one.
            dummyTile := Tile{
                Biome: tile.Biome,
                Attributes: tile.Attributes,
            }
            tileCost, moneyGainDefender, moneyLossAttacker, err := dummyTile.countDefense(gs)
            if err != nil {
                    return true, err
            }

            // counts the cost for the attacker
            attackCostStacks, moneyGainAttacker, moneyLossDefender, _, err := attackingTribe.countAttack(tile, tileCost, string(attackingTribe.Race))
            if err != nil {
                    return true, err
            }
            attackCostStacks = AddPieceStacks(attackCostStacks, []PieceStack{{Type: "Stinky Troll's Socks", Amount: 1}})

            newStacks, hasDiceBeenUsed, ok, err := attackingTribe.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
            if err != nil {
                    return true, err
            }
            if !ok {
                    return true, gs.HandleStartRedeployment(attacker.Index)
            }

            // Enact changes
            tile.handleAfterConquest(gs, attackingTribe)
            attackingTribe.postConquest(tile, gs)
            power.Tile = tile
            tile.ModifierAfterConquest["Stinky Troll's Socks"] = TileModifierAfterConquests["Stinky Troll's Socks"]
            if tile.CheckPresence() != None {
                    tile.OwningTribe.Owner.CoinPile += moneyGainDefender - moneyLossDefender
                    tile.OwningTribe.handleReturn(tile, gs, 0)
            }

            attacker.PieceStacks, _ = SubtractPieceStacks(attacker.PieceStacks, newStacks)
            tile.PieceStacks = AddPieceStacks(tile.PieceStacks, attackingTribe.countNewTileStacks(newStacks, tile, gs))

            attacker.CoinPile += moneyGainAttacker - moneyLossAttacker
            tile.OwningTribe = attackingTribe

            if hasDiceBeenUsed {
                    return true, gs.HandleStartRedeployment(attacker.Index)
            } else {
                    gs.TurnInfo.Phase = Conquest
            }

            return true, nil
        }
        return power
    },
    "Flying Doormat": func() *Power {
        power := &Power{
                Name: "Flying Doormat",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Flying Doormat Spawn")
            gs.Powers["Flying Doormat"] = power
            tribe.Owner.PieceStacks = append(tribe.Owner.PieceStacks, PieceStack{Type: "Flying Doormat", Amount: 1})
            power.Owner = tribe.Owner
            delete(t.ModifierAfterConquest, "Flying Doormat")
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for _, tile := range(gs.TileList) {
                for i, stack := range(tile.PieceStacks) {
                    if stack.Type == "Flying Doormat" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        power.Owner.PieceStacks = AddPieceStacks(power.Owner.PieceStacks, []PieceStack{stack})
                    }
                }
            }
        }
        power.HandleConquest = func(gs *GameState, tile *Tile, attackingStackType string) (bool, error) {
            if attackingStackType != "Flying Doormat" {
                return false, nil
            }
            if power.Owner.ActiveTribe == nil {
                return true, fmt.Errorf("The player does not have an active tribe")
            }
            attacker := power.Owner
            attackingTribe := attacker.ActiveTribe
            
            if ok, err := tile.specialDefense(gs, attackingTribe, attackingStackType); ok {
                    return true, err
            }

            if tile.CheckPresence() != None {
                ok, err := tile.OwningTribe.specialDefense(gs, tile, attackingTribe, attackingStackType)
                if ok {
                    return true, err
                }
            }

            ok, err := attackingTribe.specialConquest(gs, tile, attackingStackType)
            if ok {
                    return true, err
            }

            if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, attackingTribe.Race) {
                    return true, fmt.Errorf("This tile already belongs to the tribe!")
            }

            if err := attackingTribe.checkZoneAccess(tile); err != nil {
                    return true, err
            }
            // No check for adjacency

            tileCost, moneyGainDefender, moneyLossAttacker := 0, 0, 0
            if tile.CheckPresence() != None {
                    tileCost, moneyGainDefender, moneyLossAttacker, err = tile.OwningTribe.countDefense(tile, attacker, gs)
            } else {
                    tileCost, moneyGainDefender, moneyLossAttacker, err = tile.countDefense(gs)
            }
            
            if err != nil {
                    return true, err
            }

            // counts the cost for the attacker
            attackCostStacks, moneyGainAttacker, moneyLossDefender, pawnKill, err := attackingTribe.countAttack(tile, tileCost, string(attackingTribe.Race))
            if err != nil {
                    return true, err
            }
            attackCostStacks = AddPieceStacks(attackCostStacks, []PieceStack{{Type: "Flying Doormat", Amount: 1}})

            newStacks, hasDiceBeenUsed, ok, err := attackingTribe.calculateRemainingAttackingStacks(attackCostStacks, tile, gs)
            if err != nil {
                    return true, err
            }
            if !ok {
                    return true, gs.HandleStartRedeployment(attacker.Index)
            }

            // Enact changes
            tile.handleAfterConquest(gs, attackingTribe)
            attackingTribe.postConquest(tile, gs)
            power.Tile = tile
            tile.ModifierAfterConquest["Flying Doormat"] = TileModifierAfterConquests["Flying Doormat"]
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
                    return true, gs.HandleStartRedeployment(attacker.Index)
            } else {
                    gs.TurnInfo.Phase = Conquest
            }

            return true, nil
        }
        return power
    },
    "Diamond Fields": func() *Power {
        diamondsFields := &Power{
                Name: "Diamond Fields",
        }
        diamondsFields.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Diamond Fields Spawn")
            gs.Powers["Diamond Fields"] = diamondsFields
            diamondsFields.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Diamond Fields", Amount: 1})
            t.ModifierAfterConquest["Diamond Fields"] = TileModifierAfterConquests["Diamond Fields"]
            diamondsFields.Owner = tribe.Owner
        }
        diamondsFields.CountPoints = func(gs *GameState) int {
            total := 0
            for _, tile := range(gs.TileList) {
                if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, diamondsFields.Tile.OwningTribe.Race) && tile.Biome == diamondsFields.Tile.Biome {
                    total += 1
                }
            }
            return total
        }
        return diamondsFields
    },
    "Great Brass Pipe": func() *Power {
        power := &Power{
                Name: "Great Brass Pipe",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Great Brass Pipe Spawn")
            gs.Powers["Great Brass Pipe"] = power
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Great Brass Pipe", Amount: 1})
            t.ModifierAfterConquest["Great Brass Pipe"] = TileModifierAfterConquests["Great Brass Pipe"]
            if tribe != nil {
                power.Owner = tribe.Owner
                tribe.GiveTrait(Trait("Great Brass Pipe"))
            }
        }
        return power
    },
    "Fountain of Youth": func() *Power {
        power := &Power{
                Name: "Fountain of Youth",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Fountain of Youth Spawn")
            gs.Powers["Fountain of Youth"] = power
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Fountain of Youth", Amount: 1})
            t.ModifierAfterConquest["Fountain of Youth"] = TileModifierAfterConquests["Fountain of Youth"]
            if tribe != nil {
                power.Owner = tribe.Owner
            }
        }
        power.GetStacksForConquest = func(*GameState) {
            owner := power.Tile.OwningTribe.Owner
            if owner.ActiveTribe != nil {
                owner.PieceStacks = AddPieceStacks(owner.PieceStacks, []PieceStack{{Type: string(owner.ActiveTribe.Race), Amount: 1}})
            }
        }
        return power
    },
    "Keep on the Motherland": func() *Power {
        power := &Power{
                Name: "Keep on the Motherland",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Keep on the Motherland Spawn")
            gs.Powers["Keep on the Motherland"] = power
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Keep on the Motherland", Amount: 1})
            delete(t.ModifierAfterConquest, "Keep on the Motherland")
            t.ModifierPoints["Keep on the Motherland"] = TileModifierPoints["Keep on the Motherland"]
            t.ModifierDefenses["Keep on the Motherland"] = TileModifierDefenses["Keep on the Motherland"]
            if tribe != nil {
                power.Owner = tribe.Owner
            }
        }
        return power
    },
    "Mine of the Lost Dwarf": func() *Power {
        power := &Power{
                Name: "Mine of the Lost Dwarf",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Mine of the Lost Dwarf Spawn")
            gs.Powers["Mine of the Lost Dwarf"] = power
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Mine of the Lost Dwarf", Amount: 1})
            delete(t.ModifierAfterConquest, "Mine of the Lost Dwarf")
            t.ModifierPoints["Mine of the Lost Dwarf"] = TileModifierPoints["Mine of the Lost Dwarf"]
            if tribe != nil {
                power.Owner = tribe.Owner
            }
        }
        return power
    },
    "Stonehedge": func() *Power {
        power := &Power{
                Name: "Stonehedge",
                State: make(map[string]interface{}),
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Stonehedge Spawn")
            gs.Powers["Stonehedge"] = power
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Stonehedge", Amount: 1})
            delete(t.ModifierAfterConquest, "Stonehedge")
            rand.Seed(time.Now().UnixNano())
            index := rand.Intn(len(gs.TribeList) - 5) + 5
            trait := gs.TribeList[index].Trait
            if tribe != nil {
                power.Owner = tribe.Owner
                tribe.GiveTrait(trait)
            }
            power.State["trait"] = string(trait)
            gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("Stonehedge spawned with the power: %s!", trait)})
            gs.TribeList = append(gs.TribeList[:index], gs.TribeList[index+1:]...)
            t.ModifierAfterConquest["Stonehedge"] = TileModifierAfterConquests["Stonehedge"]
        }
        return power
    },
    "Altar of Souls": func() *Power {
        power := &Power{
                Name: "Altar of Souls",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Altar of Souls Spawn")
            gs.Powers["Altar of Souls"] = power
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Altar of Souls", Amount: 1})
            t.ModifierAfterConquest["Altar of Souls"] = TileModifierAfterConquests["Altar of Souls"]
            power.Owner = tribe.Owner
        }
        power.StartRedeployment = func(gs *GameState) []PieceStack {
            return []PieceStack{{Type: "Altar of Souls", Amount: 1}}
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for i, stack := range(power.Owner.PieceStacks) {
                if stack.Type == "Altar of Souls" {
                    power.Owner.PieceStacks = append(power.Owner.PieceStacks[:i], power.Owner.PieceStacks[i+1:]...)
                }
            }
        }
        power.HandleRedeploymentIn = func(tile *Tile, s string, gs *GameState) error {
            if power.Owner.ActiveTribe == nil {
                return fmt.Errorf("Player does not have an active tribe!")
            }
            if tile.OwningTribe == nil || tile.OwningTribe.IsActive {
                return fmt.Errorf("Player does not have a passive tribe on that tile!")
            }

            tile.handleAfterConquest(gs, nil)
            tile.OwningTribe.clearTile(tile, gs, 1)
            power.Owner.CoinPile += 3
            power.Owner.PieceStacks, _ = SubtractPieceStacks(power.Owner.PieceStacks, []PieceStack{{Type: "Altar of Souls", Amount: 1}})

            gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("A passive tribe was sacrificed!")})
            return nil
        }
        return power
    },
    "Crypt of the Tomb-raider": func() *Power {
        power := &Power{
                Name: "Crypt of the Tomb-raider",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Crypt of the Tomb-raider Spawn")
            gs.Powers["Crypt of the Tomb-raider"] = power
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Crypt of the Tomb-raider", Amount: 1})
            t.ModifierAfterConquest["Crypt of the Tomb-raider"] = TileModifierAfterConquests["Crypt of the Tomb-raider"]
            power.Owner = tribe.Owner
        }
        power.StartRedeployment = func(gs *GameState) []PieceStack {
            return []PieceStack{{Type: "Tomb-raider", Amount: 1}}
        }
        power.GetStacksForConquest = func(gs *GameState) {
            for _, tile := range(gs.TileList) {
                for i, stack := range(tile.PieceStacks) {
                    if stack.Type == "Tomb-raider" {
                        tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                        delete(tile.ModifierDefenses, "Tomb-raider")
                    }
                }
            }
        }
        power.HandleRedeploymentIn = func(tile *Tile, s string, gs *GameState) error {
            if tile.OwningTribe.Owner != power.Owner {
                return fmt.Errorf("This tile does not belong to you!")
            }
            for _, stack := range(tile.PieceStacks) {
                if stack.Type == "Crypt of the Tomb-raider" {
                    return fmt.Errorf("Cannot put the Tomb-raider in its own crypt!")
                }
            }
            tile.PieceStacks = append(tile.PieceStacks, PieceStack{Type: "Tomb-raider", Amount: 1})
            tile.ModifierDefenses["Tomb-raider"] = TileModifierDefenses["Tomb-raider"]
            power.Owner.PieceStacks, _ = SubtractPieceStacks(power.Owner.PieceStacks, []PieceStack{{Type: "Tomb-raider", Amount: 1}})
            return nil
        }
        return power
    },
    "Wickedest Pentacle": func() *Power {
        power := &Power{
                Name: "Wickedest Pentacle",
                State: make(map[string]interface{}),
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            delete(t.ModifierAfterConquest, "Wickedest Pentacle Spawn")
            gs.Powers["Wickedest Pentacle"] = power
            power.Tile = t
            t.PieceStacks = AddPieceStacks(t.PieceStacks, []PieceStack{{Type: "Wickedest Pentacle", Amount: 1}, {Type: "Balrog", Amount: 1}})
            power.State["hasMoved"] = false
            t.ModifierAfterConquest["Wickedest Pentacle"] = TileModifierAfterConquests["Wickedest Pentacle"]
            power.Owner = tribe.Owner
        }
        power.HandleMovement = func(s string, t1, t2 *Tile, gs *GameState) error {
            if power.State["hasMoved"].(bool) {
                return fmt.Errorf("The balrog has already moved!")
            }
            delete(t1.ModifierDefenses, "Balrog")
            t2.ModifierDefenses["Balrog"] = TileModifierDefenses["Balrog"]
            movingStack := []PieceStack{{Type: "Balrog", Amount: 1}}
            if t2.CheckPresence() != None {
                _, _, _, err := t2.OwningTribe.countDefense(t2, t2.OwningTribe.Owner, gs)
                if err != nil {
                    return err
                }
                t2.OwningTribe.handleReturn(t2, gs, 2) // Balrog destroys
            }
            t1.PieceStacks, _ = SubtractPieceStacks(t1.PieceStacks, movingStack)
            t2.PieceStacks = AddPieceStacks(t2.PieceStacks, movingStack)
            power.State["hasMoved"] = true
            return nil
        }
        power.GetStacksForConquest = func(gs *GameState) {
            power.State["hasMoved"] = false
        }

        return power
    },
}
