package server

import (
	"encoding/json"
	"log"
	"backend/internal/messages"
	"github.com/gorilla/websocket"
	"fmt"
)


func readMessages(client *Client) {
	conn := client.Conn
	for {
		select {
		// Listen for the Stop signal
		case <-client.Stop:
			log.Println("Stop signal received for client:", client)
			return

		default:
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
			handleClientMessage(client, msg)
		}
	}
}

// -----------------------------------------------------------------------------
// CLIENT MESSAGE HANDLER
// -----------------------------------------------------------------------------

func handleClientMessage(client *Client, msg messages.Message) {
	log.Println(msg.Type)
	switch msg.Type {

	// -------------------------------------------------------------------------
	// ROOM LOBBY MESSAGES
	// -------------------------------------------------------------------------
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
		sendRoomUpdate(client)
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
		startLobbyGame(client, data.RoomID)

	// -------------------------------------------------------------------------
	// IN-GAME MESSAGES (existing logic)
	// -------------------------------------------------------------------------
	default:
		// Here you can handle in-game messages once the game has started, e.g.:
		if room, ok := rooms[client.RoomID]; ok && room.InProgress {

		} else {
			log.Println("Received unknown or in-game message type:", msg.Type)
		}
	}
}


func removeClient(client *Client) {
	connectedClientsMu.Lock()
	defer connectedClientsMu.Unlock()

	delete(connectedClients, client.Conn)

	// Also remove client from any room
	if client.RoomID != "" {
		roomsMu.Lock()
		defer roomsMu.Unlock()

		room, exists := rooms[client.RoomID]
		if exists {
			newPlayers := []*Client{}
			for _, c := range room.Players {
				if c.Conn != client.Conn {
					newPlayers = append(newPlayers, c)
				}
			}
			room.Players = newPlayers
			// If all players left, remove the room or handle as you wish
			if len(room.Players) == 0 {
				delete(rooms, room.ID)
			}
		}
	}

	sendRoomsUpdateToAll()
}

func sendToAll(conns []*websocket.Conn, msg messages.Message) {
	for _, c := range conns {
		c.WriteJSON(msg)
	}
}

func sendError(client *Client, errorMsg string) {
	errMsg := messages.Message{
		Type: "error",
		Data: json.RawMessage([]byte(fmt.Sprintf(`{"message": "%s"}`, errorMsg))),
	}
	client.Conn.WriteJSON(errMsg)
}

