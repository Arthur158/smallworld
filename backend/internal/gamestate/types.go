package gamestate

type Race string;
type Trait string;

type Tribe struct {
	Owner *Player
	Race Race;
	Trait Trait;
	IsActive bool;
	State map[string]interface{};
	Minimum int;

	checkPresence func(*Tile, Race) bool;

	giveInitialStacks func() []PieceStack;

	//abandonment
	canTileBeAbandoned func(*Tile) bool;
	handleAbandonment func(*Tile, *GameState);

	// receive for conquest
	getStacksForConquestTurn func(*Player, *GameState);
	getStacksForConquest func(*Tile, *Player);

	// conquest checks
	IsStackValid func(string) bool;
	checkZoneAccess func(*Tile) error;
	checkAdjacency func(*Tile, *GameState) error;

	// conquest for attacker
	countAttack func(*Tile, int, string) ([]PieceStack, int, int, int);
	computeDiscount func(string, *Tile) int;
	countNewTileStacks func([]PieceStack, *Tile, *GameState) []PieceStack;
	calculateRemainingAttackingStacks func([]PieceStack, *Tile, *GameState) ([]PieceStack, bool, bool, error)
	specialConquest func(*GameState, *Tile, string) (bool, error);
	handleMovement func(string, *Tile, *Tile, *GameState) error

	//conquest for defender
	countDefense func(*Tile, *Player, *GameState) (int, int, int, error);
	handleReturn func(*Tile, *GameState, int)
	clearTile func(*Tile, *GameState, int);
	specialDefense func(*GameState, *Tile, *Tribe, string) (bool, error);

	// redeployment
	startRedeployment func(*GameState) []PieceStack;
	getStacksOutRedeployment func(*Tile, string) ([]PieceStack, error);
	canBeRedeployedIn func(*Tile, string, *GameState) bool;
	getRedeploymentStack func(string, []PieceStack) []PieceStack
	handleDeploymentOut func(*Tile, string, int, *GameState) error;
	handleDeploymentIn func(*Tile, string, int, *GameState) error;

	// misc
	handleOpponentAction func(string, *Player, *GameState) error;

	// end of turn
	canEndTurn func(*GameState) error;
	countPoints func(*Tile) int;
	countExtrapoints func(*GameState) int;

	// decline
	countRemovablePieces func(*Tile) []PieceStack;
	countRemovableAttackingStacks func(*Player) []PieceStack;
	canGoIntoDecline func(*GameState) bool;
	goIntoDecline func(*GameState);
	prepareRemoval func(*GameState) bool;

	handleEndOfGame func(*GameState);
}

type TribeEntry struct {
	Race Race;
	Trait Trait;
	CoinPile int;
	PiecePile int;
}


type PieceStack struct {
	Type string;
	Amount int;
	Tribe *Tribe
}

type Player struct {
	Name string
	Index int;
	ActiveTribe *Tribe;
	PassiveTribes []*Tribe;
	CoinPile int
	PieceStacks []PieceStack
	HasActiveTribe bool
	PointsEachTurn []int;
}

type Message struct {
	Receivers []int;
	Content string;
}
