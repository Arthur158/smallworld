package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/protocol"
	"context"
	"fmt"
	"log"
	"slices"
	"time"

	"github.com/google/uuid"
)

func (c *Client) displayName() string {
	if c.BotModel != "" {
		return c.BotName
	}
	return c.Username
}

func (room *Room) hasBots() bool {
	for _, p := range room.Players {
		if p != nil && p.BotModel != "" {
			return true
		}
	}
	return false
}

func (room *Room) botSeats() map[string]protocol.BotSeat {
	seats := make(map[string]protocol.BotSeat)
	for _, p := range room.Players {
		if p != nil && p.BotModel != "" {
			seats[p.Username] = protocol.BotSeat{ModelID: p.BotModel, Name: p.BotName}
		}
	}
	return seats
}

func (room *Room) useBotRules() {
	for i := range room.ExtensionChoices {
		e := &room.ExtensionChoices[i]
		e.IsChecked = false
		for j := range e.RaceChoices {
			e.RaceChoices[j].IsChecked = slices.Contains(gamestate.BotRaces(), e.RaceChoices[j].Choice)
			e.IsChecked = e.IsChecked || e.RaceChoices[j].IsChecked
		}
		for j := range e.TraitChoices {
			e.TraitChoices[j].IsChecked = slices.Contains(gamestate.BotTraits(), e.TraitChoices[j].Choice)
			e.IsChecked = e.IsChecked || e.TraitChoices[j].IsChecked
		}
	}
	room.GlobalToggle = false
	room.sendChoices()
}

func (room *Room) addBot(w *Worker, modelID string) error {
	if room.InProgress {
		return fmt.Errorf("cannot add a bot after the game starts")
	}
	for i, p := range room.Players {
		if p != nil {
			continue
		}
		if !room.hasBots() {
			room.useBotRules()
		}
		name := fmt.Sprintf("Bot %d", i+1)
		for number := i + 2; ; number++ {
			used := false
			for _, player := range room.Players {
				used = used || (player != nil && player.displayName() == name)
			}
			if !used {
				break
			}
			name = fmt.Sprintf("Bot %d", number)
		}
		room.Players[i] = &Client{Username: "bot:" + uuid.NewString(), BotModel: modelID, BotName: name, Index: i, Room: room, worker: w}
		sendRoomsUpdateToAll()
		return nil
	}
	return fmt.Errorf("room is full")
}

func (room *Room) setBotModel(w *Worker, username, modelID string) error {
	if room.InProgress {
		return fmt.Errorf("cannot change a bot after the game starts")
	}
	for _, p := range room.Players {
		if p != nil && p.Username == username && p.BotModel != "" {
			p.BotModel = modelID
			sendRoomsUpdateToAll()
			return nil
		}
	}
	return fmt.Errorf("bot seat not found")
}

// advanceBots runs only on Worker.Run's command goroutine. One decision per
// room/tick keeps commands responsive; no bot goroutine mutates room state.
// Persist before publishing or making another move. A transient Redis failure
// retries the same dirty state, never applies the move twice in this process.
func (w *Worker) advanceBots(ctx context.Context) bool {
	active := false
	for _, room := range rooms {
		if room.IsDisplayRoom {
			continue
		}
		if room.botDirty {
			active = true
			if !w.commitBotMove(ctx, room) {
				return active
			}
		}
		if !room.InProgress || !room.hasConnectedUsers() || room.botError != "" || room.Gamestate.TurnInfo == nil {
			continue
		}
		i := room.Gamestate.TurnInfo.PlayerIndex
		if i < 0 || i >= len(room.Players) {
			continue
		}
		p := room.Players[i]
		if p == nil || p.BotModel == "" {
			continue
		}
		active = true
		if time.Now().Before(room.botNextMove) {
			continue
		}
		if !w.bots.Has(p.BotModel) {
			room.sendStateMessage("A saved bot model is unavailable; this bot will use the random policy.")
			p.BotModel = "random"
			sendRoomsUpdateToAll()
		}
		if err := room.stepBot(w); err != nil {
			room.botError = err.Error()
			log.Printf("roomworker: bot in room %s paused: %v", room.ID, err)
			room.sendStateMessage("Bot paused because its move could not be validated. Leave the room to end this match.")
			continue
		}
		room.botDirty = true
		room.botNextMove = time.Now().Add(w.cfg.BotMoveDelay)
		if !w.commitBotMove(ctx, room) {
			return active
		}
	}
	return active
}

func (room *Room) stepBot(w *Worker) error {
	d, err := room.Gamestate.BotDecision()
	if err != nil {
		return err
	}
	choice, err := w.bots.Select(room.Players[d.Player].BotModel, d, nil)
	if err != nil {
		return err
	}
	before := *room.Gamestate.TurnInfo
	if err := room.Gamestate.ApplyBotAction(choice); err != nil {
		return err
	}
	after := room.Gamestate.TurnInfo
	room.botTurnEnded = before.PlayerIndex != after.PlayerIndex || before.TurnIndex != after.TurnIndex || after.Phase == gamestate.GameFinished
	if after.Phase == gamestate.GameFinished {
		room.InProgress = false
	}
	return nil
}

// A bot move has no Redis stream entry to ACK. Keep the worker's serialization
// boundary until its snapshot is durable, just like accepted human commands.
func (w *Worker) commitBotMove(ctx context.Context, room *Room) bool {
	for {
		if err := w.persistRoom(ctx, room); err == nil {
			break
		} else {
			log.Printf("roomworker: persist bot move: %v", err)
		}
		select {
		case <-ctx.Done():
			return false
		case <-time.After(500 * time.Millisecond):
		}
	}
	room.botDirty = false
	room.flushGameMessages()
	room.sendMegaUpdate()
	room.SendCoinUpdate()
	if room.botTurnEnded {
		room.AutoSave()
		room.botTurnEnded = false
	}
	if room.Gamestate.TurnInfo.Phase == gamestate.GameFinished {
		room.sendGameFinishedUpdate()
		sendRoomsUpdateToAll()
	}
	return true
}
