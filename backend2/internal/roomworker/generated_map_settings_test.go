package roomworker

import (
	"encoding/json"
	"math"
	"reflect"
	"strings"
	"testing"
)

func settingAtPath(t *testing.T, settings MapGeneratorSettings, path string) any {
	t.Helper()
	var value any = map[string]any(settings)
	for _, key := range strings.Split(path, ".") {
		object, ok := value.(map[string]any)
		if !ok {
			t.Fatalf("%s is not an object while reading %s", key, path)
		}
		value, ok = object[key]
		if !ok {
			t.Fatalf("missing setting %s", path)
		}
	}
	return value
}

func settingsWithValue(path string, value any) MapGeneratorSettings {
	settings := MapGeneratorSettings{"seed": "settings-regression"}
	object := map[string]any(settings)
	parts := strings.Split(path, ".")
	for _, key := range parts[:len(parts)-1] {
		child := map[string]any{}
		object[key] = child
		object = child
	}
	object[parts[len(parts)-1]] = value
	return settings
}

func requireNormalizedSettings(t *testing.T, players int, input MapGeneratorSettings) MapGeneratorSettings {
	t.Helper()
	settings, err := normalizedGeneratedMapSettings(players, input)
	if err != nil {
		t.Fatal(err)
	}
	return settings
}

func TestGeneratedMapSettingsFixedCanvasAndNormalizedDefaults(t *testing.T) {
	counts := map[int]int{2: 22, 3: 29, 4: 38, 5: 47, 6: 56}
	for players := 2; players <= 6; players++ {
		settings := requireNormalizedSettings(t, players, MapGeneratorSettings{
			"seed": "settings-regression", "width": 1, "height": 1, "preserveRegionCount": false,
		})
		for key, want := range map[string]any{"tileCount": float64(counts[players]), "width": 1400, "height": 900, "preserveRegionCount": true} {
			if settings[key] != want {
				t.Fatalf("%d players: client changed %s to %v; want %v", players, key, settings[key], want)
			}
		}
		// These are the generator API's established normalized values. The
		// two-player legacy preset exceeds both the wobble and building limits.
		for path, want := range map[string]float64{
			"wobble": 30, "tileShapeSettings.chonkiness": 80, "tileShapeSettings.repulsionSteps": 12,
			"buildingSettings.averagePerTile": 0.28, "buildingSettings.maxPerTile": 1,
			"buildingSettings.symbolBuildingChance": 100,
			"biomeWeights.field":                    100, "biomeWeights.grass": 100, "biomeWeights.forest": 100,
			"biomeWeights.mountain": 100, "biomeWeights.swamp": 100,
			"biomeTextureZooms.field": 60, "biomeTextureZooms.forest": 40, "biomeTextureZooms.sea": 60,
		} {
			if got := settingAtPath(t, settings, path); got != want {
				t.Fatalf("%d players: %s=%v, want normalized %v", players, path, got, want)
			}
		}
		base := generatedMapSettingsForPlayerCount(players)
		wantSize := math.Min(3, float64(base["buildingSettings"].(map[string]any)["sizeMultiplier"].(int)))
		if got := settingAtPath(t, settings, "buildingSettings.sizeMultiplier"); got != wantSize {
			t.Fatalf("building size=%v, want %v", got, wantSize)
		}
		if got := settingAtPath(t, settings, "scenerySettings.mountainObjectSize").(float64); got < 35 || got > 240 {
			t.Fatalf("unclamped default mountain size: %v", got)
		}
		if settings["shapeMode"] != "mainland" || settings["centerLake"] != true || settings["randomizeSeaCorners"] != false {
			t.Fatal("default topology flags changed")
		}
		// Defaults sent to the editor must already be canonical: JSON echoing
		// them back cannot change a value and make a preview appear stale.
		raw, err := json.Marshal(settings)
		if err != nil {
			t.Fatal(err)
		}
		var echoed MapGeneratorSettings
		if err := json.Unmarshal(raw, &echoed); err != nil {
			t.Fatal(err)
		}
		if again := requireNormalizedSettings(t, players, echoed); !reflect.DeepEqual(settings, again) {
			t.Fatalf("%d-player defaults are not stable after JSON roundtrip", players)
		}
	}
}

func TestGeneratedMapSettingsPartialNestedMergeAndIsolation(t *testing.T) {
	input := MapGeneratorSettings{
		"seed": "settings-regression", "waterSettings": map[string]any{"seaSize": 88.5},
		"featureSettings": map[string]any{"magic": map[string]any{"count": 7}},
		"seaCorners":      map[string]any{"NW": true},
	}
	before, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	defaults := requireNormalizedSettings(t, 3, MapGeneratorSettings{"seed": "settings-regression"})
	settings := requireNormalizedSettings(t, 3, input)
	if settingAtPath(t, settings, "waterSettings.seaSize") != 88.5 || settingAtPath(t, settings, "featureSettings.magic.count") != float64(7) || settingAtPath(t, settings, "seaCorners.NW") != true {
		t.Fatal("partial edits were not applied")
	}
	for _, path := range []string{"waterSettings.centerLakeSize", "featureSettings.magic.spread", "featureSettings.mine", "featureSettings.cave", "seaCorners.NE", "seaCorners.SW", "seaCorners.SE", "scenerySettings", "buildingSettings"} {
		if !reflect.DeepEqual(settingAtPath(t, settings, path), settingAtPath(t, defaults, path)) {
			t.Fatalf("partial edit dropped sibling default %s", path)
		}
	}
	after, err := json.Marshal(input)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("normalization mutated the caller's nested settings")
	}
	settings["waterSettings"].(map[string]any)["centerLakeSize"] = -100
	settings["featureSettings"].(map[string]any)["mine"].(map[string]any)["count"] = -100
	if again := requireNormalizedSettings(t, 3, MapGeneratorSettings{"seed": "settings-regression"}); !reflect.DeepEqual(defaults, again) {
		t.Fatal("settings leaked between previews or rooms")
	}
}

func TestGeneratedMapSettingsNumericBounds(t *testing.T) {
	rules := []struct {
		path      string
		low, high float64
	}{
		{"tileCount", 18, 120},
		{"wobble", 0, 30}, {"spread", 0, 100}, {"tileShapeSettings.chonkiness", 0, 100}, {"tileShapeSettings.repulsionSteps", 0, 12},
		{"waterSettings.centerLakeSize", 40, 240}, {"waterSettings.seaSize", 40, 240},
		{"scenerySettings.forestTreeSize", 30, 220}, {"scenerySettings.mountainObjectSize", 35, 240},
		{"buildingSettings.averagePerTile", 0, 5}, {"buildingSettings.maxPerTile", 0, 5}, {"buildingSettings.sizeMultiplier", 0.2, 3}, {"buildingSettings.symbolBuildingChance", 0, 100},
	}
	for _, biome := range []string{"field", "grass", "forest", "mountain", "swamp"} {
		rules = append(rules, struct {
			path      string
			low, high float64
		}{"biomeWeights." + biome, 0, 300})
	}
	for _, biome := range []string{"field", "grass", "forest", "mountain", "swamp", "sea"} {
		rules = append(rules, struct {
			path      string
			low, high float64
		}{"biomeTextureZooms." + biome, 25, 400})
	}
	for _, feature := range []string{"magic", "mine", "cave"} {
		rules = append(rules, struct {
			path      string
			low, high float64
		}{"featureSettings." + feature + ".count", 0, 40}, struct {
			path      string
			low, high float64
		}{"featureSettings." + feature + ".spread", 0, 100})
	}
	for _, rule := range rules {
		t.Run(rule.path, func(t *testing.T) {
			for _, tc := range []struct{ input, want float64 }{{-1000, rule.low}, {rule.low, rule.low}, {rule.high, rule.high}, {1000, rule.high}} {
				settings := requireNormalizedSettings(t, 2, settingsWithValue(rule.path, tc.input))
				if got := settingAtPath(t, settings, rule.path); got != tc.want {
					t.Fatalf("%s(%v)=%v, want %v", rule.path, tc.input, got, tc.want)
				}
			}
		})
	}
	for _, path := range []string{"tileShapeSettings.repulsionSteps", "buildingSettings.maxPerTile", "featureSettings.magic.count", "featureSettings.mine.count", "featureSettings.cave.count"} {
		for _, tc := range []struct{ input, want float64 }{{1.49, 1}, {1.5, 2}, {2.5, 3}} {
			settings := requireNormalizedSettings(t, 2, settingsWithValue(path, tc.input))
			if got := settingAtPath(t, settings, path); got != tc.want {
				t.Fatalf("integer setting %s(%v)=%v, want %v", path, tc.input, got, tc.want)
			}
		}
	}
	if got := settingAtPath(t, requireNormalizedSettings(t, 2, settingsWithValue("spread", json.Number("25.5"))), "spread"); got != 25.5 {
		t.Fatal("valid JSON number rejected or rounded")
	}
}

func TestGeneratedMapSettingsRejectInvalidPayloads(t *testing.T) {
	for _, players := range []int{-1, 0, 1, 7, 100} {
		if _, err := normalizedGeneratedMapSettings(players, nil); err == nil {
			t.Fatalf("accepted player count %d", players)
		}
	}
	cases := []struct {
		path  string
		value any
	}{
		{"unknown", 1}, {"waterSettings.unknown", 1}, {"featureSettings.magic.unknown", 1},
		{"seed", nil}, {"seed", strings.Repeat("x", 129)}, {"seed", 12},
		{"shapeMode", "archipelago"}, {"shapeMode", nil},
		{"waterSettings", nil}, {"waterSettings", []any{}}, {"featureSettings.magic", "bad"},
		{"spread", "5"}, {"spread", true}, {"waterSettings.seaSize", map[string]any{}},
		{"tileCount", "35"}, {"tileCount", true}, {"tileCount", nil}, {"tileCount", math.NaN()}, {"tileCount", math.Inf(1)},
		{"spread", math.NaN()}, {"waterSettings.seaSize", math.Inf(1)}, {"featureSettings.mine.count", math.Inf(-1)},
		{"spread", json.Number("bad")}, {"centerLake", "true"}, {"seaCorners.NE", 1}, {"randomizeSeaCorners", nil},
	}
	for _, tc := range cases {
		if _, err := normalizedGeneratedMapSettings(2, settingsWithValue(tc.path, tc.value)); err == nil {
			t.Errorf("accepted invalid %s=%#v", tc.path, tc.value)
		}
	}
	for _, weight := range []float64{0, -1} {
		weights := map[string]any{}
		for _, biome := range []string{"field", "grass", "forest", "mountain", "swamp"} {
			weights[biome] = weight
		}
		if _, err := normalizedGeneratedMapSettings(2, MapGeneratorSettings{"biomeWeights": weights}); err == nil {
			t.Fatalf("accepted no-positive land biome weights: %v", weight)
		}
	}
}

func TestGeneratedMapSettingsSeedDeterminism(t *testing.T) {
	input := MapGeneratorSettings{"seed": "reproducible-preview", "featureSettings": map[string]any{"mine": map[string]any{"spread": 27}}}
	first := requireNormalizedSettings(t, 4, input)
	// Intervening preview requests, including UUID-generated seeds and another
	// player count, cannot advance a shared random stream used by this seed.
	for i := 0; i < 8; i++ {
		_ = requireNormalizedSettings(t, 2+i%5, nil)
	}
	if again := requireNormalizedSettings(t, 4, input); !reflect.DeepEqual(first, again) {
		t.Fatal("same supplied seed depends on prior preview requests")
	}
	if got := requireNormalizedSettings(t, 4, MapGeneratorSettings{"seed": strings.Repeat("x", 128)})["seed"]; got != strings.Repeat("x", 128) {
		t.Fatal("valid maximum-length seed changed")
	}
	defaults := requireNormalizedSettings(t, 2, nil)
	if seed, ok := defaults["seed"].(string); !ok || seed == "" {
		t.Fatal("omitted seed did not receive a usable default")
	}
}

func TestGeneratedMapSettingsBlankSeedsAreFresh(t *testing.T) {
	seen := make(map[string]bool)
	for _, input := range []MapGeneratorSettings{nil, {"seed": ""}, {"seed": " \t\n"}} {
		for i := 0; i < 2; i++ {
			settings := requireNormalizedSettings(t, 2, input)
			seed, ok := settings["seed"].(string)
			if !ok || strings.TrimSpace(seed) == "" || seen[seed] {
				t.Fatalf("omitted or blank seed did not produce a fresh usable seed: %q", seed)
			}
			seen[seed] = true
		}
	}
}

func TestGeneratedMapSettingsCustomTileCountIsAnInteger(t *testing.T) {
	for _, tc := range []struct{ input, want float64 }{{18.49, 18}, {35.5, 36}, {119.9, 120}} {
		settings := requireNormalizedSettings(t, 2, MapGeneratorSettings{"tileCount": tc.input})
		if settings["tileCount"] != tc.want {
			t.Fatalf("tileCount(%v)=%v; want %v", tc.input, settings["tileCount"], tc.want)
		}
	}
}
