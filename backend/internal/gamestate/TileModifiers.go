package gamestate;

import "fmt"
import "log"

var TileModifierPoints = map[string]func(*Tile) int {
    "Winter" : func(tile *Tile) int {
            return -1
    },
    "Keep on the Motherland" : func(tile *Tile) (int) {
        return 1
    },
    "Mine of the Lost Dwarf" : func(tile *Tile) (int) {
        return 2
    },
}

var TileModifierAfterConquests = map[string]func(*Tile, *Tribe, *GameState) {
    "Loot" : func(tile *Tile, t *Tribe, gs *GameState) {
        if t == nil {
            return
        }
        val := tile.State["loot"]

        if t.Race == "Skags" {
            return
        }

        var loot int
        switch v := val.(type) {
        case float64:
            loot = int(v)
        case int:
            loot = v
        }
        delete(tile.ModifierDefenses, "Loot")
        delete(tile.ModifierSpecialDefenses, "Loot")
        delete(tile.ModifierAfterConquest, "Loot")
        for i := range(tile.PieceStacks) {
            if tile.PieceStacks[i].Type == "Loot" {
                tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                break
            }
        }
        gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The loot was: %d", loot)})
        gs.Players[gs.TurnInfo.PlayerIndex].CoinPile += loot
    },
    "Diamond Fields" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Diamond Fields"]
        if t != nil {
            power.Owner = t.Owner
        } else {
            power.Owner = nil
        }
    },
    "Froggy's Ring" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Froggy's Ring"]
        if t != nil {
            power.Owner = t.Owner
            delete(tile.ModifierAfterConquest, "Froggy's Ring")
            power.Tile = nil
            tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Froggy's Ring", Amount: 1}})
            t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Froggy's Ring", Amount: 1}})
        } else {
            power.Owner = nil
        }
        
    },
    "Scepter of Avarice" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Scepter of Avarice"]
        if t != nil {
            power.Owner = t.Owner
            delete(tile.ModifierAfterConquest, "Scepter of Avarice")
            power.Tile = nil
            tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Scepter of Avarice", Amount: 1}})
            t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Scepter of Avarice", Amount: 1}})
        } else {
            power.Owner = nil
        }
        
    },
    "Flying Doormat" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Flying Doormat"]
        if t != nil {
            power.Owner = t.Owner
            delete(tile.ModifierAfterConquest, "Flying Doormat")
            power.Tile = nil
            tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Flying Doormat", Amount: 1}})
            t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Flying Doormat", Amount: 1}})
        } else {
            power.Owner = nil
        }
    },
    "Stinky Troll's Socks" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Stinky Troll's Socks"]
        if t != nil {
            power.Owner = t.Owner
            delete(tile.ModifierAfterConquest, "Stinky Troll's Socks")
            power.Tile = nil
            tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Stinky Troll's Socks", Amount: 1}})
            t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Stinky Troll's Socks", Amount: 1}})
        } else {
            power.Owner = nil
        }
    },
    "Sword of the Killer Rabbit" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Sword of the Killer Rabbit"]
        if t != nil {
            power.Owner = t.Owner
            delete(tile.ModifierAfterConquest, "Sword of the Killer Rabbit")
            power.Tile = nil
            tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Sword of the Killer Rabbit", Amount: 1}})
            t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Sword of the Killer Rabbit", Amount: 1}})
        } else {
            power.Owner = nil
        }
    },
    "Shiny Orb" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Shiny Orb"]
        if t != nil {
            power.Owner = t.Owner
            delete(tile.ModifierAfterConquest, "Shiny Orb")
            power.Tile = nil
            tile.PieceStacks, _ = SubtractPieceStacks(tile.PieceStacks, []PieceStack{{Type: "Shiny Orb", Amount: 1}})
            t.Owner.PieceStacks = AddPieceStacks(t.Owner.PieceStacks, []PieceStack{{Type: "Shiny Orb", Amount: 1}})
        } else {
            power.Owner = nil
        }
    },
    "Wickedest Pentacle" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Wickedest Pentacle"]
        if t != nil {
            power.Owner = t.Owner
        } else {
            power.Owner = nil
        }
    },
    "Crypt of the Tomb-raider" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Crypt of the Tomb-raider"]
        if t != nil {
            power.Owner = t.Owner
        } else {
            power.Owner = nil
        }
        for _, tile := range(gs.TileList) {
            for i, stack := range(tile.PieceStacks) {
                if stack.Type == "Tomb-raider" {
                    tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                    delete(tile.ModifierDefenses, "Tomb-raider")
                }
            }
        }
        if tile.OwningTribe != nil {
            for i, stack := range(tile.OwningTribe.Owner.PieceStacks) {
                if stack.Type == "Tomb-raider" {
                    tile.OwningTribe.Owner.PieceStacks = append(tile.OwningTribe.Owner.PieceStacks[:i], tile.OwningTribe.Owner.PieceStacks[i+1:]...)
                    break
                }
            }
        }
    },
    "Altar of Souls" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Altar of Souls"]
        if t != nil {
            power.Owner = t.Owner
        } else {
            power.Owner = nil
        }
        if tile.OwningTribe != nil {
            for i, stack := range(tile.OwningTribe.Owner.PieceStacks) {
                if stack.Type == "Altar of Souls" {
                    tile.OwningTribe.Owner.PieceStacks = append(tile.OwningTribe.Owner.PieceStacks[:i], tile.OwningTribe.Owner.PieceStacks[i+1:]...)
                    break
                }
            }
        }
    },
    "Great Brass Pipe" : func(tile *Tile, t *Tribe,  gs *GameState) {
        log.Println(gs.Powers)
        power := gs.Powers["Great Brass Pipe"]
        if t != nil {
            power.Owner = t.Owner
        } else {
            power.Owner = nil
        }
        if tile.CheckPresence() != None {
            delete(tile.OwningTribe.checkAdjacencyMap, "Great Brass Pipe")
        }
        if t != nil {
            t.checkAdjacencyMap["Great Brass Pipe"] = func(tile *Tile, gs *GameState, err error) error {
                if err == nil {
                    return nil
                }
                if tile.Biome == power.Tile.Biome {
                    return nil
                }
                return err
            }
        }
    },
    "Fountain of Youth" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Fountain of Youth"]
        if t != nil {
            power.Owner = t.Owner
        } else {
            power.Owner = nil
        }
    },
    "Stonehedge" : func(tile *Tile, t *Tribe,  gs *GameState) {
        power := gs.Powers["Stonehedge"]
        trait := power.State["trait"].(string)
        if t != nil {
            power.Owner = t.Owner
            t.GiveTrait(Trait(trait))
        } else {
            power.Owner = nil
        }
        if tile.OwningTribe != nil {
            tile.OwningTribe.DeletePower(trait, gs)
        }
    },
}

var TileModifierDefenses = map[string]func(*Tile, *GameState) (int, int, int, error) {
    "Lava" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Cannot conquer zone with lava!")
    },
    "Balrog" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Cannot conquer zone with the balrog!")
    },
    "Tomb-raider" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Cannot conquer zone with tomb-raider!")
    },
    "Skag Attack" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Skags are attacking here!")
    },
    "Burning Zeppelin" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Cannot conquer zone with burning zeppelin!")
    },
    "Keep on the Motherland" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 1, 0, 0, nil
    },
    "Winter" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 1, 0, 0, nil
    },
}

var TileModifierSpecialDefenses = map[string]func(*Tile, *GameState, *Tribe, string) (bool, error) {
    "Loot" : func(tile *Tile, gs *GameState, tribe *Tribe, stackType string) (bool, error) {

        if tribe.Race == "Skags" {
            return false, nil
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

            for i := range(tribe.Owner.PieceStacks) {
                if tribe.Owner.PieceStacks[i].Type == string(tribe.Race) {
                    tribe.Owner.PieceStacks[i].Amount -= 1
                }
            }

            delete(tile.ModifierSpecialDefenses, "Loot")
            delete(tile.ModifierDefenses, "Loot")
            tile.ModifierDefenses["Skag Attack"] = TileModifierDefenses["Skag Attack"]
            for i := range(tile.PieceStacks) {
                if tile.PieceStacks[i].Type == "Loot" {
                    tile.PieceStacks[i].Type = "Skag Attack"
                }
            }

            gs.Messages = append(gs.Messages, Message{Content: "Skag attack!"})
            gs.ModifierTurnsAfter = append(gs.ModifierTurnsAfter, TurninfoEntry{
                    player: gs.TurnInfo.PlayerIndex,
                    TurnInfo: nil,
                    actionBefore: func(gs *GameState) {
                    delete(tile.ModifierDefenses, "Skag Attack")
                    for i := range(tile.PieceStacks) {
                        if tile.PieceStacks[i].Type == "Skag Attack" {
                            tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                            break
                        }
                    }
                    },
            })

            return true, nil
        }
        return false, nil
    },
}
