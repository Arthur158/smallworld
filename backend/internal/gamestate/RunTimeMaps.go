package gamestate

import "sync"

type RuntimeTileSpec struct {
	ID          string
	Biome      Biome
	Attributes []Attribute
	IsEdge      bool
	AdjacentIDs []string
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

	for _, spec := range specs {
		tileMap[spec.ID] = &Tile{
			Id:         spec.ID,
			Biome:     spec.Biome,
			Attributes: spec.Attributes,
			IsEdge:    spec.IsEdge,

			ModifierDefenses:        make(map[string]func(*Tile, *GameState) (int, int, int, error)),
			ModifierAfterConquest:   make(map[string]func(*Tile, *Tribe, *GameState)),
			ModifierSpecialDefenses: make(map[string]func(*Tile, *GameState, *Tribe, string) (bool, error)),
			ModifierPoints:          make(map[string]func(*Tile) int),
			State:                   make(map[string]interface{}),
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
