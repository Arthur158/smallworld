package server

import (
	"encoding/json"
	"backend/internal/messages"
	"backend/internal/gamestate"
	"fmt"
)



func (client *Client) handleTribePick (msg messages.Message) {
	var pickData struct {
		PickIndex int `json:"pickIndex"`
	}
	if err := json.Unmarshal([]byte(msg.Data), &pickData); err != nil {
		client.sendError("Invalid choice data")
		return
	}

	if err := client.Room.Gamestate.HandleTribeChoice(client.Index, pickData.PickIndex); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendPlayerUpdate()
		client.Room.sendTurnUpdate()
		client.Room.sendEntriesUpdate()
	}
}

func (client *Client) handleAbandonment (msg messages.Message) {
	var abandonmentData struct {
		TileID string `json:"tileId"`
	}
	if err := json.Unmarshal([]byte(msg.Data), &abandonmentData); err != nil {
		client.sendError("Invalid abandon data")
		return
	}

	if err := client.Room.Gamestate.HandleAbandonment(client.Index, abandonmentData.TileID); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendPlayerUpdate()
		client.Room.sendTileUpdate(abandonmentData.TileID)
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

	if err := client.Room.Gamestate.HandleConquest(conquestData.TileID, client.Index, conquestData.AttackingStackType); err != nil {
		client.sendError(err.Error())
		client.Room.sendTurnUpdate()
	} else {
		client.Room.sendPlayerUpdate()
		client.Room.sendTileUpdate(conquestData.TileID)
		client.Room.sendTurnUpdate()
	}
}

func (client *Client) handleStartRedeployment () {
	if client.Room != nil {
		if err := client.Room.Gamestate.HandleStartRedeployment(client.Index); err != nil {
			client.sendError(err.Error())
		} else {
			client.Room.sendPlayerUpdate()
			client.Room.sendTurnUpdate()
			client.Room.sendAllTileUpdate()
		}
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

	if err := client.Room.Gamestate.HandleRedeploymentIn(client.Index, deployData.TileID, deployData.StackType, 1); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendPlayerUpdate()
		client.Room.sendTileUpdate(deployData.TileID)
		client.Room.sendTurnUpdate()
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

	if err := client.Room.Gamestate.HandleRedeploymentOut(client.Index, deployData.TileID, deployData.StackType); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendPlayerUpdate()
		client.Room.sendTileUpdate(deployData.TileID)
		client.Room.sendTurnUpdate()
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


	if err := client.Room.Gamestate.HandleRedeploymentOut(client.Index, deployData.TileFromID, deployData.StackType); err != nil {
		client.sendError(err.Error())
	} else if err := client.Room.Gamestate.HandleRedeploymentIn(client.Index, deployData.TileToID, deployData.StackType, 1); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendPlayerUpdate()
		client.Room.sendTileUpdate(deployData.TileFromID)
		client.Room.sendTileUpdate(deployData.TileToID)
		client.Room.sendTurnUpdate()
	}
}


func (client *Client) handleFinishTurn () {
	if err := client.Room.Gamestate.HandleFinishTurn(client.Index); err != nil {
		client.sendError(err.Error())
	} else {
		// index, _ := SaveGameState(&client.Room.Gamestate)
		// state, err := LoadGameState(index)
		// log.Println(err)
		// client.Room.Gamestate = *state
		client.Room.sendPlayerUpdate()
		client.Room.sendTurnUpdate()
		client.Room.sendAllTileUpdate()

		pointsList := client.Room.Gamestate.Players[client.Index].PointsEachTurn
		client.Room.sendStateMessage(
		    fmt.Sprintf(
			"player %s made %d points this turn",
			client.Username,
			pointsList[len(pointsList) - 1]-pointsList[len(pointsList) - 2],
		    ),
		)

		if client.Room.Gamestate.TurnInfo.Phase == gamestate.GameFinished {
			client.Room.sendGameFinishedUpdate()
			client.Room.InProgress = false
		} 
	}
}

func (client *Client) handleDecline () {
	if err := client.Room.Gamestate.HandleDecline(client.Index); err != nil {
		client.sendError(err.Error())
	} else {
		client.Room.sendPlayerUpdate()
		client.Room.sendTurnUpdate()
		client.Room.sendAllTileUpdate()

		pointsList := client.Room.Gamestate.Players[client.Index].PointsEachTurn
		client.Room.sendStateMessage(
		    fmt.Sprintf(
			"player %s made %d points this turn",
			client.Username,
			pointsList[len(pointsList) - 1]-pointsList[len(pointsList) - 2],
		    ),
		)

		if client.Room.Gamestate.TurnInfo.Phase == gamestate.GameFinished {
			client.Room.sendGameFinishedUpdate()
		} 
	}
}

