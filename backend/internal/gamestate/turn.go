package gamestate;

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

type TurninfoEntry struct {
	TurnInfo *TurnInfo
	player int
	actionBefore func (*GameState);
	// actionAfter func (*GameState);
}
