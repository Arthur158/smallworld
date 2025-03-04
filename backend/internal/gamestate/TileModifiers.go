package gamestate;

import "fmt"

var TileModifierPoints = map[string]func() int {
    "Winter" : func() int {
            return - 1
    },
}

var TileModifierDefenses = map[string]func() (int, error) {
    "Lava" : func() (int, error) {
        return 0, fmt.Errorf("Cannot conquer zone with lava!")
    },
    "Burning Zeppelin" : func() (int, error) {
        return 0, fmt.Errorf("Cannot conquer zone with burning zeppelin!")
    },
}

var TileModifierSpecialDefenses = map[string]func(*Tile, *GameState, *Tribe, string) (bool, error) {
    "Loot" : func(tile *Tile, gs *GameState, tribe *Tribe, stackType string) (bool, error) {
        loot := tile.State["loot"].(int)
        if loot == -1 {
            gs.Messages = append(gs.Messages, "Skag attack!")

            for i := range(tribe.Owner.PieceStacks) {
                if tribe.Owner.PieceStacks[i].Type == string(tribe.Race) {
                    tribe.Owner.PieceStacks[i].Amount -= 1
                }
            }

            for i := range(tile.PieceStacks) {
                if tile.PieceStacks[i].Type == "Loot" {
                    tile.PieceStacks = append(tile.PieceStacks[:i], tile.PieceStacks[i+1:]...)
                }
            }
            delete(tile.ModifierSpecialDefenses, "Loot")

            return true, nil
        }
        return false, nil
    },
}
