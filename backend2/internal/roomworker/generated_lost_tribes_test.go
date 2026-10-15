package roomworker

import (
	"backend/internal/gamestate"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"reflect"
	"slices"
	"testing"

	"github.com/redis/go-redis/v9"
)

func lostTribeLine(length int) []gamestate.RuntimeTileSpec {
	specs := make([]gamestate.RuntimeTileSpec, length)
	for i := range specs {
		specs[i] = gamestate.RuntimeTileSpec{ID: fmt.Sprint(i), Biome: gamestate.Field, IsEdge: i == 0 || i == length-1}
		if i > 0 {
			specs[i].AdjacentIDs = append(specs[i].AdjacentIDs, fmt.Sprint(i-1))
		}
		if i+1 < length {
			specs[i].AdjacentIDs = append(specs[i].AdjacentIDs, fmt.Sprint(i+1))
		}
	}
	return specs
}

func TestGeneratedLostTribesSpreadAndStableSeed(t *testing.T) {
	specs := lostTribeLine(18)
	before, _ := json.Marshal(specs)
	selected := generatedLostTribeTileIDs(specs, "one-map")
	if len(selected) != 6 {
		t.Fatalf("got %d neutral regions, want 6", len(selected))
	}
	for _, id := range selected {
		for _, spec := range specs {
			if spec.ID == id {
				if spec.IsEdge {
					t.Fatal("blocked one of only two entry regions")
				}
				for _, neighbor := range spec.AdjacentIDs {
					if slices.Contains(selected, neighbor) {
						t.Fatal("clustered defenders on a roomy line map")
					}
				}
			}
		}
	}
	after, _ := json.Marshal(specs)
	if !bytes.Equal(before, after) {
		t.Fatal("selection changed topology, terrain, or input setup")
	}
	slices.Reverse(specs)
	if !reflect.DeepEqual(selected, generatedLostTribeTileIDs(specs, "one-map")) {
		t.Fatal("placement depends on metadata iteration order")
	}
	seen := map[string]bool{}
	for i := 0; i < 30; i++ {
		seen[fmt.Sprint(generatedLostTribeTileIDs(specs, fmt.Sprint(i)))] = true
	}
	if len(seen) < 5 {
		t.Fatal("different maps do not vary placement")
	}
}

func TestGeneratedLostTribesEntryAndTerrainLimits(t *testing.T) {
	for n := 0; n <= 12; n++ {
		specs := lostTribeLine(n)
		for i := range specs {
			specs[i].IsEdge = true
		}
		for i, biome := range []gamestate.Biome{gamestate.Water, gamestate.River, gamestate.AbyssalChasm} {
			specs = append(specs, gamestate.RuntimeTileSpec{ID: fmt.Sprintf("excluded-%d", i), Biome: biome})
		}
		specs = append(specs, gamestate.RuntimeTileSpec{ID: "sky", Biome: gamestate.Field, IsSkyIsland: true})
		selected := generatedLostTribeTileIDs(specs, "edge-map")
		if len(selected) != n/3 {
			t.Fatalf("%d all-edge land regions: got %d defenders, want %d", n, len(selected), n/3)
		}
		for _, id := range selected {
			if id == "sky" || len(id) > 2 {
				t.Fatalf("placed a neutral on ineligible terrain: %s", id)
			}
		}
	}
}

func TestGeneratedLostTribesOnRealMaps(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required to exercise the real map generator")
	}
	settings := []MapGeneratorSettings{}
	for players := 2; players <= 6; players++ {
		for i := 0; i < 20; i++ {
			config := variedGeneratedMapSettings(players, fmt.Sprintf("neutral-%d-%d", players, i))
			config["wobble"] = 30 // The API's existing clamp.
			settings = append(settings, config)
		}
	}
	input, _ := json.Marshal(settings)
	cmd := exec.Command(node, "testdata/generated_lost_tribes.mjs")
	cmd.Stdin = bytes.NewReader(input)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generate real maps: %v\n%s", err, output)
	}
	var maps [][]GeneratedServerTile
	if err := json.Unmarshal(output, &maps); err != nil {
		t.Fatal(err)
	}
	if len(maps) != 100 {
		t.Fatalf("expected 100 generated maps, got %d", len(maps))
	}
	for i, tiles := range maps {
		response := mapGeneratorResponse{}
		response.Metadata.Server.Tiles = tiles
		specs, err := convertGeneratedTiles(&response)
		if err != nil {
			t.Fatal(err)
		}
		selected := generatedLostTribeTileIDs(specs, settings[i]["seed"].(string))
		land, edges, occupiedEdges := 0, 0, 0
		for _, spec := range specs {
			if spec.Biome != gamestate.Water {
				land++
				if spec.IsEdge {
					edges++
				}
			}
			if slices.Contains(selected, spec.ID) {
				if spec.Biome == gamestate.Water {
					t.Fatal("Lost Tribe on water")
				}
				if spec.IsEdge {
					occupiedEdges++
				}
			}
		}
		if len(selected) != (land+1)/3 || occupiedEdges > edges/3 {
			t.Fatalf("map %d: %d/%d land occupied, %d/%d entry regions occupied", i, len(selected), land, occupiedEdges, edges)
		}
	}
}

func TestGeneratedLostTribesRegistrationAndRestore(t *testing.T) {
	response := mapGeneratorResponse{ImageBase64: "dGVzdA=="}
	response.Metadata.Graphics = json.RawMessage(`{}`)
	for _, spec := range lostTribeLine(18) {
		tile := GeneratedServerTile{ID: GeneratedID(spec.ID), Biome: "Field", IsEdge: spec.IsEdge}
		for _, id := range spec.AdjacentIDs {
			tile.AdjacentIDs = append(tile.AdjacentIDs, GeneratedID(id))
		}
		response.Metadata.Server.Tiles = append(response.Metadata.Server.Tiles, tile)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewEncoder(w).Encode(response); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	t.Setenv("MAP_GENERATOR_URL", server.URL)
	gameMap, err := GenerateRuntimeMapForPlayerCount(2)
	if err != nil {
		t.Fatal(err)
	}
	visuals, ok := GetGeneratedMapVisuals(gameMap.Name)
	if !ok || len(visuals.LostTribeTileIDs) != 6 {
		t.Fatalf("initial neutral setup not retained: %+v", visuals.LostTribeTileIDs)
	}
	assertMap := func(want []string) {
		t.Helper()
		tiles, ok := gamestate.BuildRuntimeMap(gameMap.Name, nil)
		if !ok {
			t.Fatal("runtime map not registered")
		}
		for id, tile := range tiles {
			if (tile.OwningTribe != nil) != slices.Contains(want, id) {
				t.Fatalf("wrong neutral ownership in %s", id)
			}
		}
	}
	assertMap(visuals.LostTribeTileIDs)
	addr := os.Getenv("SMALLWORLD_TEST_REDIS_ADDR")
	if addr == "" {
		t.Skip("initial registration passed; set SMALLWORLD_TEST_REDIS_ADDR for restoration checks")
	}
	rdb := redis.NewClient(&redis.Options{Addr: addr})
	defer rdb.Close()
	w := &Worker{redis: rdb}
	ctx := context.Background()
	defer rdb.Del(ctx, generatedMapKey(gameMap.Name))
	defer delete(mapMap, gameMap.Name)
	for _, ids := range [][]string{visuals.LostTribeTileIDs, nil} {
		visuals.LostTribeTileIDs = ids
		raw, _ := json.Marshal(generatedMapSnapshot{Map: gameMap, Visuals: visuals})
		if err := rdb.Set(ctx, generatedMapKey(gameMap.Name), raw, 0).Err(); err != nil {
			t.Fatal(err)
		}
		if err := w.restoreGeneratedMap(ctx, gameMap.Name); err != nil {
			t.Fatal(err)
		}
		assertMap(ids) // Missing old-format IDs must leave old maps empty.
	}
}
