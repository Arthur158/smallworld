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
		log.Println(client.Username)
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

	case "leaveroom":
		client.Room.removePlayer(client)
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
		log.Println("here")
		log.Println(client.Room)
		sendRoomsUpdateToAll()

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
			if player.Username == client.Username {
				room.Players[i] = client
				client.sendMessage("index", json.RawMessage([]byte(`{"index": "` + strconv.Itoa(i) + `"}`)))
			}
		}
		if room.InProgress {
			room.sendToRoomPlayers(messages.Message{Type: "gamestarted"})
		}
		room.sendBigUpdate()
		client.sendMessage("roomid", json.RawMessage([]byte(`{"roomid": "` + room.ID + `"}`)))
	}


}

func removeClient(client *Client) {
	connectedClientsMu.Lock()
	defer connectedClientsMu.Unlock()

	delete(nameSet, client.Username)
	log.Println("name delete from nameset", client.Username)
	log.Println(client.Username)
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
				playerNames = append(playerNames, player.Username)
			}
			roomList = append(roomList, map[string]interface{}{
				"id":         r.ID,
				"name":       r.Name,
				"players":    playerNames,
				"maxPlayers": r.MaxPlayers,
				"creator":  r.HostUsername,
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


