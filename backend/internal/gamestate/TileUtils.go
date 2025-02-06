package gamestate;

func CountDefense(tile *Tile) int {
    price := 2
    if tile.Biome == Mountain {
        price += 1
    }
    return price
}

func (gs *GameState) IsTribePresentOnTheBoard(race Race) bool {
    for _, tile := range gs.TileList {
        if tile.Presence != None && tile.OwningTribe.Race == race {
            return true
        }
    }
    return false
}

func (gs *GameState) GetPieceStackForConquest(player *Player) {
    if player.ActiveTribe != nil {
        player.ActiveTribe.getStacksForConquestTurn(player)
    }
    for _, tribe := range(player.PassiveTribes) {
        tribe.getStacksForConquestTurn(player)
    }
    for _, tile := range gs.TileList {
        if tile.OwningPlayer == player {
            tile.OwningTribe.getStacksForConquest(tile, player)
        }
    }
}

