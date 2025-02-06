package gamestate

type Race string;
type Trait string;

type Tribe struct {
	Race Race;
	Trait Trait;
	IsActive bool;
	State map[string]interface{};
	Minimum int;

	giveInitialStacks func() []PieceStack;

	//abandonment
	canTileBeAbandoned func(*Tile) bool;
	receiveAbandonment func(*Tile) []PieceStack;

	// receive for conquest
	getStacksForConquestTurn func(*Player);
	getStacksForConquest func(*Tile, *Player);

	// conquest checks
	IsStackValid func(string) bool;
	checkZoneAccess func(*Tile) error;
	checkAdjacency func(*Tile, *GameState) error;

	// conquest for attacker
	countAttack func(*Tile, int, string) ([]PieceStack, int, int);
	countNewTileStacks func([]PieceStack, *Tile) []PieceStack;
	calculateRemainingAttackingStacks func([]PieceStack, []PieceStack, *GameState) ([]PieceStack, []PieceStack, bool, string)
	specialConquest func(*GameState, *Tile, string, *Player) (bool, error);

	//conquest for defender
	countDefense func(*Tile) (int, int, int, error);
	countReturningStacks func(*Tile, *GameState) ([]PieceStack, []PieceStack);

	// redeployment
	startRedeployment func(*GameState) []PieceStack;
	getStacksOutRedeployment func(*Tile, string) ([]PieceStack, error);
	canBeRedeployedIn func(*Tile, string) bool;

	// end of turn
	countPoints func(*Tile) int;
	countExtrapoints func(*GameState) int;

	// decline
	countPiecesRemaining func(*Tile) []PieceStack;
	countRemainingAttackingStacks func(*Player) []PieceStack;
	canGoIntoDecline func(*GameState) bool;
	goIntoDecline func(*GameState);
	prepareRemoval func(*GameState) bool;
}

type TribeEntry struct {
	Race Race;
	Trait Trait;
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
	PointsEachTurn []int;
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
}

