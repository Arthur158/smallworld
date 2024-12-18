package gamestate

type TraitValue struct {
    Transform func(Tribe) Tribe 
    Count     int
}


var TraitMap = map[Trait]TraitValue {
	"fortunate": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 4},
	"hill": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 4},
	"field": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 4},
	"cave": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 4},
	"fortress": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 4},
	"shoed": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 4},
	"squiggly": {Transform: func(t Tribe) Tribe {
			return t
		}, Count: 4},
}
