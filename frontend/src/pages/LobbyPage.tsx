// src/pages/LobbyPage.tsx

import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../redux/store';
import { useNavigate } from 'react-router-dom';
import { sendMessageToBackend } from '../services/backendService';
import { reset } from '../redux/slices/applicationSlice';
import { Room, SaveGameInfo } from '../types/Board';

export default function LobbyPage() {
  const navigate = useNavigate();
  const dispatch = useDispatch();

  const {
    name: username,
    isAuthenticated,
    rooms,
    roomid,
    gameStarted,
    saveGames,
    error,
    saveSelectionId,
    mapChoices,
    playerStatuses,
  } = useSelector((state: RootState) => ({
    name: state.application.name,
    isAuthenticated: state.application.isAuthenticated,
    rooms: state.application.rooms,
    roomid: state.application.roomid,
    gameStarted: state.application.gameStarted,
    saveGames: state.application.saveGames as SaveGameInfo[],
    error: state.application.error,
    saveSelectionId: state.application.saveSelectionId,
    mapChoices: state.application.mapChoices,
    playerStatuses: state.application.playerStatuses,
  }));

  const [roomName, setRoomName] = useState('');

  let currentRoom: Room | null = null;
  if (rooms) {
    currentRoom = rooms.find((r) => r.id === roomid) || null;
  }

  // Redirect if not authenticated or if game started
  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/');
      return;
    }
  }, [isAuthenticated, gameStarted, navigate]);

  // Handle page refresh
  useEffect(() => {
    const handlePageRefresh = () => {
      dispatch(reset());
      sendMessageToBackend('requestrefresh', {});
    };

    window.addEventListener('beforeunload', handlePageRefresh);
    return () => {
      window.removeEventListener('beforeunload', handlePageRefresh);
    };
  }, [dispatch]);

  // Keyboard shortcut for start game: press 'p'
  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      if (event.key.toLowerCase() === 'p' && currentRoom?.creator === username) {
        handleStartGame();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [currentRoom, username]);

  // Create / Join / Start / Leave
  const handleCreateRoom = () => {
    if (!roomName.trim() || !username.trim()) return;
    sendMessageToBackend('createRoom', {
      roomName,
      maxPlayers: 2,
    });
  };

  const handleJoinRoom = (selectedRoomId: string) => {
    if (!username.trim()) return;
    sendMessageToBackend('joinRoom', {
      roomId: selectedRoomId,
    });
  };

  const handleEnterDisplayRoom = () => {
    if (!username.trim()) return;
    sendMessageToBackend('enterdisplayroom', {
    });
  };

  const handleStartGame = () => {
    if (!currentRoom) return;
    sendMessageToBackend('startGame', {
      roomId: currentRoom.id,
    });
  };

  const handleLeaveRoom = () => {
    sendMessageToBackend('leaveroom', {});
  };

  const handleGameIdClick = (gameId: number) => {
    sendMessageToBackend('loadgame', { saveId: gameId });
  };

  // Handle map change
  const handleMapChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    if (!currentRoom) return;
    const newMap = event.target.value;
    sendMessageToBackend('changeRoomMap', { roomId: currentRoom.id, newMap });
  };

  // Callbacks (backend must implement these)
  const handleMoveUp = (playerName: string) => {
    if (!currentRoom) return;
    sendMessageToBackend('moveUp', { roomId: currentRoom.id, username: playerName });
  };

  const handleMoveDown = (playerName: string) => {
    if (!currentRoom) return;
    sendMessageToBackend('moveDown', { roomId: currentRoom.id, username: playerName });
  };

  const handleKickPlayer = (playerName: string) => {
    if (!currentRoom) return;
    sendMessageToBackend('kickPlayer', { roomId: currentRoom.id, username: playerName });
  };

  const userInRoom = !!currentRoom;

  return (
    <div className="w-screen h-screen overflow-hidden bg-[#F5F5DC] font-serif text-[#5F4B32] relative">
      <div className="flex w-full h-full">
        {/* Left column */}
        <div className="w-2/3 h-full flex flex-col border border-[#5F4B32] bg-[#FDF5E6]">
          <div className="flex-1 overflow-y-auto p-4">
            <div className="mb-4">
              <span className="font-semibold">Logged in as:</span> {username}
            </div>

            {/* If not in a room, show 'Create Room' and room list */}
            {!userInRoom && (
              <div className="space-y-6">
                {/* Create Room */}
                <div className="border border-[#5F4B32] p-4 bg-white">
                  <h2 className="text-xl font-bold mb-2 underline">Create a Room</h2>
                  <div className="flex flex-col mb-2">
                    <label className="font-semibold mb-1">Room Name:</label>
                    <input
                      type="text"
                      className="border p-2"
                      value={roomName}
                      onChange={(e) => setRoomName(e.target.value)}
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handleCreateRoom}
                    className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors"
                  >
                    Create Room
                  </button>
                </div>

                {/* Available Rooms */}
                <div className="border border-[#5F4B32] p-4 bg-white">
                  <h2 className="text-xl font-bold mb-2 underline">Available Rooms</h2>
                  {rooms?.length === 0 ? (
                    <p>No rooms available. Create one!</p>
                  ) : (
                    <ul>
                      {rooms.map((rm) => (
                        <li key={rm.id} className="flex items-center justify-between my-2">
                          <div>
                            <strong>{rm.name}</strong> ({rm.players?.length || 0})
                          </div>
                          <button
                            type="button"
                            onClick={() => handleJoinRoom(rm.id)}
                            className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors ml-2"
                          >
                            Join
                          </button>
                        </li>
                      ))}
                    </ul>
                  )}
                </div>
              </div>
            )}

            {/* If in a room, show room details */}
            {userInRoom && currentRoom && (
              <div className="border border-[#5F4B32] p-4 bg-white">
                <h2 className="text-xl font-bold mb-2 underline">Room: {currentRoom.name}</h2>

                {/* Display selected map (placeholder) */}
                <div className="mb-4">
                  <p className="font-semibold">Selected Map: {currentRoom?.mapName}</p>
                </div>

                {/* Show map chooser only if you're the creator/host */}
                {currentRoom.creator === username && mapChoices && mapChoices.length > 0 && (
                  <div className="mb-4">
                    <label className="font-semibold mr-2">Choose Map:</label>
                    <select onChange={handleMapChange} className="border p-1">
                      <option value="">-- Select a Map --</option>
                      {mapChoices.map((map) => (
                        <option key={map} value={map}>
                          {map}
                        </option>
                      ))}
                    </select>
                  </div>
                )}

                <p className="font-semibold">Players in this room:</p>
                <div className="mb-4">
                  {currentRoom.players?.map((p, idx) => {
                    if (!p || !p.trim().length) {
                      return null;
                    }
                    return (
                      <div key={p} className="bg-[#EED5B7] mb-2 p-2 flex items-center rounded-lg">
                        <span className="flex-1 font-semibold">
                          {idx + 1}: {p}
                          {playerStatuses[idx] && playerStatuses[idx].trim() !== '' && (
                            <> | {playerStatuses[idx]}</>
                          )}
                        </span>
                        {currentRoom && currentRoom.creator === username && (
                          <div className="flex space-x-2">
                            <button
                              onClick={() => handleMoveUp(p)}
                              className="px-2 py-1 bg-[#8B4513] text-white rounded"
                            >
                              ↑
                            </button>
                            <button
                              onClick={() => handleMoveDown(p)}
                              className="px-2 py-1 bg-[#8B4513] text-white rounded"
                            >
                              ↓
                            </button>
                            <button
                              onClick={() => handleKickPlayer(p)}
                              className="px-2 py-1 bg-red-600 text-white rounded"
                            >
                              ✕
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>

                <div>
                  {currentRoom.creator === username && (
                    <button
                      type="button"
                      onClick={handleStartGame}
                      className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors mr-2"
                    >
                      Start Game
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={handleLeaveRoom}
                    className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors"
                  >
                    Leave Room
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Right column: Saved Games */}
        <div className="w-1/3 h-full border border-[#5F4B32] bg-[#FDF5E6] flex flex-col">
          <div className="p-4 border-b border-[#5F4B32]">
            <h2 className="text-2xl font-bold underline">Saved Games</h2>
          </div>
          <div className="flex-1 overflow-y-auto p-4">
            <button
              type="button"
              onClick={handleEnterDisplayRoom}
              className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors"
            >
             Enter Display Room 
            </button>
            {userInRoom && currentRoom && currentRoom.creator === username ? (
              saveGames && saveGames.length > 0 ? (
                <ul>
                  {saveGames.map((gameSave) => (
                    <li
                      key={gameSave.saveId}
                      className={`mb-2 p-2 cursor-pointer hover:bg-[#FFF5EE] transition-colors border-2 ${
                        gameSave.saveId === saveSelectionId ? 'border-[#8B4513]' : 'border-transparent'
                      }`}
                      onClick={() => handleGameIdClick(gameSave.saveId)}
                    >
                      <div className="font-bold">Game ID: {gameSave.saveId}</div>
                      <div>Summary: {gameSave.summary}</div>
                    </li>
                  ))}
                </ul>
              ) : (
                <p>No saved games available.</p>
              )
            ) : (
              <p>You are not the owner</p>
            )}
          </div>
        </div>
      </div>
      {/* Error Banner at the Bottom */}
      {error && (
        <div className="fixed bottom-0 left-0 w-full bg-red-500 text-white text-center py-2 z-50">
          {error}
        </div>
      )}
    </div>
  );
}
