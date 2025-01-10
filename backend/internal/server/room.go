package server

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"encoding/json"
	"log"
	"strconv"

	"github.com/google/uuid"
)

func createRoom(client *Client, roomName, username string, maxPlayers int) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	// Create a unique room ID
	roomID := uuid.New().String()
	newRoom := &Room{
		ID:           roomID,
		Name:         roomName,
		HostUsername: username,
		Players:      []*Client{},
		MaxPlayers:   maxPlayers,
		InProgress:   false,
		Gamestate:    gamestate.GameState{},
		Map: Map1,
	}
	rooms[roomID] = newRoom

	// Mark this client as the host
	client.Username = username
	client.Room = newRoom
	newRoom.Players = append(newRoom.Players, client)

	sendRoomsUpdateToAll()
	client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": "` + newRoom.ID + `"}`)))
}

func joinRoom(client *Client, roomID, username string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	// If the client was already in a different room, remove them from that old room
	oldRoom := client.Room
	if oldRoom != nil {
		oldRoom := client.Room
		var updatedPlayers []*Client
		for _, c := range oldRoom.Players {
			if c != client {
			    updatedPlayers = append(updatedPlayers, c)
			}
		}
		oldRoom.Players = updatedPlayers
		// If the old room becomes empty, optionally remove it
		if len(oldRoom.Players) == 0 {
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

	if len(newRoom.Players) >= newRoom.MaxPlayers {
		client.sendError("Room is full.")
		return
	}

	// Update client's info to reflect the new room
	client.Username = username
	client.Room = newRoom

	// Add client to the new room
	newRoom.Players = append(newRoom.Players, client)

	// Broadcast the updated rooms list to everyone
	client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": "` + newRoom.ID + `"}`)))
	sendRoomsUpdateToAll()
}

func (room *Room) removePlayer(client *Client) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	newPlayers := []*Client{}
	log.Println(room.Players)
	for _, player := range room.Players {
		if player.Username != client.Username {
			newPlayers = append(newPlayers, player)
		}
	}
	room.Players = newPlayers
	if room.HostUsername == client.Username && len(room.Players) != 0 {
		room.HostUsername = room.Players[0].Username
	}
	// If all players left, remove the room or handle as you wish
	if len(room.Players) == 0 {
		delete(rooms, room.ID)
	}

	client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": ""}`)))

}

func (room *Room) startLobbyGame(client *Client, roomID string) {
	log.Println(room.Map.Name)
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
	if len(room.Players) < 2 {
		client.sendError("Need at least 2 players to start the game.")
		return
	}

	// Mark the room as in-progress
	room.InProgress = true
	
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
	room.sendToRoomPlayers(messages.Message{Type: "gamestarted"})
	for i, client := range room.Players {
		client.sendMessage("index", json.RawMessage([]byte(`{"index": "` + strconv.Itoa(i) + `"}`)))
		client.Index = i

	}

	room.sendMapUpdate()
	room.sendEntriesUpdate()
	room.sendPlayerUpdate()
	room.sendAllTileUpdate()

	sendRoomsUpdateToAll()
}

func (room *Room) sendToRoomPlayers (msg messages.Message) {
	for _, player := range room.Players {
		room.mu.Lock()
		err := player.Conn.WriteJSON(msg)
		room.mu.Unlock()
		if err != nil {
			log.Println("Error sending message:", err)
		}
	}
}

func (room *Room) sendStateMessage (message string) {
	room.sendToRoomPlayers(messages.Message{
		Type: "message",
		Data: json.RawMessage([]byte(`{"message": "` + message + `"}`)),
	})
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




