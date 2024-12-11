package gamestate

var TraitMap = map[Trait]func(Tribe) Tribe{
	"fortunate": func(t Tribe) Tribe {
		return t
	},
	"forest": func(t Tribe) Tribe {
		return t
	},
}
