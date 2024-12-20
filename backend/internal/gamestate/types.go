package gamestate

type Race string;
type Trait string;

type Tribe struct {
	Race Race;
	Trait Trait;
	IsStackValid func(string) bool;
	countDefense func(*Tile) (int, error);
	countAttack func(*Tile, int, string) []PieceStack;
	countReturningStacks func(*Tile) []PieceStack;
	countNewTileStacks func([]PieceStack) []PieceStack;
	CanTileBeAbandoned func(*Tile) bool;
	ReceiveAbandonment func(*Tile) []PieceStack;
	startRedeployment func() []PieceStack;
	getStacksOutRedeployment func(*Tile, string) ([]PieceStack, error);
	checkZoneAccess func(*Tile) error;
	checkAdjacency func(*Tile, *GameState) error;
	GetStacksForConquest func(*Tile) []PieceStack;
	CountPoints func(*Tile) int;
	prepareDecline func(*GameState);
	prepareRemoval func(*GameState) bool;
	CanGoIntoDecline func(*GameState) bool
}

type TribeEntry struct {
	Tribe *Tribe;
	CoinPile int;
	PiecePile int;
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
}

type PieceStack struct {
	Type string;
	Amount int;
}

type Player struct {
	// Name string;
	ActiveTribe *Tribe;
	PassiveTribes []*Tribe;
	CoinPile int
	PieceStacks []PieceStack
	HasActiveTribe bool
}

type Phase int

const (
	TribeChoice Phase = iota
	DeclineChoice
	TileAbandonment
	Conquest
	Redeployment
	GameFinished
)

func (b Phase) String() string {
	switch b {
	case TribeChoice:
		return "TribeChoice"
	case DeclineChoice:
		return "DeclineChoice"
	case TileAbandonment:
		return "TileAbandonment"
	case Conquest:
		return "Conquest"
	case Redeployment:
		return "Redeployment"
	case GameFinished:
		return "GameFinished"
	default:
		return "Unknown"
	}
}


type TurnInfo struct {
	TurnIndex int;
	PlayerIndex int;
	Phase Phase;
	ConqueredPassive int;
	ConqueredActive int;
	Conquered int;
}

