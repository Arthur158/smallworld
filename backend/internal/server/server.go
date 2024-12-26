package server

import (
	"log"
	"net/http"
	"sync"
	"encoding/json"
	"strconv"
	"fmt"

	"github.com/gorilla/websocket"
	"backend/internal/gamestate"
	"backend/internal/messages"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins
    },
}

var connectedClients = make(map[*websocket.Conn]*Client)
var connectedClientsMu sync.Mutex

// Keep track of all rooms
var rooms = make(map[string]*Room)
var roomsMu sync.Mutex
var secondRoomsMu sync.Mutex

func Start() {
	http.HandleFunc("/ws", handleWebSocket)
	log.Println("WebSocket server is running on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("WebSocket upgrade error:", err)
		return
	}

	client := &Client{
		Conn:     conn,
		Stop:	  make(chan struct{}),
		Username: "",  // Will be set when user sends 'createRoom' or 'joinRoom'
		RoomID:   "",
		IsHost:   false,
	}

	connectedClientsMu.Lock()
	connectedClients[conn] = client
	connectedClientsMu.Unlock()

	sendRoomsUpdateToAll()

	log.Println("New client connected.")

	// Start goroutine to read messages from this connection
	go func(c *Client) {
		readMessages(c) // This will block until readMessages completes

		// After readMessages exits, close the connection
		c.Conn.Close()
		log.Println("Connection closed for client:", c)
	}(client)
}

func startGame(room *Room) {
	// Collect the connections from the room
	var players []*websocket.Conn
	names := []string{}

	for _, client := range room.Players {
		players = append(players, client.Conn)
		names = append(names, client.Username)
	}

	// Defer closures for each connection
	defer func() {
		for _, c := range room.Players {
			c.Conn.Close()
		}
	}()

	// EXAMPLE: create new game state with # of players = len(players)
	state, err := gamestate.New(len(players))
	if err != nil {
		log.Println("Error creating game:", err)
		sendToAll(players, messages.Message{
			Type: "error",
			Data: json.RawMessage([]byte(`{"message": "Could not create game"}`)),
		})
		return
	}

	// BROADCAST/UTILITY FUNCTIONS
	var mu sync.Mutex
	sendToAllFunc := func(msg messages.Message) {
		for _, conn := range players {
			mu.Lock()
			err := conn.WriteJSON(msg)
			mu.Unlock()
			if err != nil {
				log.Println("Error sending message:", err)
			}
		}
	}
	sendStateMessage := func(message string) {
		sendToAllFunc(messages.Message{
			Type: "message",
			Data: json.RawMessage([]byte(`{"message": "` + message + `"}`)),
		})
	}

	// Send initial states and updates
	sendTurnUpdate := func() {
		type StateInfo struct {
			TurnNumber   int    `json:"turnNumber"`
			PlayerNumber int    `json:"playerNumber"`
			Phase        string `json:"phase"`
		}
		stateInfo := StateInfo{
			TurnNumber:   state.TurnInfo.TurnIndex,
			PlayerNumber: state.TurnInfo.PlayerIndex,
			Phase:        state.TurnInfo.Phase.String(),
		}
		jsonData, _ := json.MarshalIndent(stateInfo, "", "  ")
		sendToAllFunc(messages.Message{Type: "turnupdate", Data: jsonData})
	}

	sendPlayerUpdate := func() {
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
		for i, p := range state.Players {
			var playerData Player
			playerData.Name = names[i]
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
		sendToAllFunc(messages.Message{Type: "playerupdate", Data: jsonData})
	}

	sendTileUpdate := func(tileID string) {

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
		tile := state.TileList[tileID]
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
		sendToAllFunc(messages.Message{Type: "tileupdate", Data: jsonData})
	}

	sendAllTileUpdate := func() {
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
		for _, tile := range state.TileList {
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
		sendToAllFunc(messages.Message{Type: "alltileupdate", Data: jsonData})
	}

	sendEntriesUpdate := func() {
		type Entry struct {
			Race      string `json:"race"`
			Trait     string `json:"trait"`
			CoinPile  int    `json:"coinpile"`
			PiecePile int    `json:"piecepile"`
		}
		entries := []Entry{}
		for _, entry := range state.TribeList[:5] {
			entries = append(entries, Entry{
				Race:      string(entry.Race),
				Trait:     string(entry.Trait),
				CoinPile:  entry.CoinPile,
				PiecePile: entry.PiecePile,
			})
		}
		jsonData, _ := json.MarshalIndent(entries, "", "  ")
		sendToAllFunc(messages.Message{Type: "tribeentries", Data: jsonData})
	}

	sendGameFinishedUpdate := func() {
		scores := []int{}
		for _, p := range state.Players {
			scores = append(scores, p.CoinPile)
		}
		jsonData, _ := json.MarshalIndent(scores, "", "  ")
		sendToAllFunc(messages.Message{Type: "gamefinished", Data: jsonData})
	}

	sendTurnUpdate()
	sendPlayerUpdate()
	sendAllTileUpdate()
	sendEntriesUpdate()
	sendToAllFunc(messages.Message{Type: "gamestarted"})

	// Send each player their index
	var wg sync.WaitGroup
	for index, conn := range players {
		conn.WriteJSON(messages.Message{
			Type: "index",
			Data: json.RawMessage([]byte(`{"index": "` + strconv.Itoa(index) + `"}`)),
		})
		wg.Add(1)
		go handlePlayerConnection(
			conn,
			state,
			index,
			names,
			sendToAllFunc,
			sendStateMessage,
			sendPlayerUpdate,
			sendEntriesUpdate,
			sendTileUpdate,
			sendAllTileUpdate,
			sendTurnUpdate,
			sendGameFinishedUpdate,
			&wg,
		)
	}

	wg.Wait()
	log.Println("Game ended in room:", room.Name)
}

func handlePlayerConnection(
	conn *websocket.Conn,
	state *gamestate.GameState,
	index int,
	names []string,
	sendToAll func(msg messages.Message),
	sendStateMessage func(message string),
	sendPlayerUpdate func(),
	sendEntriesUpdate func(),
	sendTileUpdate func(tileID string),
	sendAllTileUpdate func(),
	sendTurnUpdate func(),
	sendGameFinishedUpdate func(),
	wg *sync.WaitGroup,
) {
	defer wg.Done()

	var mu sync.Mutex

	sendError := func(message string) {
		conn.WriteJSON(messages.Message{Type: "error", Data: json.RawMessage([]byte(`{"message": "` + message + `"}`))})	
	}

	//Handling functions
	handleTribePick := func(msg messages.Message) {
		var pickData struct {
			PickIndex int `json:"pickIndex"`
		}
		if err := json.Unmarshal([]byte(msg.Data), &pickData); err != nil {
			sendError("Invalid choice data")
			return
		}

		if err := state.HandleTribeChoice(index, pickData.PickIndex); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTurnUpdate()
			sendEntriesUpdate()
		}
	}

	handleAbandonment := func(msg messages.Message) {
		var abandonmentData struct {
			TileID string `json:"tileId"`
		}
		if err := json.Unmarshal([]byte(msg.Data), &abandonmentData); err != nil {
			sendError("Invalid abandon data")
			return
		}

		if err := state.HandleAbandonment(index, abandonmentData.TileID); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTileUpdate(abandonmentData.TileID)
		}
	}

	handleConquest := func(msg messages.Message) {
		var conquestData struct {
			TileID             string `json:"tileId"`
			AttackingStackType string `json:"attackingStackType"`
		}

		if err := json.Unmarshal([]byte(msg.Data), &conquestData); err != nil {
			sendError("Invalid conquest data")
			return
		}

		if err := state.HandleConquest(conquestData.TileID, index, conquestData.AttackingStackType); err != nil {
			sendError(err.Error())
			sendTurnUpdate()
		} else {
			sendPlayerUpdate()
			sendTileUpdate(conquestData.TileID)
			sendTurnUpdate()
		}
	}

	handleStartRedeployment := func() {
		if err := state.HandleStartRedeployment(index); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTurnUpdate()
			sendAllTileUpdate()
		}
	}

	handleRedeploymentIn := func (msg messages.Message) {
		var deployData struct {
			TileID          string `json:"tileId"`
			StackType	string `json:"stackType"`
		}

		if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
			sendError("Invalid deploy data")
			return
		}

		if err := state.HandleRedeploymentIn(index, deployData.TileID, deployData.StackType); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTileUpdate(deployData.TileID)
			sendTurnUpdate()
		}
	}

	handleRedeploymentOut := func (msg messages.Message) {
		var deployData struct {
		    TileID    string `json:"tileId"`
		    StackType string `json:"stackType"`
		}

		if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
			sendError("Invalid deploy data")
			return
		}

		if err := state.HandleRedeploymentOut(index, deployData.TileID, deployData.StackType); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTileUpdate(deployData.TileID)
			sendTurnUpdate()
		}
	}

	handleRedeploymentThrough := func (msg messages.Message) {
		var deployData struct {
			TileFromID          string `json:"tileFromId"`
			TileToID          string `json:"tileToId"`
			StackType	string `json:"stackType"`
		}

		if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
			sendError("Invalid deploy data")
			return
		}


		if err := state.HandleRedeploymentOut(index, deployData.TileFromID, deployData.StackType); err != nil {
			sendError(err.Error())
		} else if err := state.HandleRedeploymentIn(index, deployData.TileToID, deployData.StackType); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTileUpdate(deployData.TileFromID)
			sendTileUpdate(deployData.TileToID)
			sendTurnUpdate()
		}
	}


	handleFinishTurn := func () {
		if err := state.HandleFinishTurn(index); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTurnUpdate()
			sendAllTileUpdate()

			pointsList := state.Players[index].PointsEachTurn
			sendStateMessage(
			    fmt.Sprintf(
				"player %s made %d points this turn",
				names[index],
				pointsList[len(pointsList) - 1]-pointsList[len(pointsList) - 2],
			    ),
			)

			if state.TurnInfo.Phase == gamestate.GameFinished {
				sendGameFinishedUpdate()
			} 
		}
	}

	handleDecline := func () {
		if err := state.HandleDecline(index); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTurnUpdate()
			sendAllTileUpdate()

			pointsList := state.Players[index].PointsEachTurn
			sendStateMessage(
			    fmt.Sprintf(
				"player %s made %d points this turn",
				names[index],
				pointsList[len(pointsList) - 1]-pointsList[len(pointsList) - 2],
			    ),
			)

			if state.TurnInfo.Phase == gamestate.GameFinished {
				sendGameFinishedUpdate()
			} 
		}
	}

	// New Main loop
	for {
		log.Println("waiting for msg")
		var msg messages.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Player disconnected:", err)
			sendStateMessage("Other player disconnected")
			return
		}
		log.Println(msg.Type)

		mu.Lock()
		switch msg.Type {
		case "tribepick":
			handleTribePick(msg)
		case "abandonment":
			handleAbandonment(msg)
		case "Conquest":
			handleConquest(msg)
		case "startredeployment":
			handleStartRedeployment()
		case "deploymentin":
			handleRedeploymentIn(msg)
		case "deploymentout":
			handleRedeploymentOut(msg)
		case "deploymentthrough":
			handleRedeploymentThrough(msg)
		case "finishturn":
			handleFinishTurn()
		case "decline":
			handleDecline()
		default:
			sendError("Unknown message type")
		}
		mu.Unlock()
	}
}

