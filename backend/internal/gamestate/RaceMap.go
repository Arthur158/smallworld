package gamestate

type RaceValue struct {
    Transform func(*Tribe)
    Count     int
}

var RaceMap = map[Race]RaceValue {
	"Troll": {Transform: func(t *Tribe) {
		// make a newly conquered region contain a lair
		oldCountNewTileStacks := t.countNewTileStacks
		t.countNewTileStacks = func(stacks []PieceStack, tile *Tile) []PieceStack {
			oldstacks := oldCountNewTileStacks(stacks, tile)
			return AddPieceStacks(oldstacks, []PieceStack{{Type: "Lair", Amount: 1}})
		}

		// make the defense of the tile + 1
		oldCountDefense := t.countDefense
		t.countDefense = func(tile *Tile) (int, error) {
			old, err := oldCountDefense(tile)
			if err != nil {
				return old, err
			}
			return old+1, nil
		}

		oldCountPiecesRemaining := t.countPiecesRemaining
		t.countPiecesRemaining = func(tile *Tile) []PieceStack {
			oldstacks := oldCountPiecesRemaining(tile)
			return AddPieceStacks(oldstacks, []PieceStack{{Type: "Lair", Amount: 1}})
		}
		}, Count: 5},
	"Wizard": {Transform: func(t *Tribe) {
		oldCountPoints := t.CountPoints
		t.CountPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			println("were here")
			for _, attr := range tile.Attributes {
				if attr == Magic {
					count += 1
				}
			}
			println(count)
			return count
		}
		}, Count: 5},
	"White Lady": {Transform: func(t *Tribe) {
		}, Count: 6},
	"Wendigo": {Transform: func(t *Tribe) {
		}, Count: 5},
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
