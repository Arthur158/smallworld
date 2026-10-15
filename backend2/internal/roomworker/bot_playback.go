package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/protocol"
	"backend/internal/store"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// Empty seats disqualify a room, including before a generated map compacts its
// roster. Playback controls never change the pacing of human or mixed games.
func (room *Room) onlyBots() bool {
	if len(room.Players) == 0 {
		return false
	}
	for _, player := range room.Players {
		if player == nil || player.BotModel == "" {
			return false
		}
	}
	return true
}

func validBotSpeed(speed float64) bool {
	switch speed {
	case 0.25, 0.5, 1, 2, 3:
		return true
	default:
		return false
	}
}

func (room *Room) botPlayback() *protocol.BotPlayback {
	if !room.onlyBots() {
		return nil
	}
	speed := room.botSpeed
	if !validBotSpeed(speed) {
		speed = 1
	}
	return &protocol.BotPlayback{Paused: room.botPaused, Speed: speed, CanGoBack: len(room.botHistory) > 0}
}

func (w *Worker) botMoveDelay(room *Room) time.Duration {
	playback := room.botPlayback()
	if playback == nil {
		return w.cfg.BotMoveDelay
	}
	delay := time.Duration(float64(w.cfg.BotMoveDelay) / playback.Speed)
	return max(100*time.Millisecond, delay)
}

func requireBotPlaybackHost(client *Client, cmd protocol.Command) (*Room, error) {
	room := client.Room
	if room == nil || room.ID != cmd.RoomID || room.IsDisplayRoom || cmd.Scope != protocol.ScopeRoom || !roomHasUser(room, client.Username) {
		return nil, errors.New("client not in the requested room")
	}
	if room.HostUsername != client.Username {
		return nil, errors.New("only the room host can control bot playback")
	}
	finished := room.Gamestate.TurnInfo != nil && room.Gamestate.TurnInfo.Phase == gamestate.GameFinished
	if room.Gamestate.TurnInfo == nil || ((!room.InProgress || finished) && !(finished && cmd.Message.Type == "previousBotMove")) {
		return nil, errors.New("bot playback requires a game in progress")
	}
	if !room.onlyBots() {
		return nil, errors.New("bot playback is only available in games with all bot seats")
	}
	return room, nil
}

func (w *Worker) handleBotPlayback(client *Client, cmd protocol.Command) error {
	room, err := requireBotPlaybackHost(client, cmd)
	if err != nil {
		return err
	}
	if cmd.Message.Type == "previousBotMove" {
		if !room.botPaused {
			return errors.New("pause bot playback before going back one move")
		}
		return room.previousBotMove()
	}
	if cmd.Message.Type == "stepBot" {
		if !room.botPaused {
			return errors.New("pause bot playback before advancing one move")
		}
		if room.botError != "" {
			return fmt.Errorf("bot is stopped: %s", room.botError)
		}
		if err := room.applyBotMove(w); err != nil {
			room.botError = err.Error()
			return fmt.Errorf("bot move could not be validated: %w", err)
		}
		// processResults persists the move together with the command ID before
		// publishing. No queued step can survive a subsequent resume command.
		return nil
	}

	var data struct {
		Paused *bool    `json:"paused"`
		Speed  *float64 `json:"speed"`
	}
	if err := json.Unmarshal(cmd.Message.Data, &data); err != nil || (data.Paused == nil && data.Speed == nil) {
		return errors.New("provide paused as a boolean or a supported bot speed")
	}
	if data.Speed != nil && !validBotSpeed(*data.Speed) {
		return errors.New("bot speed must be 0.25, 0.5, 1, 2, or 3")
	}
	if data.Paused != nil && !*data.Paused && room.botError != "" {
		return fmt.Errorf("bot is stopped: %s", room.botError)
	}
	if data.Paused != nil {
		room.botPaused = *data.Paused
	}
	if data.Speed != nil {
		room.botSpeed = *data.Speed
	}
	// Start a fresh interval on resume and speed changes. Paused wall-clock
	// time never accumulates into a burst of catch-up moves.
	room.botNextMove = time.Now().Add(w.botMoveDelay(room))
	room.botPlaybackDirty = true
	return nil
}

func (room *Room) applyBotMove(w *Worker) error {
	i := room.Gamestate.TurnInfo.PlayerIndex
	if i < 0 || i >= len(room.Players) || room.Players[i] == nil || room.Players[i].BotModel == "" {
		return errors.New("current player is not a bot")
	}
	var before json.RawMessage
	if room.onlyBots() {
		var err error
		before, err = store.EncodeGameState(&room.Gamestate)
		if err != nil {
			return fmt.Errorf("save bot playback history: %w", err)
		}
	}
	p := room.Players[i]
	if !w.bots.Has(p.BotModel) {
		room.sendStateMessage("A saved bot model is unavailable; this bot will use the random policy.")
		p.BotModel = "random"
		room.botPlaybackDirty = true
	}
	if before != nil && len(room.botFuture) > 0 {
		if err := room.restoreBotState(room.botFuture[len(room.botFuture)-1]); err != nil {
			return err
		}
		room.botFuture = room.botFuture[:len(room.botFuture)-1]
	} else if err := room.stepBot(w); err != nil {
		return err
	}
	if before != nil {
		if len(room.botHistory) == 0 {
			room.botPlaybackDirty = true
		}
		room.botHistory = append(room.botHistory, before)
		room.trimBotHistory()
		if room.Gamestate.TurnInfo.Phase == gamestate.GameFinished {
			room.botPaused = true
			room.botPlaybackDirty = true
		}
	}
	room.botDirty = true
	room.botNextMove = time.Now().Add(w.botMoveDelay(room))
	return nil
}
