package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"log"
	"strconv"
)

func createDisplayRoom(client *Client) {
	roomsMu.Lock()
	defer roomsMu.Unlock()

	gameMap, ok := mapMap["map2players"]
	if !ok {
		log.Println("Problem logging map")
	}

	roomID := client.CommandRoomID
	if roomID == "" {
		roomID = uuid.New().String()
	}
	room := &Room{
		ID:             roomID,
		Name:           "displayroom",
		HostUsername:   client.Username,
		Players:        make([]*Client, 1), // Create a fixed-size slice with nil values
		InProgress:     false,
		Gamestate:      gamestate.GameState{},
		Map:            gameMap,
		saveId:         -1,
		playerStatuses: []string{},
		IsDisplayRoom:  true,
	}

	rooms[room.ID] = room
	room.Players[0] = client
	if err := client.bindDisplay(room); err != nil {
		client.sendError("failed to enter display room")
		delete(rooms, room.ID)
		return
	}

	newstate, err := gamestate.New([]string{}, room.Map.Name, []string{}, []string{}, []string{})
	if err != nil {
		log.Println("error creating state:", err)
		client.sendError("error creating display state")
		return
	}
	room.Gamestate = *newstate

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
	if currentWorker != nil {
		_ = currentWorker.restoreGeneratedMap(context.Background(), mapName)
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
	newstate, err := gamestate.New([]string{}, mapName, []string{}, []string{}, []string{})
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
	if room == nil || len(room.Players) == 0 || room.Players[0] == nil {
		return
	}
	client := room.Players[0]
	delete(rooms, room.ID)
	data := json.RawMessage([]byte(`{"leavedisplayroom": ` + strconv.FormatInt(1, 10) + `}`))
	_ = client.clearDisplay(messages.Message{Type: "leavedisplayroom", Data: data})
}
