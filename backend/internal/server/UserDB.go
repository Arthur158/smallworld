package server

import (
    "database/sql"
    "encoding/json"
    "errors"
    "fmt"
    "log"

    "golang.org/x/crypto/bcrypt"
)

// CreateUsersTable creates the users table with a savegameids column
func CreateUsersTable() {
    query := `
  CREATE TABLE IF NOT EXISTS users (
    id INTEGER PRIMARY KEY AUTOINCREMENT, 
    username TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    savegameids TEXT NOT NULL DEFAULT '[]'
  );`

    _, err := db.Exec(query)
    if err != nil {
        log.Fatal("Error creating users table:", err)
    }

    log.Println("Users table created successfully!")
}

// AddUser inserts a new user with an empty list of saved game IDs
func AddUser(username, password string) error {
    hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
    if err != nil {
        return err
    }

    query := "INSERT INTO users (username, password_hash, savegameids) VALUES (?, ?, ?);"
    _, err = db.Exec(query, username, string(hashedPassword), "[]")
    if err != nil {
        log.Println("Error inserting user:", err)
        return err
    }

    log.Println("User added successfully:", username)
    return nil
}

// AddGameIDToUser adds a game ID to the user's savegameids list
func AddGameIDToUser(username string, gameID int64) error {
    // Get current savegameids
    var saveGameIDsStr string
    query := "SELECT savegameids FROM users WHERE username = ?;"
    err := db.QueryRow(query, username).Scan(&saveGameIDsStr)
    if err != nil {
        return err
    }

    // Unmarshal JSON
    var saveGameIDs []int64
    if err := json.Unmarshal([]byte(saveGameIDsStr), &saveGameIDs); err != nil {
        return err
    }

    // Append new ID
    saveGameIDs = append(saveGameIDs, gameID)

    // Marshal back to JSON
    newSaveGameIDsStr, err := json.Marshal(saveGameIDs)
    if err != nil {
        return err
    }

    // Update database
    updateQuery := "UPDATE users SET savegameids = ? WHERE username = ?;"
    _, err = db.Exec(updateQuery, string(newSaveGameIDsStr), username)
    if err != nil {
        return err
    }

    log.Printf("Game ID %d added to user %s", gameID, username)
    return nil
}

// RemoveGameIDFromUser removes a game ID from the user's savegameids list
func RemoveGameIDFromUser(username string, gameID int64) error {
    // Get current savegameids
    var saveGameIDsStr string
    query := "SELECT savegameids FROM users WHERE username = ?;"
    err := db.QueryRow(query, username).Scan(&saveGameIDsStr)
    if err != nil {
        return err
    }

    // Unmarshal JSON
    var saveGameIDs []int64
    if err := json.Unmarshal([]byte(saveGameIDsStr), &saveGameIDs); err != nil {
        return err
    }

    // Find and remove ID
    found := false
    for i, id := range saveGameIDs {
        if id == gameID {
            saveGameIDs = append(saveGameIDs[:i], saveGameIDs[i+1:]...)
            found = true
            break
        }
    }

    if !found {
        return errors.New("game ID not found in user's save list")
    }

    // Marshal back to JSON
    newSaveGameIDsStr, err := json.Marshal(saveGameIDs)
    if err != nil {
        return err
    }

    // Update database
    updateQuery := "UPDATE users SET savegameids = ? WHERE username = ?;"
    _, err = db.Exec(updateQuery, string(newSaveGameIDsStr), username)
    if err != nil {
        return err
    }

    log.Printf("Game ID %d removed from user %s", gameID, username)
    return nil
}

// AuthenticateUser remains unchanged
func AuthenticateUser(username, password string) error {
    var storedHash string
    query := "SELECT password_hash FROM users WHERE username = ?;"

    err := db.QueryRow(query, username).Scan(&storedHash)
    if err != nil {
        if err == sql.ErrNoRows {
            return errors.New("user not found")
        }
        return err
    }

    err = bcrypt.CompareHashAndPassword([]byte(storedHash), []byte(password))
    if err != nil {
        return errors.New("incorrect password")
    }

    return nil
}

// GetUserSaveGameIDs retrieves the list of saved game IDs for a user
func GetUserSaveGameIDs(username string) ([]int64, error) {
    var saveGameIDsStr string
    query := "SELECT savegameids FROM users WHERE username = ?;"
    err := db.QueryRow(query, username).Scan(&saveGameIDsStr)
    if err != nil {
        if err == sql.ErrNoRows {
            return nil, errors.New("user not found")
        }
        return nil, err
    }

    var saveGameIDs []int64
    if err := json.Unmarshal([]byte(saveGameIDsStr), &saveGameIDs); err != nil {
        return nil, fmt.Errorf("failed to parse save game IDs: %v", err)
    }

    log.Printf("Retrieved %d save game IDs for user %s", len(saveGameIDs), username)
    return saveGameIDs, nil
}
