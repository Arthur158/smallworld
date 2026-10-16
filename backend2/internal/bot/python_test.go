package bot

import (
	"backend/internal/gamestate"
	"encoding/json"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"
)

func TestPythonHelperProcess(t *testing.T) {
	mode := os.Getenv("SMALLWORLD_EVALUATOR_TEST")
	if mode == "" {
		return
	}
	encoder := json.NewEncoder(os.Stdout)
	digest := "sha"
	if mode == "wrongweights" {
		digest = "different"
	}
	encoder.Encode(evaluatorReply{Ready: true, Protocol: 1, Models: map[string]string{"test-v1": digest}})
	decoder := json.NewDecoder(os.Stdin)
	for {
		var req struct {
			ID uint64 `json:"id"`
		}
		if decoder.Decode(&req) != nil {
			os.Exit(0)
		}
		reply := evaluatorReply{ID: req.ID, Scores: []float64{0, 3}}
		switch mode {
		case "exit":
			os.Exit(2)
		case "hang":
			time.Sleep(time.Hour)
		case "wrongid":
			reply.ID++
		case "wrongsize":
			reply.Scores = []float64{1}
		case "error":
			reply.Error = "invalid request"
		}
		encoder.Encode(reply)
	}
}

func testEvaluator(mode string) *pythonEvaluator {
	return &pythonEvaluator{expected: map[string]string{"test-v1": "sha"}, startupTimeout: 3 * time.Second, timeout: 100 * time.Millisecond,
		command: func() *exec.Cmd {
			c := exec.Command(os.Args[0], "-test.run=^TestPythonHelperProcess$")
			c.Env = append(os.Environ(), "SMALLWORLD_EVALUATOR_TEST="+mode)
			return c
		}}
}
func testDecision() gamestate.BotDecision {
	return gamestate.BotDecision{Schema: gamestate.BotSchema, State: make([]float64, gamestate.BotStateFeatures), Candidates: []gamestate.BotCandidate{
		{Features: make([]float64, gamestate.BotActionFeatures)}, {Features: make([]float64, gamestate.BotActionFeatures)}}}
}

func TestPersistentEvaluatorReuseSerializationAndClose(t *testing.T) {
	p := testEvaluator("ok")
	if err := p.start(); err != nil {
		t.Fatal(err)
	}
	command := p.cmd
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			scores, err := p.scores("test-v1", testDecision())
			if err != nil || len(scores) != 2 || scores[1] != 3 {
				t.Errorf("bad reply: %v %v", scores, err)
			}
		}()
	}
	wg.Wait()
	if p.cmd != command || p.requestID != 10 {
		t.Fatal("did not reuse process")
	}
	p.Close()
	if command.ProcessState == nil {
		t.Fatal("child not reaped")
	}
	if _, err := p.scores("test-v1", testDecision()); err == nil {
		t.Fatal("closed evaluator restarted")
	}
}

func TestPersistentEvaluatorFailureFallsBackAndRecovers(t *testing.T) {
	for _, mode := range []string{"exit", "hang", "wrongid", "wrongsize", "error"} {
		t.Run(mode, func(t *testing.T) {
			p := testEvaluator(mode)
			defer p.Close()
			m := testModel()
			r := &Registry{models: map[string]*Model{m.ID: &m}, python: p}
			started := time.Now()
			action, err := r.Select(m.ID, testDecision(), nil)
			if err != nil || action != 0 {
				t.Fatalf("Go fallback failed: %v %d", err, action)
			}
			if time.Since(started) > 2*time.Second {
				t.Fatal("failed request did not time out")
			}
			if p.cmd != nil {
				t.Fatal("failed child not reaped")
			}
			p.command = testEvaluator("ok").command
			p.retryAfter = time.Time{}
			action, err = r.Select(m.ID, testDecision(), nil)
			if err != nil || action != 1 {
				t.Fatalf("CPU evaluator did not recover: %v %d", err, action)
			}
		})
	}
	p := testEvaluator("wrongweights")
	defer p.Close()
	if err := p.start(); err == nil {
		t.Fatal("accepted different model weights")
	}
}
