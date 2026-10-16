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

// A mixed room uses the intersection supported by its trained models.
// Random and greedy policies can use either action schema.
func (room *Room) surfaceBotRules() bool {
	graph := false
	for _, p := range room.Players {
		if p == nil || p.BotModel == "" || p.BotModel == "random" || p.BotModel == "greedy" {
			continue
		}
		w := p.worker
		if w == nil {
			w = currentWorker
		}
		if w == nil || w.bots.Schema(p.BotModel) != gamestate.BotSchemaV2 {
			return false
		}
		graph = true
	}
	return graph
}
func (room *Room) validateBotState(state *gamestate.GameState) error {
	if room.surfaceBotRules() {
		return state.ValidateBotStateV2()
	}
	return state.ValidateBotState()
}
func (room *Room) validateBotChoices(races, traits, powers []string) error {
	if room.surfaceBotRules() {
		return gamestate.ValidateBotChoicesV2(races, traits, powers)
	}
	return gamestate.ValidateBotChoices(races, traits, powers)
}

func (room *Room) useBotRules() {
	races, traits := gamestate.BotRaces(), gamestate.BotTraits()
	if room.surfaceBotRules() {
		races, traits = gamestate.BotV2Races(), gamestate.BotV2Traits()
	}
	for i := range room.ExtensionChoices {
		e := &room.ExtensionChoices[i]
		e.IsChecked = false
		for j := range e.RaceChoices {
			e.RaceChoices[j].IsChecked = slices.Contains(races, e.RaceChoices[j].Choice) && gamestate.DefaultRaceEnabled(e.RaceChoices[j].Choice)
			e.IsChecked = e.IsChecked || e.RaceChoices[j].IsChecked
		}
		for j := range e.TraitChoices {
			e.TraitChoices[j].IsChecked = slices.Contains(traits, e.TraitChoices[j].Choice) && gamestate.DefaultTraitEnabled(e.TraitChoices[j].Choice)
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
		hadBots, surface := room.hasBots(), room.surfaceBotRules()
		name, err := room.newBotName(w, modelID, nil)
		if err != nil {
			return err
		}
		room.Players[i] = &Client{Username: "bot:" + uuid.NewString(), BotModel: modelID, BotName: name, Index: i, Room: room, worker: w}
		if !hadBots || surface != room.surfaceBotRules() {
			room.useBotRules()
		}
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
			if p.BotModel == modelID {
				return nil
			}
			name, err := room.newBotName(w, modelID, p)
			if err != nil {
				return err
			}
			surface := room.surfaceBotRules()
			p.BotModel = modelID
			p.BotName = name
			if surface != room.surfaceBotRules() {
				room.useBotRules()
			}
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
		if room.onlyBots() && room.botPaused {
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
		if err := room.applyBotMove(w); err != nil {
			room.botError = err.Error()
			log.Printf("roomworker: bot in room %s paused: %v", room.ID, err)
			room.sendStateMessage("Bot paused because its move could not be validated. Leave the room to end this match.")
			continue
		}
		if !w.commitBotMove(ctx, room) {
			return active
		}
	}
	return active
}

func (room *Room) stepBot(w *Worker) error {
	id := room.Players[room.Gamestate.TurnInfo.PlayerIndex].BotModel
	surface := w.bots.Schema(id) == gamestate.BotSchemaV2 || ((id == "random" || id == "greedy") && room.surfaceBotRules())
	var d gamestate.BotDecision
	var err error
	if surface {
		d, err = room.Gamestate.BotDecisionV2()
	} else {
		d, err = room.Gamestate.BotDecision()
	}
	if err != nil {
		return err
	}
	choice, err := w.bots.Select(room.Players[d.Player].BotModel, d, nil)
	if err != nil {
		return err
	}
	before := *room.Gamestate.TurnInfo
	if surface {
		_, err = room.Gamestate.ApplyBotActionV2(choice)
	} else {
		err = room.Gamestate.ApplyBotAction(choice)
	}
	if err != nil {
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
	room.publishBotMove()
	return true
}

// publishBotMove is also called after a manual step's state and command ID have
// been persisted together. It must never publish a move ahead of that journal.
func (room *Room) publishBotMove() {
	room.botDirty = false
	room.flushGameMessages()
	room.sendMegaUpdate()
	room.SendCoinUpdate()
	if room.botTurnEnded || room.botForceSave {
		room.AutoSave()
		room.botTurnEnded = false
		room.botForceSave = false
	}
	if room.Gamestate.TurnInfo.Phase == gamestate.GameFinished {
		room.sendGameFinishedUpdate()
		room.botPlaybackDirty = true
	}
	if room.botPlaybackDirty {
		sendRoomsUpdateToAll()
		room.botPlaybackDirty = false
	}
}
