package gamestate

type Power struct {
    Name string
    Tile *Tile
    Spawn func(*Tile, *Tribe, *GameState)
    CountPoints func(*GameState) int;
    GetStacksForConquest func()
}

var PowerMap = map[string]func()*Power {
    "Diamond Fields": func() *Power {
        diamondsFields := &Power{
                Name: "Diamond Fields",
        }
        diamondsFields.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            diamondsFields.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Diamond Fields", Amount: 1})
            t.ModifierAfterConquest["Diamond Fields"] = TileModifierAfterConquests["Diamond Fields"]
            tribe.Owner.Powers = append(tribe.Owner.Powers, diamondsFields)
        }
        diamondsFields.CountPoints = func(gs *GameState) int {
            total := 0
            for _, tile := range(gs.TileList) {
                if tile.CheckPresence() != None && tile.OwningTribe.checkPresence(tile, diamondsFields.Tile.OwningTribe.Race) {
                    total += 1
                }
            }
            return total
        }
        diamondsFields.GetStacksForConquest = func() {}
        return diamondsFields
    },
    "Great Brass Pipe": func() *Power {
        power := &Power{
                Name: "Great Brass Pipe",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Great Brass Pipe", Amount: 1})
            t.ModifierAfterConquest["Great Brass Pipe"] = TileModifierAfterConquests["Great Brass Pipe"]
            tribe.Owner.Powers = append(tribe.Owner.Powers, power)
            tribe.checkAdjacencyMap["Great Brass Pipe"] = func(t *Tile, gs *GameState, err error) error {
                if err == nil {
                    return nil
                }
                if t.Biome == power.Tile.Biome {
                    return nil
                }
                return err
            }
        }
        power.CountPoints = func(gs *GameState) int {
            return 0
        }
        power.GetStacksForConquest = func() {}
        return power
    },
    "Fountain of Youth": func() *Power {
        power := &Power{
                Name: "Fountain of Youth",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Fountain of Youth", Amount: 1})
            t.ModifierAfterConquest["Fountain of Youth"] = TileModifierAfterConquests["Fountain of Youth"]
            tribe.Owner.Powers = append(tribe.Owner.Powers, power)
        }
        power.CountPoints = func(gs *GameState) int {
            return 0
        }
        power.GetStacksForConquest = func() {
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
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Keep on the Motherland", Amount: 1})
            delete(t.ModifierAfterConquest, "Keep on the Motherland")
            t.ModifierPoints["Keep on the Motherland"] = TileModifierPoints["Keep on the Motherland"]
            t.ModifierDefenses["Keep on the Motherland"] = TileModifierDefenses["Keep on the Motherland"]
            tribe.Owner.Powers = append(tribe.Owner.Powers, power)
        }
        power.CountPoints = func(gs *GameState) int {
            return 0
        }
        power.GetStacksForConquest = func() {}
        return power
    },
    "Mine of the Lost Dwarf": func() *Power {
        power := &Power{
                Name: "Mine of the Lost Dwarf",
        }
        power.Spawn = func(t *Tile, tribe *Tribe, gs *GameState) {
            power.Tile = t
            t.PieceStacks = append(t.PieceStacks, PieceStack{Type: "Mine of the Lost Dwarf", Amount: 1})
            delete(t.ModifierAfterConquest, "Mine of the Lost Dwarf")
            t.ModifierPoints["Mine of the Lost Dwarf"] = TileModifierPoints["Mine of the Lost Dwarf"]
            tribe.Owner.Powers = append(tribe.Owner.Powers, power)
        }
        power.CountPoints = func(gs *GameState) int {
            return 0
        }
        power.GetStacksForConquest = func() {}
        return power
    },
}
