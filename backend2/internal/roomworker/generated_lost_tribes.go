package roomworker

import (
	"backend/internal/gamestate"
	"hash/fnv"
	"math/rand"
	"sort"
)

// generatedLostTribeTileIDs distributes single neutral defenders over about a
// third of the land. Keep at least two thirds of entry regions open, and spread
// defenders by land-graph distance instead of building a wall around one area.
// This setup uses its own stream, independent of combat and training RNG.
func generatedLostTribeTileIDs(specs []gamestate.RuntimeTileSpec, seed string) []string {
	land := make(map[string]gamestate.RuntimeTileSpec)
	ids := []string{}
	edges := 0
	for _, spec := range specs {
		if spec.IsSkyIsland {
			continue
		}
		switch spec.Biome {
		case gamestate.Forest, gamestate.Hill, gamestate.Field, gamestate.Swamp, gamestate.Mountain:
		default:
			continue
		}
		land[spec.ID] = spec
		ids = append(ids, spec.ID)
		if spec.IsEdge {
			edges++
		}
	}
	edgeLimit := edges / 3
	target := min((len(ids)+1)/3, len(ids)-edges+edgeLimit)
	if target == 0 {
		return nil
	}
	sort.Strings(ids)
	hash := fnv.New64a()
	_, _ = hash.Write([]byte("lost-tribes:" + seed))
	rng := rand.New(rand.NewSource(int64(hash.Sum64())))
	rng.Shuffle(len(ids), func(i, j int) { ids[i], ids[j] = ids[j], ids[i] })

	distance := make(map[string]int, len(ids))
	for _, id := range ids {
		distance[id] = len(ids) + 1
	}
	selected := make(map[string]bool, target)
	result := make([]string, 0, target)
	usedEdges := 0
	for len(result) < target {
		best := ""
		found := false
		for _, id := range ids {
			if selected[id] || (land[id].IsEdge && usedEdges >= edgeLimit) {
				continue
			}
			// On equal distances, avoid stacking a neutral's extra defense on
			// mountains. Remaining ties follow the seeded shuffle above.
			if !found || distance[id] > distance[best] ||
				(distance[id] == distance[best] && land[best].Biome == gamestate.Mountain && land[id].Biome != gamestate.Mountain) {
				best, found = id, true
			}
		}
		if !found {
			break
		}
		result = append(result, best)
		selected[best] = true
		if land[best].IsEdge {
			usedEdges++
		}

		// Multi-source shortest distances, incrementally updated after each
		// pick. Water never provides a shortcut between land regions.
		distance[best] = 0
		queue := []string{best}
		for head := 0; head < len(queue); head++ {
			id := queue[head]
			for _, adjacent := range land[id].AdjacentIDs {
				if _, ok := land[adjacent]; !ok || distance[adjacent] <= distance[id]+1 {
					continue
				}
				distance[adjacent] = distance[id] + 1
				queue = append(queue, adjacent)
			}
		}
	}
	return result
}

func markGeneratedLostTribes(specs []gamestate.RuntimeTileSpec, ids []string) {
	selected := make(map[string]bool, len(ids))
	for _, id := range ids {
		selected[id] = true
	}
	for i := range specs {
		specs[i].LostTribe = selected[specs[i].ID]
	}
}
