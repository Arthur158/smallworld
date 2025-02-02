package server

import (
	"log"
	"golang.org/x/crypto/bcrypt"
	"database/sql"
	"errors"
)

func CreateUsersTable() {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id INTEGER PRIMARY KEY AUTOINCREMENT, 
		username TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL
	);`

	_, err := db.Exec(query)
	if err != nil {
		log.Fatal("Error creating users table:", err)
	}

	log.Println("Users table created successfully!")
}

func AddUser(username, password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	query := "INSERT INTO users (username, password_hash) VALUES (?, ?);"

	_, err = db.Exec(query, username, string(hashedPassword))
	if err != nil {
		log.Println("Error inserting user:", err)
		return err
	}

	log.Println("User added successfully:", username)
	return nil
}

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
