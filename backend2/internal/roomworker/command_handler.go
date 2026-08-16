package roomworker

import (
	"backend/internal/messages"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"log"

	"github.com/redis/go-redis/v9"
)

func (w *Worker) HandleCommand(ctx context.Context, cmd protocol.Command) error {
	if cmd.Username == "" && cmd.Message.Type != "gatewaydisconnect" {
		return errors.New("command has no authenticated username")
	}

	// For every non-create command, rehydrate a room after worker restart before
	// looking up the logical client. Snapshot restoration also re-creates the
	// room's logical player objects.
	if cmd.Message.Type != "createRoom" && cmd.Message.Type != "enterdisplayroom" {
		if _, err := w.ensureRoomLoaded(ctx, cmd.RoomID); err != nil {
			return fmt.Errorf("load room: %w", err)
		}
	}

	client := w.logicalClient(cmd)
	if room := rooms[cmd.RoomID]; room != nil {
		if cmd.Scope == protocol.ScopeDisplay {
			client.DisplayRoom = room
		} else if roomHasUser(room, client.Username) {
			client.Room = room
			client.IsSpectator = roomHasSpectator(room, client.Username)
		}
	}

	switch cmd.Message.Type {
	case "createRoom":
		if err := w.requireNoOtherRoom(ctx, cmd.Username, cmd.RoomID); err != nil {
			return err
		}
		var data struct {
			RoomName string `json:"roomName"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid createRoom data")
		}
		createRoom(client, data.RoomName, cmd.Username)
		client.sendUserSaves()
		return nil

	case "enterdisplayroom":
		createDisplayRoom(client)
		client.sendUserSaves()
		client.sendMessage("displayroom", json.RawMessage([]byte(`{"index": -1}`)))
		return nil

	case "joinRoom":
		if err := w.requireNoOtherRoom(ctx, cmd.Username, cmd.RoomID); err != nil {
			return err
		}
		if rooms[cmd.RoomID] == nil {
			return errors.New("that room does not exist")
		}
		joinRoom(client, cmd.RoomID, cmd.Username)
		return nil

	case "spectateRoom":
		if err := w.requireNoOtherRoom(ctx, cmd.Username, cmd.RoomID); err != nil {
			return err
		}
		if rooms[cmd.RoomID] == nil {
			return errors.New("that room does not exist")
		}
		spectateRoom(client, cmd.RoomID)
		return nil

	case "reconnect":
		sendRoomsUpdateToAll()
		return w.handleReconnect(ctx, client, cmd)

	case "gatewaydisconnect":
		client.Connected = false
		client.GatewayID = ""
		client.ConnectionID = ""
		if cmd.Scope == protocol.ScopeDisplay && client.DisplayRoom != nil {
			roomID := client.DisplayRoom.ID
			delete(rooms, roomID)
			client.DisplayRoom = nil
			_ = w.redis.Del(ctx, roomSnapshotKey(roomID)).Err()
		}
		sendRoomsUpdateToAll()
		return nil

	case "leaveroom":
		if client.Room == nil {
			return errors.New("client not in a room")
		}
		if client.IsSpectator {
			client.Room.removeSpectator(client.Username)
		} else {
			client.Room.removePlayer(client.Username)
		}
		return nil

	case "leavedisplayroom":
		if client.DisplayRoom == nil {
			return errors.New("client not in a display room")
		}
		client.DisplayRoom.EndDisplayRoom()
		return nil

	case "startGame":
		room, err := requireNormalRoom(client)
		if err != nil {
			return err
		}
		room.startLobbyGame(client, cmd.RoomID)
		return nil

	case "moveUp", "moveDown":
		room, err := requireHost(client)
		if err != nil {
			return err
		}
		var data struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid move player data")
		}
		direction := "up"
		if cmd.Message.Type == "moveDown" {
			direction = "down"
		}
		if !room.MovePlayer(data.Username, direction) {
			return errors.New("could not move player")
		}
		return nil

	case "changeRoomMap":
		room, err := requireHost(client)
		if err != nil {
			return err
		}
		var data struct {
			NewMap string `json:"newMap"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid map data")
		}
		if !room.ChangeMap(data.NewMap) {
			return errors.New("error changing map")
		}
		room.playerStatuses = []string{}
		room.saveId = -1
		room.sendPlayerStatuses()
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": -1}`)))
		return nil

	case "kickPlayer":
		room, err := requireHost(client)
		if err != nil {
			return err
		}
		var data struct {
			Username string `json:"username"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid kick data")
		}
		if data.Username == "" {
			return errors.New("username is required")
		}
		room.removePlayer(data.Username)
		return nil

	case "toggleRace":
		room, err := requireHost(client)
		if err != nil {
			return err
		}
		var data struct {
			ExtensionName string `json:"extensionName"`
			RaceChoice    string `json:"raceChoice"`
			Checked       bool   `json:"checked"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid toggleRace data")
		}
		room.toggleRace(data.ExtensionName, data.RaceChoice, data.Checked)
		return nil

	case "toggleTrait":
		room, err := requireHost(client)
		if err != nil {
			return err
		}
		var data struct {
			ExtensionName string `json:"extensionName"`
			TraitChoice   string `json:"traitChoice"`
			Checked       bool   `json:"checked"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid toggleTrait data")
		}
		room.toggleTrait(data.ExtensionName, data.TraitChoice, data.Checked)
		return nil

	case "toggleExtension":
		room, err := requireHost(client)
		if err != nil {
			return err
		}
		var data struct {
			ExtensionName string `json:"extensionName"`
			Checked       bool   `json:"checked"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid toggleExtension data")
		}
		room.toggleExtension(data.ExtensionName, data.Checked)
		return nil

	case "toggleAll":
		room, err := requireHost(client)
		if err != nil {
			return err
		}
		var data struct {
			Checked bool `json:"checked"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid toggleAll data")
		}
		room.toggleAll(data.Checked)
		return nil

	case "savegame":
		room, err := requirePlayableRoom(client)
		if err != nil {
			return err
		}
		id, err := w.store.SaveGameState(ctx, &room.Gamestate, client.Index, room.Map.Name)
		if err != nil {
			return fmt.Errorf("save game: %w", err)
		}
		if err := w.store.AddSaveToUser(ctx, client.Username, id); err != nil {
			return fmt.Errorf("attach save: %w", err)
		}
		client.sendMessage("message", json.RawMessage([]byte(`{"message":"Game successfully saved"}`)))
		client.sendUserSaves()
		return nil

	case "rollback":
		room, err := requirePlayableRoom(client)
		if err != nil {
			return err
		}
		room.RollBack(client)
		return nil

	case "loadgame":
		room, err := requireNormalRoom(client)
		if err != nil {
			return err
		}
		var data struct {
			SaveID int64 `json:"saveId"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid save id")
		}
		if data.SaveID == -1 {
			room.playerStatuses = []string{}
			room.saveId = -1
			client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": -1}`)))
			room.sendPlayerStatuses()
			return nil
		}
		index, mapName, statuses, err := w.store.LoadGameInfo(ctx, data.SaveID)
		if err != nil {
			return err
		}
		_ = w.restoreGeneratedMap(ctx, mapName)
		if !room.ChangeMap(mapName) {
			return errors.New("error changing map")
		}
		if index < 0 || index >= len(room.Players) {
			return errors.New("saved player index does not fit selected map")
		}
		if !room.MovePlayerWithIndex(client.Username, index) {
			return errors.New("error moving player")
		}
		room.playerStatuses = statuses
		room.saveId = data.SaveID
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": `+strconv.FormatInt(data.SaveID, 10)+`}`)))
		room.sendPlayerStatuses()
		return nil

	case "loadgamedisplay":
		if client.DisplayRoom == nil {
			return errors.New("client not in a display room")
		}
		var data struct {
			SaveID int64 `json:"saveId"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid save id")
		}
		client.DisplayRoom.LoadSave(client, data.SaveID)
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": `+strconv.FormatInt(data.SaveID, 10)+`}`)))
		return nil

	case "loadmapdisplay":
		if client.DisplayRoom == nil {
			return errors.New("client not in a display room")
		}
		var data struct {
			Name string `json:"mapName"`
		}
		if err := json.Unmarshal(cmd.Message.Data, &data); err != nil {
			return errors.New("invalid map name")
		}
		client.DisplayRoom.LoadMap(client, data.Name)
		client.sendMessage("saveSelection", json.RawMessage([]byte(`{"index": -1}`)))
		return nil

	case "tribepick":
		client.handleTribePick(cmd.Message)
	case "entryaction":
		client.handleEntryAction(cmd.Message)
	case "abandonment":
		client.handleAbandonment(cmd.Message)
	case "Conquest":
		client.handleConquest(cmd.Message)
	case "startredeployment":
		client.handleStartRedeployment()
	case "deploymentin":
		client.handleRedeploymentIn(cmd.Message)
	case "deploymentout":
		client.handleRedeploymentOut(cmd.Message)
	case "deploymentthrough":
		client.handleRedeploymentThrough(cmd.Message)
	case "movement":
		client.handleMovement(cmd.Message)
	case "opponentaction":
		client.handleOppponentAction(cmd.Message)
	case "finishturn":
		client.handleFinishTurn()
	case "decline":
		client.handleDecline()
	default:
		return fmt.Errorf("unsupported worker message type %q", cmd.Message.Type)
	}

	if client.Room != nil {
		client.Room.flushGameMessages()
	}
	return nil
}

func (w *Worker) handleReconnect(ctx context.Context, client *Client, cmd protocol.Command) error {
	room := rooms[cmd.RoomID]
	if room == nil || room.IsDisplayRoom {
		_ = w.clearMembership(ctx, cmd.Username)
		_ = client.sendMessages(&protocol.Binding{Action: protocol.BindingClear, Scope: protocol.ScopeRoom},
			messages.Message{Type: "lobby"})
		return errors.New("room no longer exists")
	}

	spectator := roomHasSpectator(room, cmd.Username)
	index := roomPlayerIndex(room, cmd.Username)
	if index < 0 && !spectator {
		_ = w.clearMembership(ctx, cmd.Username)
		return errors.New("user is no longer a member of that room")
	}
	client.Room = room
	client.IsSpectator = spectator
	if index >= 0 {
		client.Index = index
	}

	roomIDData, _ := json.Marshal(map[string]string{"roomid": room.ID})
	client.sendMessage("roomid", roomIDData)
	if spectator {
		client.sendMessage("spectate", json.RawMessage([]byte(`{"index":"1"}`)))
	} else if room.InProgress {
		client.sendMessage("index", json.RawMessage([]byte(`{"index":"`+strconv.Itoa(index)+`"}`)))
	}
	room.sendSmallMapUpdateToClient(client)
	room.sendGeneratedMapVisualsToClient(client)
	room.sendMegaUpdateToClient(client)
	room.sendPlayerStatusesToClient(client)
	return nil
}

func (w *Worker) requireNoOtherRoom(ctx context.Context, username, targetRoom string) error {
	value, err := w.redis.HGet(ctx, protocol.UserRoomsHashKey, username).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	var membership protocol.RoomMembership
	if json.Unmarshal([]byte(value), &membership) == nil && membership.RoomID != "" && membership.RoomID != targetRoom {
		return errors.New("leave your current room before joining another room")
	}
	return nil
}

func requireNormalRoom(client *Client) (*Room, error) {
	if client.Room == nil {
		return nil, errors.New("client not in a room")
	}
	return client.Room, nil
}

func requireHost(client *Client) (*Room, error) {
	room, err := requireNormalRoom(client)
	if err != nil {
		return nil, err
	}
	if room.HostUsername != client.Username {
		return nil, errors.New("only the room host can do that")
	}
	if room.InProgress {
		return nil, errors.New("game already started")
	}
	return room, nil
}

func requirePlayableRoom(client *Client) (*Room, error) {
	room, err := requireNormalRoom(client)
	if err != nil {
		return nil, err
	}
	if !room.InProgress {
		return nil, errors.New("game has not started")
	}
	if client.IsSpectator {
		return nil, errors.New("spectators cannot do that")
	}
	return room, nil
}

func roomHasUser(room *Room, username string) bool {
	return roomPlayerIndex(room, username) >= 0 || roomHasSpectator(room, username)
}
func roomHasSpectator(room *Room, username string) bool {
	for _, c := range room.Spectators {
		if c != nil && c.Username == username {
			return true
		}
	}
	return false
}
func roomPlayerIndex(room *Room, username string) int {
	for i, c := range room.Players {
		if c != nil && c.Username == username {
			return i
		}
	}
	return -1
}

// The original UI broadcasts these as room messages, but reconnect happens
// after a new gateway subscription and should not blast a full update to every
// other player just because one socket came back.
func (room *Room) sendPlayerStatusesToClient(c *Client) {
    statuses := room.playerStatuses

    if statuses == nil {
        statuses = []string{}
    }

    raw, err := json.Marshal(statuses)
    if err != nil {
        log.Printf("marshal player statuses: %v", err)
        return
    }

    c.sendMessage("playerStatuses", raw)
}

func (room *Room) sendSmallMapUpdateToClient(c *Client) {
	raw, _ := json.Marshal(map[string]any{"mapName": room.Map.Name, "offset": room.Map.Offset, "fontSize": room.Map.FontSize})
	c.sendMessage("smallmapupdate", raw)
}

func (room *Room) sendMegaUpdateToClient(c *Client) {
	if !room.InProgress && !room.IsDisplayRoom {
		return
	}
	// megaUpdate is room-wide state, so rebroadcasting it on reconnect is safe
	// and keeps this path using exactly the same frontend payload as normal play.
	room.sendMegaUpdate()
	if !room.IsDisplayRoom {
		room.SendCoinUpdate()
	}
}
