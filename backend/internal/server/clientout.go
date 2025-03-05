package server

import (
	"encoding/json"
	"log"
	"backend/internal/messages"
	"fmt"
	"strconv"
	"github.com/gorilla/websocket"
)

type Client struct {
	Conn		*websocket.Conn
	Username	string
	IsAuthenticated bool
	Index		int
	Room		*Room
	DisplayRoom	*Room
	IsSpectator	bool
}

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
	log.Println(client)
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
	case "movement":
		client.handleMovement(msg)
	case "opponentaction":
		client.handleOppponentAction(msg)
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
		createRoom(client, data.RoomName, client.Username)
		client.sendUserSaves()
	case "enterdisplayroom":
		client.sendUserSaves()
		createDisplayRoom(client)
		client.DisplayRoom.sendMapChoices()
		client.sendMessage("displayroom", json.RawMessage([]byte(`{"index": ` + strconv.FormatInt(-1, 10) + `}`)))
	case "leaveroom":
		if client.IsSpectator {
			client.Room.removeSpectator(client.Username)
			return;
		}
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
	case "spectateRoom":
		var data struct {
			RoomID   string `json:"roomId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling joinRoom data:", err)
			return
		}
		spectateRoom(client, data.RoomID)

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
	case "logout":
		client.handleLogout()
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
	case "changeRoomMap":
		var data struct {
			RoomId string `json:"roomId"`
			NewMap string `json:"newMap"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling register data:", err)
			return
		}
		client.Room.ChangeMap(data.NewMap)
		client.Room.playerStatuses = []string{}
		client.Room.saveId = -1
		client.Room.sendPlayerStatuses()
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": ` + strconv.FormatInt(-1, 10) + `}`)))
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
		id, err := SaveGameState(&client.Room.Gamestate, client.Index, client.Room.Map.Name)
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
	case "rollback":
		client.Room.RollBack(client)
	case "loadgame":
		var data struct {
			SaveId int64 `json:"saveId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)

		if data.SaveId == -1 {
			client.Room.playerStatuses = []string{}
			client.Room.saveId = data.SaveId
			client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": ` + strconv.FormatInt(data.SaveId, 10) + `}`)))
			client.Room.sendPlayerStatuses()
			return
		} 
		index, mapName, playerStatuses, err := LoadGameInfo(data.SaveId)
		if err != nil {
			log.Println(err)
			return
		}
		ok := client.Room.ChangeMap(mapName)
		if !ok {
			client.sendError("Error changing map")
		}
		ok = client.Room.MovePlayerWithIndex(client.Username, index)
		if !ok {
			client.sendError("Error moving player")
		}
		client.Room.playerStatuses = playerStatuses
		client.Room.saveId = data.SaveId
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": ` + strconv.FormatInt(data.SaveId, 10) + `}`)))
		client.Room.sendPlayerStatuses()
	case "deletesave":
		var data struct {
			SaveId int64 `json:"saveId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)
		RemoveGameIDFromUser(client.Username, data.SaveId)
		client.sendUserSaves()
	case "loadgamedisplay":
		var data struct {
			SaveId int64 `json:"saveId"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)
		if client.DisplayRoom == nil {
			client.sendError("client not in a display room!")
		}
		client.DisplayRoom.LoadSave(client, data.SaveId)
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": ` + strconv.FormatInt(data.SaveId, 10) + `}`)))
	case "loadmapdisplay":
		var data struct {
			Name string `json:"mapName"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)
		if client.DisplayRoom == nil {
			client.sendError("client not in a display room!")
		}
		log.Println(data.Name)
		client.DisplayRoom.LoadMap(client, data.Name)
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": ` + strconv.FormatInt(-1, 10) + `}`)))
	case "leavedisplayroom":
		client.DisplayRoom.EndDisplayRoom()
		sendRoomsUpdateToAll()
	case "toggleRace":
		if client.Room == nil {
			return
		}
		if client.Username != client.Room.HostUsername {
			return
		}
		var data struct {
			ExtensionName string `json:"extensionName"`
			RaceChoice string `json:"raceChoice"`
			Checked bool `json:"checked"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)

		client.Room.toggleRace(data.ExtensionName, data.RaceChoice, data.Checked)
	case "toggleTrait":
		if client.Room == nil {
			return
		}
		if client.Username != client.Room.HostUsername {
			return
		}
		var data struct {
			ExtensionName string `json:"extensionName"`
			TraitChoice string `json:"traitChoice"`
			Checked bool `json:"checked"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)

		client.Room.toggleTrait(data.ExtensionName, data.TraitChoice, data.Checked)
	case "toggleExtension":
		if client.Room == nil {
			return
		}
		if client.Username != client.Room.HostUsername {
			return
		}
		var data struct {
			ExtensionName string `json:"extensionName"`
			Checked bool `json:"checked"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)

		client.Room.toggleExtension(data.ExtensionName, data.Checked)
	case "toggleAll":
		if client.Room == nil {
			return
		}
		if client.Username != client.Room.HostUsername {
			return
		}
		var data struct {
			Checked bool `json:"checked"`
		}
		if err := json.Unmarshal(msg.Data, &data); err != nil {
			log.Println("Error unmarshalling loadGame data:", err)
			client.sendError("Error unmarshalling game")
			log.Println("Raw Data:", string(msg.Data)) // Debug log
			return
		}
		log.Println("Successfully parsed:", data)

		client.Room.toggleAll(data.Checked)
	default:
		log.Println("Received unknown or in-game message type:", msg.Type)
	}
}

func (client *Client) handleLogout() {
	delete(nameSet, client.Username)
	client.Username = ""
	client.IsAuthenticated = false
	client.sendMessage("unauth", json.RawMessage([]byte(`{"name": "` + client.Username + `"}`)))
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
				client.Index = i
				client.sendMessage("index", json.RawMessage([]byte(`{"index": "` + strconv.Itoa(i) + `"}`)))
			}
		}
		if room.InProgress {
			room.sendToRoomPlayers(messages.Message{Type: "gamestarted"})
			room.sendSmallMapUpdate()
			room.sendBigUpdate()
		}
		client.Room.sendMapChoices()
		client.sendUserSaves()
		client.Room.sendChoices()
		client.Room.sendPlayerStatuses()
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
		summary, err := LoadSummary(id)
		if err != nil {
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
	if client.Conn == nil {
		log.Println("Client is not connected!")
		return
	}
	err := client.Conn.WriteJSON(errMsg)
	if err != nil {
		log.Println(err)
	}
}

func (client *Client) sendMessage (msgType string, msgData json.RawMessage) {
	errMsg := messages.Message{
		Type: msgType,
		Data: msgData	}
	if client == nil || client.Conn == nil {
		log.Println("Client is not connected!")
		return
	}
	err := client.Conn.WriteJSON(errMsg)
	if err != nil {
		log.Println(err)
	}
}

