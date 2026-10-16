package main

import (
	"backend/internal/bot"
	"backend/internal/gamestate"
	"fmt"
	"reflect"
	"testing"
)

func matchEnvironment(t *testing.T) *environment {
	t.Helper()
	gamestate.InitRaceMap()
	gamestate.InitTraitMap()
	r, err := bot.NewRegistry("")
	if err != nil {
		t.Fatal(err)
	}
	return &environment{policies: r, limit: 8192}
}

func TestMatchRestoresIdenticalStateAndReplays(t *testing.T) {
	for _, rules := range []string{"legacy", "surface"} {
		for players := 2; players <= 6; players++ {
			t.Run(fmt.Sprintf("%s/%d", rules, players), func(t *testing.T) {
				e := matchEnvironment(t)
				gameMap := fmt.Sprintf("map%dplayers", players)
				if players == 6 {
					gameMap += "2islands"
				}
				setup, err := e.handle(request{Op: "match_setup", Rules: rules, Players: players, Map: gameMap, Seed: 71891})
				if err != nil {
					t.Fatal(err)
				}
				policies := make([]string, players)
				for i := range policies {
					policies[i] = "greedy"
					if i%2 == 0 {
						policies[i] = "random"
					}
				}
				req := request{Op: "match", Rules: rules, Policies: policies, Seed: 71891, InitialState: setup.InitialState, InitialHash: setup.InitialHash}
				first, err := e.handle(req)
				if err != nil {
					t.Fatal(err)
				}
				second, err := e.handle(req)
				if err != nil {
					t.Fatal(err)
				}
				if !first.Done || first.Truncated || !reflect.DeepEqual(first.Match, second.Match) {
					t.Fatalf("non deterministic or incomplete: %+v %+v", first.Match, second.Match)
				}
				if first.Match.InitialHash != setup.InitialHash || len(first.Match.Scores) != players || first.Match.Players != players {
					t.Fatal("invalid restored match")
				}
				total := 0
				for seat, counts := range first.Match.SuccessfulActions {
					for _, count := range counts {
						total += count
					}
					total += first.Match.Rejections[seat]
				}
				if total != first.Match.Steps {
					t.Fatal("missing action diagnostics")
				}
				req.InitialHash = "wrong"
				if _, err = e.handle(req); err == nil {
					t.Fatal("accepted wrong fixture checksum")
				}
				req.InitialHash = setup.InitialHash
				req.Policies[0] = "missing"
				if _, err = e.handle(req); err == nil {
					t.Fatal("accepted unknown model")
				}
			})
		}
	}
}

func TestComparisonRejectsPartialGameAndWrongPlayerCount(t *testing.T) {
	e := matchEnvironment(t)
	for _, count := range []int{0, 1, 7} {
		if _, err := e.handle(request{Op: "match_setup", Players: count}); err == nil {
			t.Fatalf("accepted %d player comparison", count)
		}
	}
	if _, err := e.handle(request{Op: "match_setup", Players: 4, Focus: true}); err == nil {
		t.Fatal("accepted forced curriculum")
	}
	setup, err := e.handle(request{Op: "match_setup", Rules: "legacy", Players: 4, Map: "map4players", Seed: 71})
	if err != nil {
		t.Fatal(err)
	}
	req := request{Op: "match", Rules: "legacy", Policies: []string{"greedy", "greedy", "greedy", "greedy"}, InitialState: setup.InitialState, InitialHash: setup.InitialHash}
	for _, count := range []int{1, 2, 3, 5, 6, 7} {
		bad := req
		bad.Policies = make([]string, count)
		for i := range bad.Policies {
			bad.Policies[i] = "greedy"
		}
		if _, err := e.handle(bad); err == nil {
			t.Fatalf("accepted %d policies for a 4-player snapshot", count)
		}
	}
	bad := req
	bad.Players = 3
	if _, err := e.handle(bad); err == nil {
		t.Fatal("accepted conflicting explicit player count")
	}
	e.limit = 1
	if _, err = e.handle(req); err == nil {
		t.Fatal("reported truncated game as complete")
	}
}
