package roomworker

import (
	"backend/internal/gamestate"
	"backend/internal/protocol"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
)

func sendRoomsUpdateToAll() {
	if currentWorker == nil {
		return
	}
	if err := currentWorker.syncLobbyMetadata(context.Background()); err != nil {
		log.Printf("roomworker: sync lobby metadata: %v", err)
	}
}

func (w *Worker) syncLobbyMetadata(ctx context.Context) error {
	local := make(map[string]protocol.RoomMetadata, len(rooms))
	for id, room := range rooms {
		if room == nil || room.IsDisplayRoom {
			continue
		}
		if !room.hasConnectedUsers() {
			continue
		}


		players := make([]string, len(room.Players))
		for i, player := range room.Players {
			if player != nil {
				players[i] = player.Username
			}
		}
		local[id] = protocol.RoomMetadata{
			ID:         room.ID,
			Name:       room.Name,
			Players:    players,
			MaxPlayers: room.Map.Capacity,
			MapName:    room.Map.Name,
			Creator:    room.HostUsername,
			InProgress: room.InProgress,
		}
	}
	pipe := w.redis.TxPipeline()
	for id, meta := range local {
		b, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		pipe.HSet(ctx, protocol.RoomsHashKey, id, b)
	}

	w.publishedMu.Lock()
	for id := range w.publishedRooms {
		if _, ok := local[id]; !ok {
			pipe.HDel(ctx, protocol.RoomsHashKey, id)
		}
	}
	w.publishedRooms = make(map[string]struct{}, len(local))
	for id := range local {
		w.publishedRooms[id] = struct{}{}
	}
	w.publishedMu.Unlock()

	pipe.Publish(ctx, protocol.LobbyChangedChannel, fmt.Sprintf("worker-%d", w.cfg.WorkerID))
	_, err := pipe.Exec(ctx)
	return err
}

func SaveGameState(state *gamestate.GameState, saverIndex int, mapName string) (int64, error) {
	if currentWorker == nil {
		return 0, errors.New("room worker not initialized")
	}
	return currentWorker.store.SaveGameState(context.Background(), state, saverIndex, mapName)
}

func LoadGameState(id int64) (*gamestate.GameState, int, error) {
	if currentWorker == nil {
		return nil, 0, errors.New("room worker not initialized")
	}
	return currentWorker.store.LoadGameState(context.Background(), id)
}

func LoadGameInfo(id int64) (int, string, []string, error) {
	if currentWorker == nil {
		return 0, "", nil, errors.New("room worker not initialized")
	}
	return currentWorker.store.LoadGameInfo(context.Background(), id)
}

func DeleteGameState(id int64) error {
	if currentWorker == nil || id < 0 {
		return nil
	}
	return currentWorker.store.DeleteGameState(context.Background(), id)
}

func AddGameIDToUser(username string, gameID int64) error {
	if currentWorker == nil {
		return errors.New("room worker not initialized")
	}
	return currentWorker.store.AddSaveToUser(context.Background(), username, gameID)
}

func RemoveGameIDFromUser(username string, gameID int64) error {
	if currentWorker == nil || gameID < 0 {
		return nil
	}
	return currentWorker.store.RemoveSaveLink(context.Background(), username, gameID)
}
