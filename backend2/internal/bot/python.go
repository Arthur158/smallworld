package bot

import (
	"backend/internal/gamestate"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// Each registry owns a persistent local child. Requests are pure inference;
// game mutation and format-3 group selection remain on the Go worker thread.
type pythonEvaluator struct {
	mu                      sync.Mutex
	command                 func() *exec.Cmd
	expected                map[string]string
	cmd                     *exec.Cmd
	input                   io.WriteCloser
	output                  *json.Decoder
	requestID               uint64
	closed                  bool
	retryAfter              time.Time
	startupTimeout, timeout time.Duration
}

type evaluatorReply struct {
	ID       uint64            `json:"id"`
	Ready    bool              `json:"ready"`
	Protocol int               `json:"protocol"`
	Models   map[string]string `json:"models"`
	Scores   []float64         `json:"scores"`
	Error    string            `json:"error"`
}

type evaluatorResult struct {
	reply evaluatorReply
	err   error
}

func (p *pythonEvaluator) exchange(request interface{}, timeout time.Duration) (evaluatorReply, error) {
	input, output := p.input, p.output
	done := make(chan evaluatorResult, 1)
	go func() {
		var reply evaluatorReply
		var err error
		if request != nil {
			err = json.NewEncoder(input).Encode(request)
		}
		if err == nil {
			err = output.Decode(&reply)
		}
		done <- evaluatorResult{reply, err}
	}()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case result := <-done:
		return result.reply, result.err
	case <-timer.C:
		p.stop()
		return evaluatorReply{}, fmt.Errorf("CPU evaluator timed out after %s", timeout)
	}
}

func (p *pythonEvaluator) start() error {
	if p.closed {
		return fmt.Errorf("CPU evaluator is closed")
	}
	cmd := p.command()
	cmd.Stderr = os.Stderr
	input, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	output, err := cmd.StdoutPipe()
	if err != nil {
		input.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		input.Close()
		output.Close()
		return err
	}
	p.cmd, p.input, p.output = cmd, input, json.NewDecoder(output)
	ready, err := p.exchange(nil, p.startupTimeout)
	if err == nil && (!ready.Ready || ready.Protocol != 1 || len(ready.Models) != len(p.expected)) {
		err = fmt.Errorf("CPU evaluator handshake mismatch")
	}
	if err == nil {
		for id, digest := range p.expected {
			if ready.Models[id] != digest {
				err = fmt.Errorf("CPU evaluator weights differ for %s", id)
				break
			}
		}
	}
	if err != nil {
		p.stop()
		return err
	}
	log.Printf("bot: persistent PyTorch CPU evaluator ready (%d models)", len(p.expected))
	return nil
}

func (p *pythonEvaluator) stop() {
	if p.cmd != nil {
		p.input.Close()
		_ = p.cmd.Process.Kill()
		_ = p.cmd.Wait()
		p.cmd, p.input, p.output = nil, nil, nil
	}
}

func (p *pythonEvaluator) scores(id string, d gamestate.BotDecision) ([]float64, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return nil, fmt.Errorf("CPU evaluator is closed")
	}
	if p.cmd == nil {
		if time.Now().Before(p.retryAfter) {
			return nil, fmt.Errorf("CPU evaluator restart pending")
		}
		if err := p.start(); err != nil {
			p.retryAfter = time.Now().Add(30 * time.Second)
			log.Printf("bot: CPU evaluator restart failed; using Go temporarily: %v", err)
			return nil, err
		}
	}
	p.requestID++
	request := struct {
		ID       uint64                `json:"id"`
		Model    string                `json:"model"`
		Decision gamestate.BotDecision `json:"decision"`
	}{p.requestID, id, d}
	reply, err := p.exchange(request, p.timeout)
	if err == nil && reply.Error != "" {
		err = fmt.Errorf("CPU evaluator: %s", reply.Error)
	}
	if err == nil && (reply.ID != request.ID || len(reply.Scores) != len(d.Candidates)) {
		err = fmt.Errorf("CPU evaluator response mismatch")
	}
	if err == nil {
		for _, score := range reply.Scores {
			if !finite(score) {
				err = fmt.Errorf("non-finite CPU evaluator score")
				break
			}
		}
	}
	if err != nil {
		p.stop()
		p.retryAfter = time.Now().Add(30 * time.Second)
		log.Printf("bot: CPU evaluator failed; using Go temporarily: %v", err)
		return nil, err
	}
	return reply.Scores, nil
}

func (p *pythonEvaluator) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.closed = true
	p.stop()
	return nil
}

// EnablePython is called once at startup, after Go validates every export.
// The child verifies identical SHA256 weights before the worker becomes ready.
// Formats 1-6 retain the Go fallback. Format 7 requires this evaluator and
// propagates errors rather than silently substituting its inherited actor.
func (r *Registry) EnablePython(executable, dir string) error {
	if executable == "" {
		for _, model := range r.models {
			if model.FormatVersion == 7 {
				return fmt.Errorf("specialist models require --inference-python or BOT_INFERENCE_PYTHON")
			}
		}
	}
	if executable == "" || len(r.models) == 0 {
		return nil
	}
	if r.python != nil {
		return fmt.Errorf("CPU evaluator already enabled")
	}
	expected := make(map[string]string)
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		hash := sha256.New()
		_, err = io.Copy(hash, f)
		f.Close()
		if err != nil {
			return err
		}
		// Compare to the exact file already validated by NewRegistry, avoiding
		// model substitution between registry load and evaluator startup.
		digest := fmt.Sprintf("%x", hash.Sum(nil))
		id, ok := r.sources[entry.Name()]
		if !ok || r.digests[id] != digest {
			return fmt.Errorf("model changed during evaluator startup: %s", entry.Name())
		}
		expected[id] = digest
	}
	if len(expected) != len(r.models) {
		return fmt.Errorf("model catalog changed during evaluator startup")
	}
	absolute, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	p := &pythonEvaluator{expected: expected, startupTimeout: 180 * time.Second, timeout: 5 * time.Second,
		command: func() *exec.Cmd {
			return exec.Command(executable, "-u", "-m", "smallworld_ai.cpu_evaluator", "--model-dir", absolute, "--threads", "1")
		}}
	if err := p.start(); err != nil {
		return err
	}
	r.python = p
	return nil
}

func (r *Registry) Close() error {
	if r != nil && r.python != nil {
		return r.python.Close()
	}
	return nil
}
