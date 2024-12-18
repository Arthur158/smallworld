package main

import (
	"log"
	"backend/internal/gamestate"
)

func main() {
	log.Println("Starting the gamestate...")

	state, err := gamestate.New(3)

	log.Println(err)

	log.Println(state.GetTribeEntries())

}
