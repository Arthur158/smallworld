// botenv is an offline, line-delimited JSON bridge to the real game engine.
// It has no network listener and requires no Redis, PostgreSQL or account.
package main

import (
	"backend/internal/bot"
	"backend/internal/gamestate"
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"math/rand"
	"os"
)

type request struct {
	Tiles   []gamestate.RuntimeTileSpec `json:"tiles"`
	Op      string                      `json:"op"`
	Seed    uint64                      `json:"seed"`
	Players int                         `json:"players"`
	Map     string                      `json:"map"`
	Action  *int                        `json:"action"`
	Policy  string                      `json:"policy"`
}

type response struct {
	PolicyScores []float64              `json:"policy_scores,omitempty"`
	Decision     *gamestate.BotDecision `json:"decision,omitempty"`
	Done         bool                   `json:"done"`
	Truncated    bool                   `json:"truncated"`
	Scores       []int                  `json:"scores,omitempty"`
	Steps        int                    `json:"steps"`
	Error        string                 `json:"error,omitempty"`
}

type environment struct {
	state    *gamestate.GameState
	policies *bot.Registry
	rng      *rand.Rand
	steps    int
	limit    int
}

func (e *environment) handle(req request) (response, error) {
	switch req.Op {
	case "reset":
		if req.Players < 2 || req.Players > 6 {
			return response{}, fmt.Errorf("players must be between 2 and 6")
		}
		mapName := req.Map
		if req.Map == "generated" {
			if err := installTrainingMap(req.Tiles); err != nil {
				return response{}, err
			}
			mapName = "botenv-generated"
		} else if _, ok := gamestate.MapRegistry[req.Map]; !ok {
			return response{}, fmt.Errorf("unknown built-in map %q", req.Map)
		}
		names := make([]string, req.Players)
		for i := range names {
			names[i] = fmt.Sprintf("Player %d", i+1)
		}
		state, err := gamestate.NewSeeded(names, mapName, gamestate.BotRaces(), gamestate.BotTraits(), nil, req.Seed)
		if err != nil {
			return response{}, err
		}
		e.state, e.steps = state, 0
		e.rng = rand.New(rand.NewSource(int64(req.Seed)))
	case "observe", "scores":
		if e.state == nil {
			return response{}, fmt.Errorf("reset first")
		}
	case "step":
		if e.state == nil {
			return response{}, fmt.Errorf("reset first")
		}
		if e.state.TurnInfo.Phase == gamestate.GameFinished || e.steps >= e.limit {
			return response{}, fmt.Errorf("episode ended; reset first")
		}
		if (req.Action == nil) == (req.Policy == "") {
			return response{}, fmt.Errorf("supply exactly one action index or policy id")
		}
		choice := 0
		if req.Action != nil {
			choice = *req.Action
		} else {
			d, err := e.state.BotDecision()
			if err != nil {
				return response{}, err
			}
			choice, err = e.policies.Select(req.Policy, d, e.rng)
			if err != nil {
				return response{}, err
			}
		}
		if err := e.state.ApplyBotAction(choice); err != nil {
			return response{}, err
		}
		e.steps++
		// Logs are irrelevant to learning and must not accumulate in long runs.
		e.state.Messages = nil
	default:
		return response{}, fmt.Errorf("unknown operation %q", req.Op)
	}
	d, err := e.state.BotDecision()
	if err != nil {
		return response{}, err
	}
	out := response{Decision: &d, Steps: e.steps, Done: e.state.TurnInfo.Phase == gamestate.GameFinished}
	out.Truncated = !out.Done && e.steps >= e.limit
	for _, p := range e.state.Players {
		out.Scores = append(out.Scores, p.CoinPile)
	}
	if req.Op == "scores" {
		out.PolicyScores, err = e.policies.Scores(req.Policy, d)
		if err != nil {
			return response{}, err
		}
	}
	return out, nil
}

func main() {
	models := flag.String("model-dir", "", "directory of exported model JSON files")
	maxSteps := flag.Int("max-steps", 4096, "maximum engine decisions per episode")
	flag.Parse()
	if *maxSteps < 1 {
		fmt.Fprintln(os.Stderr, "max-steps must be positive")
		os.Exit(2)
	}
	registry, err := bot.NewRegistry(*models)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	e := environment{policies: registry, limit: *maxSteps}
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Buffer(make([]byte, 4096), 1<<20)
	writer := bufio.NewWriter(os.Stdout)
	encoder := json.NewEncoder(writer)
	for scanner.Scan() {
		var req request
		err := json.Unmarshal(scanner.Bytes(), &req)
		var out response
		if err == nil {
			out, err = e.handle(req)
		}
		if err != nil {
			out = response{Error: err.Error()}
		}
		if err := encoder.Encode(out); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		if err := writer.Flush(); err != nil {
			os.Exit(1)
		}
	}
	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
