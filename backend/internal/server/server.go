package server

import (
	"log"
	"net/http"
	"sync"
	"encoding/json"

	"github.com/gorilla/websocket"
	"backend/internal/gamestate"
	"backend/internal/messages"
)

var upgrader = websocket.Upgrader{
    CheckOrigin: func(r *http.Request) bool {
        return true // Allow all origins
    },
}
var waitingConnection *websocket.Conn
var waitingMutex sync.Mutex

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

	waitingMutex.Lock()
	if waitingConnection == nil {
		// First player waits for another player
		waitingConnection = conn
		waitingMutex.Unlock()
		log.Println("A player is waiting for an opponent...")
		return
	}

	// Second player connects, start the game
	player1 := waitingConnection
	player2 := conn
	waitingConnection = nil
	waitingMutex.Unlock()

	log.Println("Two players connected, starting the game...")
	go startGame(player1, player2)
}

func startGame(c1, c2 *websocket.Conn) {
	defer c1.Close()
	defer c2.Close()

	players := []*websocket.Conn{c1, c2}

	var mu sync.Mutex

	// Send a message to all players
	sendToAll := func(msg messages.Message) {
		for _, conn := range players {
			conn.WriteJSON(msg)
		}
	}

	state, err := gamestate.New(len(players))

	if err != nil {
		log.Println("Error creating game", err)
		sendToAll(messages.Message{Type: "error", Data: "Could not create game"})
	}

	// Notify players that the game has started
	jsonData, err := json.MarshalIndent(state.Players, "", "  ")
	for index, conn := range players {
		conn.WriteJSON(messages.Message{Type: "index", Data: string(index)})
	}
	sendToAll(messages.Message{Type: "info", Data: "Game has started!"})
	if err != nil {
		log.Fatal("Error marshaling players:", err)
	}
	sendToAll(messages.Message{Type: "playerupdate", Data: string(jsonData)})
	players[state.TurnInfo.PlayerIndex].WriteJSON(messages.Message{Type: "tribechoice"})

	var wg sync.WaitGroup

	for index, conn := range players {
		wg.Add(1)
		go handlePlayerConnection(conn, state, index, &wg, &mu, sendToAll)
	}

	// Block until all player goroutines have finished
	wg.Wait()
	log.Println("Game ended: All players disconnected.")
}

func handlePlayerConnection(conn *websocket.Conn, state *gamestate.GameState, index int, wg *sync.WaitGroup, mu *sync.Mutex, sendToAll func(msg messages.Message)) {
	defer wg.Done()

	// Helper functions
	sendError := func(message string) {
		conn.WriteJSON(messages.Message{Type: "error", Data: message})
	}

	sendStateMessage := func(message string) {
		sendToAll(messages.Message{Type: "state", Data: message})
	}

	sendPlayerUpdate := func() {
		jsonData, err := json.MarshalIndent(state.Players, "", "  ")
		if err != nil {
			log.Fatal("Error marshaling players:", err)
		}
		sendToAll(messages.Message{Type: "playerupdate", Data: string(jsonData)})
	}

	sendTileUpdate := func(tileID string) {
		jsonData, err := json.MarshalIndent(state.GetTileStacks(tileID), "", "  ")
		if err != nil {
			log.Fatal("Error marshaling tile stacks:", err)
		}
		sendToAll(messages.Message{Type: "tileupdate", Data: string(jsonData)})
	}

	sendTurnUpdate := func() {
		jsonData, err := json.MarshalIndent(state.TurnInfo, "", "  ")
		if err != nil {
			log.Fatal("Error marshaling turn info:", err)
		}
		sendToAll(messages.Message{Type: "turnupdate", Data: string(jsonData)})
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
		}
	}

	handleRedeploymentIn := func (msg messages.Message) {
		var deployData struct {
			TileID          string `json:"tileId"`
			stackType	string `json:"stackType"`
		}

		if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
			sendError("Invalid deploy data")
			return
		}

		if err := state.HandleRedeploymentIn(index, deployData.TileID, deployData.stackType); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTileUpdate(deployData.TileID)
			sendTurnUpdate()
		}
	}

	handleRedeploymentOut := func (msg messages.Message) {
		var deployData struct {
			TileID          string `json:"tileId"`
			stackType	string `json:"stackType"`
		}

		if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
			sendError("Invalid deploy data")
			return
		}

		if err := state.HandleRedeploymentOut(index, deployData.TileID, deployData.stackType); err != nil {
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
			stackType	string `json:"stackType"`
		}

		if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
			sendError("Invalid deploy data")
			return
		}

		if err := state.HandleRedeploymentOut(index, deployData.TileFromID, deployData.stackType); err != nil {
			sendError(err.Error())
		} else if err := state.HandleRedeploymentIn(index, deployData.TileToID, deployData.stackType); err != nil {
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
		}
	}

	handleDecline := func () {
		if err := state.HandleDecline(index); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTurnUpdate()
		}
	}

	// Main loop
	for {
		var msg messages.Message
		if err := conn.ReadJSON(&msg); err != nil {
			log.Println("Player disconnected:", err)
			sendStateMessage("Other player disconnected")
			return
		}

		mu.Lock()
		switch msg.Type {
		case "tribepick":
			handleTribePick(msg)
		case "abandonment":
			handleAbandonment(msg)
		case "conquest":
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

