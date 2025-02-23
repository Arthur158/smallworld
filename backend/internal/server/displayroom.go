package server;

import (
	"log"
	"github.com/google/uuid"
	"backend/internal/gamestate"
	"encoding/json"
	"strconv"
)

func createDisplayRoom(client *Client) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, ok := mapMap["map2players"]
	if !ok {
		log.Println("Problem logging map")
	}

	room := &Room{
		ID:           uuid.New().String(),
		Name:         "displayroom",
		HostUsername: client.Username,
		Players:      make([]*Client, 1), // Create a fixed-size slice with nil values
		InProgress:   false,
		Gamestate:    gamestate.GameState{},
		Map:          gameMap,
		saveId:	      -1,
		playerStatuses: []string{},
		IsDisplayRoom: true,
	}

	client.DisplayRoom = room
	room.Players[0] = client

	newstate, err := gamestate.New([]string{}, client.DisplayRoom.Map.Name)
	if err != nil {
		log.Println("error creating state")
	}
	client.DisplayRoom.Gamestate = *newstate

	room.sendMapChoices()
	room.sendSmallMapUpdate()
	room.sendMegaUpdate()
}

func (room *Room) LoadSave(client *Client, id int64) {
	_, mapName, _, err := LoadGameInfo(id)
	if err != nil {
		log.Println(err)
		return
	}
	ok := room.ChangeMap(mapName)
	room.sendSmallMapUpdate()
	newstate, _, err := LoadGameState(id)
	if err != nil {
		log.Println("Error loading game", err)
		client.sendError("error loading game")
		return
	}
	client.DisplayRoom.Gamestate = *newstate
	if !ok {
		client.sendError("Error changing map")
	}
	client.DisplayRoom.saveId = id
	room.sendMegaUpdate()
}
func (room *Room) LoadMap(client *Client, mapName string) {
	ok := room.ChangeMap(mapName)
	room.sendSmallMapUpdate()
	newstate, err := gamestate.New([]string{}, mapName)
	if err != nil {
		log.Println("Error loading game", err)
		client.sendError("error loading game")
		return
	}
	client.DisplayRoom.Gamestate = *newstate
	if !ok {
		client.sendError("Error changing map")
	}
	room.sendMegaUpdate()
}

func (room *Room) EndDisplayRoom() {
	client := room.Players[0]
	client.DisplayRoom = nil
	client.sendMessage("leavedisplayroom", json.RawMessage([]byte(`{"leavedisplayroom": ` + strconv.FormatInt(1, 10) + `}`)))
}

