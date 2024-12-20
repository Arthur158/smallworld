package gamestate

type RaceValue struct {
    Transform func(*Tribe)
    Count     int
}

var RaceMap = map[Race]RaceValue {
	"Orc": {Transform: func(t *Tribe) {
		}, Count: 5},
	"Gypsies": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Humans": {Transform: func(t *Tribe) {
		}, Count: 5},
	"Elves": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Dwarves": {Transform: func(t *Tribe) {
		}, Count: 3},
	"Giants": {Transform: func(t *Tribe) {
		}, Count: 6},
}
