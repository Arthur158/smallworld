package gamestate

// RaceFunctions is a map of race names to functions that modify a Tribe.
var RaceMap = map[Race]func(Tribe) Tribe{
	"Orc": func(t Tribe) Tribe {
		return t
	},
	"Elf": func(t Tribe) Tribe {
		return t
	},
}
