package roomworker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"reflect"
	"testing"
)

func TestVariedGeneratedMapSettings(t *testing.T) {
	seen := make(map[string]bool)
	seaCounts := make(map[int]bool)
	for players := 2; players <= 6; players++ {
		for n := 0; n < 200; n++ {
			seed := fmt.Sprintf("layout-check-%d", n)
			base := generatedMapSettingsForPlayerCount(players)
			varied := variedGeneratedMapSettings(players, seed)
			if !reflect.DeepEqual(varied, variedGeneratedMapSettings(players, seed)) {
				t.Fatalf("%d players: settings are not reproducible for %q", players, seed)
			}
			if varied["seed"] != seed || varied["preserveRegionCount"] != true {
				t.Fatal("missing deterministic seed or fixed-count mode")
			}
			spread := varied["spread"].(int)
			if spread < 0 || spread > 35 {
				t.Fatalf("unsafe biome spread: %d", spread)
			}
			variation := []int{spread}
			features := varied["featureSettings"].(map[string]any)
			baseFeatures := base["featureSettings"].(map[string]any)
			for _, feature := range []string{"magic", "mine", "cave"} {
				featureSettings := features[feature].(map[string]any)
				featureSpread := featureSettings["spread"].(int)
				if featureSpread < 5 || featureSpread > 40 {
					t.Fatalf("unsafe %s spread: %d", feature, featureSpread)
				}
				variation = append(variation, featureSpread)
				featureSettings["spread"] = baseFeatures[feature].(map[string]any)["spread"]
			}
			water := varied["waterSettings"].(map[string]any)
			baseWater := base["waterSettings"].(map[string]any)
			for name, percent := range map[string]int{"centerLakeSize": 10, "seaSize": 15} {
				value, original := water[name].(int), baseWater[name].(int)
				if value < (original*(100-percent)+50)/100 || value > (original*(100+percent)+50)/100 {
					t.Fatalf("unsafe %s size: %d", name, value)
				}
				variation = append(variation, value)
			}
			seaCount := 0
			for _, enabled := range varied["seaCorners"].(map[string]any) {
				if enabled.(bool) {
					seaCount++
				}
			}
			if seaCount < 1 || seaCount > 3 {
				t.Fatalf("unsafe sea count: %d", seaCount)
			}
			seaCounts[seaCount] = true
			scenery := varied["scenerySettings"].(map[string]any)
			for _, name := range []string{"forestTreeSize", "mountainObjectSize"} {
				original := 100
				if preset, ok := base["scenerySettings"].(map[string]any); ok {
					original = preset[name].(int)
				}
				if name == "mountainObjectSize" && original > 240 {
					original = 240
				}
				value := scenery[name].(int)
				if value < (original*92+50)/100 || value > (original*108+50)/100 || name == "mountainObjectSize" && value > 240 {
					t.Fatalf("unsafe %s scenery: %d", name, value)
				}
			}
			seen[fmt.Sprint(variation)] = true
			delete(varied, "seed")
			delete(varied, "preserveRegionCount")
			for _, name := range []string{"spread", "waterSettings", "seaCorners", "scenerySettings"} {
				if original, ok := base[name]; ok {
					varied[name] = original
				} else {
					delete(varied, name)
				}
			}
			// Everything else is the unchanged preset, including border style,
			// region count, dimensions, land-biome weights and feature counts.
			if !reflect.DeepEqual(base, varied) {
				t.Fatalf("%d players: variation changed a fixed setting", players)
			}
			if !reflect.DeepEqual(base, generatedMapSettingsForPlayerCount(players)) {
				t.Fatal("variation mutated the baseline settings")
			}
		}
	}
	if len(seen) < 150 || len(seaCounts) != 3 {
		t.Fatalf("too little variation: %d layouts, %d sea counts", len(seen), len(seaCounts))
	}
}

func TestVariedGeneratedMapSettingsPreserveGeneratedStructure(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("Node.js is required to exercise the real map generator")
	}
	type generationCase struct {
		Players int                  `json:"players"`
		Base    MapGeneratorSettings `json:"base"`
		Varied  MapGeneratorSettings `json:"varied"`
	}
	cases := make([]generationCase, 0, 1000)
	for players := 2; players <= 6; players++ {
		for n := 0; n < 200; n++ {
			seed := fmt.Sprintf("layout-check-%d", n)
			base := generatedMapSettingsForPlayerCount(players)
			base["seed"] = seed
			cases = append(cases, generationCase{players, base, variedGeneratedMapSettings(players, seed)})
		}
	}
	// Exercise every 1–3-corner configuration at the water-size boundaries,
	// in addition to the seeded production-setting sample above.
	for players := 2; players <= 6; players++ {
		for mask := 1; mask < 15; mask++ {
			for _, lakePercent := range []int{90, 110} {
				for _, seaPercent := range []int{85, 115} {
					seed := fmt.Sprintf("water-boundary-%d-%d-%d", mask, lakePercent, seaPercent)
					base := generatedMapSettingsForPlayerCount(players)
					base["seed"] = seed
					varied := variedGeneratedMapSettings(players, seed)
					water := varied["waterSettings"].(map[string]any)
					baseWater := base["waterSettings"].(map[string]any)
					water["centerLakeSize"] = (baseWater["centerLakeSize"].(int)*lakePercent + 50) / 100
					water["seaSize"] = (baseWater["seaSize"].(int)*seaPercent + 50) / 100
					for index, corner := range []string{"NW", "NE", "SW", "SE"} {
						varied["seaCorners"].(map[string]any)[corner] = mask&(1<<index) != 0
					}
					cases = append(cases, generationCase{players, base, varied})
				}
			}
		}
	}
	payload, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(node, "testdata/map_settings_structure.mjs")
	cmd.Stdin = bytes.NewReader(payload)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("real map generator invariant check failed: %v\n%s", err, output)
	}
	t.Log(string(output))
}
