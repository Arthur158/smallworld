// Package bot runs the same candidate-scoring policy exported by smallworld-export.
// Go inference is always available; workers may enable a local PyTorch CPU child.
package bot

import (
	"backend/internal/gamestate"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"regexp"
	"slices"
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
	Specialists      json.RawMessage  `json:"specialists,omitempty"`
	SpecialistRaces  []string         `json:"specialist_races,omitempty"`
	SpecialistTraits []string         `json:"specialist_traits,omitempty"`
	Entity           *EntityModel     `json:"entity,omitempty"`
	Relational       *RelationalModel `json:"relational,omitempty"`
	Info
	FormatVersion  int     `json:"format_version"`
	Schema         string  `json:"schema"`
	StateFeatures  int     `json:"state_features"`
	ActionFeatures int     `json:"action_features"`
	Layers         []Layer `json:"layers"`
	NodeFeatures   int     `json:"node_features,omitempty"`
	GraphLayers    []Layer `json:"graph_layers,omitempty"`
	GroupLayers    []Layer `json:"group_layers,omitempty"`
}

type Registry struct {
	models  map[string]*Model
	sources map[string]string
	digests map[string]string
	python  *pythonEvaluator
}

func NewRegistry(dir string) (*Registry, error) {
	r := &Registry{models: make(map[string]*Model), sources: make(map[string]string), digests: make(map[string]string)}
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
		hash := sha256.New()
		m, err := ReadModel(io.TeeReader(f, hash))
		f.Close()
		if err != nil {
			return nil, fmt.Errorf("model %s: %w", entry.Name(), err)
		}
		if r.Has(m.ID) {
			return nil, fmt.Errorf("duplicate or reserved model id %q", m.ID)
		}
		r.models[m.ID] = m
		r.sources[entry.Name()] = m.ID
		r.digests[m.ID] = fmt.Sprintf("%x", hash.Sum(nil))
	}
	return r, nil
}

func ReadModel(reader io.Reader) (*Model, error) {
	data, err := io.ReadAll(io.LimitReader(reader, (384<<20)+1))
	if err != nil {
		return nil, err
	}
	if len(data) > 384<<20 {
		return nil, fmt.Errorf("model exceeds 384 MiB")
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
	if m.FormatVersion == 7 {
		if len(m.Specialists) == 0 || !slices.Equal(m.SpecialistRaces, gamestate.BotTrainingRaces()) || !slices.Equal(m.SpecialistTraits, gamestate.BotTrainingTraits()) {
			return fmt.Errorf("invalid specialist export vocabulary")
		}
		// Tensor shapes and finite values are checked by the mandatory Python
		// evaluator before readiness. Validate the inherited format-6 body here.
		base := *m
		base.FormatVersion = 6
		return base.Validate()
	}
	if !modelID.MatchString(m.ID) || m.Name == "" || len(m.Name) > 80 || len(m.Description) > 300 {
		return fmt.Errorf("invalid model identity")
	}
	legacy := m.FormatVersion == 1 && m.Schema == gamestate.BotSchema && m.StateFeatures == gamestate.BotStateFeatures && m.ActionFeatures == gamestate.BotActionFeatures && len(m.GraphLayers) == 0
	full := (m.FormatVersion >= 2 && m.FormatVersion <= 6) && m.Schema == gamestate.BotSchemaV2 && m.StateFeatures == gamestate.BotV2StateWidth() && m.ActionFeatures == gamestate.BotV2ActionWidth() && m.NodeFeatures == gamestate.BotV2NodeWidth()
	if full {
		if err := validateGraphLayers(m.GraphLayers, m.NodeFeatures, m.FormatVersion); err != nil {
			return err
		}
	}
	if m.FormatVersion >= 3 && m.FormatVersion <= 6 {
		if err := m.validateGroupLayers(); err != nil {
			return err
		}
	} else if len(m.GroupLayers) != 0 {
		return fmt.Errorf("unexpected group layers")
	}
	if m.FormatVersion >= 5 && m.FormatVersion <= 6 {
		if err := m.validateRelational(); err != nil {
			return err
		}
	} else if m.Relational != nil {
		return fmt.Errorf("unexpected relational layers")
	}
	if m.FormatVersion == 6 {
		if err := m.validateEntity(); err != nil {
			return err
		}
	} else if m.Entity != nil {
		return fmt.Errorf("unexpected entity layers")
	}
	if !legacy && !full {
		return fmt.Errorf("model schema or feature dimensions do not match this server")
	}
	if len(m.Layers) != 3 {
		return fmt.Errorf("expected three actor layers")
	}
	width := m.StateFeatures + m.ActionFeatures
	if full {
		width += 3 * m.graphWidth()
		if m.FormatVersion >= 3 && m.FormatVersion <= 6 {
			width += m.graphWidth()
		}
	}
	maxHidden := 256
	if m.FormatVersion >= 3 && m.FormatVersion <= 6 {
		maxHidden = 384
	}
	for i, l := range m.Layers {
		if len(l.Bias) < 1 || len(l.Bias) > maxHidden || len(l.Weights) != len(l.Bias) {
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
	if m.FormatVersion == 7 {
		return nil, fmt.Errorf("specialist models require the PyTorch CPU evaluator")
	}
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
	// The state is shared by every candidate. Reuse its projection without
	// changing the arithmetic order of the exported actor's first layer.
	shared := make([]float64, len(m.Layers[0].Bias))
	for j, row := range m.Layers[0].Weights {
		shared[j] = m.Layers[0].Bias[j]
		for n, v := range d.State {
			shared[j] += row[n] * v
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
					x = append(x, make([]float64, m.graphWidth())...)
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
				start := 0
				if k == 0 {
					y[j] = shared[j]
					start = m.StateFeatures
				}
				for n := start; n < len(row); n++ {
					y[j] += row[n] * x[n]
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
	if m.FormatVersion >= 3 && m.FormatVersion <= 6 {
		var err error
		scores, err = m.hierarchicalScores(d, pooled, scores)
		if err != nil {
			return nil, err
		}
		if m.FormatVersion >= 5 && m.FormatVersion <= 6 {
			scores, err = m.relationalScores(d, scores)
			if err != nil {
				return nil, err
			}
			if m.FormatVersion == 6 {
				return m.entityScores(d, scores)
			}
			return scores, nil
		}
		return scores, nil
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
		scores, err = r.Scores(id, d)
		if err != nil {
			return 0, err
		}
		if r.models[id].FormatVersion == 3 {
			return selectHierarchical(d, scores), nil
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
	if r.python != nil {
		if scores, err := r.python.scores(id, d); err == nil {
			return scores, nil
		} else if r.models[id].FormatVersion == 7 {
			return nil, fmt.Errorf("specialist evaluator: %w", err)
		}
	}
	return r.models[id].Scores(d)
}
