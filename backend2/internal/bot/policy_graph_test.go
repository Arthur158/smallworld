package bot

import (
	"backend/internal/gamestate"
	"testing"
)

func TestGraphRepeatedConnectionsDoNotReweightNeighbors(t *testing.T) {
	for _, version := range []int{2, 4, 6} {
		m := Model{FormatVersion: version, NodeFeatures: 1}
		width := m.graphWidth()
		input := Layer{Bias: make([]float64, width), Weights: make([][]float64, width)}
		neighbor := Layer{Bias: make([]float64, width), Weights: make([][]float64, width)}
		for i := 0; i < width; i++ {
			input.Weights[i] = make([]float64, 1)
			neighbor.Weights[i] = make([]float64, 2*width)
		}
		input.Weights[0][0] = 1
		neighbor.Weights[0][width] = 1
		m.GraphLayers = []Layer{input, neighbor}
		d := gamestate.BotDecision{Graph: &gamestate.BotGraph{
			Nodes: [][]float64{{0}, {2}, {6}},
			Edges: [][2]int{{0, 1}, {0, 2}, {0, 2}},
		}}
		nodes, _, err := m.graphEmbeddings(d)
		if err != nil || nodes[0][0] != 4 {
			t.Fatalf("format %d: repeated connection changed equal-neighbor average: %v %v", version, nodes, err)
		}
	}
}
