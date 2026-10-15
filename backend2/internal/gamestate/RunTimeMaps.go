package gamestate

import "sync"

type RuntimeTileSpec struct {
	IsSkyIsland bool
	ID          string
	Biome       Biome
	Attributes  []Attribute
	IsEdge      bool
	AdjacentIDs []string
	// Optional game setup, supplied by the backend rather than map geometry.
	// The zero value keeps existing runtime maps and training inputs unchanged.
	LostTribe bool
}

var runtimeMapMu sync.RWMutex
var runtimeMapRegistry = map[string][]RuntimeTileSpec{}

func RegisterRuntimeMap(name string, specs []RuntimeTileSpec) {
	runtimeMapMu.Lock()
	defer runtimeMapMu.Unlock()

	copied := make([]RuntimeTileSpec, len(specs))
	copy(copied, specs)

	runtimeMapRegistry[name] = copied
}

func BuildRuntimeMap(name string, gs *GameState) (map[string]*Tile, bool) {
	runtimeMapMu.RLock()
	specs, ok := runtimeMapRegistry[name]
	runtimeMapMu.RUnlock()

	if !ok {
		return nil, false
	}

	tileMap := make(map[string]*Tile, len(specs))
	var lostTribe *Tribe

	for _, spec := range specs {
		tileMap[spec.ID] = &Tile{
			Id:          spec.ID,
			IsSkyIsland: spec.IsSkyIsland,
			Biome:       spec.Biome,
			Attributes:  spec.Attributes,
			IsEdge:      spec.IsEdge,

			ModifierDefenses:        make(map[string]func(*Tile, *GameState) (int, int, int, error)),
			ModifierAfterConquest:   make(map[string]func(*Tile, *Tribe, *GameState)),
			ModifierSpecialDefenses: make(map[string]func(*Tile, *GameState, *Tribe, string) (bool, error)),
			ModifierPoints:          make(map[string]func(*Tile) int),
			State:                   make(map[string]interface{}),
		}
		if spec.LostTribe && !spec.IsSkyIsland && spec.Biome != Water && spec.Biome != River && spec.Biome != AbyssalChasm {
			if lostTribe == nil {
				lostTribe = CreateBaseTribe()
				lostTribe.Race = "Lost Tribe"
				lostTribe.Trait = "Lost"
				lostTribe.IsActive = false
				lostTribe.Owner = &Player{Index: -1, ActiveTribe: lostTribe, PieceStacks: []PieceStack{}}
			}
			tileMap[spec.ID].OwningTribe = lostTribe
			tileMap[spec.ID].PieceStacks = []PieceStack{{Type: "Lost Tribe", Amount: 1}}
		}
	}

	for _, spec := range specs {
		tile := tileMap[spec.ID]

		for _, adjacentID := range spec.AdjacentIDs {
			if adjacentTile, ok := tileMap[adjacentID]; ok {
				tile.AdjacentTiles = append(tile.AdjacentTiles, adjacentTile)
			}
		}
	}

	return tileMap, true
}
