package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/store"
	"encoding/json"
	"errors"
	"fmt"
)

const (
	botHistoryLimit     = 100
	botHistoryByteLimit = 4 << 20
)

// Keep history outside GameState and the AI contract. A forward stack preserves
// the exact previously played decisions, including random-policy choices that
// use the policy's process-wide RNG rather than GameState's dice stream.
func (room *Room) trimBotHistory() {
	bytes := 0
	for _, states := range [][]json.RawMessage{room.botHistory, room.botFuture} {
		for _, state := range states {
			bytes += len(state)
		}
	}
	for len(room.botHistory)+len(room.botFuture) > botHistoryLimit || bytes > botHistoryByteLimit {
		if len(room.botHistory) > 0 {
			bytes -= len(room.botHistory[0])
			room.botHistory[0] = nil
			room.botHistory = room.botHistory[1:]
		} else if len(room.botFuture) > 0 {
			bytes -= len(room.botFuture[0])
			room.botFuture[0] = nil
			room.botFuture = room.botFuture[1:]
		} else {
			break
		}
	}
}

func (room *Room) restoreBotState(data json.RawMessage) error {
	state, err := store.DecodeGameState(data)
	if err != nil {
		return fmt.Errorf("restore bot playback: %w", err)
	}
	if state.TurnInfo == nil || len(state.Players) != len(room.Players) {
		return errors.New("saved bot playback does not match the room")
	}
	if err := room.validateBotState(state); err != nil {
		return fmt.Errorf("restore bot playback: %w", err)
	}
	for i, player := range state.Players {
		player.Name = room.Players[i].displayName()
	}
	before, after := room.Gamestate.TurnInfo, state.TurnInfo
	room.botTurnEnded = before.PlayerIndex != after.PlayerIndex || before.TurnIndex != after.TurnIndex || after.Phase == gamestate.GameFinished
	room.Gamestate = *state
	room.InProgress = after.Phase != gamestate.GameFinished
	return nil
}

func (room *Room) previousBotMove() error {
	if len(room.botHistory) == 0 {
		return errors.New("no earlier bot move is available")
	}
	current, err := store.EncodeGameState(&room.Gamestate)
	if err != nil {
		return fmt.Errorf("save bot playback: %w", err)
	}
	previous := room.botHistory[len(room.botHistory)-1]
	if err := room.restoreBotState(previous); err != nil {
		return err
	}
	room.botHistory = room.botHistory[:len(room.botHistory)-1]
	room.botFuture = append(room.botFuture, current)
	room.trimBotHistory()
	room.botPaused = true
	room.botError = ""
	room.botDirty = true
	room.botPlaybackDirty = true
	room.botForceSave = true
	room.Gamestate.Messages = append(room.Gamestate.Messages, gamestate.Message{Content: "Rewound one bot move."})
	return nil
}
