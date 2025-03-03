package server

import (
	"backend/internal/gamestate"
	"backend/internal/messages"
	"encoding/json"
)



func (client *Client) handleTribePick (msg messages.Message) {
	var pickData struct {
		PickIndex int `json:"pickIndex"`
	}
	if err := json.Unmarshal([]byte(msg.Data), &pickData); err != nil {
		client.sendError("Invalid choice data")
		return
	}

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleTribeChoice(client.Index, pickData.PickIndex); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}

func (client *Client) handleAbandonment (msg messages.Message) {
	var abandonmentData struct {
		TileID string `json:"tileId"`
		StackType string `json:"stackType"`
	}
	if err := json.Unmarshal([]byte(msg.Data), &abandonmentData); err != nil {
		client.sendError("Invalid abandon data")
		return
	}

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleAbandonment(client.Index, abandonmentData.TileID, abandonmentData.StackType); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}

func (client *Client) handleConquest (msg messages.Message) {
	var conquestData struct {
		TileID             string `json:"tileId"`
		AttackingStackType string `json:"attackingStackType"`
	}

	if err := json.Unmarshal([]byte(msg.Data), &conquestData); err != nil {
		client.sendError("Invalid conquest data")
		return
	}

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleConquest(conquestData.TileID, client.Index, conquestData.AttackingStackType); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}

func (client *Client) handleStartRedeployment () {

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleStartRedeployment(client.Index); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}

func (client *Client) handleRedeploymentIn (msg messages.Message) {
	var deployData struct {
		TileID          string `json:"tileId"`
		StackType	string `json:"stackType"`
	}

	if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
		client.sendError("Invalid deploy data")
		return
	}

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleRedeploymentIn(client.Index, deployData.TileID, deployData.StackType, 1); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}

func (client *Client) handleRedeploymentOut (msg messages.Message) {
	var deployData struct {
	    TileID    string `json:"tileId"`
	    StackType string `json:"stackType"`
	}

	if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
		client.sendError("Invalid deploy data")
		return
	}

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleRedeploymentOut(client.Index, deployData.TileID, deployData.StackType); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}

func (client *Client) handleRedeploymentThrough (msg messages.Message) {
	var deployData struct {
		TileFromID          string `json:"tileFromId"`
		TileToID          string `json:"tileToId"`
		StackType	string `json:"stackType"`
	}

	if err := json.Unmarshal([]byte(msg.Data), &deployData); err != nil {
		client.sendError("Invalid deploy data")
		return
	}

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleRedeploymentOut(client.Index, deployData.TileFromID, deployData.StackType); err != nil {
		client.sendError(err.Error())
	} else if err := client.Room.Gamestate.HandleRedeploymentIn(client.Index, deployData.TileToID, deployData.StackType, 1); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}


func (client *Client) handleFinishTurn () {
	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleFinishTurn(client.Index); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
		client.Room.AutoSave()

		if client.Room.Gamestate.TurnInfo.Phase == gamestate.GameFinished {
			client.Room.sendGameFinishedUpdate()
			client.Room.InProgress = false
		} 
	}
}

func (client *Client) handleDecline () {
	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}

	if err := client.Room.Gamestate.HandleDecline(client.Index); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
		client.Room.AutoSave()

		if client.Room.Gamestate.TurnInfo.Phase == gamestate.GameFinished {
			client.Room.sendGameFinishedUpdate()
		} 
	}
}

func (client *Client) handleOppponentAction (msg messages.Message) {
	var data struct {
		OpponentName          string `json:"opponentName"`
		StackType	string `json:"stackType"`
	}

	if err := json.Unmarshal([]byte(msg.Data), &data); err != nil {
		client.sendError("Invalid deploy data")
		return
	}

	if client.Room == nil {
		client.sendError("Client not in a room")
		return
	}
	if !client.Room.InProgress {
		client.sendError("Client's room's game has not started yet")
		return
	}
	opponentIndex := 0
	for _, client := range(client.Room.Players) {
		if client.Username == data.OpponentName {
			opponentIndex = client.Index
		}
	}

	if err := client.Room.Gamestate.HandleOpponentAction(client.Index, opponentIndex, data.StackType); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendBigUpdate()
	}
}
