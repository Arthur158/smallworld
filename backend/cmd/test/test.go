package main

import (
	"log"
	"backend/internal/gamestate"
)

func main() {
	log.Println("Starting the gamestate...")

	// start := time.Now()
	state, err := gamestate.New(3)
	// duree := time.Since(start)
	// println("creating a game takes", duree)

	log.Println(err)

	// log.Println(state.GetTribeEntries())

	// start = time.Now()
	state.HandleTribeChoice(0, 0)
	// duree = time.Since(start)



	// for _, tile := range state.TileList {
	// 	log.Println(tile.Id)
	// 	log.Println(tile.IsEdge)
	// }

	// println(state.TurnInfo.Phase.String())
	// println(state.Players[0].ActiveTribe.Race)
	tribe := state.Players[0].ActiveTribe.Race

	// start = time.Now()
	state.HandleConquest("0", 0, string(tribe))
	// duree = time.Since(start)
	// println("Handling conquest takes", duree)
	// start = time.Now()
	state.HandleStartRedeployment(0)
	// duree = time.Since(start)
	// println("handling start redeployment takes", duree)
	log.Println(state.TileList["0"].PieceStacks)
	// start = time.Now()
	log.Println(state.HandleRedeploymentIn(0, "0", string(tribe)))
	// duree = time.Since(start)
	// println("handling redeployment takes", duree)
	log.Println(state.TileList["0"].PieceStacks)
	log.Println(state.Players[0].CoinPile)
	log.Println(state.HandleFinishTurn(0))
	log.Println(state.Players[0].CoinPile)

}
