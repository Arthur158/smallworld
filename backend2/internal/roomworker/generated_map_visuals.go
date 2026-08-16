package roomworker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sync"
)

type GeneratedID string

func (id *GeneratedID) UnmarshalJSON(data []byte) error {
	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		*id = GeneratedID(stringValue)
		return nil
	}

	var numberValue json.Number
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&numberValue); err == nil {
		*id = GeneratedID(numberValue.String())
		return nil
	}

	return fmt.Errorf("generated map id must be string or number, got %s", string(data))
}

func (id GeneratedID) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(id))
}

func (id GeneratedID) String() string {
	return string(id)
}

type GeneratedServerTile struct {
	ID          GeneratedID   `json:"id"`
	Biome       string        `json:"biome"`
	Attributes  []string      `json:"attributes"`
	IsEdge      bool          `json:"isEdge"`
	AdjacentIDs []GeneratedID `json:"adjacentIds"`
}

type GeneratedMapVisuals struct {
	MapName     string                `json:"mapName"`
	ImageBase64 string                `json:"imageBase64"`
	Graphics    json.RawMessage       `json:"graphics"`
	Settings    json.RawMessage       `json:"settings"`
	ServerTiles []GeneratedServerTile `json:"serverTiles"`
	Offset      float64               `json:"offset"`
}

var generatedMapVisualsMu sync.RWMutex
var generatedMapVisuals = map[string]GeneratedMapVisuals{}

func StoreGeneratedMapVisuals(visuals GeneratedMapVisuals) {
	generatedMapVisualsMu.Lock()
	defer generatedMapVisualsMu.Unlock()

	generatedMapVisuals[visuals.MapName] = visuals
}

func GetGeneratedMapVisuals(mapName string) (GeneratedMapVisuals, bool) {
	generatedMapVisualsMu.RLock()
	defer generatedMapVisualsMu.RUnlock()

	visuals, ok := generatedMapVisuals[mapName]
	return visuals, ok
}
