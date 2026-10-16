package bot

import (
	"backend/internal/gamestate"
	"fmt"
	"math"
)

const relationWidth = 128
const relationHeads = 4

type RelationNorm struct {
	Weights []float64 `json:"weights"`
	Bias    []float64 `json:"bias"`
}
type RelationBlock struct {
	Norm1      RelationNorm `json:"norm1"`
	QKV        Layer        `json:"qkv"`
	Projection Layer        `json:"projection"`
	Norm2      RelationNorm `json:"norm2"`
	FF1        Layer        `json:"ff1"`
	FF2        Layer        `json:"ff2"`
}
type RelationalModel struct {
	State  []Layer         `json:"state"`
	Node   Layer           `json:"node"`
	Blocks []RelationBlock `json:"blocks"`
	Actor  []Layer         `json:"actor"`
}

func validateRelationLayer(l Layer, input, output int) error {
	if len(l.Weights) != output || len(l.Bias) != output {
		return fmt.Errorf("invalid relational layer dimensions")
	}
	for i, row := range l.Weights {
		if len(row) != input || !finite(l.Bias[i]) || math.Abs(l.Bias[i]) > 1e6 {
			return fmt.Errorf("invalid relational bias or width")
		}
		for _, v := range row {
			if !finite(v) || math.Abs(v) > 1e6 {
				return fmt.Errorf("invalid relational weight")
			}
		}
	}
	return nil
}
func validateRelationNorm(n RelationNorm) error {
	if len(n.Weights) != relationWidth || len(n.Bias) != relationWidth {
		return fmt.Errorf("invalid relational norm width")
	}
	for i, v := range n.Weights {
		if !finite(v) || !finite(n.Bias[i]) || math.Abs(v) > 1e6 || math.Abs(n.Bias[i]) > 1e6 {
			return fmt.Errorf("invalid relational norm")
		}
	}
	return nil
}
func (m *Model) validateRelational() error {
	r := m.Relational
	if r == nil || len(r.State) != 2 || len(r.Blocks) != 3 || len(r.Actor) != 3 {
		return fmt.Errorf("invalid relational architecture")
	}
	checks := []struct {
		l       Layer
		in, out int
	}{
		{r.State[0], m.StateFeatures, 256}, {r.State[1], 256, 128}, {r.Node, m.NodeFeatures, 128},
		{r.Actor[0], m.ActionFeatures + 5*128, 512}, {r.Actor[1], 512, 256}, {r.Actor[2], 256, 1},
	}
	for _, b := range r.Blocks {
		if err := validateRelationNorm(b.Norm1); err != nil {
			return err
		}
		if err := validateRelationNorm(b.Norm2); err != nil {
			return err
		}
		checks = append(checks, struct {
			l       Layer
			in, out int
		}{b.QKV, 128, 384}, struct {
			l       Layer
			in, out int
		}{b.Projection, 128, 128}, struct {
			l       Layer
			in, out int
		}{b.FF1, 128, 256}, struct {
			l       Layer
			in, out int
		}{b.FF2, 256, 128})
	}
	for _, c := range checks {
		if err := validateRelationLayer(c.l, c.in, c.out); err != nil {
			return err
		}
	}
	return nil
}
func relationLinear(l Layer, x []float64) []float64 {
	y := make([]float64, len(l.Bias))
	copy(y, l.Bias)
	for i, row := range l.Weights {
		for j, w := range row {
			y[i] += w * x[j]
		}
	}
	return y
}
func relationNorm(n RelationNorm, x []float64) []float64 {
	mean := 0.
	for _, v := range x {
		mean += v / float64(len(x))
	}
	variance := 0.
	for _, v := range x {
		variance += (v - mean) * (v - mean) / float64(len(x))
	}
	scale := 1 / math.Sqrt(variance+1e-5)
	y := make([]float64, len(x))
	for i, v := range x {
		y[i] = (v-mean)*scale*n.Weights[i] + n.Bias[i]
	}
	return y
}
func (r *RelationalModel) board(d gamestate.BotDecision) ([][]float64, []float64) {
	context := linearReLU(r.State[1], linearReLU(r.State[0], d.State))
	n := len(d.Graph.Nodes)
	h := make([][]float64, n)
	allowed := make([][]bool, n)
	for i := range allowed {
		allowed[i] = make([]bool, n)
		allowed[i][i] = true
	}
	for _, e := range d.Graph.Edges {
		allowed[e[0]][e[1]] = true
	}
	for i, node := range d.Graph.Nodes {
		h[i] = relationLinear(r.Node, node)
		for k := range h[i] {
			h[i][k] = math.Max(0, h[i][k]+context[k])
		}
	}
	for blockIndex, b := range r.Blocks {
		qkv := make([][]float64, n)
		for i := range h {
			qkv[i] = relationLinear(b.QKV, relationNorm(b.Norm1, h[i]))
		}
		next := make([][]float64, n)
		for i := range h {
			attended := make([]float64, 128)
			for head := 0; head < relationHeads; head++ {
				offset := head * 32
				scores := make([]float64, n)
				maximum := math.Inf(-1)
				for j := range h {
					if blockIndex < 2 && !allowed[i][j] {
						scores[j] = math.Inf(-1)
						continue
					}
					for k := 0; k < 32; k++ {
						scores[j] += qkv[i][offset+k] * qkv[j][128+offset+k]
					}
					scores[j] /= math.Sqrt(32)
					maximum = math.Max(maximum, scores[j])
				}
				total := 0.
				for j := range scores {
					scores[j] = math.Exp(scores[j] - maximum)
					total += scores[j]
				}
				for j := range h {
					for k := 0; k < 32; k++ {
						attended[offset+k] += scores[j] / total * qkv[j][256+offset+k]
					}
				}
			}
			next[i] = relationLinear(b.Projection, attended)
			for k := range next[i] {
				next[i][k] += h[i][k]
			}
			f := relationLinear(b.FF2, linearReLU(b.FF1, relationNorm(b.Norm2, next[i])))
			for k := range next[i] {
				next[i][k] += f[k]
			}
		}
		h = next
	}
	pooled := append([]float64(nil), context...)
	mean := make([]float64, 128)
	maximum := make([]float64, 128)
	for k := range maximum {
		maximum[k] = math.Inf(-1)
	}
	for _, row := range h {
		for k, v := range row {
			mean[k] += v / float64(n)
			maximum[k] = math.Max(maximum[k], v)
		}
	}
	pooled = append(pooled, mean...)
	pooled = append(pooled, maximum...)
	return h, pooled
}
func (m *Model) relationalScores(d gamestate.BotDecision, scores []float64) ([]float64, error) {
	r := m.Relational
	nodes, context := r.board(d)
	// Share the player and whole-board projection across all candidates.
	first := r.Actor[0]
	prefix := m.ActionFeatures + 2*relationWidth
	shared := make([]float64, len(first.Bias))
	copy(shared, first.Bias)
	for i, row := range first.Weights {
		for j, v := range context {
			shared[i] += row[prefix+j] * v
		}
	}
	for i, c := range d.Candidates {
		local := append([]float64(nil), c.Features...)
		for _, target := range []int{c.Target, c.Source} {
			if target < 0 {
				local = append(local, make([]float64, 128)...)
			} else {
				local = append(local, nodes[target]...)
			}
		}
		x := append([]float64(nil), shared...)
		for j, row := range first.Weights {
			for k, v := range local {
				x[j] += row[k] * v
			}
			x[j] = math.Max(0, x[j])
		}
		extra := relationLinear(r.Actor[2], linearReLU(r.Actor[1], x))[0]
		scores[i] += extra
		if !finite(scores[i]) {
			return nil, fmt.Errorf("non-finite relational output")
		}
	}
	return scores, nil
}
