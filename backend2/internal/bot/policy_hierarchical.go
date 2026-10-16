package bot

import (
	"backend/internal/gamestate"
	"fmt"
	"math"
)

// This is an architecture partition only: each action kind/piece identity has
// the same learned scorer. No move values or piece-specific bonuses are used.
var groupFeatures = gamestate.BotV2GroupFeatures()

func (m *Model) validateGroupLayers() error {
	if len(m.GroupLayers) != 2 {
		return fmt.Errorf("expected two group layers")
	}
	width := m.StateFeatures + 2*m.graphWidth() + groupFeatures
	for i, layer := range m.GroupLayers {
		output := 128
		if i == 1 {
			output = 1
		}
		if len(layer.Bias) != output || len(layer.Weights) != output {
			return fmt.Errorf("invalid group dimensions")
		}
		for j, row := range layer.Weights {
			if len(row) != width || !finite(layer.Bias[j]) || math.Abs(layer.Bias[j]) > 1e6 {
				return fmt.Errorf("invalid group layer")
			}
			for _, value := range row {
				if !finite(value) || math.Abs(value) > 1e6 {
					return fmt.Errorf("invalid group weight")
				}
			}
		}
		width = output
	}
	return nil
}

func (m *Model) hierarchicalScores(d gamestate.BotDecision, pooled, targets []float64) ([]float64, error) {
	type group struct{ max, sum, score float64 }
	groups := map[string]*group{}
	keys := make([]string, len(d.Candidates))
	for i, candidate := range d.Candidates {
		key := candidate.Action.Kind + "\x00" + candidate.Action.Stack
		keys[i] = key
		g := groups[key]
		if g == nil {
			x := append(append([]float64(nil), d.State...), pooled...)
			x = append(x, candidate.Features[gamestate.BotActionFeatures:gamestate.BotActionFeatures+groupFeatures]...)
			h := linearReLU(m.GroupLayers[0], x)
			score := m.GroupLayers[1].Bias[0]
			for j, v := range h {
				score += m.GroupLayers[1].Weights[0][j] * v
			}
			if !finite(score) {
				return nil, fmt.Errorf("non-finite group output")
			}
			g = &group{max: targets[i], score: score}
			groups[key] = g
		}
		g.max = math.Max(g.max, targets[i])
	}
	for i, value := range targets {
		g := groups[keys[i]]
		g.sum += math.Exp(value - g.max)
	}
	for i, value := range targets {
		g := groups[keys[i]]
		if m.FormatVersion >= 4 {
			targets[i] = value + g.score
		} else {
			targets[i] = value - g.max - math.Log(g.sum) + g.score
		}
	}
	return targets, nil
}

// Decode the action/piece group first, then its learned best destination.
// Comparing joint probabilities directly would favor singleton actions over
// a group whose probability is spread across many tiles.
func selectHierarchical(d gamestate.BotDecision, scores []float64) int {
	type group struct {
		max, sum float64
		best     int
	}
	groups := map[string]*group{}
	order := []string{}
	for i, c := range d.Candidates {
		key := c.Action.Kind + "\x00" + c.Action.Stack
		g := groups[key]
		if g == nil {
			g = &group{max: scores[i], best: i}
			groups[key] = g
			order = append(order, key)
		}
		if scores[i] > g.max {
			g.max = scores[i]
			g.best = i
		}
	}
	for i, c := range d.Candidates {
		g := groups[c.Action.Kind+"\x00"+c.Action.Stack]
		g.sum += math.Exp(scores[i] - g.max)
	}
	best := 0
	maximum := math.Inf(-1)
	for _, key := range order {
		g := groups[key]
		score := g.max + math.Log(g.sum)
		if score > maximum {
			maximum = score
			best = g.best
		}
	}
	return best
}
