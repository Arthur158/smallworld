package gamestate

type TribeEntry struct {
    Race      Race
    Trait     Trait
    CoinPile  int
    PiecePile int
}

type PieceStack struct {
    Type   string
    Amount int
    Tribe  *Tribe
}

type Player struct {
    Name           string
    Index          int
    ActiveTribe    *Tribe
    PassiveTribes  []*Tribe
    CoinPile       int
    PieceStacks    []PieceStack
    PointsEachTurn []int
}

type Message struct {
    Receivers []int
    Content   string
}
