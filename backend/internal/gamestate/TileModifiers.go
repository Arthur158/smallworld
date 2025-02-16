package gamestate;

import "fmt"

var TileModifierPoints = map[string]func(int) int {
    "Winter" : func(i int) int {
            return i - 1
    },
}

var TileModifierDefenses = map[string]func(int, error) (int, error) {
    "Lava" : func(i int, err error) (int, error) {
        return i, fmt.Errorf("Cannot conquer zone with lava")
    },
}
