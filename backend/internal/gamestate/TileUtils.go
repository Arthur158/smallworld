package gamestate;
// import "log"

func CountDefense(tile *Tile) int {
    price := 2
    if tile.Biome == Mountain {
        price += 1
    }
    return price
}

func (gs *GameState) IsTribePresentOnTheBoard(race Race) bool {
    for _, tile := range gs.TileList {
        // log.Println(race)
        // log.Println(tile)
        // log.Println(tile.OwningTribe)
        if tile.Presence != None && tile.OwningTribe.checkPresence(tile, race) {
            return true
        }
    }
    return false
}

func (gs *GameState) GetPieceStackForConquest(player *Player) {
    // if player.ActiveTribe != nil {
    player.ActiveTribe.getStacksForConquestTurn(player)
    // }
    for _, tribe := range(player.PassiveTribes) {
        tribe.getStacksForConquestTurn(player)
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

