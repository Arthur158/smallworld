package gamestate

type Race string;
type Trait string;

type Tribe struct {
	Race      Race                  `json:"race"`
	Trait     Trait                 `json:"trait"`
	IsActive  bool                  `json:"is_active"`
	State     map[string]interface{} `json:"state"`

	// All function fields get json:"-" to exclude them
	giveInitialStacks                 func() []PieceStack          `json:"-"`
	canTileBeAbandoned                func(*Tile) bool             `json:"-"`
	receiveAbandonment                func(*Tile) []PieceStack     `json:"-"`
	getStacksForConquest              func(*Tile, *Player)         `json:"-"`
	IsStackValid                      func(string) bool            `json:"-"`
	checkZoneAccess                   func(*Tile) error            `json:"-"`
	checkAdjacency                    func(*Tile, *GameState) error `json:"-"`
	countAttack                       func(*Tile, int, string) ([]PieceStack, int, int) `json:"-"`
	countNewTileStacks                func([]PieceStack, *Tile) []PieceStack `json:"-"`
	calculateRemainingAttackingStacks func([]PieceStack, []PieceStack) ([]PieceStack, []PieceStack, bool, error) `json:"-"`
	countDefense                      func(*Tile) (int, int, int, error) `json:"-"`
	countReturningStacks              func(*Tile) ([]PieceStack, []PieceStack) `json:"-"`
	startRedeployment                 func(*GameState) []PieceStack `json:"-"`
	getStacksOutRedeployment          func(*Tile, string) ([]PieceStack, error) `json:"-"`
	canBeRedeployedIn                 func(*Tile, string) bool     `json:"-"`
	countPoints                       func(*Tile) int              `json:"-"`
	countExtrapoints                  func() int                   `json:"-"`
	countPiecesRemaining              func(*Tile) []PieceStack     `json:"-"`
	countRemainingAttackingStacks     func(*Player) []PieceStack   `json:"-"`
	canGoIntoDecline                  func(*GameState) bool        `json:"-"`
	goIntoDecline                     func(*GameState)            `json:"-"`
	prepareRemoval                    func(*GameState) bool        `json:"-"`
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

