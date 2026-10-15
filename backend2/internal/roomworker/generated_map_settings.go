package roomworker

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
)

// Custom settings are confined to lobby previews. Training and automatic map
// requests retain their existing defaults and normalization path.
func normalizedGeneratedMapSettings(players int, input MapGeneratorSettings) (MapGeneratorSettings, error) {
	if players < 2 || players > 6 {
		return nil, fmt.Errorf("generated maps support 2–6 players")
	}
	seed := "generated-" + uuid.NewString()
	if value, ok := input["seed"]; ok {
		text, ok := value.(string)
		if !ok || len(text) > 128 {
			return nil, fmt.Errorf("seed must be a string of at most 128 characters")
		}
		if strings.TrimSpace(text) != "" {
			seed = text
		}
	}
	settings := variedGeneratedMapSettings(players, seed)
	settings["buildingSettings"].(map[string]any)["symbolBuildingChance"] = 100
	type numberRule struct {
		low, high float64
		integer   bool
	}
	rules := map[string]any{
		"seed": "string", "tileCount": numberRule{18, 120, true}, "width": "fixed", "height": "fixed", "preserveRegionCount": "fixed",
		"shapeMode": "shape", "wobble": numberRule{0, 30, false}, "spread": numberRule{0, 100, false},
		"tileShapeSettings": map[string]any{"chonkiness": numberRule{0, 100, false}, "repulsionSteps": numberRule{0, 12, true}},
		"waterSettings":     map[string]any{"centerLakeSize": numberRule{40, 240, false}, "seaSize": numberRule{40, 240, false}},
		"scenerySettings":   map[string]any{"forestTreeSize": numberRule{30, 220, false}, "mountainObjectSize": numberRule{35, 240, false}},
		"biomeWeights":      map[string]any{}, "biomeTextureZooms": map[string]any{},
		"buildingSettings":    map[string]any{"averagePerTile": numberRule{0, 5, false}, "maxPerTile": numberRule{0, 5, true}, "sizeMultiplier": numberRule{0.2, 3, false}, "symbolBuildingChance": numberRule{0, 100, false}},
		"featureSettings":     map[string]any{},
		"seaCorners":          map[string]any{"NW": "bool", "NE": "bool", "SW": "bool", "SE": "bool"},
		"randomizeSeaCorners": "bool", "centerLake": "bool",
	}
	for _, biome := range []string{"field", "grass", "forest", "mountain", "swamp"} {
		rules["biomeWeights"].(map[string]any)[biome] = numberRule{0, 300, false}
		rules["biomeTextureZooms"].(map[string]any)[biome] = numberRule{25, 400, false}
	}
	rules["biomeTextureZooms"].(map[string]any)["sea"] = numberRule{25, 400, false}
	for _, feature := range []string{"magic", "mine", "cave"} {
		rules["featureSettings"].(map[string]any)[feature] = map[string]any{"count": numberRule{0, 40, true}, "spread": numberRule{0, 100, false}}
	}
	var merge func(map[string]any, map[string]any, map[string]any, string) error
	merge = func(dst, src, schema map[string]any, prefix string) error {
		for key := range src {
			if _, ok := schema[key]; !ok {
				return fmt.Errorf("unknown map setting %s%s", prefix, key)
			}
		}
		for key, rule := range schema {
			value, supplied := src[key]
			if !supplied {
				value = dst[key]
			}
			switch r := rule.(type) {
			case map[string]any:
				child, ok := value.(map[string]any)
				if !ok {
					return fmt.Errorf("%s%s must be an object", prefix, key)
				}
				if err := merge(dst[key].(map[string]any), child, r, prefix+key+"."); err != nil {
					return err
				}
			case numberRule:
				var number float64
				switch n := value.(type) {
				case float64:
					number = n
				case int:
					number = float64(n)
				case json.Number:
					var err error
					number, err = n.Float64()
					if err != nil {
						return fmt.Errorf("%s%s must be a number", prefix, key)
					}
				default:
					return fmt.Errorf("%s%s must be a number", prefix, key)
				}
				if math.IsNaN(number) || math.IsInf(number, 0) {
					return fmt.Errorf("%s%s must be finite", prefix, key)
				}
				number = math.Max(r.low, math.Min(r.high, number))
				if r.integer {
					number = math.Round(number)
				}
				dst[key] = number
			case string:
				switch r {
				case "fixed": // The frontend may echo these read-only values, but cannot change them.
				case "string":
					dst[key] = seed
				case "bool":
					if _, ok := value.(bool); !ok {
						return fmt.Errorf("%s%s must be a boolean", prefix, key)
					}
					dst[key] = value
				case "shape":
					if value != "mainland" && value != "islands" {
						return fmt.Errorf("shapeMode must be mainland or islands")
					}
					dst[key] = value
				}
			}
		}
		return nil
	}
	if err := merge(settings, input, rules, ""); err != nil {
		return nil, err
	}
	total := 0.0
	for _, value := range settings["biomeWeights"].(map[string]any) {
		total += value.(float64)
	}
	if total <= 0 {
		return nil, fmt.Errorf("at least one land biome needs a positive weight")
	}
	return settings, nil
}
