package server

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
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
		Username: "",  // Will be set when user sends 'createRoom' or 'joinRoom'
		Room: nil,
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

