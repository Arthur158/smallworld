package server

import (
    "backend/internal/gamestate"
    "backend/internal/messages"
    "database/sql"
    "encoding/json"
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
    CreateGameStatesTable()

    gamestate.InitTraitMap()
    gamestate.InitRaceMap()

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
        Conn:            conn,
        Username:        "", // Will be set when user sends 'createRoom' or 'joinRoom'
        IsAuthenticated: false,
    }

    connectedClientsMu.Lock()
    connectedClients[conn] = client
    connectedClientsMu.Unlock()

    sendRoomsUpdateToAll()

    log.Println("New client connected.")

    // Start goroutine to read messages from this connection
    go func(c *Client) {
        readMessages(c)

        // After readMessages exits, close the connection
        c.Conn.Close()
        log.Println("Connection closed for client:", c)
    }(client)
}

func sendRoomsUpdateToAll() {
    // Build a list of rooms that are not in progress or in progress (up to you)
    secondRoomsMu.Lock()
    defer secondRoomsMu.Unlock()

    var roomList []map[string]interface{}
    var roomInProgressList []map[string]interface{}
    for _, r := range rooms {
        if !r.InProgress {
            playerNames := []string{}
            for _, player := range r.Players {
                if player != nil {
                    playerNames = append(playerNames, player.Username)
                } else {
                    playerNames = append(playerNames, "")
                }
            }
            roomList = append(roomList, map[string]interface{}{
                "id":         r.ID,
                "name":       r.Name,
                "players":    playerNames,
                "maxPlayers": r.Map.Capacity,
                "mapName":    r.Map.Name,
                "creator":    r.HostUsername,
            })
        } else {
            playerNames := []string{}
            for _, player := range r.Players {
                if player != nil {
                    playerNames = append(playerNames, player.Username)
                } else {
                    playerNames = append(playerNames, "")
                }
            }
            roomInProgressList = append(roomList, map[string]interface{}{
                "id":         r.ID,
                "name":       r.Name,
                "players":    playerNames,
                "maxPlayers": r.Map.Capacity,
                "mapName":    r.Map.Name,
                "creator":    r.HostUsername,
            })
        }
    }

    data, err := json.Marshal(roomList)
    if err != nil {
        log.Println("Error marshaling room list:", err)
        return
    }

    data2, err := json.Marshal(roomInProgressList)
    if err != nil {
        log.Println("Error marshaling room list:", err)
        return
    }

    // Send to all clients that are NOT currently in a game or even all, depending on your logic
    for _, cli := range connectedClients {
        // Optionally skip if client is in a game
        if cli.Room == nil || (cli.Room != nil && !cli.Room.InProgress) {
            if cli.Conn == nil {
                log.Println("Client is not connected!")
                break
            }
            cli.Conn.WriteJSON(messages.Message{
                Type: "roomEntriesUpdate",
                Data: data,
            })
            cli.Conn.WriteJSON(messages.Message{
                Type: "roomsInProgress",
                Data: data2,
            })
        }
    }
}
