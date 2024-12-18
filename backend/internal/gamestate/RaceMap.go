package gamestate

type RaceValue struct {
    Transform func(Tribe) Tribe 
    Count     int
}

var RaceMap = map[Race]RaceValue {
	"Orc": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 5},
	"Gypsies": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 6},
	"Humans": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 5},
	"Elves": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 6},
	"Dwarves": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 3},
	"Giants": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 6},
}
