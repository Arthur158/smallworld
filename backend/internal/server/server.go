package server

import (
	"log"
	"net/http"
	"sync"
	"encoding/json"
	"strconv"

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

	names := []string{"ulysse", "achille"}

	players := []*websocket.Conn{c1, c2}

	var mu sync.Mutex
	var localMu sync.Mutex

	// Send a message to all players
	sendToAll := func(msg messages.Message) {

		for _, conn := range players {
			localMu.Lock()
			err := conn.WriteJSON(msg)
			localMu.Unlock()
			if err != nil {
				log.Println("err sending mess", err)
			}

		}
	}

	state, err := gamestate.New(len(players))

	if err != nil {
		log.Println("Error creating game", err)
		sendToAll(messages.Message{Type: "error", Data: json.RawMessage([]byte(`{"message": "Could not create game"}`))})
	}

	var wg sync.WaitGroup

	for index, conn := range players {
		wg.Add(1)
		go handlePlayerConnection(conn, state, index, &wg, &mu, sendToAll, names)
	}

	// Block until all player goroutines have finished
	wg.Wait()
	log.Println("Game ended: All players disconnected.")
}

func handlePlayerConnection(conn *websocket.Conn, state *gamestate.GameState, index int, wg *sync.WaitGroup, mu *sync.Mutex, sendToAll func(msg messages.Message), names []string) {
	defer wg.Done()

	// Helper functions
	sendError := func(message string) {
		conn.WriteJSON(messages.Message{Type: "error", Data: json.RawMessage([]byte(`{"message": "` + message + `"}`))})	
	}

	sendStateMessage := func(message string) {
		sendToAll(messages.Message{Type: "state", Data: json.RawMessage([]byte(`{"message": "` + message + `"}`))})
	}
	type Tribe struct {
		Race  string `json:"race"`
		Trait string `json:"trait"`
	}

	sendPlayerUpdate := func() {

		// Player represents the player object.
		type Player struct {
			Name	      string     `json:"name"`
			ActiveTribe   Tribe       `json:"activeTribe"`   // Pointer to allow null value
			PassiveTribes []Tribe      `json:"passiveTribes"` // List of passive tribes
			PieceStacks   []gamestate.PieceStack `json:"pieceStacks"`
		}

		playerInfo := []Player {}
		for i, player := range state.Players {
			var playerData Player
			if player.ActiveTribe != nil {
				playerData.ActiveTribe = Tribe{Race: string(player.ActiveTribe.Race), Trait: string(player.ActiveTribe.Trait)}
			} else {
				playerData.ActiveTribe = Tribe{Race: "", Trait: ""}
			}
			playerData.Name = names[i]			
			for _, tribe := range player.PassiveTribes {
				playerData.PassiveTribes = append(playerData.PassiveTribes, Tribe{Race: string(tribe.Race), Trait: string(tribe.Trait)})
			}
			playerData.PieceStacks = player.PieceStacks
			playerInfo = append(playerInfo, playerData)
		}

		jsonData, err := json.MarshalIndent(playerInfo, "", "  ")
		if err != nil {
			log.Fatal("Error marshaling players:", err)
		}
		sendToAll(messages.Message{Type: "playerupdate", Data: jsonData})
	}

	sendEntriesUpdate := func() {
		type Entry struct {
			Race string `json:race`
			Trait string `json:trait`
			CoinPile int      `json:"coinpile"` // List of passive tribes
			PiecePile   int `json:"piecepile"`
		}

		entries := []Entry {}

		for _, entry := range state.GetTribeEntries() {
			entries = append(entries, Entry{Race: string(entry.Tribe.Race), Trait: string(entry.Tribe.Trait), CoinPile: entry.CoinPile, PiecePile: entry.PiecePile})
		}

		jsonData, err := json.MarshalIndent(entries, "", "  ")
		if err != nil {
			log.Fatal("Error marshaling tribe entries:", err)
		}
		sendToAll(messages.Message{Type: "tribeentries", Data: jsonData})
	}

	sendTileUpdate := func(tileID string) {
		// Define a structure to include both tileID and its stacks
		type TileUpdate struct {
			TileID string      `json:"tileID"`
			Stacks interface{} `json:"stacks"`
		}

		// Create the TileUpdate object with tileID and stacks
		tileUpdate := TileUpdate{
			TileID: tileID,
			Stacks: state.GetTileStacks(tileID),
		}

		// Marshal the combined structure into JSON
		jsonData, err := json.MarshalIndent(tileUpdate, "", "  ")
		if err != nil {
			log.Fatal("Error marshaling tile update:", err)
		}

		// Send the message with the new combined structure
		sendToAll(messages.Message{Type: "tileupdate", Data: jsonData})
	}

	sendAllTileUpdate := func() {
		// Define a structure to include both tileID and its stacks
		type TileUpdate struct {
			TileID string      `json:"tileID"`
			Stacks interface{} `json:"stacks"`
		}

		// Create the TileUpdate object with tileID and stacks
		tileUpdates := []TileUpdate{}

		for _, tile := range state.TileList {
			tileUpdates = append(tileUpdates, TileUpdate{
				tile.Id,
				tile.PieceStacks,
			})
		}

		// Marshal the combined structure into JSON
		jsonData, err := json.MarshalIndent(tileUpdates, "", "  ")
		if err != nil {
			log.Fatal("Error marshaling tile update:", err)
		}

		// Send the message with the new combined structure
		sendToAll(messages.Message{Type: "alltileupdate", Data: jsonData})
	}

	sendTurnUpdate := func() {
		type StateInfo struct {
			TurnNumber int `json:"turnNumber"`
			PlayerNumber int `json:"playerNumber"`
			Phase string `json:"phase`
}
		stateInfo := StateInfo{
			TurnNumber: state.TurnInfo.TurnIndex,
			PlayerNumber: state.TurnInfo.PlayerIndex,
			Phase : state.TurnInfo.Phase.String(),
		}

		jsonData, err := json.MarshalIndent(stateInfo, "", "  ")
		if err != nil {
			log.Fatal("Error marshaling turn info:", err)
		}
		sendToAll(messages.Message{Type: "turnupdate", Data: jsonData})
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
		log.Println(conquestData.TileID)
		log.Println(conquestData.AttackingStackType)

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
			sendTurnUpdate()
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
		log.Println("deployment out")
		log.Println(deployData.TileID)
		log.Println(deployData.StackType)

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

		log.Println(deployData.TileFromID)
		log.Println(deployData.StackType)

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
		}
	}

	handleDecline := func () {
		if err := state.HandleDecline(index); err != nil {
			sendError(err.Error())
		} else {
			sendPlayerUpdate()
			sendTurnUpdate()
			sendAllTileUpdate()
		}
	}

	mu.Lock()
	sendPlayerUpdate()
	sendTurnUpdate()
	sendEntriesUpdate()
	conn.WriteJSON(messages.Message{Type: "index", Data: json.RawMessage([]byte(`{"index": "` + strconv.Itoa(index) + `"}`))})
	mu.Unlock()


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

