import React, { useEffect, useState } from 'react';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService'
import { setIsStackFromBank, setSelectedStack, setSelectedTile } from '../../redux/slices/applicationSlice'
import { useSelector, useDispatch } from 'react-redux';


export default function TurnInfoBlock() {
  const playerNumber = useSelector((state: RootState) => state.application.playerNumber);
  const phase = useSelector((state: RootState) => state.application.phase);
  const turnNumber = useSelector((state: RootState) => state.application.turnNumber);
  const players = useSelector((state: RootState) => state.application.players);
  const dispatch = useDispatch()

  const currentPlayer = players[playerNumber]?.name || 'Unknown Player';
  const placeholderNumber = 5;

  const handleDecline = () => {
    sendMessageToBackend("decline", {})
    dispatch(setIsStackFromBank(false))
    dispatch(setSelectedTile(null))
    dispatch(setSelectedStack(null))
  }

  const handleRedeploy = () => {
    sendMessageToBackend("startredeployment", {})
    dispatch(setIsStackFromBank(false))
    dispatch(setSelectedTile(null))
    dispatch(setSelectedStack(null))
  }

  const handleEndTurn = () => {
    sendMessageToBackend("finishturn", {})
    dispatch(setIsStackFromBank(false))
    dispatch(setSelectedTile(null))
    dispatch(setSelectedStack(null))
  }
  const handleLeaveGame = () => {
    sendMessageToBackend("leaveroom", {})
    dispatch(setIsStackFromBank(false))
    dispatch(setSelectedTile(null))
    dispatch(setSelectedStack(null))
  }
  const handleSaveGame = () => {
    sendMessageToBackend("savegame", {})
  }

  useEffect(() => {
    const handleKeyPress = (event: KeyboardEvent) => {
      if (event.key === 'd') { // Replace 't' with the key you want to trigger the action
        handleDecline()
      }
      if (event.key === 'r') { // Replace 't' with the key you want to trigger the action
        handleRedeploy()
      }
      if (event.key === 'e') { // Replace 't' with the key you want to trigger the action
        handleEndTurn()
      }
    };

    window.addEventListener('keydown', handleKeyPress);

    return () => {
      window.removeEventListener('keydown', handleKeyPress);
    };
  }, []);


  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6]">
      <h2 className="text-xl font-bold underline mb-2">Turn Information</h2>
      <p><span className="font-semibold">Current Player:</span> {currentPlayer}</p>
      <p><span className="font-semibold">Turn Number:</span> {turnNumber}</p>
      <p><span className="font-semibold">Phase:</span> {phase}</p>
      <p><span className="font-semibold">Number (placeholder):</span> {placeholderNumber}</p>

      <div className="flex space-x-2 mt-4">
        <button onClick={handleDecline} className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          Decline
        </button>
        <button onClick={handleRedeploy} className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          Redeploy
        </button>
        <button onClick={handleEndTurn} className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          End Turn
        </button>
        <button onClick={handleSaveGame} className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          Save Game
        </button>
        <button onClick={handleLeaveGame} className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          Leave Game
        </button>
      </div>
    </div>
  );
}
