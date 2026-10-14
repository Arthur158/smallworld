package bot

import (
	"backend/internal/gamestate"
	"bytes"
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func testModel() Model {
	m := Model{Info: Info{ID: "test-v1", Name: "Test"}, FormatVersion: 1, Schema: gamestate.BotSchema,
		StateFeatures: gamestate.BotStateFeatures, ActionFeatures: gamestate.BotActionFeatures}
	width := m.StateFeatures + m.ActionFeatures
	for _, outputs := range []int{8, 8, 1} {
		layer := Layer{Weights: make([][]float64, outputs), Bias: make([]float64, outputs)}
		for i := range layer.Weights {
			layer.Weights[i] = make([]float64, width)
		}
		m.Layers = append(m.Layers, layer)
		width = outputs
	}
	return m
}

func TestModelRejectsIncompatibleAndInvalidWeights(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Model)
	}{
		{"schema", func(m *Model) { m.Schema = "future-v2" }},
		{"version", func(m *Model) { m.FormatVersion = 2 }},
		{"path id", func(m *Model) { m.ID = "../model" }},
		{"missing layer", func(m *Model) { m.Layers = m.Layers[:2] }},
		{"row width", func(m *Model) { m.Layers[0].Weights[0] = nil }},
		{"nonfinite", func(m *Model) { m.Layers[0].Weights[0][0] = math.NaN() }},
		{"overflow bias", func(m *Model) { m.Layers[0].Bias[0] = 1e308 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := testModel()
			tc.mutate(&m)
			if m.Validate() == nil {
				t.Fatal("invalid model accepted")
			}
		})
	}
	if _, err := ReadModel(bytes.NewReader(make([]byte, (8<<20)+1))); err == nil {
		t.Fatal("oversized model accepted")
	}
}

func TestCatalogRejectsReservedAndDuplicateIDs(t *testing.T) {
	dir := t.TempDir()
	write := func(name string, m Model) {
		data, err := json.Marshal(m)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	m := testModel()
	write("first.json", m)
	r, err := NewRegistry(dir)
	if err != nil || len(r.Catalog()) != 3 || !r.Has("test-v1") {
		t.Fatalf("valid catalog failed: %v", err)
	}
	write("second.json", m)
	if _, err := NewRegistry(dir); err == nil {
		t.Fatal("duplicate model accepted")
	}
	m.ID = "random"
	write("second.json", m)
	if _, err := NewRegistry(dir); err == nil {
		t.Fatal("builtin override accepted")
	}
}
