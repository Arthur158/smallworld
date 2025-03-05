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
	ModifierPoints map[string]func(*Tile) int;
	ModifierDefenses map[string]func(*Tile, *GameState) (int, int, int, error);
	ModifierAfterConquest map[string]func(*Tile, *GameState);
	ModifierSpecialDefenses map[string]func(*Tile, *GameState, *Tribe, string) (bool, error);
}

func (tile *Tile) countPoints() int {
    value := 1
    for _, modifier := range(tile.ModifierPoints) {
	value += modifier(tile)
    }
    return value
}

func (tile *Tile) countDefense(gs *GameState) (int, int, int, error) {
    a := 2
    b := 0
    c := 0
    err := error(nil)
    if tile.Biome == Mountain {
        a += 1
    }
    for _, modifier := range(tile.ModifierDefenses) {
	x, y, z, err := modifier(tile, gs)
	if err != nil {
	    return a, b, c, err
	}
	a += x
	b += y
	c += z
    }
    return a, b, c, err
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

func (tile *Tile) handleAfterConquest(gs *GameState) {
	for _, modifier := range(tile.ModifierAfterConquest) {
		modifier(tile, gs)
	}
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

