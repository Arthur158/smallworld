package main

import (
	"backend/internal/gamestate"
	"fmt"
	"slices"
)

// Imported layouts come from the same Voronoi generator as live rooms. Validate
// the graph at the engine boundary before replacing the single reusable entry.
func installTrainingMap(tiles []gamestate.RuntimeTileSpec) error {
	if len(tiles) < 12 || len(tiles) > 160 {
		return fmt.Errorf("generated map needs 12..160 tiles")
	}
	byID := make(map[string]gamestate.RuntimeTileSpec, len(tiles))
	for _, tile := range tiles {
		if tile.ID == "" || len(tile.ID) > 80 {
			return fmt.Errorf("invalid generated tile ID")
		}
		if _, ok := byID[tile.ID]; ok {
			return fmt.Errorf("duplicate generated tile %q", tile.ID)
		}
		if tile.Biome < gamestate.Forest || tile.Biome > gamestate.AbyssalChasm {
			return fmt.Errorf("invalid generated biome")
		}
		if len(tile.Attributes) > 3 {
			return fmt.Errorf("too many tile attributes")
		}
		for _, a := range tile.Attributes {
			if a < gamestate.Magic || a > gamestate.Cave {
				return fmt.Errorf("invalid generated attribute")
			}
		}
		byID[tile.ID] = tile
	}
	for _, tile := range tiles {
		seen := make(map[string]bool)
		for _, id := range tile.AdjacentIDs {
			n, ok := byID[id]
			if !ok || id == tile.ID || seen[id] || !slices.Contains(n.AdjacentIDs, tile.ID) {
				return fmt.Errorf("invalid adjacency for tile %q", tile.ID)
			}
			seen[id] = true
		}
	}
	visit := func(start string, landOnly bool) map[string]bool {
		seen := map[string]bool{start: true}
		queue := []string{start}
		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]
			for _, id := range byID[current].AdjacentIDs {
				if !seen[id] && (!landOnly || byID[id].Biome != gamestate.Water && byID[id].Biome != gamestate.AbyssalChasm) {
					seen[id] = true
					queue = append(queue, id)
				}
			}
		}
		return seen
	}
	if len(visit(tiles[0].ID, false)) != len(tiles) {
		return fmt.Errorf("generated graph is disconnected")
	}
	landSeen := make(map[string]bool)
	landCount := 0
	for _, tile := range tiles {
		if tile.Biome == gamestate.Water || tile.Biome == gamestate.AbyssalChasm {
			continue
		}
		landCount++
		if landSeen[tile.ID] {
			continue
		}
		component := visit(tile.ID, true)
		entry := false
		for id := range component {
			landSeen[id] = true
			entry = entry || byID[id].IsEdge
		}
		if !entry {
			return fmt.Errorf("land component has no entry tile")
		}
	}
	if landCount < 8 {
		return fmt.Errorf("generated map needs at least 8 land tiles")
	}
	gamestate.RegisterRuntimeMap("botenv-generated", tiles)
	return nil
}
