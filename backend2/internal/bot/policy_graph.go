package bot

import (
	"backend/internal/gamestate"
	"fmt"
	"math"
)

const graphWidth = 64

func validateGraphLayers(layers []Layer, n int) error {
	if len(layers) != 3 {
		return fmt.Errorf("expected three graph layers")
	}
	for i, l := range layers {
		width := 2 * graphWidth
		if i == 0 {
			width = n
		}
		if len(l.Bias) != graphWidth || len(l.Weights) != graphWidth {
			return fmt.Errorf("invalid graph layer dimensions")
		}
		for j, row := range l.Weights {
			if len(row) != width || !finite(l.Bias[j]) || math.Abs(l.Bias[j]) > 1e6 {
				return fmt.Errorf("invalid graph layer")
			}
			for _, w := range row {
				if !finite(w) || math.Abs(w) > 1e6 {
					return fmt.Errorf("invalid graph weight")
				}
			}
		}
	}
	return nil
}
func linearReLU(l Layer, x []float64) []float64 {
	y := make([]float64, len(l.Bias))
	for i, row := range l.Weights {
		y[i] = l.Bias[i]
		for j, w := range row {
			y[i] += w * x[j]
		}
		y[i] = math.Max(0, y[i])
	}
	return y
}
func (m *Model) graphEmbeddings(d gamestate.BotDecision) ([][]float64, []float64, error) {
	g := d.Graph
	if g == nil || len(g.Nodes) < 1 || len(g.Nodes) > 160 {
		return nil, nil, fmt.Errorf("invalid board graph")
	}
	h := make([][]float64, len(g.Nodes))
	neighbors := make([][]int, len(g.Nodes))
	for i, node := range g.Nodes {
		if len(node) != m.NodeFeatures {
			return nil, nil, fmt.Errorf("invalid graph features")
		}
		for _, v := range node {
			if !finite(v) {
				return nil, nil, fmt.Errorf("non-finite graph input")
			}
		}
		h[i] = linearReLU(m.GraphLayers[0], node)
	}
	for _, edge := range g.Edges {
		i, j := edge[0], edge[1]
		if i < 0 || j < 0 || i >= len(h) || j >= len(h) {
			return nil, nil, fmt.Errorf("invalid graph edge")
		}
		neighbors[i] = append(neighbors[i], j)
	}
	for _, layer := range m.GraphLayers[1:] {
		next := make([][]float64, len(h))
		for i := range h {
			x := make([]float64, 2*graphWidth)
			copy(x, h[i])
			if len(neighbors[i]) > 0 {
				for _, j := range neighbors[i] {
					for k, v := range h[j] {
						x[graphWidth+k] += v / float64(len(neighbors[i]))
					}
				}
			}
			next[i] = linearReLU(layer, x)
		}
		h = next
	}
	pooled := make([]float64, graphWidth)
	for _, node := range h {
		for k, v := range node {
			pooled[k] += v / float64(len(h))
		}
	}
	return h, pooled, nil
}
func (r *Registry) Schema(id string) string {
	if r != nil && r.models[id] != nil {
		return r.models[id].Schema
	}
	return gamestate.BotSchema
}
