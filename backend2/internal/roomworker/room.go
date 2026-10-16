package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/google/uuid"
)

type Room struct {
	generatedDraft          *generatedMapPreview
	generatedSelected       *generatedMapPreview
	generatedPending        *pendingGeneratedMap
	generatedSelectionDirty bool
	botHistory              []json.RawMessage
	botFuture               []json.RawMessage
	botForceSave            bool
	botPaused               bool
	botSpeed                float64
	botPlaybackDirty        bool
	botNextMove             time.Time
	botDirty                bool
	botTurnEnded            bool
	botError                string
	ID                      string
	Name                    string
	HostUsername            string
	Players                 []*Client
	Spectators              []*Client
	InProgress              bool
	Gamestate               gamestate.GameState
	mu                      sync.Mutex
	Map                     Map
	saveId                  int64
	autoSaveId              int64
	playerStatuses          []string
	IsDisplayRoom           bool
	ExtensionChoices        []Extension
	GlobalToggle            bool
	processedCommandIDs     []string
}

const processedCommandHistory = 256

func (room *Room) hasProcessedCommand(id string) bool {
	if room == nil || id == "" {
		return false
	}
	for _, existing := range room.processedCommandIDs {
		if existing == id {
			return true
		}
	}
	return false
}

func (room *Room) markProcessedCommand(id string) {
	if room == nil || id == "" || room.hasProcessedCommand(id) {
		return
	}
	room.processedCommandIDs = append(room.processedCommandIDs, id)
	if len(room.processedCommandIDs) > processedCommandHistory {
		room.processedCommandIDs = append([]string(nil), room.processedCommandIDs[len(room.processedCommandIDs)-processedCommandHistory:]...)
	}
}

func createRoom(client *Client, roomName, username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, choices, err := rulesPresetSettings("standard")
	if err != nil {
		client.sendError(err.Error())
		return
	}

	roomID := client.CommandRoomID
	if roomID == "" {
		roomID = uuid.New().String()
	}

	room := &Room{
		ID:             roomID,
		Name:           roomName,
		HostUsername:   username,
		Players:        make([]*Client, gameMap.Capacity), // Create a fixed-size slice with nil values
		Spectators:     []*Client{},
		InProgress:     false,
		Gamestate:      gamestate.GameState{},
		Map:            gameMap,
		saveId:         -1,
		playerStatuses: []string{},
		GlobalToggle:   true,
	}
	rooms[room.ID] = room

	for i := range gameMap.Capacity {
		room.Players[i] = nil
	}

	room.ExtensionChoices = choices
	room.GlobalToggle = false

	room.Players[0] = client

	roomIDData, _ := json.Marshal(map[string]string{"roomid": room.ID})
	if err := client.bindRoom(room, false, messages.Message{Type: "roomid", Data: roomIDData}); err != nil {
		log.Printf("roomworker: bind creator to room: %v", err)
	}

	sendRoomsUpdateToAll()
	log.Println("creating room ...")

	room.sendPlayerStatuses()
	room.sendMapChoices()
	room.sendChoices()
	room.sendSaveSelectionToHost()
}

func (room *Room) sendSaveSelectionToHost() {
	if host := room.hostClient(); host != nil {
		raw, _ := json.Marshal(map[string]int64{"index": room.saveId})
		host.sendMessage("saveSelection", raw)
	}
}

func (room *Room) sendPlayerStatuses() {
	statuses := room.playerStatuses

	if statuses == nil {
		statuses = []string{}
	}

	playerStatusesJSON, err := json.Marshal(statuses)
	if err != nil {
		log.Println("Error marshalling playerStatuses:", err)
		return
	}

	room.sendToRoomPlayers(messages.Message{
		Type: "playerStatuses",
		Data: json.RawMessage(playerStatusesJSON),
	})
}

func (room *Room) sendMapChoices() {
	mapChoices := []string{}
	for key := range mapMap {
		mapChoices = append(mapChoices, key)
	}

	host := room.hostClient()

	message := map[string]interface{}{
		"mapChoices": mapChoices, // This ensures it's an empty list if no maps exist
	}

	if host == nil {
		return
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		host.sendError("Failed to marshal message")
		return
	}
	host.sendMessage("mapChoices", json.RawMessage(messageJSON))
}

func (room *Room) sendChoices() {
	host := room.hostClient()

	message := map[string]interface{}{
		"extensionChoices": room.ExtensionChoices,
		"globalToggle":     room.GlobalToggle,
	}

	if host == nil {
		return
	}
	messageJSON, err := json.Marshal(message)
	if err != nil {
		host.sendError("Failed to marshal message")
		return
	}
	host.sendMessage("choices", json.RawMessage(messageJSON))
}

func (room *Room) toggleRace(extension string, race string, check bool) {
	for i, entry := range room.ExtensionChoices {
		if entry.ExtensionName == extension {
			for j, entry2 := range entry.RaceChoices {
				if entry2.Choice == race {
					room.ExtensionChoices[i].RaceChoices[j].IsChecked = check
				}
			}
		}
	}
	room.sendChoices()
}
func (room *Room) toggleTrait(extension string, trait string, check bool) {
	for i, entry := range room.ExtensionChoices {
		if entry.ExtensionName == extension {
			for j, entry2 := range entry.TraitChoices {
				if entry2.Choice == trait {
					room.ExtensionChoices[i].TraitChoices[j].IsChecked = check
				}
			}
		}
	}
	room.sendChoices()
}

func (room *Room) toggleExtension(name string, check bool) {
	for i := range room.ExtensionChoices {
		if room.ExtensionChoices[i].ExtensionName == name {
			room.ExtensionChoices[i].IsChecked = check
			for j := range room.ExtensionChoices[i].RaceChoices {
				room.ExtensionChoices[i].RaceChoices[j].IsChecked = check
			}
			for j := range room.ExtensionChoices[i].TraitChoices {
				room.ExtensionChoices[i].TraitChoices[j].IsChecked = check
			}
		}
	}
	room.sendChoices()
}
func (room *Room) toggleAll(check bool) {
	room.GlobalToggle = check
	for i := range room.ExtensionChoices {
		room.ExtensionChoices[i].IsChecked = check
		for j := range room.ExtensionChoices[i].RaceChoices {
			room.ExtensionChoices[i].RaceChoices[j].IsChecked = check
		}
		for j := range room.ExtensionChoices[i].TraitChoices {
			room.ExtensionChoices[i].TraitChoices[j].IsChecked = check
		}
	}
	room.sendChoices()
}

func (room *Room) ChangeMap(newMap string) bool {

	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, ok := mapMap[newMap]
	if !ok {
		log.Println("Problem logging map")
		return false
	}

	occupied := 0
	for _, p := range room.Players {
		if p != nil {
			occupied++
		}
	}
	if !room.IsDisplayRoom && occupied > gameMap.Capacity {
		return false
	}
	room.clearGeneratedMapCustomization()
	room.Map = gameMap

	newSize := gameMap.Capacity

	// First, remove any nil players from the slice
	cleanedPlayers := make([]*Client, 0, len(room.Players))
	for _, p := range room.Players {
		if p != nil {
			cleanedPlayers = append(cleanedPlayers, p)
		}
	}
	room.Players = cleanedPlayers

	currentPlayers := len(room.Players)

	// Now adjust the slice length
	if newSize < currentPlayers {
		// Truncate if the new size is smaller
		room.Players = room.Players[:newSize]
	} else {
		// Expand with nil slots if the new size is larger
		for len(room.Players) < newSize {
			room.Players = append(room.Players, nil)
		}
	}

	// Ensure there is still a valid host
	if len(room.Players) > 0 && room.HostUsername == "" {
		for _, player := range room.Players {
			if player != nil && player.BotModel == "" {
				room.HostUsername = player.Username
				break
			}
		}
	}

	log.Printf("Room %s resized to %d players.\n", room.ID, newSize)

	sendRoomsUpdateToAll()
	return true
}

func joinRoom(client *Client, roomID, username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	// If the client was already in a different room, remove them from that old room
	oldRoom := client.Room
	if oldRoom != nil {
		for i, c := range oldRoom.Players {
			if c == client {
				oldRoom.Players[i] = nil // Mark the slot as empty
				break
			}
		}
		// If the old room becomes empty, optionally remove it
		empty := true
		for _, c := range oldRoom.Players {
			if c != nil {
				empty = false
				break
			}
		}
		if empty {
			delete(rooms, oldRoom.ID)
		}
	}

	// Now try to join the new room
	newRoom, exists := rooms[roomID]
	if !exists {
		client.sendError("That room does not exist.")
		return
	}

	if newRoom.InProgress {
		client.sendError("That game is already in progress.")
		return
	}

	// Check for available slots
	slotFound := false
	for i := range newRoom.Players {
		if newRoom.Players[i] == nil { // Find the first empty spot
			newRoom.Players[i] = client
			slotFound = true
			break
		}
	}

	if !slotFound {
		client.sendError("Room is full.")
		return
	}

	// Update client's info to reflect the new room
	client.Username = username

	// Bind first so the gateway subscribes to room events before normal room broadcasts.
	roomIDData, _ := json.Marshal(map[string]string{"roomid": newRoom.ID})
	if err := client.bindRoom(newRoom, false, messages.Message{Type: "roomid", Data: roomIDData}); err != nil {
		client.sendError("failed to join room")
		return
	}
	newRoom.sendPlayerStatuses()

	log.Println("adding player ...")
	log.Println(client.Room.Players)

	// Broadcast the updated rooms list to everyone
	sendRoomsUpdateToAll()
}

func spectateRoom(client *Client, roomID string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	// If the client was already in a different room, remove them from that old room
	oldRoom := client.Room
	if oldRoom != nil {
		for i, c := range oldRoom.Players {
			if c == client {
				oldRoom.Players[i] = nil // Mark the slot as empty
				break
			}
		}
		// If the old room becomes empty, optionally remove it
		empty := true
		for _, c := range oldRoom.Players {
			if c != nil {
				empty = false
				break
			}
		}
		if empty {
			delete(rooms, oldRoom.ID)
		}
	}

	// Now try to join the new room
	newRoom, exists := rooms[roomID]
	if !exists {
		client.sendError("That room does not exist.")
		return
	}

	newRoom.Spectators = append(newRoom.Spectators, client)

	roomIDData, _ := json.Marshal(map[string]string{"roomid": newRoom.ID})
	if err := client.bindRoom(newRoom, true, messages.Message{Type: "roomid", Data: roomIDData}); err != nil {
		client.sendError("failed to spectate room")
		return
	}

	if newRoom.InProgress {
		client.sendMessage("spectate", json.RawMessage([]byte(`{"index": "`+strconv.Itoa(1)+`"}`)))
		newRoom.sendSmallMapUpdate()
		newRoom.sendGeneratedMapVisualsToClient(client)
		newRoom.sendBigUpdate()
	}
	client.IsSpectator = true
	newRoom.sendPlayerStatuses()

	// Broadcast the updated rooms list to everyone
	sendRoomsUpdateToAll()
}

// hostClient includes hosts who watch a game without occupying a player seat.
func (room *Room) hostClient() *Client {
	for _, members := range [][]*Client{room.Players, room.Spectators} {
		for _, client := range members {
			if client != nil && client.BotModel == "" && client.Username == room.HostUsername {
				return client
			}
		}
	}
	return nil
}

func (room *Room) setHostSpectator(client *Client, spectator bool) error {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	if room.InProgress {
		return errors.New("game already started")
	}
	if client.Username != room.HostUsername || !roomHasUser(room, client.Username) {
		return errors.New("only the room host can do that")
	}
	if roomHasSpectator(room, client.Username) == spectator {
		return nil
	}

	seat := roomPlayerIndex(room, client.Username)
	if !spectator {
		seat = -1
		for i, player := range room.Players {
			if player == nil {
				seat = i
				break
			}
		}
		if seat < 0 {
			return errors.New("room is full; remove a bot or wait for a free seat to play")
		}
	}

	// Persist the gateway role before changing the roster. A failed write must
	// leave both the host's seat and their spectator state unchanged.
	if err := client.worker.setMembership(context.Background(), client.Username, room.ID, spectator); err != nil {
		return fmt.Errorf("update host role: %w", err)
	}
	if spectator {
		room.Players[seat] = nil
		room.Spectators = append(room.Spectators, client)
	} else {
		for i, member := range room.Spectators {
			if member != nil && member.Username == client.Username {
				room.Spectators = append(room.Spectators[:i], room.Spectators[i+1:]...)
				break
			}
		}
		room.Players[seat] = client
		client.Index = seat
	}
	client.IsSpectator = spectator
	if err := client.sendMessages(&protocol.Binding{
		Action: protocol.BindingSet, Scope: protocol.ScopeRoom, RoomID: room.ID, IsSpectator: spectator,
	}); err != nil {
		// The role is already durable. Keep the command successful so the room
		// snapshot is saved; reconnect recovers a missed gateway notification.
		log.Printf("roomworker: publish host role: %v", err)
	}
	room.sendMapChoices()
	room.sendChoices()
	sendRoomsUpdateToAll()
	return nil
}

// updateHostAfterLeave runs with roomsMu held. Humans in either role keep a
// room alive; bots alone do not. Prefer a seated human when a host leaves.
func (room *Room) updateHostAfterLeave() {
	if room.hostClient() != nil {
		return
	}
	if room.generatedPending != nil {
		room.generatedPending.cancel()
		room.generatedPending = nil
	}
	room.HostUsername = ""
	for _, members := range [][]*Client{room.Players, room.Spectators} {
		for _, client := range members {
			if client != nil && client.BotModel == "" {
				room.HostUsername = client.Username
				room.sendSaveSelectionToHost()
				return
			}
		}
	}
	delete(rooms, room.ID)
	room.InProgress = false
	for _, player := range room.Players {
		if player != nil {
			player.Room = nil
		}
	}
}

func (room *Room) removeSpectator(username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	if room == nil {
		return
	}
	for i, spectator := range room.Spectators {
		if spectator == nil || spectator.Username != username {
			continue
		}
		room.Spectators = append(room.Spectators[:i], room.Spectators[i+1:]...)
		_ = spectator.clearRoom(messages.Message{Type: "lobby"})
		room.updateHostAfterLeave()
		room.sendMapChoices()
		room.sendChoices()
		sendRoomsUpdateToAll()
		return
	}
}

func (room *Room) removePlayer(username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	var client *Client

	if room == nil || room.Players == nil {
		return
	}
	for i, player := range room.Players {
		if player != nil && player.Username == username {
			client = player
			room.Players[i] = nil
			break
		}
	}
	if client == nil {
		return
	}
	if room.InProgress {
		room.sendStateMessage("A player left the game, game ended!")
		room.sendGameFinishedUpdate()
		room.InProgress = false
	}
	if client.BotModel == "" {
		_ = client.clearRoom(messages.Message{Type: "lobby"})
	} else {
		client.Room = nil
	}
	room.updateHostAfterLeave()
	room.sendMapChoices()
	room.sendChoices()
	sendRoomsUpdateToAll()
}

func (r *Room) MovePlayer(username string, direction string) bool {
	for i := range r.Players {
		// Find the player
		if r.Players[i] != nil && r.Players[i].Username == username {
			switch direction {
			case "up":
				// Swap with the slot above, even if it's nil
				if i > 0 {
					r.Players[i], r.Players[i-1] = r.Players[i-1], r.Players[i]
					sendRoomsUpdateToAll()
					return true
				}
			case "down":
				// Swap with the slot below, even if it's nil
				if i < len(r.Players)-1 {
					r.Players[i], r.Players[i+1] = r.Players[i+1], r.Players[i]
					sendRoomsUpdateToAll()
					return true
				}
			}
		}
	}
	return false
}

func (r *Room) MovePlayerWithIndex(username string, index int) bool {
	for i := range r.Players {
		if r.Players[i] != nil && r.Players[i].Username == username {
			r.Players[i], r.Players[index] = r.Players[index], r.Players[i]
			sendRoomsUpdateToAll()
			return true
		}
	}
	return false
}

func (room *Room) startLobbyGame(client *Client, roomID string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		client.sendError("That room does not exist.")
		return
	}

	if room.HostUsername != client.Username {
		client.sendError("Only the room host can start the game.")
		return
	}

	// Keep the waiting lobby intact until all validation and game creation pass.
	players, gameMap := room.Players, room.Map
	if IsGeneratedMapChoice(gameMap.Name) {
		nonNilPlayers := []*Client{}

		for _, player := range players {
			if player != nil {
				nonNilPlayers = append(nonNilPlayers, player)
			}
		}

		if len(nonNilPlayers) < 2 {
			client.sendError("Need at least 2 players to start the game.")
			return
		}

		if room.generatedSelected != nil && room.generatedSelected.PlayerCount != len(nonNilPlayers) {
			client.sendError("The player count changed; generate a new preview or return to automatic maps.")
			return
		}
		players = nonNilPlayers
		gameMap.Capacity = len(nonNilPlayers)
	} else {
		for _, player := range players {
			if player == nil {
				log.Println("The room is not full")
				client.sendError("The room is not full")
				return
			}
		}

		if len(players) < 2 {
			client.sendError("Need at least 2 players to start the game.")
			return
		}
	}

	playerNames := make([]string, len(players))
	for i, client := range players {
		playerNames[i] = client.displayName()
	}

	if room.saveId == -1 {
		raceKeys := []string{}
		traitKeys := []string{}
		powerKeys := []string{}

		for _, entry := range room.ExtensionChoices {
			if entry.ExtensionName == "Powers" {
				for _, entry2 := range entry.RaceChoices {
					if entry2.IsChecked {
						powerKeys = append(powerKeys, entry2.Choice)
					}
				}
				for _, entry2 := range entry.TraitChoices {
					if entry2.IsChecked {
						powerKeys = append(powerKeys, entry2.Choice)
					}
				}
				continue
			}
			for _, entry2 := range entry.RaceChoices {
				if entry2.IsChecked {
					raceKeys = append(raceKeys, entry2.Choice)
				}
			}
			for _, entry2 := range entry.TraitChoices {
				if entry2.IsChecked {
					traitKeys = append(traitKeys, entry2.Choice)
				}
			}
		}

		if room.hasBots() {
			if err := room.validateBotChoices(raceKeys, traitKeys, powerKeys); err != nil {
				client.sendError(err.Error() + ". Click Use bot rules in the lobby.")
				return
			}
			if min(len(raceKeys), len(traitKeys)) < len(playerNames) {
				client.sendError("Bots need at least one tribe per player")
				return
			}
		}
		if IsGeneratedMapChoice(gameMap.Name) {
			var generatedMap Map
			var err error
			if room.generatedSelected != nil {
				generatedMap, err = room.useSelectedGeneratedMap(len(playerNames))
			} else {
				generatedMap, err = GenerateRuntimeMapForPlayerCount(len(playerNames))
			}
			if err != nil {
				log.Println("Error generating runtime map:", err)
				client.sendError("Could not generate map: " + err.Error())
				return
			}

			gameMap = generatedMap
		}
		newstate, err := gamestate.New(playerNames, gameMap.Name, raceKeys, traitKeys, powerKeys)
		if err != nil {
			log.Println(err)
			client.sendError("Could not start game: " + err.Error())
			return
		}
		if room.hasBots() {
			if err := room.validateBotState(newstate); err != nil {
				client.sendError(err.Error())
				return
			}
		}
		room.Gamestate = *newstate
	} else {
		newstate, _, err := LoadGameState(room.saveId)
		if err != nil {
			log.Println("Error loading game", err)
			client.sendError("error loading game")
			return
		}
		if len(newstate.Players) != len(playerNames) {
			client.sendError("saved game player count does not match room")
			return
		}
		for i := range playerNames {
			newstate.Players[i].Name = playerNames[i]
		}
		if room.hasBots() {
			if err := room.validateBotState(newstate); err != nil {
				client.sendError("Saved game is incompatible with bots: " + err.Error())
				return
			}
		}
		room.Gamestate = *newstate
	}

	room.Players, room.Map = players, gameMap
	mapMap[gameMap.Name] = gameMap

	// Preview drafts are lobby-only; the active map now has its own durable record.
	room.clearGeneratedMapCustomization()
	// Mark the room as in-progress
	room.InProgress = true
	room.botError = ""
	room.botPaused = false
	room.botHistory, room.botFuture = nil, nil
	room.botNextMove = time.Now().Add(350 * time.Millisecond)

	id, err := SaveGameState(&room.Gamestate, 0, room.Map.Name)
	if err != nil {
		log.Println("Problem saving the game:", err)
		client.sendError("game started, but initial autosave failed")
	} else {
		room.autoSaveId = id
	}

	if id > 0 {
		for _, player := range room.saveOwners() {
			if err := AddGameIDToUser(player.Username, id); err != nil {
				log.Println("Problem adding new save for player:", err)
			}
		}
	}

	room.flushGameMessages()

	room.sendSmallMapUpdate()
	room.sendGeneratedMapVisuals()

	for i, client := range room.Players {
		client.sendMessage("index", json.RawMessage([]byte(`{"index": "`+strconv.Itoa(i)+`"}`)))
		client.Index = i

	}

	for _, spectator := range room.Spectators {
		spectator.sendMessage("spectate", json.RawMessage([]byte(`{"index": "`+strconv.Itoa(1)+`"}`)))
	}
	room.sendBigUpdate()

	sendRoomsUpdateToAll()
}

func (room *Room) sendToRoomPlayers(msg messages.Message) {
	if room == nil || currentWorker == nil {
		return
	}
	if err := currentWorker.publishRoom(context.Background(), room.ID, msg); err != nil {
		log.Printf("roomworker: publish room %s message %s: %v", room.ID, msg.Type, err)
	}
}

func (room *Room) sendNextPlayerReady() {
	if !room.InProgress || room.Gamestate.TurnInfo == nil {
		return
	}
	index := room.Gamestate.TurnInfo.PlayerIndex
	if index < 0 || index >= len(room.Players) || room.Players[index] == nil {
		return
	}
	room.Players[index].sendMessage("ready", nil)
}

func (room *Room) flushGameMessages() {
	if room == nil || len(room.Gamestate.Messages) == 0 {
		return
	}
	for _, msg := range room.Gamestate.Messages {
		if msg.Receivers == nil {
			room.sendStateMessage(msg.Content)
			continue
		}
		raw, _ := json.Marshal(map[string]string{"message": msg.Content})
		for _, index := range msg.Receivers {
			if index >= 0 && index < len(room.Players) && room.Players[index] != nil {
				room.Players[index].sendMessage("message", raw)
			}
		}
	}
	room.Gamestate.Messages = room.Gamestate.Messages[:0]
}

func (room *Room) sendStateMessage(message string) {
	room.sendToRoomPlayers(messages.Message{
		Type: "message",
		Data: json.RawMessage([]byte(`{"message": "` + message + `"}`)),
	})
}

func (room *Room) sendBigUpdate() {
	room.flushGameMessages()
	if room.InProgress {
		room.sendMegaUpdate()
		room.SendCoinUpdate()
		// room.sendNextPlayerReady()
	}
}

func (room *Room) sendTurnUpdate() {
	type StateInfo struct {
		TurnNumber   int    `json:"turnNumber"`
		PlayerNumber int    `json:"playerNumber"`
		Phase        string `json:"phase"`
	}
	stateInfo := StateInfo{
		TurnNumber:   room.Gamestate.TurnInfo.TurnIndex,
		PlayerNumber: room.Gamestate.TurnInfo.PlayerIndex,
		Phase:        room.Gamestate.TurnInfo.Phase.String(),
	}
	jsonData, _ := json.MarshalIndent(stateInfo, "", "  ")
	room.sendToRoomPlayers(messages.Message{Type: "turnupdate", Data: jsonData})
}

func (room *Room) sendPlayerUpdate() {
	type Tribe struct {
		Race  string `json:"race"`
		Trait string `json:"trait"`
	}
	type PieceStack struct {
		Type     string `json:"type"`
		Amount   int    `json:"amount"`
		IsActive bool   `json:"isActive"`
	}
	type Player struct {
		Name          string       `json:"name"`
		ActiveTribe   Tribe        `json:"activeTribe"`
		PassiveTribes []Tribe      `json:"passiveTribes"`
		PieceStacks   []PieceStack `json:"pieceStacks"`
	}

	playerInfo := []Player{}
	for i, p := range room.Gamestate.Players {
		var playerData Player
		if p == nil {
			continue
		}
		playerData.Name = room.Players[i].displayName()
		if p.ActiveTribe != nil {
			playerData.ActiveTribe = Tribe{
				Race:  string(p.ActiveTribe.Race),
				Trait: string(p.ActiveTribe.Trait),
			}
		} else {
			playerData.ActiveTribe = Tribe{Race: "", Trait: ""}
		}
		for _, tribe := range p.PassiveTribes {
			playerData.PassiveTribes = append(playerData.PassiveTribes, Tribe{
				Race:  string(tribe.Race),
				Trait: string(tribe.Trait),
			})
		}
		for _, stack := range p.PieceStacks {
			playerData.PieceStacks = append(playerData.PieceStacks, PieceStack{
				Type:     stack.Type,
				Amount:   stack.Amount,
				IsActive: *copyOrDefault(stack.Tribe, true),
			})
		}
		playerInfo = append(playerInfo, playerData)
	}
	jsonData, _ := json.MarshalIndent(playerInfo, "", "  ")
	room.sendToRoomPlayers(messages.Message{Type: "playerupdate", Data: jsonData})
}

func (room *Room) sendTileUpdate(tileID string) {
	type PieceStack struct {
		Type     string `json:"type"`
		Amount   int    `json:"amount"`
		IsActive bool   `json:"isActive"`
	}

	// Define a structure to include both tileID and its stacks
	type TileUpdate struct {
		TileID string       `json:"tileID"`
		Stacks []PieceStack `json:"stacks"`
	}

	// Create the TileUpdate object with tileID and stacks
	tileUpdate := TileUpdate{
		TileID: tileID,
		Stacks: []PieceStack{},
	}
	tile := room.Gamestate.TileList[tileID]
	for _, stack := range tile.PieceStacks {
		tileUpdate.Stacks = append(tileUpdate.Stacks, PieceStack{
			Type:     stack.Type,
			Amount:   stack.Amount,
			IsActive: *copyOrDefault(stack.Tribe, tile.CheckPresence() == gamestate.Active),
		})
	}

	// Marshal the combined structure into JSON
	jsonData, err := json.MarshalIndent(tileUpdate, "", "  ")
	if err != nil {
		log.Printf("roomworker: marshal tile update: %v", err)
		return
	}

	// Send the message with the new combined structure
	room.sendToRoomPlayers(messages.Message{Type: "tileupdate", Data: jsonData})
}
func (room *Room) sendAllTileUpdate() {
	type PieceStack struct {
		Type     string `json:"type"`
		Amount   int    `json:"amount"`
		IsActive bool   `json:"isActive"`
	}
	type TileUpdate struct {
		TileID string       `json:"tileID"`
		Stacks []PieceStack `json:"stacks"`
	}
	tileUpdates := []TileUpdate{}
	for _, tile := range room.Gamestate.TileList {
		tu := TileUpdate{
			TileID: tile.Id,
			Stacks: []PieceStack{},
		}
		for _, stack := range tile.PieceStacks {
			tu.Stacks = append(tu.Stacks, PieceStack{
				Type:     stack.Type,
				Amount:   stack.Amount,
				IsActive: *copyOrDefault(stack.Tribe, tile.CheckPresence() == gamestate.Active),
			})
		}
		tileUpdates = append(tileUpdates, tu)
	}
	jsonData, _ := json.MarshalIndent(tileUpdates, "", "  ")
	room.sendToRoomPlayers(messages.Message{Type: "alltileupdate", Data: jsonData})
}

func (room *Room) sendEntriesUpdate() {
	type Entry struct {
		Race      string `json:"race"`
		Trait     string `json:"trait"`
		CoinPile  int    `json:"coinCount"`
		PiecePile int    `json:"pieceCount"`
	}
	entries := []Entry{}
	for _, entry := range room.Gamestate.TribeList[:5] {
		entries = append(entries, Entry{
			Race:      string(entry.Race),
			Trait:     string(entry.Trait),
			CoinPile:  entry.CoinPile,
			PiecePile: entry.PiecePile,
		})
	}
	jsonData, _ := json.MarshalIndent(entries, "", "  ")
	room.sendToRoomPlayers(messages.Message{Type: "tribeentries", Data: jsonData})
}

func (room *Room) sendGameFinishedUpdate() {
	scores := []int{}
	for _, p := range room.Gamestate.Players {
		scores = append(scores, p.CoinPile)
	}
	jsonData, _ := json.MarshalIndent(scores, "", "  ")
	room.sendToRoomPlayers(messages.Message{Type: "gamefinished", Data: jsonData})
}

// Function that sends map updates to all players
func (room *Room) sendSmallMapUpdate() {
	type mapUpdateData struct {
		MapName  string  `json:"mapName"`
		OffSet   float64 `json:"offset"`
		FontSize int     `json:"fontSize"`
	}

	update := mapUpdateData{
		MapName:  room.Map.Name,
		OffSet:   room.Map.Offset,
		FontSize: room.Map.FontSize,
	}

	jsonData, _ := json.MarshalIndent(update, "", "  ")
	room.sendToRoomPlayers(struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}{
		Type: "smallmapupdate",
		Data: jsonData,
	})
}

// Function that sends map updates to all players
func (room *Room) sendMapUpdate() {
	type mapUpdateData struct {
		Picture string  `json:"picture"`
		OffSet  float64 `json:"offset"`
	}

	// Load the image from disk
	imgPath := room.Map.ImagePath("./assets/maps") // base path for images
	base64Img, err := getMapImageAsBase64(imgPath)
	if err != nil {
		// handle error; you might default to an empty string or log
		base64Img = ""
	}

	update := mapUpdateData{
		// Leave the path or stream data blank (or assign appropriately)
		Picture: base64Img,
		OffSet:  room.Map.Offset,
	}

	jsonData, _ := json.MarshalIndent(update, "", "  ")
	room.sendToRoomPlayers(struct {
		Type string          `json:"type"`
		Data json.RawMessage `json:"data"`
	}{
		Type: "mapupdate",
		Data: jsonData,
	})
}

func copyOrDefault(t *gamestate.Tribe, defaultVal bool) *bool {
	if t == nil {
		return &defaultVal
	}
	return &t.IsActive
}

func (room *Room) sendMegaUpdate() {
	if !room.InProgress {
		log.Println("Game has not started, sending MegaUpdate anyway might be pointless.")
	}

	type MegaUpdate struct {
		TurnInfo struct {
			TurnNumber   int    `json:"turnNumber"`
			PlayerNumber int    `json:"playerNumber"`
			Phase        string `json:"phase"`
		} `json:"turnInfo"`

		Players []struct {
			Name        string `json:"name"`
			ActiveTribe struct {
				Race  string `json:"race"`
				Trait string `json:"trait"`
			} `json:"activeTribe"`
			PassiveTribes []struct {
				Race  string `json:"race"`
				Trait string `json:"trait"`
			} `json:"passiveTribes"`
			PieceStacks []struct {
				Type     string `json:"type"`
				Amount   int    `json:"amount"`
				IsActive bool   `json:"isActive"`
			} `json:"pieceStacks"`
		} `json:"players"`

		TribeEntries []struct {
			Race      string `json:"race"`
			Trait     string `json:"trait"`
			CoinPile  int    `json:"coinCount"`
			PiecePile int    `json:"pieceCount"`
		} `json:"tribeEntries"`

		AllTiles []struct {
			TileID string `json:"tileID"`
			Stacks []struct {
				Type     string `json:"type"`
				Amount   int    `json:"amount"`
				IsActive bool   `json:"isActive"`
			} `json:"stacks"`
		} `json:"allTiles"`

		NextPlayerIndex int `json:"nextPlayerIndex"`
	}

	var mega MegaUpdate

	// -- Turn Info --
	mega.TurnInfo.TurnNumber = room.Gamestate.TurnInfo.TurnIndex
	mega.TurnInfo.PlayerNumber = room.Gamestate.TurnInfo.PlayerIndex
	mega.TurnInfo.Phase = room.Gamestate.TurnInfo.Phase.String()

	// -- Players --
	for _, p := range room.Gamestate.Players {
		if p == nil {
			continue
		}

		var playerData struct {
			Name        string `json:"name"`
			ActiveTribe struct {
				Race  string `json:"race"`
				Trait string `json:"trait"`
			} `json:"activeTribe"`
			PassiveTribes []struct {
				Race  string `json:"race"`
				Trait string `json:"trait"`
			} `json:"passiveTribes"`
			PieceStacks []struct {
				Type     string `json:"type"`
				Amount   int    `json:"amount"`
				IsActive bool   `json:"isActive"`
			} `json:"pieceStacks"`
		}
		// Make sure empty slices don't become null
		playerData.PassiveTribes = make([]struct {
			Race  string `json:"race"`
			Trait string `json:"trait"`
		}, 0)
		playerData.PieceStacks = make([]struct {
			Type     string `json:"type"`
			Amount   int    `json:"amount"`
			IsActive bool   `json:"isActive"`
		}, 0)

		playerData.Name = p.Name
		if p.ActiveTribe != nil {
			playerData.ActiveTribe.Race = string(p.ActiveTribe.Race)
			playerData.ActiveTribe.Trait = string(p.ActiveTribe.Trait)
		} else {
			playerData.ActiveTribe.Race = ""
			playerData.ActiveTribe.Trait = ""
		}

		for _, tribe := range p.PassiveTribes {
			playerData.PassiveTribes = append(playerData.PassiveTribes, struct {
				Race  string `json:"race"`
				Trait string `json:"trait"`
			}{
				Race:  string(tribe.Race),
				Trait: string(tribe.Trait),
			})
		}

		for _, stack := range p.PieceStacks {
			isActive := true
			if stack.Tribe != nil {
				isActive = stack.Tribe.IsActive
			}
			playerData.PieceStacks = append(playerData.PieceStacks, struct {
				Type     string `json:"type"`
				Amount   int    `json:"amount"`
				IsActive bool   `json:"isActive"`
			}{
				Type:     stack.Type,
				Amount:   stack.Amount,
				IsActive: isActive,
			})
		}

		mega.Players = append(mega.Players, playerData)
	}

	maxEntries := 5
	if len(room.Gamestate.TribeList) < maxEntries {
		maxEntries = len(room.Gamestate.TribeList)
	}
	// -- Tribe Entries --
	for _, entry := range room.Gamestate.TribeList[:maxEntries] {
		mega.TribeEntries = append(mega.TribeEntries, struct {
			Race      string `json:"race"`
			Trait     string `json:"trait"`
			CoinPile  int    `json:"coinCount"`
			PiecePile int    `json:"pieceCount"`
		}{
			Race:      string(entry.Race),
			Trait:     string(entry.Trait),
			CoinPile:  entry.CoinPile,
			PiecePile: entry.PiecePile,
		})
	}

	// -- All Tiles --
	for _, tile := range room.Gamestate.TileList {
		var tileData struct {
			TileID string `json:"tileID"`
			Stacks []struct {
				Type     string `json:"type"`
				Amount   int    `json:"amount"`
				IsActive bool   `json:"isActive"`
			} `json:"stacks"`
		}
		tileData.TileID = tile.Id
		// Make sure empty slices don't become null
		tileData.Stacks = make([]struct {
			Type     string `json:"type"`
			Amount   int    `json:"amount"`
			IsActive bool   `json:"isActive"`
		}, 0)

		for _, stack := range tile.PieceStacks {
			isActive := (tile.CheckPresence() != gamestate.Passive)
			if stack.Tribe != nil {
				isActive = stack.Tribe.IsActive
			}
			tileData.Stacks = append(tileData.Stacks, struct {
				Type     string `json:"type"`
				Amount   int    `json:"amount"`
				IsActive bool   `json:"isActive"`
			}{
				Type:     stack.Type,
				Amount:   stack.Amount,
				IsActive: isActive,
			})
		}
		mega.AllTiles = append(mega.AllTiles, tileData)
	}

	jsonData, _ := json.MarshalIndent(mega, "", "  ")
	room.sendToRoomPlayers(messages.Message{
		Type: "megaUpdate",
		Data: jsonData,
	})
}

func (room *Room) SendCoinUpdate() {
	for i, client := range room.Players {
		client.sendMessage("coins", json.RawMessage([]byte(`{"coins": `+strconv.FormatInt(int64(room.Gamestate.Players[i].CoinPile), 10)+`}`)))
	}
}

func (room *Room) sendGeneratedMapVisuals() {
	visuals, ok := GetGeneratedMapVisuals(room.Map.Name)
	if !ok {
		return
	}

	jsonData, err := json.Marshal(visuals)
	if err != nil {
		log.Println("failed to marshal generated map visuals:", err)
		return
	}

	room.sendToRoomPlayers(messages.Message{
		Type: "generatedmapvisuals",
		Data: jsonData,
	})
}

// Ordinary spectators do not own saves, but a watching host must be able to
// load their bot-only matches just like a host who occupies a player seat.
func (room *Room) saveOwners() []*Client {
	owners := make([]*Client, 0, len(room.Players)+1)
	for _, player := range room.Players {
		if player != nil && player.BotModel == "" {
			owners = append(owners, player)
		}
	}
	if roomHasSpectator(room, room.HostUsername) {
		if host := room.hostClient(); host != nil {
			owners = append(owners, host)
		}
	}
	return owners
}

func (room *Room) AutoSave() {
	if !room.InProgress && (room.Gamestate.TurnInfo == nil || room.Gamestate.TurnInfo.Phase != gamestate.GameFinished) {
		return
	}
	id, err := SaveGameState(&room.Gamestate, 0, room.Map.Name)
	if err != nil {
		log.Println("Problem saving the game:", err)
		return
	}
	oldID := room.autoSaveId
	room.autoSaveId = id
	for _, player := range room.saveOwners() {
		if oldID > 0 {
			_ = RemoveGameIDFromUser(player.Username, oldID)
		}
		if err := AddGameIDToUser(player.Username, id); err != nil {
			log.Println("Problem adding new autosave for player:", err)
		}
	}
	if oldID > 0 {
		_ = DeleteGameState(oldID)
	}
}

func (room *Room) RollBack(client *Client) {
	if room.Gamestate.TurnInfo.PlayerIndex != client.Index {
		client.sendError("Not able to roll back on someone else's turn!")
		return
	}
	state, _, err := LoadGameState(room.autoSaveId)
	if err != nil {
		client.sendError("Error rolling back")
		return
	}
	playerNames := make([]string, len(room.Players))
	for i, client := range room.Players {
		playerNames[i] = client.displayName()
	}
	for i := range playerNames {
		state.Players[i].Name = playerNames[i]
	}
	room.Gamestate = *state
	room.sendMegaUpdate()
}

func (room *Room) sendGeneratedMapVisualsToClient(client *Client) {
	visuals, ok := GetGeneratedMapVisuals(room.Map.Name)
	if !ok || client == nil {
		return
	}
	jsonData, err := json.Marshal(visuals)
	if err != nil {
		log.Println("failed to marshal generated map visuals:", err)
		return
	}
	client.sendMessage("generatedmapvisuals", jsonData)
}

func (room *Room) hasConnectedUsers() bool {
	for _, player := range room.Players {
		if player != nil && player.Connected {
			return true
		}
	}

	for _, spectator := range room.Spectators {
		if spectator != nil && spectator.Connected {
			return true
		}
	}

	return false
}
