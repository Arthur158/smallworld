package server
import (
	"log"
	"github.com/google/uuid"
	"encoding/json"
	"backend/internal/messages"
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
	}
	rooms[roomID] = newRoom

	// Mark this client as the host
	client.Username = username
	client.RoomID = roomID
	client.IsHost = true
	newRoom.Players = append(newRoom.Players, client)

	sendRoomsUpdateToAll()
	sendRoomUpdate(client)
}

func joinRoom(client *Client, roomID, username string) {
    roomsMu.Lock()
    defer roomsMu.Unlock()

    // If the client was already in a different room, remove them from that old room
    if client.RoomID != "" && client.RoomID != roomID {
        if oldRoom, ok := rooms[client.RoomID]; ok {
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
    }

    // Now try to join the new room
    newRoom, exists := rooms[roomID]
    if !exists {
        sendError(client, "That room does not exist.")
        return
    }

    if newRoom.InProgress {
        sendError(client, "That game is already in progress.")
        return
    }

    if len(newRoom.Players) >= newRoom.MaxPlayers {
        sendError(client, "Room is full.")
        return
    }

    // Update client's info to reflect the new room
    client.Username = username
    client.RoomID = roomID
    client.IsHost = false

    // Add client to the new room
    newRoom.Players = append(newRoom.Players, client)

    // Broadcast the updated rooms list to everyone
    sendRoomUpdate(client)
    sendRoomsUpdateToAll()
}

func startLobbyGame(client *Client, roomID string) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	room, exists := rooms[roomID]
	if !exists {
		sendError(client, "That room does not exist.")
		return
	}
	// if room.HostUsername != client.Username {
	// 	sendError(client, "Only the room host can start the game.")
	// 	return
	// }
	if len(room.Players) < 2 {
		sendError(client, "Need at least 2 players to start the game.")
		return
	}

	// Mark the room as in-progress
	room.InProgress = true
	log.Println("Starting game for room:", room.Name)
	close(client.Stop) // This will stop the `readMessages` loop
	startGame(room) // reuses the existing game logic, modified for multiple players

	// Update all clients about new room states
	sendRoomsUpdateToAll()
}


func sendRoomsUpdateToAll() {
	// Build a list of rooms that are not in progress or in progress (up to you)
	secondRoomsMu.Lock()
	defer secondRoomsMu.Unlock()

	var roomList []map[string]interface{}
	for _, r := range rooms {
		if !r.InProgress {
			roomList = append(roomList, map[string]interface{}{
				"id":         r.ID,
				"name":       r.Name,
				"players":    getRoomPlayers(r),
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
		if cli.RoomID == "" || (cli.RoomID != "" && !rooms[cli.RoomID].InProgress) {
			cli.Conn.WriteJSON(messages.Message{
				Type: "roomEntriesUpdate",
				Data: data,
			})
		}
	}
}

func sendRoomUpdate(client *Client) {
	// Build a list of rooms that are not in progress or in progress (up to you)
	secondRoomsMu.Lock()
	defer secondRoomsMu.Unlock()


	r, ok := rooms[client.RoomID]
	if !ok {
		log.Println("room doesnt exist")
		return
	}

	roomInfo := map[string]interface{}{
		"id":         r.ID,
		"name":       r.Name,
		"players":    getRoomPlayers(r),
		"maxPlayers": r.MaxPlayers,
		"creator":    r.HostUsername,
	}

	data, err := json.Marshal(roomInfo)
	if err != nil {
		log.Println("Error marshaling room list:", err)
		return
	}



	client.Conn.WriteJSON(messages.Message{
		Type: "roomUpdate",
		Data: data,
	})
}

func getRoomPlayers(r *Room) []string {
	var players []string
	for _, c := range r.Players {
		players = append(players, c.Username)
	}
	return players
}
