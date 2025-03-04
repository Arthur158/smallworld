package gamestate

type Tile struct {
	Id string;
	AdjacentTiles []*Tile;
	PieceStacks []PieceStack;
	OwningTribe *Tribe;
	Biome Biome;
	Attributes []Attribute;
	Presence Presence;
	IsEdge bool;

	State map[string]interface{};
	ModifierPoints map[string]func() int;
	ModifierDefenses map[string]func() (int, error);
	ModifierSpecialDefenses map[string]func(*Tile, *GameState, *Tribe, string) (bool, error);
}

func (tile *Tile) countPoints() int {
    value := 1
    for _, modifier := range(tile.ModifierPoints) {
	value += modifier()
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
	extra, err := modifier()
	if err != nil {
	    return price, err
	}
	price += extra
    }
    return price, err
}

func (tile *Tile) specialDefense(gs *GameState, attackingTribe *Tribe, attackingStackType string) (bool, error) {
	for _, modifier := range(tile.ModifierSpecialDefenses) {
		ok, err := modifier(tile, gs, attackingTribe, attackingStackType)
		if ok {
			return ok, err
		}
	}
	return false, nil
}

type Biome int

// Enum values for Biome
const (
	Forest Biome = iota
	Hill
	Field
	Swamp
	Water
	Mountain
)

func (b Biome) String() string {
	switch b {
	case Forest:
		return "Forest"
	case Hill:
		return "Hill"
	case Field:
		return "Field"
	case Swamp:
		return "Swamp"
	case Water:
		return "Water"
	case Mountain:
		return "Mountain"
	default:
		return "Unknown"
	}
}

type Attribute int;

const (
	Magic Attribute = iota
	Mine
	Cave
)

func (b Attribute) String() string {
	switch b {
	case Magic:
		return "Magic"
	case Mine:
		return "Mine"
	case Cave:
		return "Cave"
	default:
		return "Unknown"
	}
}

type Presence int;

const (
	None Presence = iota
	Active
	Passive
)

func (b Presence) String() string {
	switch b {
	case None:
		return "None"
	case Active:
		return "Active"
	case Passive:
		return "Passive"
	default:
		return "Unknown"
	}
}

