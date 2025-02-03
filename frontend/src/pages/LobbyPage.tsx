// src/pages/LobbyPage.tsx

import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../redux/store';
import { useNavigate } from 'react-router-dom';
import { sendMessageToBackend } from '../services/backendService';
import { setTiles, reset } from '../redux/slices/applicationSlice';
import { Room } from '../types/Board';

export default function LobbyPage() {
  const navigate = useNavigate();
  const dispatch = useDispatch();

  // Redux state
  const {
    name: username,
    isAuthenticated,
    rooms,
    roomid,
    gameStarted,
    saveGames,
  } = useSelector((state: RootState) => state.application);

  // Local state
  const [roomName, setRoomName] = useState('');

  // Identify current room (if any)
  let currentRoom: Room | null = null;
  if (rooms) {
    currentRoom = rooms.find((r) => r.id === roomid) || null;
  }

  // If user not authenticated, redirect home
  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/');
      return;
    }
    if (gameStarted) {
      navigate('/game');
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

  // Create room
  const handleCreateRoom = () => {
    if (!roomName.trim() || !username.trim()) return;
    sendMessageToBackend('createRoom', {
      roomName,
      maxPlayers: 5,
    });
  };

  // Join room
  const handleJoinRoom = (selectedRoomId: string) => {
    if (!username.trim()) return;
    sendMessageToBackend('joinRoom', {
      roomId: selectedRoomId,
    });
  };

  // Start game
  const handleStartGame = () => {
    if (!currentRoom) return;
    sendMessageToBackend('startGame', {
      roomId: currentRoom.id,
    });
  };

  // Leave room
  const handleLeaveRoom = () => {
    sendMessageToBackend('leaveroom', {});
  };

  // Placeholder for handling clicks on game IDs
  // Replace with your own logic as needed
  const handleGameIdClick = (gameId: number) => {
    sendMessageToBackend('loadgame', {saveId: gameId});
  };

  // Check if user is in a room
  const userInRoom = !!currentRoom;

  return (
    <div className="w-screen h-screen overflow-hidden bg-[#F5F5DC] font-serif text-[#5F4B32] relative">
      <div className="flex w-full h-full">
        {/* Left Column */}
        <div className="w-2/3 h-full flex flex-col border border-[#5F4B32] bg-[#FDF5E6]">
          <div className="flex-1 overflow-y-auto p-4">
            <h1 className="text-3xl font-bold mb-4">Welcome to the Lobby</h1>
            <div className="mb-4">
              <span className="font-semibold">Logged in as:</span> {username}
            </div>

            {/* If user is not in a room, show Create Room + Available Rooms */}
            {!userInRoom && (
              <div className="space-y-6">
                {/* Create Room Section */}
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

                {/* Available Rooms Section */}
                <div className="border border-[#5F4B32] p-4 bg-white">
                  <h2 className="text-xl font-bold mb-2 underline">Available Rooms</h2>
                  {rooms?.length === 0 ? (
                    <p>No rooms available. Create one!</p>
                  ) : (
                    <ul>
                      {rooms.map((rm) => (
                        <li
                          key={rm.id}
                          className="flex items-center justify-between my-2"
                        >
                          <div>
                            <strong>{rm.name}</strong> ({rm.players?.length})
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

            {/* If user is in a room, show room details + Start/Leave buttons */}
            {userInRoom && currentRoom && (
              <div className="border border-[#5F4B32] p-4 bg-white">
                <h2 className="text-xl font-bold mb-2 underline">Room: {currentRoom.name}</h2>
                <p className="font-semibold">Players in this room:</p>
                <ul className="list-disc list-inside mb-4 ml-4">
                  {currentRoom.players?.map((p) => (
                    <li key={p}>{p}</li>
                  ))}
                </ul>
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

        {/* Right Column: Display saved game IDs */}
        <div className="w-1/3 h-full border border-[#5F4B32] bg-[#FDF5E6] flex flex-col">
          <div className="p-4 border-b border-[#5F4B32]">
            <h2 className="text-2xl font-bold underline">Saved Games</h2>
          </div>
          <div className="flex-1 overflow-y-auto p-4">
            {userInRoom && currentRoom && currentRoom.creator === username ? (
              saveGames && saveGames.length > 0 ? (
                <ul>
                  {saveGames.map((gameId) => (
                    <li
                      key={gameId}
                      className="mb-2 p-2 cursor-pointer hover:bg-[#FFF5EE] transition-colors"
                      onClick={() => handleGameIdClick(gameId)}
                    >
                      Game ID: {gameId}
                    </li>
                  ))}
                </ul>
              ) : (
                <p>No saved games available.</p>
              )
            ) : <p>You are not the owner</p>}
          </div>
        </div>
      </div>
    </div>
  );
}
