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
		// sendToAll(messages.Message{Type: "error", Data: "Could not create game"})
	}

	// Notify players that the game has started
	jsonData, err := json.MarshalIndent(state.GetPlayers(), "", "  ")
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

		go func(player *websocket.Conn) {
			defer wg.Done()


			for {
				var msg messages.Message
				if err := player.ReadJSON(&msg); err != nil {
					log.Println("Player disconnected:", err)
					sendToAll(messages.Message{Type: "state", Data: "Other player disconnected"})
					return
				}

				mu.Lock()
				switch msg.Type {
				case "action":
					state.ApplyAction(msg.Data)
					actions := state.GetActions()
					sendToAll(messages.Message{Type: "state", Data: "Actions: " + actions[len(actions)-1]})
				case "tribepick":
					var pickData struct {
						pickIndex	int `json:"pickIndex"`
					}

					if err := json.Unmarshal([]byte(msg.Data), &pickData); err != nil {
						log.Println("Error parsing conquest message:", err)
						conn.WriteJSON(messages.Message{Type: "error", Data: "Invalid choice data"})
						return
					}

					if err := state.HandleTribeChoice(index, pickData.pickIndex);
					err != nil {
						log.Println("Error choosing tribe", err)
						conn.WriteJSON(messages.Message{Type: "error", Data: err.Error()})
					} else {
						jsonData, err := json.MarshalIndent(state.GetPlayers(), "", "  ")
						if err != nil {
							log.Fatal("Error marshaling players:", err)
						}
						sendToAll(messages.Message{Type: "playerupdate", Data: string(jsonData)})

						jsonData, err = json.MarshalIndent(state.TurnInfo, "", "  ")
						if err != nil {
							log.Fatal("Error marshaling players:", err)
						}
						sendToAll(messages.Message{Type: "turnupdate", Data: string(jsonData)})
					}

				case "conquest":
					var conquestData struct {
						TileID            string `json:"tileId"`
						AttackingStackType string `json:"attackingStackType"`
					    }

					// Parse the JSON data into conquestData
					if err := json.Unmarshal([]byte(msg.Data), &conquestData); err != nil {
						log.Println("Error parsing conquest message:", err)
						conn.WriteJSON(messages.Message{Type: "error", Data: "Invalid conquest data"})
						return
					}

					// Call the HandleConquest method with parsed parameters
					if err := state.HandleConquest(conquestData.TileID, index, conquestData.AttackingStackType); err != nil {
						log.Println("Error handling conquest:", err)
						conn.WriteJSON(messages.Message{Type: "error", Data: err.Error()})
					} else {
						jsonData, err := json.MarshalIndent(state.GetPlayers(), "", "  ")
						if err != nil {
							log.Fatal("Error marshaling players:", err)
						}
						sendToAll(messages.Message{Type: "playerupdate", Data: string(jsonData)})

						jsonData, err = json.MarshalIndent(state.GetTileStacks(conquestData.TileID), "", "  ")
						if err != nil {
							log.Fatal("Error marshaling players:", err)
						}
						sendToAll(messages.Message{Type: "tileupdate", Data: string(jsonData)})
						log.Println("conquest successful!:")
					}
				}
				mu.Unlock()

			}
		}(conn)
	}


	// Block until all player goroutines have finished
	wg.Wait()
	log.Println("Game ended: All players disconnected.")
}
