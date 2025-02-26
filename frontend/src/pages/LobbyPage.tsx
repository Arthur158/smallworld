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

  // Create / Join / Start / Leave handlers
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
    sendMessageToBackend('enterdisplayroom', {});
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

  const handleDeleteGame = (gameId: number) => {
    sendMessageToBackend('deletesave', { saveId: gameId });
  };

  const handleMapChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    if (!currentRoom) return;
    const newMap = event.target.value;
    sendMessageToBackend('changeRoomMap', { roomId: currentRoom.id, newMap });
  };

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

  const handleLogout = () => {
    sendMessageToBackend('logout', {});
  };

  const userInRoom = !!currentRoom;

  return (
    <div className="w-screen h-screen overflow-hidden bg-[#F5F5DC] font-serif text-[#5F4B32] relative">
      <div className="flex w-full h-full">
        {/* Left column */}
        <div className="w-2/3 h-full flex flex-col border-r border-[#5F4B32] bg-[#FDF5E6]">
          <div className="flex-1 overflow-y-auto p-6">
            {/* Header */}
            <div className="mb-6 flex items-center justify-between">
              <span className="text-lg font-semibold">Logged in as: {username}</span>
              <button
                onClick={handleLogout}
                className="bg-red-600 hover:bg-red-700 text-white py-1 px-4 rounded shadow-md transition-colors"
              >
                Log Out
              </button>
            </div>
            {/* Not in room: Create Room and Room List */}
            {!userInRoom && (
              <div className="space-y-8">
                {/* Create Room Card */}
                <div className="p-6 bg-white rounded-lg shadow-md border border-[#5F4B32]">
                  <h2 className="text-xl font-bold mb-4 underline">Create a Room</h2>
                  <div className="mb-4">
                    <label className="block font-semibold mb-1">Room Name:</label>
                    <input
                      type="text"
                      className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-[#8B4513]"
                      value={roomName}
                      onChange={(e) => setRoomName(e.target.value)}
                    />
                  </div>
                  <button
                    type="button"
                    onClick={handleCreateRoom}
                    className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 rounded transition-colors shadow-md"
                  >
                    Create Room
                  </button>
                </div>
                {/* Available Rooms Card */}
                <div className="p-6 bg-white rounded-lg shadow-md border border-[#5F4B32]">
                  <h2 className="text-xl font-bold mb-4 underline">Available Rooms</h2>
                  {rooms?.length === 0 ? (
                    <p className="text-gray-600">No rooms available. Create one!</p>
                  ) : (
                    <ul className="space-y-3">
                      {rooms.map((rm) => (
                        <li
                          key={rm.id}
                          className="flex items-center justify-between p-3 bg-gray-50 rounded border border-transparent hover:border-[#8B4513] transition-colors"
                        >
                          <div className="text-md font-medium">
                            <strong>{rm.name}</strong>{' '}
                            <span className="text-sm text-gray-600">
                              ({rm.players?.length || 0})
                            </span>
                          </div>
                          <button
                            type="button"
                            onClick={() => handleJoinRoom(rm.id)}
                            className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors"
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
            {/* In room: Room Details */}
            {userInRoom && currentRoom && (
              <div className="p-6 bg-white rounded-lg shadow-md border border-[#5F4B32]">
                <h2 className="text-xl font-bold mb-4 underline">Room: {currentRoom.name}</h2>
                <div className="mb-4">
                  <p className="font-semibold">Selected Map: {currentRoom?.mapName}</p>
                </div>
                {currentRoom.creator === username && mapChoices && mapChoices.length > 0 && (
                  <div className="mb-4">
                    <label className="font-semibold mr-2">Choose Map:</label>
                    <select onChange={handleMapChange} className="border p-2 rounded focus:outline-none">
                      <option value="">-- Select a Map --</option>
                      {mapChoices.map((map) => (
                        <option key={map} value={map}>
                          {map}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
                <p className="font-semibold mb-3">Players in this room:</p>
                <div className="space-y-3 mb-4">
                  {currentRoom.players?.map((p, idx) => {
                    if (!p || !p.trim().length) return null;
                    return (
                      <div
                        key={p}
                        className="flex items-center justify-between p-3 bg-[#EED5B7] rounded-lg shadow-sm"
                      >
                        <span className="font-semibold">
                          {idx + 1}: {p}
                          {playerStatuses[idx] && playerStatuses[idx].trim() !== '' && (
                            <span className="text-sm font-normal"> | {playerStatuses[idx]}</span>
                          )}
                        </span>
                        {currentRoom?.creator === username && (
                          <div className="flex space-x-2">
                            <button
                              onClick={() => handleMoveUp(p)}
                              className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded"
                            >
                              ↑
                            </button>
                            <button
                              onClick={() => handleMoveDown(p)}
                              className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded"
                            >
                              ↓
                            </button>
                            <button
                              onClick={() => handleKickPlayer(p)}
                              className="bg-red-600 hover:bg-red-700 text-white py-1 px-3 rounded"
                            >
                              ✕
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
                <div className="flex space-x-4">
                  {currentRoom.creator === username && (
                    <button
                      type="button"
                      onClick={handleStartGame}
                      className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-4 rounded transition-colors shadow-md"
                    >
                      Start Game
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={handleLeaveRoom}
                    className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-4 rounded transition-colors shadow-md"
                  >
                    Leave Room
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>
        {/* Right column: Saved Games */}
        <div className="w-1/3 h-full flex flex-col bg-[#FDF5E6] border-l border-[#5F4B32]">
          <div className="p-6 border-b border-[#5F4B32]">
            <h2 className="text-2xl font-bold underline">Saved Games</h2>
          </div>
          <div className="flex-1 overflow-y-auto p-6">
            <button
              type="button"
              onClick={handleEnterDisplayRoom}
              className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 rounded transition-colors shadow-md mb-6"
            >
              Enter Display Room
            </button>
            {userInRoom && currentRoom && currentRoom.creator === username ? (
              <>
                {saveGames && saveGames.length > 0 ? (
                  <ul className="space-y-4">
                    {saveGames.map((gameSave) => (
                      <li
                        key={gameSave.saveId}
                        className={`relative flex items-center p-4 cursor-pointer bg-white rounded border-2 transition-colors ${
                          gameSave.saveId === saveSelectionId ? 'border-[#8B4513]' : 'border-transparent'
                        }`}
                        onClick={() => handleGameIdClick(gameSave.saveId)}
                      >
                        <div className="flex-1">
                          <div className="font-bold">Game ID: {gameSave.saveId}</div>
                          <div className="text-sm text-gray-600">Summary: {gameSave.summary}</div>
                        </div>
                        <button
                          onClick={(e) => {
                            e.stopPropagation();
                            handleDeleteGame(gameSave.saveId);
                          }}
                          className="absolute right-4 bg-red-600 hover:bg-red-700 text-white py-1 px-3 rounded transition-colors"
                        >
                          Delete
                        </button>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <p className="text-gray-600">No saved games available.</p>
                )}

                {/* RENDER EXTRA CHOICES BELOW SAVED GAMES IF HOST */}
                {saveSelectionId === -1 && <ExtraChoices />}
              </>
            ) : (
              <p className="text-gray-600">You are not the owner</p>
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

function ExtraChoices() {
  const { extensionChoices, globalToggle } = useSelector((state: RootState) => ({
    extensionChoices: state.application.extensionChoices,
    globalToggle: state.application.globalToggle,
  }));

  const handleToggleAll = (checked: boolean) => {
    sendMessageToBackend('toggleAll', { checked });
  };

  const handleToggleExtension = (extensionName: string, checked: boolean) => {
    sendMessageToBackend('toggleExtension', {
      extensionName,
      checked,
    });
  };

  const handleToggleRace = (
    extensionName: string,
    raceChoice: string,
    checked: boolean
  ) => {
    sendMessageToBackend('toggleRace', {
      extensionName,
      raceChoice,
      checked,
    });
  };

  const handleToggleTrait = (
    extensionName: string,
    traitChoice: string,
    checked: boolean
  ) => {
    sendMessageToBackend('toggleTrait', {
      extensionName,
      traitChoice,
      checked,
    });
  };

  return (
    <div className="mt-8 p-4 bg-white rounded border border-[#8B4513]">
      <div className="mb-4">
        <label className="flex items-center space-x-2">
          <input
            type="checkbox"
            checked={globalToggle}
            onChange={(e) => handleToggleAll(e.target.checked)}
          />
          <span>Toggle All</span>
        </label>
      </div>

      <div className="space-y-6">
        {extensionChoices.map((ext, index) => (
          <div key={index} className="p-4 border rounded border-[#8B4513]">
            <label className="flex items-center space-x-2 mb-4">
              <input
                type="checkbox"
                checked={ext.isChecked}
                onChange={(e) =>
                  handleToggleExtension(ext.extensionName, e.target.checked)
                }
              />
              <span>{ext.extensionName}</span>
            </label>
            <div className="flex space-x-8">
              <div className="w-1/2">
                <h4 className="font-bold text-md mb-2">Races</h4>
                <div className="space-y-2">
                  {ext.raceChoices.map((race, rIdx) => (
                    <label key={rIdx} className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        checked={race.isChecked}
                        onChange={(e) =>
                          handleToggleRace(
                            ext.extensionName,
                            race.choice,
                            e.target.checked
                          )
                        }
                      />
                      <span>{race.choice}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="w-1/2">
                <h4 className="font-bold text-md mb-2">Traits</h4>
                <div className="space-y-2">
                  {ext.traitChoices.map((trait, tIdx) => (
                    <label key={tIdx} className="flex items-center space-x-2">
                      <input
                        type="checkbox"
                        checked={trait.isChecked}
                        onChange={(e) =>
                          handleToggleTrait(
                            ext.extensionName,
                            trait.choice,
                            e.target.checked
                          )
                        }
                      />
                      <span>{trait.choice}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
