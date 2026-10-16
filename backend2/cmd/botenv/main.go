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
	"sort"
)

type request struct {
	Policies     []string                    `json:"policies,omitempty"`
	InitialState json.RawMessage             `json:"initial_state,omitempty"`
	InitialHash  string                      `json:"initial_sha256,omitempty"`
	Rules        string                      `json:"rules"`
	Curriculum   int                         `json:"curriculum"`
	FocusPlayer  int                         `json:"focus_player"`
	FocusRace    string                      `json:"focus_race,omitempty"`
	FocusTrait   string                      `json:"focus_trait,omitempty"`
	Focus        bool                        `json:"focus"`
	Tiles        []gamestate.RuntimeTileSpec `json:"tiles"`
	Op           string                      `json:"op"`
	Seed         uint64                      `json:"seed"`
	Players      int                         `json:"players"`
	Map          string                      `json:"map"`
	Action       *int                        `json:"action"`
	Policy       string                      `json:"policy"`
}

type response struct {
	Match        *matchResult           `json:"match,omitempty"`
	InitialState json.RawMessage        `json:"initial_state,omitempty"`
	InitialHash  string                 `json:"initial_sha256,omitempty"`
	Packed       *packedDecision        `json:"packed_decision,omitempty"`
	Action       *gamestate.BotAction   `json:"action,omitempty"`
	Rejected     bool                   `json:"rejected,omitempty"`
	Coverage     map[string][]string    `json:"coverage,omitempty"`
	Spec         map[string]interface{} `json:"spec,omitempty"`
	PolicyScores []float64              `json:"policy_scores,omitempty"`
	Decision     *gamestate.BotDecision `json:"decision,omitempty"`
	Done         bool                   `json:"done"`
	Truncated    bool                   `json:"truncated"`
	Scores       []int                  `json:"scores,omitempty"`
	Steps        int                    `json:"steps"`
	Error        string                 `json:"error,omitempty"`
}

type environment struct {
	action                  *gamestate.BotAction
	full                    bool
	rejected                bool
	coverage                map[string]map[string]bool
	focusRace, focusTrait   string
	focusPending            bool
	focusPlayer, curriculum int
	state                   *gamestate.GameState
	policies                *bot.Registry
	rng                     *rand.Rand
	steps                   int
	limit                   int
}

func (e *environment) handle(req request) (response, error) {
	e.rejected = false
	e.action = nil
	switch req.Op {
	case "match_setup":
		return e.matchSetup(req)
	case "match":
		return e.playMatch(req)
	case "spec":
		return response{Spec: gamestate.BotV2Spec()}, nil
	case "reset":
		if req.Curriculum < 0 {
			return response{}, fmt.Errorf("curriculum must be nonnegative")
		}
		if req.Players < 2 || req.Players > 6 {
			return response{}, fmt.Errorf("players must be between 2 and 6")
		}
		if req.FocusRace != "" || req.FocusTrait != "" {
			if !req.Focus || req.Rules != "surface" || !contains(gamestate.BotTrainingRaces(), req.FocusRace) || !contains(gamestate.BotTrainingTraits(), req.FocusTrait) {
				return response{}, fmt.Errorf("explicit focus requires an enabled surface race and trait")
			}
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
		races, traits, powers := gamestate.BotRaces(), gamestate.BotTraits(), []string(nil)
		if req.Rules != "" && req.Rules != "legacy" && req.Rules != "surface" {
			return response{}, fmt.Errorf("unknown rules preset")
		}
		e.full = req.Rules == "surface"
		if e.full {
			races, traits = gamestate.BotTrainingRaces(), gamestate.BotTrainingTraits()
		}
		state, err := gamestate.NewSeeded(names, mapName, races, traits, powers, req.Seed)
		if err != nil {
			return response{}, err
		}
		if e.full && req.Map == "generated" {
			state.PopulateBotSurfaceMap()
		}
		e.state, e.steps = state, 0
		e.coverage = map[string]map[string]bool{"races": {}, "traits": {}, "powers": {}, "actions": {}}
		if req.FocusPlayer < 0 || req.FocusPlayer >= req.Players {
			return response{}, fmt.Errorf("invalid focus player")
		}
		e.focusPending, e.focusPlayer, e.curriculum = e.full && req.Focus, req.FocusPlayer, req.Curriculum
		e.focusRace, e.focusTrait = req.FocusRace, req.FocusTrait
		// Reserve the assigned pair beyond the visible market until the learner
		// gets its first turn. Earlier seats must not take its training assignment.
		if e.focusRace != "" {
			if err := placeFocusPair(state, e.focusRace, e.focusTrait, len(state.TribeList)-1); err != nil {
				return response{}, err
			}
		}
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
			d, err := e.decision()
			if err != nil {
				return response{}, err
			}
			choice, err = e.policies.Select(req.Policy, d, e.rng)
			if err != nil {
				return response{}, err
			}
		}
		if e.full {
			d, err := e.decision()
			if err != nil {
				return response{}, err
			}
			if choice < 0 || choice >= len(d.Candidates) {
				return response{}, fmt.Errorf("action out of range")
			}
			selected := d.Candidates[choice].Action
			e.action = &selected
			e.rejected, err = e.state.ApplyBotActionV2(choice)
			if err != nil {
				return response{}, err
			}
			if !e.rejected {
				e.coverage["actions"][d.Candidates[choice].Action.Kind+":"+d.Candidates[choice].Action.Stack] = true
			}
		} else if err := e.state.ApplyBotAction(choice); err != nil {
			return response{}, err
		}
		e.steps++
		// Logs are irrelevant to learning and must not accumulate in long runs.
		e.state.Messages = nil
	default:
		return response{}, fmt.Errorf("unknown operation %q", req.Op)
	}
	if err := e.applyFocus(); err != nil {
		return response{}, err
	}
	d, err := e.decision()
	if err != nil {
		return response{}, err
	}
	out := response{Action: e.action, Decision: &d, Steps: e.steps, Done: e.state.TurnInfo.Phase == gamestate.GameFinished, Rejected: e.rejected}
	if e.full {
		for _, p := range e.state.Players {
			tribes := append([]*gamestate.Tribe{p.ActiveTribe}, p.PassiveTribes...)
			for _, t := range tribes {
				if t != nil {
					e.coverage["races"][string(t.Race)] = true
					e.coverage["traits"][string(t.Trait)] = true
				}
			}
		}
		for name := range e.state.Powers {
			e.coverage["powers"][name] = true
		}
		out.Coverage = map[string][]string{}
		for k, values := range e.coverage {
			for v := range values {
				out.Coverage[k] = append(out.Coverage[k], v)
			}
			sort.Strings(out.Coverage[k])
		}
	}
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

func (e *environment) decision() (gamestate.BotDecision, error) {
	if e.full {
		return e.state.BotDecisionV2()
	}
	return e.state.BotDecision()
}

func main() {
	python := flag.String("inference-python", os.Getenv("BOT_INFERENCE_PYTHON"), "optional persistent PyTorch CPU interpreter")
	models := flag.String("model-dir", "", "directory of exported model JSON files")
	packed := flag.Bool("packed-observations", false, "encode observation arrays as float32 for the trainer")
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
	if err := registry.EnablePython(*python, *models); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer registry.Close()
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
		if *packed && out.Decision != nil {
			out.Packed = packDecision(out.Decision)
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

func (e *environment) applyFocus() error {
	if e.focusPending && e.state.TurnInfo.PlayerIndex == e.focusPlayer && e.state.TurnInfo.Phase == gamestate.TribeChoice {
		state := e.state
		races, traits := gamestate.BotTrainingRaces(), gamestate.BotTrainingTraits()
		// Rotate traits across deck cycles, covering every pairing even when
		// the two roster sizes share a factor.
		race, trait := curriculumPair(e.curriculum, races, traits)
		if e.focusRace != "" {
			if err := placeFocusPair(state, e.focusRace, e.focusTrait, 0); err != nil {
				return err
			}
			if err := state.HandleTribeChoice(e.focusPlayer, 0); err != nil {
				return err
			}
			e.focusPending = false
			return nil
		}
		for i, entry := range state.TribeList {
			if string(entry.Race) == race {
				state.TribeList[0].Race, state.TribeList[i].Race = state.TribeList[i].Race, state.TribeList[0].Race
				break
			}
		}
		for i, entry := range state.TribeList {
			if string(entry.Trait) == trait {
				state.TribeList[0].Trait, state.TribeList[i].Trait = state.TribeList[i].Trait, state.TribeList[0].Trait
				break
			}
		}
		// A trait omitted by a shorter deck still receives curriculum exposure.
		state.TribeList[0].Trait = gamestate.Trait(trait)
		for _, entry := range state.TribeList {
			entry.PiecePile = gamestate.RaceMap[entry.Race].Count + gamestate.TraitMap[entry.Trait].Count
		}
		if err := state.HandleTribeChoice(e.focusPlayer, 0); err != nil {
			return err
		}
		e.focusPending = false
	}
	return nil
}

func curriculumPair(cursor int, races, traits []string) (string, string) {
	race := cursor % len(races)
	return races[race], traits[(cursor/len(races)+race)%len(traits)]
}

func contains(items []string, item string) bool {
	for _, value := range items {
		if value == item {
			return true
		}
	}
	return false
}

func placeFocusPair(state *gamestate.GameState, race, trait string, destination int) error {
	found := false
	for i, entry := range state.TribeList {
		if string(entry.Race) == race {
			state.TribeList[destination].Race, state.TribeList[i].Race = state.TribeList[i].Race, state.TribeList[destination].Race
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("assigned race is missing from training deck")
	}
	for i, entry := range state.TribeList {
		if string(entry.Trait) == trait {
			state.TribeList[destination].Trait, state.TribeList[i].Trait = state.TribeList[i].Trait, state.TribeList[destination].Trait
			break
		}
	}
	state.TribeList[destination].Trait = gamestate.Trait(trait)
	for _, entry := range state.TribeList {
		entry.PiecePile = gamestate.RaceMap[entry.Race].Count + gamestate.TraitMap[entry.Trait].Count
	}
	return nil
}
