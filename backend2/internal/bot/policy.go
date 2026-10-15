// Package bot runs the same candidate-scoring policy exported by smallworld-export.
// Production inference needs neither Python, network access nor a GPU.
package bot

import (
	"backend/internal/gamestate"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

type Info struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Layer struct {
	Weights [][]float64 `json:"weights"`
	Bias    []float64   `json:"bias"`
}

type Model struct {
	Info
	FormatVersion  int     `json:"format_version"`
	Schema         string  `json:"schema"`
	StateFeatures  int     `json:"state_features"`
	ActionFeatures int     `json:"action_features"`
	Layers         []Layer `json:"layers"`
	NodeFeatures   int     `json:"node_features,omitempty"`
	GraphLayers    []Layer `json:"graph_layers,omitempty"`
}

type Registry struct{ models map[string]*Model }

func NewRegistry(dir string) (*Registry, error) {
	r := &Registry{models: make(map[string]*Model)}
	if dir == "" {
		return r, nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("bot model directory: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		f, err := os.Open(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		m, err := ReadModel(f)
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("model %s: %w", entry.Name(), err)
		}
		if r.Has(m.ID) {
			return nil, fmt.Errorf("duplicate or reserved model id %q", m.ID)
		}
		r.models[m.ID] = m
	}
	return r, nil
}

func ReadModel(reader io.Reader) (*Model, error) {
	data, err := io.ReadAll(io.LimitReader(reader, (32<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 32<<20 {
		return nil, fmt.Errorf("model exceeds 32 MiB")
	}
	var m Model
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	if err := m.Validate(); err != nil {
		return nil, err
	}
	return &m, nil
}

var modelID = regexp.MustCompile(`^[a-z0-9][a-z0-9_-]{0,63}$`)

func (m *Model) Validate() error {
	if !modelID.MatchString(m.ID) || m.Name == "" || len(m.Name) > 80 || len(m.Description) > 300 {
		return fmt.Errorf("invalid model identity")
	}
	legacy := m.FormatVersion == 1 && m.Schema == gamestate.BotSchema && m.StateFeatures == gamestate.BotStateFeatures && m.ActionFeatures == gamestate.BotActionFeatures && len(m.GraphLayers) == 0
	full := m.FormatVersion == 2 && m.Schema == gamestate.BotSchemaV2 && m.StateFeatures == gamestate.BotV2StateWidth() && m.ActionFeatures == gamestate.BotV2ActionWidth() && m.NodeFeatures == gamestate.BotV2NodeWidth()
	if full {
		if err := validateGraphLayers(m.GraphLayers, m.NodeFeatures); err != nil {
			return err
		}
	}
	if !legacy && !full {
		return fmt.Errorf("model schema or feature dimensions do not match this server")
	}
	if len(m.Layers) != 3 {
		return fmt.Errorf("expected three actor layers")
	}
	width := m.StateFeatures + m.ActionFeatures
	if full {
		width += 3 * graphWidth
	}
	for i, l := range m.Layers {
		if len(l.Bias) < 1 || len(l.Bias) > 256 || len(l.Weights) != len(l.Bias) {
			return fmt.Errorf("invalid layer %d dimensions", i)
		}
		for j, row := range l.Weights {
			if len(row) != width || !finite(l.Bias[j]) || math.Abs(l.Bias[j]) > 1e6 {
				return fmt.Errorf("invalid layer %d row %d", i, j)
			}
			for _, x := range row {
				if !finite(x) || math.Abs(x) > 1e6 {
					return fmt.Errorf("invalid model weight")
				}
			}
		}
		width = len(l.Bias)
	}
	if width != 1 {
		return fmt.Errorf("actor must output one score per candidate")
	}
	return nil
}

func finite(x float64) bool { return !math.IsNaN(x) && !math.IsInf(x, 0) }

func (r *Registry) Has(id string) bool {
	if id == "random" || id == "greedy" {
		return true
	}
	return r != nil && r.models[id] != nil
}

func (r *Registry) Catalog() []Info {
	result := []Info{
		{ID: "random", Name: "Random bot", Description: "Chooses randomly from available moves. A baseline opponent."},
		{ID: "greedy", Name: "Greedy bot", Description: "Prefers cheap conquests and larger armies. A scripted benchmark."},
	}
	if r != nil {
		ids := make([]string, 0, len(r.models))
		for id := range r.models {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		for _, id := range ids {
			result = append(result, r.models[id].Info)
		}
	}
	return result
}

func (m *Model) Scores(d gamestate.BotDecision) ([]float64, error) {
	if d.Schema != m.Schema || len(d.State) != m.StateFeatures {
		return nil, fmt.Errorf("incompatible decision")
	}
	scores := make([]float64, len(d.Candidates))
	var nodes [][]float64
	var pooled []float64
	if m.Schema == gamestate.BotSchemaV2 {
		var err error
		nodes, pooled, err = m.graphEmbeddings(d)
		if err != nil {
			return nil, err
		}
	}
	for i, c := range d.Candidates {
		if len(c.Features) != m.ActionFeatures {
			return nil, fmt.Errorf("incompatible action features")
		}
		x := append(append([]float64(nil), d.State...), c.Features...)
		if nodes != nil {
			for _, target := range []int{c.Target, c.Source} {
				if target < -1 || target >= len(nodes) {
					return nil, fmt.Errorf("invalid graph action target")
				}
				if target == -1 {
					x = append(x, make([]float64, graphWidth)...)
				} else {
					x = append(x, nodes[target]...)
				}
			}
			x = append(x, pooled...)
		}
		for _, value := range x {
			if !finite(value) {
				return nil, fmt.Errorf("non-finite input")
			}
		}
		for k, l := range m.Layers {
			y := make([]float64, len(l.Bias))
			for j, row := range l.Weights {
				y[j] = l.Bias[j]
				for n, w := range row {
					y[j] += w * x[n]
				}
				if k < len(m.Layers)-1 {
					y[j] = math.Max(y[j], 0)
				}
			}
			x = y
		}
		if !finite(x[0]) {
			return nil, fmt.Errorf("non-finite actor output")
		}
		scores[i] = x[0]
	}
	return scores, nil
}

// Select never receives an arbitrary model URL/path from a room participant.
func (r *Registry) Select(id string, d gamestate.BotDecision, rng *rand.Rand) (int, error) {
	if len(d.Candidates) == 0 {
		return 0, fmt.Errorf("no bot moves available")
	}
	if !r.Has(id) {
		return 0, fmt.Errorf("unknown bot model %q", id)
	}
	if id == "random" {
		if rng != nil {
			return rng.Intn(len(d.Candidates)), nil
		}
		return rand.Intn(len(d.Candidates)), nil
	}
	scores := make([]float64, len(d.Candidates))
	if id == "greedy" {
		for i, c := range d.Candidates {
			switch c.Action.Kind {
			case "pick":
				scores[i] = c.Features[10]*20 + c.Features[9]*10 - c.Features[8]*5
			case "conquer":
				scores[i] = 10 - c.Features[7]*20
			case "redeploy":
				scores[i] = 2 - c.Features[52]
			case "finish", "redeploy_start":
				scores[i] = -5
			case "decline":
				scores[i] = -6
				if d.State[10]*20 < 3 {
					scores[i] = 5
				}
			case "abandon":
				scores[i] = -20
			}
		}
	} else {
		var err error
		scores, err = r.models[id].Scores(d)
		if err != nil {
			return 0, err
		}
	}
	best := 0
	for i := 1; i < len(scores); i++ {
		if scores[i] > scores[best] {
			best = i
		}
	}
	return best, nil
}

// Scores exposes the exported actor's logits for cross-runtime parity checks.
func (r *Registry) Scores(id string, d gamestate.BotDecision) ([]float64, error) {
	if r == nil || r.models[id] == nil {
		return nil, fmt.Errorf("no exported model %q", id)
	}
	return r.models[id].Scores(d)
}
