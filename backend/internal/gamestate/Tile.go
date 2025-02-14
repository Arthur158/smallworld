package gamestate


type Tile struct {
	Id string;
	AdjacentTiles []*Tile;
	PieceStacks []PieceStack;
	OwningPlayer *Player;
	OwningTribe *Tribe;
	Biome Biome;
	Attributes []Attribute;
	Presence Presence;
	IsEdge bool;
	ModifierPoints map[string]func(int) int;
	ModifierDefenses map[string]func(int, error) (int, error);
}

func (tile *Tile) countPoints() int {
    value := 1
    for _, modifier := range(tile.ModifierPoints) {
	value = modifier(value)
    }
    return value
}

func (tile *Tile) countDefense() (int, error) {
    price := 2
    err := error(nil)
    if tile.Biome == Mountain {
        price += 1
    }
    if err != nil {
	return price, err
    }
    for _, modifier := range(tile.ModifierDefenses) {
	price, err = modifier(price, err)
	if err != nil {
	    return price, err
	}
    }
    return price, err
}
