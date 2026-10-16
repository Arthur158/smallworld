package main

import (
	"backend/internal/gamestate"
	"encoding/binary"
	"math"
)

// Transport optimization only. The trainer already uses float32 tensors; send
// their exact little-endian representation rather than parsing thousands of
// JSON numbers for every game action. Normal botenv clients keep readable JSON.
type packedDecision struct {
	State       []byte `json:"state"`
	Candidates  []byte `json:"candidates"`
	Nodes       []byte `json:"nodes"`
	StateWidth  int    `json:"state_width"`
	ActionWidth int    `json:"action_width"`
	NodeWidth   int    `json:"node_width"`
	NodeCount   int    `json:"node_count"`
}

func appendFloats(dst []byte, values []float64) []byte {
	for _, value := range values {
		dst = binary.LittleEndian.AppendUint32(dst, math.Float32bits(float32(value)))
	}
	return dst
}
func packDecision(d *gamestate.BotDecision) *packedDecision {
	p := &packedDecision{StateWidth: len(d.State), State: appendFloats(nil, d.State)}
	d.State = nil
	if len(d.Candidates) > 0 {
		p.ActionWidth = len(d.Candidates[0].Features)
		p.Candidates = make([]byte, 0, len(d.Candidates)*p.ActionWidth*4)
	}
	for i := range d.Candidates {
		p.Candidates = appendFloats(p.Candidates, d.Candidates[i].Features)
		d.Candidates[i].Features = nil
	}
	if d.Graph != nil && len(d.Graph.Nodes) > 0 {
		p.NodeWidth = len(d.Graph.Nodes[0])
		p.NodeCount = len(d.Graph.Nodes)
		p.Nodes = make([]byte, 0, p.NodeWidth*p.NodeCount*4)
		for _, node := range d.Graph.Nodes {
			p.Nodes = appendFloats(p.Nodes, node)
		}
		d.Graph.Nodes = nil
	}
	return p
}
