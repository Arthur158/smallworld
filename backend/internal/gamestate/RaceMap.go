package gamestate

type RaceValue struct {
    Transform func(*Tribe)
    Count     int
}

var RaceMap = map[Race]RaceValue {
	"Wizard": {Transform: func(t *Tribe) {
		}, Count: 5},
	"White Lady": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Wendigo": {Transform: func(t *Tribe) {
		}, Count: 5},
	"Troll": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Triton": {Transform: func(t *Tribe) {
		}, Count: 3},
	"Sorcerer": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Skeletton": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Shrubman": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Scavenger": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Scarecrow": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Ratman": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Pygmy": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Priestess": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Pixy": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Orc": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Leprechaun": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Kobold": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Khan": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Human": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Halfling": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Gypsy": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Goblin": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Giant": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Faun": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Elf": {Transform: func(t *Tribe) {
		}, Count: 6},
}
