package server

import (
	"bytes"
	"backend/internal/gamestate"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/google/uuid"
)

type MapGeneratorSettings map[string]any

type mapGeneratorResponse struct {
	ImageBase64 string          `json:"imageBase64"`
	Settings    json.RawMessage `json:"settings"`
	Metadata    struct {
		Server struct {
			Tiles []GeneratedServerTile `json:"tiles"`
		} `json:"server"`

		Graphics json.RawMessage `json:"graphics"`
	} `json:"metadata"`
}

func mapGeneratorURL() string {
	if value := os.Getenv("MAP_GENERATOR_URL"); value != "" {
		return value
	}

	return "http://127.0.0.1:3001"
}

func generatedMapSettingsForPlayerCount(playerCount int) MapGeneratorSettings {
	settingsByPlayerCount := map[int]MapGeneratorSettings{
		2: map[string]any{
			"tileCount": 22,
			"width": 1400,
			"height": 900,
			"shapeMode": "mainland",

			"wobble": 34,
			"spread": 0,

			"tileShapeSettings": map[string]any{
			  "chonkiness": 80,
			  "repulsionSteps": 12,
			},

			"randomizeSeaCorners": false,
			"centerLake": true,

			"seaCorners": map[string]any{
			  "NW": true,
			  "NE": false,
			  "SW": false,
			  "SE": true,
			},

			"waterSettings": map[string]any{
			  "centerLakeSize": 140,
			  "seaSize": 120,
			},

			"biomeWeights": map[string]any{
			  "field": 100,
			  "grass": 100,
			  "forest": 100,
			  "mountain": 100,
			  "swamp": 100,
			},

			"biomeTextureZooms": map[string]any{
			  "field": 60,
			  "grass": 60,
			  "forest": 40,
			  "mountain": 60,
			  "swamp": 60,
			  "sea": 60,
			},

			"buildingSettings": map[string]any{
			  "averagePerTile": 0.28,
			  "maxPerTile": 1,
			  "sizeMultiplier": 2,
			},

			"featureSettings": map[string]any{
			  "magic": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "mine": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "cave": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			},
		},
		3: map[string]any{
			"tileCount": 29,
			"width": 1400,
			"height": 900,
			"shapeMode": "mainland",

			"wobble": 34,
			"spread": 0,

			"tileShapeSettings": map[string]any{
			  "chonkiness": 80,
			  "repulsionSteps": 12,
			},

			"randomizeSeaCorners": false,
			"centerLake": true,

			"seaCorners": map[string]any{
			  "NW": true,
			  "NE": false,
			  "SW": false,
			  "SE": true,
			},

			"waterSettings": map[string]any{
			  "centerLakeSize": 140,
			  "seaSize": 120,
			},

			"biomeWeights": map[string]any{
			  "field": 100,
			  "grass": 100,
			  "forest": 100,
			  "mountain": 100,
			  "swamp": 100,
			},

			"biomeTextureZooms": map[string]any{
			  "field": 60,
			  "grass": 60,
			  "forest": 40,
			  "mountain": 60,
			  "swamp": 60,
			  "sea": 60,
			},

			"buildingSettings": map[string]any{
			  "averagePerTile": 0.28,
			  "maxPerTile": 1,
			  "sizeMultiplier": 2,
			},

			"featureSettings": map[string]any{
			  "magic": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "mine": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "cave": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			},
		},
		4: map[string]any{
			"tileCount": 38,
			"width": 1400,
			"height": 900,
			"shapeMode": "mainland",

			"wobble": 34,
			"spread": 0,

			"tileShapeSettings": map[string]any{
			  "chonkiness": 80,
			  "repulsionSteps": 12,
			},

			"randomizeSeaCorners": false,
			"centerLake": true,

			"seaCorners": map[string]any{
			  "NW": true,
			  "NE": false,
			  "SW": false,
			  "SE": true,
			},

			"waterSettings": map[string]any{
			  "centerLakeSize": 140,
			  "seaSize": 120,
			},

			"biomeWeights": map[string]any{
			  "field": 100,
			  "grass": 100,
			  "forest": 100,
			  "mountain": 100,
			  "swamp": 100,
			},

			"biomeTextureZooms": map[string]any{
			  "field": 60,
			  "grass": 60,
			  "forest": 40,
			  "mountain": 60,
			  "swamp": 60,
			  "sea": 60,
			},

			"buildingSettings": map[string]any{
			  "averagePerTile": 0.28,
			  "maxPerTile": 1,
			  "sizeMultiplier": 2,
			},

			"featureSettings": map[string]any{
			  "magic": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "mine": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "cave": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			},
		},
		5: map[string]any{
			"tileCount": 47,
			"width": 1400,
			"height": 900,
			"shapeMode": "mainland",

			"wobble": 34,
			"spread": 0,

			"tileShapeSettings": map[string]any{
			  "chonkiness": 80,
			  "repulsionSteps": 12,
			},

			"randomizeSeaCorners": false,
			"centerLake": true,

			"seaCorners": map[string]any{
			  "NW": true,
			  "NE": false,
			  "SW": false,
			  "SE": true,
			},

			"waterSettings": map[string]any{
			  "centerLakeSize": 140,
			  "seaSize": 120,
			},

			"biomeWeights": map[string]any{
			  "field": 100,
			  "grass": 100,
			  "forest": 100,
			  "mountain": 100,
			  "swamp": 100,
			},

			"biomeTextureZooms": map[string]any{
			  "field": 60,
			  "grass": 60,
			  "forest": 40,
			  "mountain": 60,
			  "swamp": 60,
			  "sea": 60,
			},

			"buildingSettings": map[string]any{
			  "averagePerTile": 0.28,
			  "maxPerTile": 1,
			  "sizeMultiplier": 2,
			},

			"featureSettings": map[string]any{
			  "magic": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "mine": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "cave": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			},
		},
		6: map[string]any{
			"tileCount": 56,
			"width": 1400,
			"height": 900,
			"shapeMode": "mainland",

			"wobble": 34,
			"spread": 0,

			"tileShapeSettings": map[string]any{
			  "chonkiness": 80,
			  "repulsionSteps": 12,
			},

			"randomizeSeaCorners": false,
			"centerLake": true,

			"seaCorners": map[string]any{
			  "NW": true,
			  "NE": false,
			  "SW": false,
			  "SE": true,
			},

			"waterSettings": map[string]any{
			  "centerLakeSize": 140,
			  "seaSize": 120,
			},

			"biomeWeights": map[string]any{
			  "field": 100,
			  "grass": 100,
			  "forest": 100,
			  "mountain": 100,
			  "swamp": 100,
			},

			"biomeTextureZooms": map[string]any{
			  "field": 60,
			  "grass": 60,
			  "forest": 40,
			  "mountain": 60,
			  "swamp": 60,
			  "sea": 60,
			},

			"buildingSettings": map[string]any{
			  "averagePerTile": 0.28,
			  "maxPerTile": 1,
			  "sizeMultiplier": 2,
			},

			"featureSettings": map[string]any{
			  "magic": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "mine": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			  "cave": map[string]any{
				"count": 4,
				"spread": 0,
			  },
			},
		},
	}

	base, ok := settingsByPlayerCount[playerCount]
	if !ok {
		base = settingsByPlayerCount[2]
	}

	settings := MapGeneratorSettings{}
	for key, value := range base {
		settings[key] = value
	}

	return settings
}

func requestGeneratedMap(settings MapGeneratorSettings) (*mapGeneratorResponse, error) {
	body, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}

	url := mapGeneratorURL() + "/api/generate"

	resp, err := http.Post(url, "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to contact map generator at %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		raw, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("map generator returned %s: %s", resp.Status, string(raw))
	}

	var decoded mapGeneratorResponse
	if err := json.NewDecoder(resp.Body).Decode(&decoded); err != nil {
		return nil, err
	}

	if decoded.ImageBase64 == "" {
		return nil, fmt.Errorf("map generator returned no imageBase64")
	}

	if len(decoded.Metadata.Server.Tiles) == 0 {
		return nil, fmt.Errorf("map generator returned no server tiles")
	}

	if len(decoded.Metadata.Graphics) == 0 {
		return nil, fmt.Errorf("map generator returned no graphics metadata")
	}

	return &decoded, nil
}

func generatedBiomeToGameBiome(value string) (gamestate.Biome, error) {
	switch value {
	case "Forest":
		return gamestate.Forest, nil
	case "Hill":
		return gamestate.Hill, nil
	case "Field":
		return gamestate.Field, nil
	case "Swamp":
		return gamestate.Swamp, nil
	case "Water":
		return gamestate.Water, nil
	case "River":
		return gamestate.River, nil
	case "Mountain":
		return gamestate.Mountain, nil
	default:
		return gamestate.Forest, fmt.Errorf("unknown generated biome: %s", value)
	}
}

func generatedAttributeToGameAttribute(value string) (gamestate.Attribute, error) {
	switch value {
	case "Magic":
		return gamestate.Magic, nil
	case "Mine":
		return gamestate.Mine, nil
	case "Cave":
		return gamestate.Cave, nil
	default:
		return gamestate.Magic, fmt.Errorf("unknown generated attribute: %s", value)
	}
}

func convertGeneratedTiles(resp *mapGeneratorResponse) ([]gamestate.RuntimeTileSpec, error) {
	specs := make([]gamestate.RuntimeTileSpec, 0, len(resp.Metadata.Server.Tiles))

	for _, tile := range resp.Metadata.Server.Tiles {
		biome, err := generatedBiomeToGameBiome(tile.Biome)
		if err != nil {
			return nil, err
		}

		attributes := []gamestate.Attribute{}
		for _, attrName := range tile.Attributes {
			attr, err := generatedAttributeToGameAttribute(attrName)
			if err != nil {
				return nil, err
			}

			attributes = append(attributes, attr)
		}

		specs = append(specs, gamestate.RuntimeTileSpec{
			ID:          tile.ID.String(),
			Biome:      biome,
			Attributes: attributes,
			IsEdge:      tile.IsEdge,
			AdjacentIDs: generatedIDsToStrings(tile.AdjacentIDs),
		})
	}

	return specs, nil
}

func GenerateRuntimeMapForPlayerCount(playerCount int) (Map, error) {
	mapName := "generated-" + uuid.NewString()

	settings := generatedMapSettingsForPlayerCount(playerCount)
	settings["seed"] = mapName

	resp, err := requestGeneratedMap(settings)
	if err != nil {
		return Map{}, err
	}

	specs, err := convertGeneratedTiles(resp)
	if err != nil {
		return Map{}, err
	}

	gamestate.RegisterRuntimeMap(mapName, specs)

	generatedMap := Map{
		Name:     mapName,
		Offset:   0.808,
		FontSize: 60,
		Capacity: playerCount,
	}

	StoreGeneratedMapVisuals(GeneratedMapVisuals{
		MapName:     mapName,
		ImageBase64: resp.ImageBase64,
		Graphics:    resp.Metadata.Graphics,
		Settings:    resp.Settings,
		ServerTiles: resp.Metadata.Server.Tiles,
		Offset:      0.808,
	})

	return generatedMap, nil
}

func generatedIDsToStrings(ids []GeneratedID) []string {
	result := make([]string, 0, len(ids))

	for _, id := range ids {
		result = append(result, id.String())
	}

	return result
}
