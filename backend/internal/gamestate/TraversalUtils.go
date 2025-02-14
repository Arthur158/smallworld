package gamestate;

import (
    "fmt"
    "math/rand"
    "time"
)


func (gs *GameState) IsTribePresentOnTheBoard(race Race) bool {
    for _, tile := range gs.TileList {
        if tile.Presence != None && tile.OwningTribe.checkPresence(tile, race) {
            return true
        }
    }
    return false
}

func (gs *GameState) GetPieceStackForConquest(player *Player) {
    player.ActiveTribe.getStacksForConquestTurn(player, gs)
    // }
    for _, tribe := range(player.PassiveTribes) {
        tribe.getStacksForConquestTurn(player, gs)
    }
    for _, tile := range gs.TileList {
        if tile.Presence != None {
            if tile.OwningTribe.checkPresence(tile, player.ActiveTribe.Race) {
                player.ActiveTribe.getStacksForConquest(tile, player)
            }
            for _, tribe := range(player.PassiveTribes) {
                if tile.OwningTribe.checkPresence(tile, tribe.Race) {
                    tribe.getStacksForConquest(tile, player)
                }
            }
        }
    }
}

func (gs *GameState) CheckJump(tile *Tile, otherTile *Tile) bool {
    for _, neighbor := range(tile.AdjacentTiles) {
        for _, neighbor2 := range(neighbor.AdjacentTiles) {
            adjacent := false
            for _, neighbor1bis := range(tile.AdjacentTiles) {
                adjacent = adjacent || neighbor1bis == neighbor2
            }
            if neighbor2 != tile && !adjacent && neighbor2 == otherTile {
                return true
            }
        }
    }
    return false
}
// pickTwoRandom selects two distinct strings from a given slice.
func pickTwoRandom(strings []string) (string, string, error) {
	if len(strings) < 2 {
		return "", "", fmt.Errorf("not enough elements to pick two")
	}

	rand.Seed(time.Now().UnixNano())

	// Pick first random index
	i := rand.Intn(len(strings))

	// Pick second random index, ensuring it's different from the first
	j := rand.Intn(len(strings) - 1)
	if j >= i {
		j++
	}

	return strings[i], strings[j], nil
}

func (gs *GameState) countPoints(player *Player) int {
	total := 0
	for _, tile := range gs.TileList {
		if tile.Presence != None {
			if player.HasActiveTribe && tile.OwningTribe.checkPresence(tile, player.ActiveTribe.Race) {
				total += player.ActiveTribe.countPoints(tile)
			}
			for _, tribe := range(player.PassiveTribes) {
			    if tile.OwningTribe.checkPresence(tile, tribe.Race) {
				total += tribe.countPoints(tile)
			    }
			}
		}
	}
	if player.HasActiveTribe {
		total += player.ActiveTribe.countExtrapoints(gs)
	}
	for _, passiveTribe := range player.PassiveTribes {
		total += passiveTribe.countExtrapoints(gs)
	}

        for _, modifier := range(gs.ModifierPoints) {
            total = modifier(total, player)
        }
    

	return total
}
