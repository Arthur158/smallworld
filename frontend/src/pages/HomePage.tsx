import React, { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { sendMessageToBackend } from '../services/backendService';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../redux/store';
import { setName, reset, setTiles } from '../redux/slices/applicationSlice'; // <--- new actions to track selection
import { connectWebSocket } from '../services/backendService';
import { parseAreaFile } from '../utility/MapParser';
import { Room } from '../types/Board';


export default function HomePage() {
  const navigate = useNavigate();
  const dispatch = useDispatch();

  // Pull the rooms array from Redux
  const rooms = useSelector((state: RootState) => state.application.rooms);
  const roomid = useSelector((state: RootState) => state.application.roomid);
  const gameStarted = useSelector((state: RootState) => state.application.gameStarted);
  let room : Room | null = null
  if (rooms) {
    for (const r of rooms) {
      if (r.id == roomid) {
        room = r
      }
    }
  }

  // Local state for the username and roomName the user inputs
  const username = useSelector((state: RootState) => state.application.name);
  const [roomName, setRoomName] = useState('');

  // We'll use Redux to store which room the user joined/created (if any).
  // But you can keep it local if you prefer. For clarity, let's do a simple approach:
    useEffect(() => {
    if (gameStarted) {
      navigate('/game');
    }
  }, [gameStarted, navigate]);


  // On component load, you might want to request the rooms list from the server
  // if the server doesn't automatically push them. For example:
  useEffect(() => {
    connectWebSocket();

    // If your server implements "getRooms", you could do:
    // sendMessageToBackend('getRooms', {});
    // Then the server replies with a "roomsUpdate" message.
  }, []);

  // Create a new room
  const handleCreateRoom = () => {
    if (!roomName.trim() || !username.trim()) return;

    sendMessageToBackend('createRoom', {
      roomName,
      username,
      maxPlayers: 5, // can be dynamic
    });

    // We'll assume the server triggers a "roomsUpdate" broadcast. 
    // Also set local user fields in Redux
  };

  useEffect(() => {

    const loadAreas = async () => {
      try {
        const response = await fetch('/maps/map.txt');
        const text = await response.text();
        const polygons = parseAreaFile(text);

        let id = 0;
        const tileData = polygons.map((polygon) => ({
          id: id++,
          pieceStack: [],
          polygon,
        }));

        dispatch(setTiles(tileData));
      } catch (error) {
        console.error('Error loading file:', error);
      }
    };
    loadAreas();
  }, [dispatch]);


  useEffect(() => {
    const handlePageRefresh = () => {
      dispatch(reset());
      sendMessageToBackend('requestrefresh', {})
    };

    window.addEventListener('beforeunload', handlePageRefresh);

    return () => {
      window.removeEventListener('beforeunload', handlePageRefresh);
    };
  }, [dispatch]);

  // Join a room
  const handleJoinRoom = (roomid: string) => {
    if (!username.trim()) return;

    sendMessageToBackend('joinRoom', {
      roomId: roomid,
      username,
    });

    // The server will send a "roomsUpdate" that includes you in the players array
    // Also, let's see if we're the host
  };

  // Host can start the game
  const handleStartGame = () => {
    if (!room) return;
    sendMessageToBackend('startGame', {
      roomId: room.id,
    });
  };
  const handleLeaveRoom = () => {
    if (!room) return;
    sendMessageToBackend('leaveroom', {})
  };

  // Decide if the user is in a room based on `selectedRoom`
  const userInRoom = !!room;
  console.log("since we here")
  console.log(room)
  console.log(rooms)

  return (
    <div className="flex flex-col items-center min-h-screen p-5 space-y-6">
      <h1 className="text-2xl font-bold">Welcome to the Lobby</h1>

      {/* Username Input */}
      <div className="flex flex-col items-start">
        <label className="mb-1">Username:</label>
        <input
          type="text"
          className="border p-1"
          value={username}
          onChange={(e) => dispatch(setName(e.target.value))}
        />
      </div>

      {/* If the user is not yet in a room, show Room creation + Room list */}
      {!userInRoom && (
        <>
          <div className="border p-4 w-full max-w-md space-y-3 bg-gray-100">
            {/* Create Room Section */}
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
                {rooms?.map((room) => (
                  <li key={room.id} className="flex items-center justify-between my-2">
                    <div>
                      <strong>{room.name}</strong> ({room?.players?.length})
                    </div>
                    <button
                      type="button"
                      onClick={() => {
                        handleJoinRoom(room?.id);
                      }}
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

      {/* If user is in a room (created or joined), show room details and Start Game if host */}
      {room && (
        <div className="border p-4 w-full max-w-md bg-gray-100">
          <h2 className="font-bold">Room: {room?.name}</h2>
          <p>Players in this room:</p>
          <ul className="list-disc list-inside mb-4">
            {room?.players?.map((p) => (
              <li key={p}>{p}</li>
            ))}
          </ul>
          {room?.creator == username && (
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
            className="btn btn-primary"
          >
            Leave Room
          </button>
        </div>
      )}
    </div>
  );
}
