package server

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"encoding/json"
	"log"
	"strconv"
	"time"

	"github.com/google/uuid"
)

func createRoom(client *Client, roomName, username string, maxPlayers int) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, ok := mapMap[maxPlayers]
	if !ok {
		log.Println("Problem logging map")
	}

	room := &Room{
		ID:           uuid.New().String(),
		Name:         roomName,
		HostUsername: username,
		Players:      make([]*Client, maxPlayers), // Create a fixed-size slice with nil values
		MaxPlayers:   maxPlayers,
		InProgress:   false,
		Gamestate:    gamestate.GameState{},
		Map:          gameMap,
	}
	rooms[room.ID] = room

	for i := range(maxPlayers) {
		room.Players[i] = nil
	}

	tempState, err := gamestate.New(len(room.Players), room.Map.Name)
	if err != nil {
		log.Println("Error creating game:", err)
		room.sendToRoomPlayers(messages.Message{
			Type: "error",
			Data: json.RawMessage([]byte(`{"message": "Could not create game"}`)),
		})
		return
	}
	room.Gamestate = *tempState


	// Mark this client as the host
	client.Username = username
	client.Room = room
	room.Players[0] = client

	sendRoomsUpdateToAll()
	client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": "` + room.ID + `"}`)))
}

// ChangeSize modifies the room size while preserving existing players
func (room *Room) ChangeSize(newSize int) {
	// Ensure the new size is between 2 and 5
	if newSize < 2 || newSize > 5 {
		log.Println("Invalid room size. Must be between 2 and 5.")
		return
	}

	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, ok := mapMap[newSize]
	if !ok {
		log.Println("Problem logging map")
	}

	room.Map = gameMap

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

	// Assign new max size
	room.MaxPlayers = newSize

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

	// Broadcast the updated rooms list to everyone
	sendRoomsUpdateToAll()
}

func (room *Room) removePlayer(username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()
	var client Client

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
					return true
				}
			case "down":
				// Swap with the slot below, even if it's nil
				if i < len(r.Players)-1 {
					r.Players[i], r.Players[i+1] = r.Players[i+1], r.Players[i]
					return true
				}
			}
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

	// Mark the room as in-progress
	room.InProgress = true
	
	room.sendToRoomPlayers(messages.Message{Type: "gamestarted"})
	for i, client := range room.Players {
		client.sendMessage("index", json.RawMessage([]byte(`{"index": "` + strconv.Itoa(i) + `"}`)))
		client.Index = i

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

	room.sendMapUpdate()
	room.sendEntriesUpdate()
	room.sendPlayerUpdate()
	room.sendAllTileUpdate()

	sendRoomsUpdateToAll()
}

func (room *Room) sendToRoomPlayers (msg messages.Message) {
	for _, player := range room.Players {
		log.Println(player)
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

func (room *Room) sendStateMessage (message string) {
	room.sendToRoomPlayers(messages.Message{
		Type: "message",
		Data: json.RawMessage([]byte(`{"message": "` + message + `"}`)),
	})
}

func (room *Room) sendBigUpdate() {
	if (room.InProgress) {
		room.sendMapUpdate()
		room.sendAllTileUpdate()
		room.sendTurnUpdate()
		room.sendPlayerUpdate()
		room.sendEntriesUpdate()
	}
	sendRoomsUpdateToAll()
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
				IsActive: true,
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
		IsActive: tile.OwningTribe.IsActive,
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
				IsActive: tile.Presence == gamestate.Active,
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
func (room *Room) sendMapUpdate() {
	type mapUpdateData struct {
		Picture string    `json:"picture"`
		Zones   []TileData `json:"zones"`
		OffSet float64 `json:"offset"`
	}

	zones := room.Map.populateMap()

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
		Zones:   zones,
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


