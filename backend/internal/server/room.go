package server

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"encoding/json"
	"log"
	"strconv"
	"time"
	"sync"
	"github.com/google/uuid"
)

type Room struct {
	ID           string
	Name         string
	HostUsername string
	Players      []*Client
	InProgress   bool
	Gamestate    gamestate.GameState
	mu	     sync.Mutex
	Map	     Map
	saveId	     int64
	autoSaveId   int64
	playerStatuses []string
	IsDisplayRoom  bool
}

func createRoom(client *Client, roomName, username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, ok := mapMap["map2players"]
	if !ok {
		log.Println("Problem logging map")
	}

	room := &Room{
		ID:           uuid.New().String(),
		Name:         roomName,
		HostUsername: username,
		Players:      make([]*Client, gameMap.Capacity), // Create a fixed-size slice with nil values
		InProgress:   false,
		Gamestate:    gamestate.GameState{},
		Map:          gameMap,
		saveId:	      -1,
		playerStatuses: []string{},
	}
	rooms[room.ID] = room

	for i := range(gameMap.Capacity) {
		room.Players[i] = nil
	}

	client.Room = room
	room.Players[0] = client

	sendRoomsUpdateToAll()
	client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": "` + room.ID + `"}`)))

	room.sendMapChoices()
}

func (room *Room) sendPlayerStatuses() {
		playerStatusesJSON, err := json.Marshal(room.playerStatuses)
		if err != nil {
			log.Println("Error marshalling playerStatuses:", err)
			return
		}

		room.sendToRoomPlayers(messages.Message{Type: "playerStatuses", Data: json.RawMessage(playerStatusesJSON)})
}

func (room *Room) sendMapChoices() {
	mapChoices := []string{}
	for key := range(mapMap) {
		mapChoices = append(mapChoices, key)
	}

	var host *Client
	for i, client := range(room.Players) {
		if client != nil && client.Username == room.HostUsername {
			host = room.Players[i]
			break
		}
	}

	message := map[string]interface{}{
		"mapChoices": mapChoices, // This ensures it's an empty list if no maps exist
	}

	messageJSON, err := json.Marshal(message)
	if err != nil {
		host.sendMessage("error", json.RawMessage(`{"error": "Failed to marshal message"}`))
		return
	}

	host.sendMessage("mapChoices", json.RawMessage(messageJSON))
}

// ChangeSize modifies the room size while preserving existing players
func (room *Room) ChangeMap(newMap string) bool {

	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, ok := mapMap[newMap]
	if !ok {
		log.Println("Problem logging map")
		return false
	}

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
			if player != nil {
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
	client.Room = newRoom

	// Notify client about successful join
	client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": "` + newRoom.ID + `"}`)))
	client.Room.sendPlayerStatuses()

	// Broadcast the updated rooms list to everyone
	sendRoomsUpdateToAll()
}

func (room *Room) removePlayer(username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	var client Client

	if room.Players == nil {
		log.Println("room uninitialized or something")
		return
	}

	for i, player := range room.Players {
		if player != nil && player.Username == username {
			client = *player
			room.Players[i] = nil
		}
	}
	if room.HostUsername == client.Username && len(room.Players) != 0 {
		for _, player := range room.Players {
			if player != nil {
				room.HostUsername = player.Username
			}
		}
	}
	if room.InProgress {
		room.sendStateMessage("A player left the game, game ended!") // in the future replace with a bot
		// also allow for players to juggle between multiple games in the future.
		room.sendGameFinishedUpdate()
		room.InProgress = false
	}
	// If all players left, remove the room or handle as you wish
	count := 0
	for _, player := range(room.Players) {
		if player != nil {
			count += 1
		}
	}
	if count == 0 {
		delete(rooms, room.ID)
	}

	client.Room = nil
	client.sendMessage("lobby", nil)

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

	for _, player := range room.Players {
		if player == nil {
			log.Println("The room is not full")
			client.sendError("The room is not full")
			return
		}
	}

	room, exists := rooms[roomID]
	if !exists {
		client.sendError("That room does not exist.")
		return
	}
	if room.HostUsername != client.Username {
		client.sendError("Only the room host can start the game.")
		return
	}
	if len(room.Players) < 2 {
		client.sendError("Need at least 2 players to start the game.")
		return
	}

	playerNames := make([]string, len(room.Players))
	for i, client := range(room.Players) {
		playerNames[i] = client.Username
		if client.DisplayRoom != nil {
			client.DisplayRoom.EndDisplayRoom()
		}
	}

	if room.saveId == -1 {
		newstate, err := gamestate.New(playerNames, client.Room.Map.Name)
		if err != nil {
			log.Println("error creating state")
		}
		client.Room.Gamestate = *newstate
	} else {
		newstate, _, err := LoadGameState(room.saveId)
		for i := range(playerNames) {
			newstate.Players[i].Name = playerNames[i]
		}
		if err != nil {
			log.Println("Error loading game", err)
			client.sendError("error loading game")
			return
		}
		client.Room.Gamestate = *newstate
	}


	// Mark the room as in-progress
	room.InProgress = true
	
	id, err := SaveGameState(&room.Gamestate, 0, room.Map.Name)
	if err != nil {
		log.Println("Problem saving the game")
	}
	room.autoSaveId = id

	for _, client := range(room.Players) {
		err = AddGameIDToUser(client.Username, id)
		if err != nil {
			log.Println("Problem adding new save for player")
		}
	}

	go func() {
		for {
			// Lock to safely access Messages
			// room.mu.Lock()

			if len(room.Gamestate.Messages) > 0 {
				log.Println("Sending messages to players...")
				
				// Send messages to all players
				for _, msg := range room.Gamestate.Messages {
					log.Println("Message:", msg)
					room.sendStateMessage(msg)
				}

				// Clear messages after sending (without setting it to nil)
				room.Gamestate.Messages = room.Gamestate.Messages[:0]
			}

			// room.mu.Unlock()

			// Sleep to avoid busy-waiting
			time.Sleep(100 * time.Millisecond)
		}
	}()

	room.sendSmallMapUpdate()

	for i, client := range room.Players {
		client.sendMessage("index", json.RawMessage([]byte(`{"index": "` + strconv.Itoa(i) + `"}`)))
		client.Index = i

	}
	room.sendBigUpdate()

	sendRoomsUpdateToAll()
}

func (room *Room) sendToRoomPlayers (msg messages.Message) {
	for _, player := range room.Players {
		if player != nil {
			room.mu.Lock()
			err := player.Conn.WriteJSON(msg)
			room.mu.Unlock()
			if err != nil {
				log.Println("Error sending message:", err)
			}
		}
	}
}

func (room *Room) sendNextPlayerReady() {
	if !room.InProgress {
		log.Println("Game has not started")
		return
	}
	player := room.Players[room.Gamestate.TurnInfo.PlayerIndex]
	if player == nil {
		log.Println("Problem with the player")
		return
	}
	room.mu.Lock()
	err := player.Conn.WriteJSON(messages.Message{Type: "ready"})
	room.mu.Unlock()
	if err != nil {
		log.Println("Error sending message:", err)
	}
}

func (room *Room) sendStateMessage (message string) {
	room.sendToRoomPlayers(messages.Message{
		Type: "message",
		Data: json.RawMessage([]byte(`{"message": "` + message + `"}`)),
	})
}



func (room *Room) sendBigUpdate() {
	if room.InProgress {
		room.sendMegaUpdate()
		room.sendNextPlayerReady()
	}
	// if (room.InProgress) {
	// 	room.sendAllTileUpdate()
	// 	room.sendTurnUpdate()
	// 	room.sendPlayerUpdate()
	// 	room.sendEntriesUpdate()
	// 	room.sendNextPlayerReady()
	// }
	// sendRoomsUpdateToAll()
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

func (room *Room) sendPlayerUpdate () {
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
		playerData.Name = room.Players[i].Username
		if p.HasActiveTribe {
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

func (room *Room) sendTileUpdate (tileID string) {
	type PieceStack struct {
		Type     string `json:"type"`
		Amount   int    `json:"amount"`
		IsActive bool   `json:"isActive"`
	}

	// Define a structure to include both tileID and its stacks
	type TileUpdate struct {
		TileID string      `json:"tileID"`
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
		IsActive: *copyOrDefault(stack.Tribe, tile.Presence == gamestate.Active),
	})
	}


	// Marshal the combined structure into JSON
	jsonData, err := json.MarshalIndent(tileUpdate, "", "  ")
	if err != nil {
		log.Fatal("Error marshaling tile update:", err)
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
			IsActive: *copyOrDefault(stack.Tribe, tile.Presence == gamestate.Active),
			})
		}
		tileUpdates = append(tileUpdates, tu)
	}
	jsonData, _ := json.MarshalIndent(tileUpdates, "", "  ")
	room.sendToRoomPlayers(messages.Message{Type: "alltileupdate", Data: jsonData})
	}

func (room *Room) sendEntriesUpdate () {
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

func (room *Room) sendGameFinishedUpdate () {
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
		MapName string `json: "mapName"`
		OffSet float64 `json:"offset"`
		FontSize int `json:"fontSize`
	}

	update := mapUpdateData{
		MapName: room.Map.Name,
		OffSet: room.Map.Offset,
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
		Picture string    `json:"picture"`
		OffSet float64 `json:"offset"`
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
		OffSet: room.Map.Offset,
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
			Name          string `json:"name"`
			ActiveTribe   struct {
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
			Name          string `json:"name"`
			ActiveTribe   struct {
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
		if p.HasActiveTribe {
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

	// -- Tribe Entries --
	for _, entry := range room.Gamestate.TribeList[:5] {
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
			isActive := (tile.Presence == gamestate.Active)
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

func (room *Room) AutoSave() {
	if !room.InProgress {
		log.Println("game not started")
		return
	}
	id, err := SaveGameState(&room.Gamestate, 0, room.Map.Name)
	if err != nil {
		log.Println("Problem saving the game")
	}
	for _, client := range(room.Players) {
		err := RemoveGameIDFromUser(client.Username, room.autoSaveId)
		if err != nil {
			log.Println("Problem removing old save")
		}
	}

	DeleteGameState(room.autoSaveId)
	room.autoSaveId = id

	for _, client := range(room.Players) {
		err = AddGameIDToUser(client.Username, id)
		if err != nil {
			log.Println("Problem adding new save for player")
		}
	}
}

func (room *Room) RollBack(client *Client) {
	if room.Gamestate.TurnInfo.PlayerIndex != client.Index {
		client.sendError("Not able to roll back on someone else's turn!")
	}
	state, _, err := LoadGameState(room.autoSaveId)
	if err != nil {
		client.sendError("Error rolling back")
	}
	room.Gamestate = *state
	room.sendMegaUpdate()
}
