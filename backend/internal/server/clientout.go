package server

import (
	"encoding/json"
	"log"
	"backend/internal/messages"
	"fmt"
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
			Username  string `json:"username"`
			MaxPlayers int   `json:"maxPlayers"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling createRoom data:", err)
			return
		}
		createRoom(client, data.RoomName, data.Username, data.MaxPlayers)

	case "requestrefresh":
		sendRoomsUpdateToAll()

	case "joinRoom":
		var data struct {
			RoomID   string `json:"roomId"`
			Username string `json:"username"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling joinRoom data:", err)
			return
		}
		joinRoom(client, data.RoomID, data.Username)

	case "startGame":
		var data struct {
			RoomID string `json:"roomId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling startGame data:", err)
			return
		}
		client.Room.startLobbyGame(client, data.RoomID)

	default:
			log.Println("Received unknown or in-game message type:", msg.Type)
	}
}

func removeClient(client *Client) {
	connectedClientsMu.Lock()
	defer connectedClientsMu.Unlock()

	delete(connectedClients, client.Conn)

	// Also remove client from any room
	if client.Room != nil {
		room := client.Room
		roomsMu.Lock()
		defer roomsMu.Unlock()

		newPlayers := []*Client{}
		for _, player := range room.Players {
			if player.Username != client.Username {
				newPlayers = append(newPlayers, player)
			}
		}
		room.Players = newPlayers
		// If all players left, remove the room or handle as you wish
		if len(room.Players) == 0 {
			delete(rooms, room.ID)
		}
	}

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


