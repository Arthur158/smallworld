package gamestate

type TraitValue struct {
    Transform func(*Tribe)
    Count     int
}


var TraitMap = map[Trait]TraitValue {
	"Hill": {Transform: func(t *Tribe) {
		oldCountPoints := t.CountPoints
		t.CountPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && tile.Biome == Hill {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 4},
	"Merchant": {Transform: func(t *Tribe) {
		oldCountPoints := t.CountPoints
		t.CountPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 2},
	"Forest": {Transform: func(t *Tribe) {
		oldCountPoints := t.CountPoints
		t.CountPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && tile.Biome == Forest {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 4},
	"Goldsmith": {Transform: func(t *Tribe) {
		oldCountPoints := t.CountPoints
		t.CountPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			containsMine := false
			for _, attr := range tile.Attributes {
				if attr == Mine {
					containsMine = true
				}
			}
			if t.IsActive && containsMine {
				count += 2
			} else if t.IsActive {
				count -= 1
			}
			return max(0, count)
		}
		}, Count: 4},
	"Aquatic": {Transform: func(t *Tribe) {
		oldCountPoints := t.CountPoints
		t.CountPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			isNextToWater := false
			for _, neighbour := range tile.AdjacentTiles {
				if neighbour.Biome == Water {
					isNextToWater = true
				}
			}
			if t.IsActive && isNextToWater {
				count += 1
			} else if t.IsActive {
				count -= 1
			}
			return max(0, count)
		}
		}, Count: 4},
	"Swamp": {Transform: func(t *Tribe) {
		oldCountPoints := t.CountPoints
		t.CountPoints = func(tile *Tile) int {
			count := oldCountPoints(tile)
			if t.IsActive && tile.Biome == Swamp {
				count += 1
			} 
			return max(0, count)
		}
		}, Count: 4},
	"Fortified": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Alchemist": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Barricade": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Behemoth": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Berserk": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Bivouacking": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Catapult": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Commando": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Corrupt": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Dragon Master": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Fireball": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Flying": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Heroic": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Hordes of": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Imperial": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Mercenary": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Mounted": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Peace-loving": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Ransacking": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Seafaring": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Spirit": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Stout": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Underworld": {Transform: func(t *Tribe) {
		}, Count: 4},
	"Wealthy": {Transform: func(t *Tribe) {
		}, Count: 4},
}
