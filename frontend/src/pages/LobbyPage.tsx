// src/pages/LobbyPage.tsx
import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../redux/store';
import { useNavigate } from 'react-router-dom';
import { sendMessageToBackend } from '../services/backendService';
import { setTiles, reset } from '../redux/slices/applicationSlice';
import { parseAreaFile } from '../utility/MapParser';
import { Room } from '../types/Board';

export default function LobbyPage() {
  const navigate = useNavigate();
  const dispatch = useDispatch();

  // We no longer allow user to type username; we read it from Redux
  const { name: username, isAuthenticated, rooms, roomid, gameStarted } = useSelector(
    (state: RootState) => state.application
  );

  // If we have multiple players, we might store them differently, but let's follow your original code:
  const [roomName, setRoomName] = useState('');

  // This is the room that the user is currently in
  let currentRoom: Room | null = null;
  if (rooms) {
    currentRoom = rooms.find((r) => r.id === roomid) || null;
  }

  // If user is not authenticated, redirect to "/"
  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/');
      return;
    }
    if (gameStarted) {
      navigate('/game');
    }
  }, [isAuthenticated, gameStarted, navigate]);

  // Connect WebSocket on mount
  // On page refresh, request updated room list
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

  // Are we in a room?
  const userInRoom = !!currentRoom;

  return (
    <div className="flex flex-col items-center min-h-screen p-5 space-y-6">
      <h1 className="text-2xl font-bold">Welcome to the Lobby</h1>
      <div className="mb-4">Logged in as: <strong>{username}</strong></div>

      {/* If user not in a room, show available rooms + create room form */}
      {!userInRoom && (
        <>
          <div className="border p-4 w-full max-w-md space-y-3 bg-gray-100">
            <h2 className="font-bold">Create a Room</h2>
            <div className="flex flex-col items-start">
              <label className="mb-1">Room Name:</label>
              <input
                type="text"
                className="border p-1"
                value={roomName}
                onChange={(e) => setRoomName(e.target.value)}
              />
            </div>
            <button
              type="button"
              onClick={handleCreateRoom}
              className="btn btn-primary mt-2"
            >
              Create Room
            </button>
          </div>

          <div className="border p-4 w-full max-w-md bg-gray-100">
            <h2 className="font-bold">Available Rooms</h2>
            {rooms?.length === 0 ? (
              <p>No rooms available. Create one!</p>
            ) : (
              <ul>
                {rooms.map((rm) => (
                  <li key={rm.id} className="flex items-center justify-between my-2">
                    <div>
                      <strong>{rm.name}</strong> ({rm.players?.length})
                    </div>
                    <button
                      type="button"
                      onClick={() => handleJoinRoom(rm.id)}
                      className="btn btn-secondary ml-2"
                    >
                      Join
                    </button>
                  </li>
                ))}
              </ul>
            )}
          </div>
        </>
      )}

      {/* If user is in a room, show room details and Start Game if host */}
      {userInRoom && currentRoom && (
        <div className="border p-4 w-full max-w-md bg-gray-100">
          <h2 className="font-bold">Room: {currentRoom.name}</h2>
          <p>Players in this room:</p>
          <ul className="list-disc list-inside mb-4">
            {currentRoom.players?.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
          {currentRoom.creator === username && (
            <button
              type="button"
              onClick={handleStartGame}
              className="btn btn-primary"
            >
              Start Game
            </button>
          )}
          <button
            type="button"
            onClick={handleLeaveRoom}
            className="btn btn-primary ml-2"
          >
            Leave Room
          </button>
        </div>
      )}
    </div>
  );
}
