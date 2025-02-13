package server

import (
	"encoding/json"
	"log"
	"backend/internal/messages"
	"fmt"
	"strconv"
)


func readMessages(client *Client) {
	conn := client.Conn
	for {
		// Read message from the WebSocket connection
		_, messageBytes, err := conn.ReadMessage()
		if err != nil {
			// Client has disconnected or an error occurred
			removeClient(client)
			log.Println("Client disconnected:", err)
			return
		}

		// Parse the incoming message
		var msg messages.Message
		if err := json.Unmarshal(messageBytes, &msg); err != nil {
			log.Println("Invalid message format:", err)
			continue
		}

		// Handle the incoming message
		client.handleClientMessage(msg)
	}
}

// -----------------------------------------------------------------------------
// CLIENT MESSAGE HANDLER
// -----------------------------------------------------------------------------

func (client *Client) handleClientMessage(msg messages.Message) {
	switch msg.Type {

	// -------------------------------------------------------------------------
	// ROOM LOBBY MESSAGES
	// -------------------------------------------------------------------------
	case "tribepick":
		client.handleTribePick(msg)
	case "abandonment":
		client.handleAbandonment(msg)
	case "Conquest":
		client.handleConquest(msg)
	case "startredeployment":
		client.handleStartRedeployment()
	case "deploymentin":
		client.handleRedeploymentIn(msg)
	case "deploymentout":
		client.handleRedeploymentOut(msg)
	case "deploymentthrough":
		client.handleRedeploymentThrough(msg)
	case "finishturn":
		client.handleFinishTurn()
	case "decline":
		client.handleDecline()
	case "createRoom":
		var data struct {
			RoomName  string `json:"roomName"`
			MaxPlayers int   `json:"maxPlayers"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling createRoom data:", err)
			return
		}
		createRoom(client, data.RoomName, client.Username, data.MaxPlayers)
		client.sendUserSaves()

	case "leaveroom":
		client.Room.removePlayer(client.Username)
		sendRoomsUpdateToAll()
	case "requestrefresh":
		sendRoomsUpdateToAll()

	case "joinRoom":
		var data struct {
			RoomID   string `json:"roomId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling joinRoom data:", err)
			return
		}
		joinRoom(client, data.RoomID, client.Username)

	case "startGame":
		var data struct {
			RoomID string `json:"roomId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling startGame data:", err)
			return
		}
		client.Room.startLobbyGame(client, data.RoomID)

	case "register":
		var data struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling register data:", err)
			return
		}

		if err := AddUser(data.UserName, data.Password); err != nil {
			log.Println("Error adding user", err)
			client.sendError("error adding user")
			return
		}

		client.handleLogin(data.UserName, data.Password)
		sendRoomsUpdateToAll()
		
	case "login":
		var data struct {
			UserName string `json:"username"`
			Password string `json:"password"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling register data:", err)
			return
		}
		client.handleLogin(data.UserName, data.Password)
		sendRoomsUpdateToAll()
	case "moveUp":
		var data struct {
			RoomId string `json:"roomId"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling register data:", err)
			return
		}
		client.Room.MovePlayer(data.Username, "up")
		sendRoomsUpdateToAll()
	case "moveDown":
		var data struct {
			RoomId string `json:"roomId"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling register data:", err)
			return
		}
		client.Room.MovePlayer(data.Username, "down")
		sendRoomsUpdateToAll()
	case "changeRoomSize":
		var data struct {
			RoomId string `json:"roomId"`
			NewSize int `json:"newSize"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling register data:", err)
			return
		}
		client.Room.ChangeSize(data.NewSize)
	case "kickPlayer":
		var data struct {
			RoomId string `json:"roomId"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling register data:", err)
			return
		}
		client.Room.removePlayer(data.Username)
		sendRoomsUpdateToAll()
		
	case "savegame":
		id, err := SaveGameState(&client.Room.Gamestate, client.Index)
		if err != nil {
			log.Println("Error saving game", err)
			client.sendError("error saving game")
			return
		}
		err = AddGameIDToUser(client.Username, id)
		if err != nil {
			log.Println("Error adding game", err)
			client.sendError("error adding game")
			return
		}
		client.sendMessage("message", json.RawMessage([]byte(`{"message": "Game successfully saved"}`)))
	case "loadgame":
		var data struct {
			SaveId int64 `json:"saveId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)

		// here also need to pull map name and game size to change the room so that it corresponds to the saved game
		client.Room.saveId = data.SaveId
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": ` + strconv.FormatInt(data.SaveId, 10) + `}`)))
	default:
		log.Println("Received unknown or in-game message type:", msg.Type)
	}
}

func (client *Client) handleLogin(userName string, password string) {
	_, exists := nameSet[userName]
	if exists {
		log.Println("user tried to auth twice")
		client.sendError("user with that username already active")
		return
	}

	if err := AuthenticateUser(userName, password); err != nil {
		log.Println("Error adding user", err)
		return
	}

	client.Username = userName
	nameSet[client.Username] = struct{}{}
	client.IsAuthenticated = true
	client.sendMessage("auth", json.RawMessage([]byte(`{"name": "` + client.Username + `"}`)))

	room, exists := disconnectedUsers[client.Username]; if exists {
		client.Room = room
		delete(disconnectedUsers, client.Username)
		for i, player := range room.Players {
			if player != nil && player.Username == client.Username {
				room.Players[i] = client
				client.sendMessage("index", json.RawMessage([]byte(`{"index": "` + strconv.Itoa(i) + `"}`)))
			}
		}
		if room.InProgress {
			room.sendToRoomPlayers(messages.Message{Type: "gamestarted"})
		}
		room.sendSmallMapUpdate()
		room.sendBigUpdate()
		client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": "` + room.ID + `"}`)))
	}
}

func (client *Client) sendUserSaves() {
	saveIds, err := GetUserSaveGameIDs(client.Username)
	if err != nil {
		log.Println("unable to load save ids:", err)
		return
	}

	type SaveInfo struct {
		SaveID  int64  `json:"saveId"`
		Summary string `json:"summary"`
	}

	saves := []SaveInfo{{SaveID: -1, Summary: "New game"}}

	for _, id := range saveIds {
		// For each save ID, retrieve the summary from the database
		var summary string
		row := db.QueryRow("SELECT summary FROM game_states WHERE id = ?", id)
		if err := row.Scan(&summary); err != nil {
			log.Printf("Could not retrieve summary for save ID %d: %v\n", id, err)
			continue
		}

		saves = append(saves, SaveInfo{
			SaveID:  id,
			Summary: summary,
		})
	}

	// Marshal the slice of saves (each with { saveId, summary }) into JSON
	savesJSON, err := json.Marshal(map[string]interface{}{
		"saves": saves,
	})
	if err != nil {
		log.Println("unable to marshal saves:", err)
		return
	}

	// Send the message of type "loadSaves" with the save info
	client.sendMessage("loadSaves", json.RawMessage(savesJSON))
}

func removeClient(client *Client) {
	connectedClientsMu.Lock()
	defer connectedClientsMu.Unlock()

	delete(nameSet, client.Username)
	if (client.Room != nil) {
		disconnectedUsers[client.Username] = client.Room
	}

	delete(connectedClients, client.Conn)

	sendRoomsUpdateToAll()
}

func (client *Client) sendError (errorMsg string) {
	errMsg := messages.Message{
		Type: "error",
		Data: json.RawMessage([]byte(fmt.Sprintf(`{"message": "%s"}`, errorMsg))),
	}
	client.Conn.WriteJSON(errMsg)
}

func (client *Client) sendMessage (msgType string, msgData json.RawMessage) {
	errMsg := messages.Message{
		Type: msgType,
		Data: msgData	}
	client.Conn.WriteJSON(errMsg)
}

func sendRoomsUpdateToAll() {
	// Build a list of rooms that are not in progress or in progress (up to you)
	secondRoomsMu.Lock()
	defer secondRoomsMu.Unlock()

	var roomList []map[string]interface{}
	for _, r := range rooms {
		if !r.InProgress {
			playerNames := []string{}
			for _, player := range r.Players {
				if player != nil {
					playerNames = append(playerNames, player.Username)
				} else {
					playerNames = append(playerNames, "")
				}
			}
			roomList = append(roomList, map[string]interface{}{
				"id":         r.ID,
				"name":       r.Name,
				"players":    playerNames,
				"maxPlayers": r.MaxPlayers,
				"creator":  r.HostUsername,
				"capacity": r.MaxPlayers,
			})
		}
	}

	data, err := json.Marshal(roomList)
	if err != nil {
		log.Println("Error marshaling room list:", err)
		return
	}

	// Send to all clients that are NOT currently in a game or even all, depending on your logic
	for _, cli := range connectedClients {
		// Optionally skip if client is in a game
		if cli.Room == nil || (cli.Room != nil && !cli.Room.InProgress) {
			cli.Conn.WriteJSON(messages.Message{
				Type: "roomEntriesUpdate",
				Data: data,
			})
		}
	}
}


