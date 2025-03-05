package gamestate;

import "fmt"

var TileModifierPoints = map[string]func(*Tile) int {
    "Winter" : func(tile *Tile) int {
            return -1
    },
}

var TileModifierAfterConquests = map[string]func(*Tile, *GameState) {
    "Loot" : func(tile *Tile, gs *GameState) {
        val := tile.State["loot"]

        if gs.Players[gs.TurnInfo.PlayerIndex].HasActiveTribe && gs.Players[gs.TurnInfo.PlayerIndex].ActiveTribe.Race == "Skags" {
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
        for i := range(tile.PieceStacks) {
            if tile.PieceStacks[i].Type == "Loot" {
                tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                break
            }
        }
        gs.Messages = append(gs.Messages, Message{Content: fmt.Sprintf("The loot was: %d", loot)})
        gs.Players[gs.TurnInfo.PlayerIndex].CoinPile += loot
    },
}

var TileModifierDefenses = map[string]func(*Tile, *GameState) (int, int, int, error) {
    "Lava" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Cannot conquer zone with lava!")
    },
    "Skag Attack" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Skags are attacking here!")
    },
    "Burning Zeppelin" : func(tile *Tile, gs *GameState) (int, int, int, error) {
        return 0, 0, 0, fmt.Errorf("Cannot conquer zone with burning zeppelin!")
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
