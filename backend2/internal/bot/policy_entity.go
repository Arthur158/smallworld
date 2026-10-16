package bot

import (
	"backend/internal/gamestate"
	"fmt"
	"math"
)

const entityWidth = 256
const entityHeads = 8

type EntityBlock struct {
	Norm1    RelationNorm `json:"norm1"`
	Norm2    RelationNorm `json:"norm2"`
	QKV      Layer        `json:"qkv"`
	Out      Layer        `json:"out"`
	FF       []Layer      `json:"ff"`
	EdgeBias []float64    `json:"edge_bias"`
}
type EntityModel struct {
	State   Layer         `json:"state"`
	Tile    Layer         `json:"tile"`
	Player  Layer         `json:"player"`
	Stack   Layer         `json:"stack"`
	StackID [][]float64   `json:"stack_id"`
	Action  Layer         `json:"action"`
	Actor   []Layer       `json:"actor"`
	Blocks  []EntityBlock `json:"blocks"`
}

func entitySchema() (r, t, k, p, reserve, tileStack, actionStack int) {
	s := gamestate.BotV2Spec()
	r, t, k = len(s["races"].([]string)), len(s["traits"].([]string)), len(s["stacks"].([]string))
	p = 1 + 2*r + 3*t + k
	return r, t, k, p, 1 + r + 2*t, 23 + r + 2*t, gamestate.BotActionFeatures + len(s["action_kinds"].([]string))
}

func (m *Model) validateEntity() error {
	e := m.Entity
	_, _, k, p, _, _, _ := entitySchema()
	if e == nil || len(e.Blocks) != 6 || len(e.Actor) != 2 {
		return fmt.Errorf("invalid entity architecture")
	}
	checks := []struct {
		layer   Layer
		in, out int
	}{
		{e.State, m.StateFeatures, 256}, {e.Tile, m.NodeFeatures, 256}, {e.Player, p + 6, 256},
		{e.Stack, 14, 256}, {Layer{Weights: e.StackID, Bias: make([]float64, k)}, 256, k},
		{e.Action, m.ActionFeatures + 3*256, 256}, {e.Actor[0], 512, 256}, {e.Actor[1], 256, 1},
	}
	for _, b := range e.Blocks {
		if len(b.FF) != 2 || len(b.EdgeBias) != 8 {
			return fmt.Errorf("invalid entity block")
		}
		for _, v := range b.EdgeBias {
			if !finite(v) || math.Abs(v) > 1e6 {
				return fmt.Errorf("invalid entity edge bias")
			}
		}
		for _, norm := range []RelationNorm{b.Norm1, b.Norm2} {
			if len(norm.Weights) != 256 || len(norm.Bias) != 256 {
				return fmt.Errorf("invalid entity norm")
			}
			for i, v := range norm.Weights {
				if !finite(v) || !finite(norm.Bias[i]) || math.Abs(v) > 1e6 || math.Abs(norm.Bias[i]) > 1e6 {
					return fmt.Errorf("invalid entity norm weight")
				}
			}
		}
		checks = append(checks, struct {
			layer   Layer
			in, out int
		}{b.QKV, 256, 768}, struct {
			layer   Layer
			in, out int
		}{b.Out, 256, 256},
			struct {
				layer   Layer
				in, out int
			}{b.FF[0], 256, 1024}, struct {
				layer   Layer
				in, out int
			}{b.FF[1], 1024, 256})
	}
	for _, c := range checks {
		if err := validateRelationLayer(c.layer, c.in, c.out); err != nil {
			return err
		}
	}
	return nil
}

func entityGELU(x []float64) []float64 {
	for i, v := range x {
		x[i] = .5 * v * (1 + math.Erf(v/math.Sqrt2))
	}
	return x
}

func (e *EntityModel) encode(d gamestate.BotDecision) [][]float64 {
	_, _, k, p, reserve, tileStack, actionStack := entitySchema()
	n := len(d.Graph.Nodes)
	count := 1 + n + 6 + k
	x := make([][]float64, count)
	valid := make([]bool, count)
	x[0], valid[0] = relationLinear(e.State, d.State), true
	counts := make([][]float64, k)
	for i := range counts {
		counts[i] = make([]float64, 14)
	}
	for i, node := range d.Graph.Nodes {
		x[i+1], valid[i+1] = relationLinear(e.Tile, node), true
		for stack := 0; stack < k; stack++ {
			for owner := 0; owner < 8; owner++ {
				counts[stack][6+owner] += node[14+owner] * node[tileStack+stack]
			}
		}
	}
	for player := 0; player < 6; player++ {
		start := gamestate.BotStateFeatures + player*p
		features := append([]float64(nil), d.State[start:start+p]...)
		order := make([]float64, 6)
		order[player] = 1
		valid[n+1+player] = features[0] > 0
		for stack := 0; stack < k; stack++ {
			counts[stack][player] = features[reserve+stack] * features[0]
		}
		x[n+1+player] = relationLinear(e.Player, append(features, order...))
	}
	for stack := 0; stack < k; stack++ {
		i := n + 7 + stack
		x[i] = relationLinear(e.Stack, counts[stack])
		for j, v := range e.StackID[stack] {
			x[i][j] += v
		}
		for _, v := range counts[stack] {
			if v != 0 {
				valid[i] = true
			}
		}
		for _, c := range d.Candidates {
			if c.Features[actionStack+stack] > 0 {
				valid[i] = true
				break
			}
		}
	}
	for i := range x {
		if !valid[i] {
			x[i] = make([]float64, 256)
		}
	}
	edges := make([][]bool, count)
	for i := range edges {
		edges[i] = make([]bool, count)
	}
	for _, edge := range d.Graph.Edges {
		edges[edge[0]+1][edge[1]+1] = true
	}
	for _, b := range e.Blocks {
		qkv := make([][]float64, count)
		for i := range x {
			if valid[i] {
				qkv[i] = relationLinear(b.QKV, relationNorm(b.Norm1, x[i]))
			}
		}
		next := make([][]float64, count)
		for i := range x {
			if !valid[i] {
				next[i] = make([]float64, 256)
				continue
			}
			attended := make([]float64, 256)
			for head := 0; head < 8; head++ {
				offset := head * 32
				scores := make([]float64, count)
				maximum := math.Inf(-1)
				for j := range x {
					if !valid[j] {
						scores[j] = math.Inf(-1)
						continue
					}
					for z := 0; z < 32; z++ {
						scores[j] += qkv[i][offset+z] * qkv[j][256+offset+z]
					}
					scores[j] /= math.Sqrt(32)
					if edges[i][j] {
						scores[j] += b.EdgeBias[head]
					}
					maximum = math.Max(maximum, scores[j])
				}
				total := 0.
				for j := range scores {
					scores[j] = math.Exp(scores[j] - maximum)
					total += scores[j]
				}
				for j := range x {
					if valid[j] {
						for z := 0; z < 32; z++ {
							attended[offset+z] += scores[j] / total * qkv[j][512+offset+z]
						}
					}
				}
			}
			next[i] = relationLinear(b.Out, attended)
			for z := range next[i] {
				next[i][z] += x[i][z]
			}
			ff := relationLinear(b.FF[1], entityGELU(relationLinear(b.FF[0], relationNorm(b.Norm2, next[i]))))
			for z := range ff {
				next[i][z] += ff[z]
			}
		}
		x = next
	}
	return x
}

func (m *Model) entityScores(d gamestate.BotDecision, scores []float64) ([]float64, error) {
	e := m.Entity
	x := e.encode(d)
	n := len(d.Graph.Nodes)
	_, _, k, _, _, _, actionStack := entitySchema()
	shared := append([]float64(nil), e.Actor[0].Bias...)
	for i, row := range e.Actor[0].Weights {
		for j, v := range x[0] {
			shared[i] += row[256+j] * v
		}
	}
	for i, c := range d.Candidates {
		features := append([]float64(nil), c.Features...)
		for _, target := range []int{c.Target, c.Source} {
			if target < 0 {
				features = append(features, make([]float64, 256)...)
			} else {
				features = append(features, x[target+1]...)
			}
		}
		stack, mass := 0, 0.
		for j := 0; j < k; j++ {
			mass += c.Features[actionStack+j]
			if c.Features[actionStack+j] > c.Features[actionStack+stack] {
				stack = j
			}
		}
		piece := make([]float64, 256)
		for j, v := range x[n+7+stack] {
			piece[j] = mass * v
		}
		query := entityGELU(relationLinear(e.Action, append(features, piece...)))
		hidden := append([]float64(nil), shared...)
		for j, row := range e.Actor[0].Weights {
			for z, v := range query {
				hidden[j] += row[z] * v
			}
		}
		scores[i] += relationLinear(e.Actor[1], entityGELU(hidden))[0]
		if !finite(scores[i]) {
			return nil, fmt.Errorf("non-finite entity score")
		}
	}
	return scores, nil
}
