package server

import (
	"database/sql"
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
	_ "modernc.org/sqlite" // SQLite driver
)

// WebSocket upgrader settings
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins (for testing; restrict in production)
	},
}

// Connected clients
var connectedClients = make(map[*websocket.Conn]*Client)
var nameSet = make(map[string]struct{})
var disconnectedUsers = make(map[string]*Room)
var connectedClientsMu sync.Mutex

// Rooms
var rooms = make(map[string]*Room)
var roomsMu sync.Mutex
var secondRoomsMu sync.Mutex

// Global DB connection
var db *sql.DB

// Initialize and start the WebSocket server
func Start() {
	// SQLite database file
	dbSource := "file:data.db?cache=shared&mode=rwc"

	var err error
	db, err = sql.Open("sqlite", dbSource)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer db.Close()

	// Verify the database connection
	err = db.Ping()
	if err != nil {
		log.Fatal("Database ping error:", err)
	}

	log.Println("Connected to SQLite database successfully!")

	// Create users table
	CreateUsersTable()

	// Start WebSocket server
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
		Username: "placeholder",  // Will be set when user sends 'createRoom' or 'joinRoom'
		IsAuthenticated: false,
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
