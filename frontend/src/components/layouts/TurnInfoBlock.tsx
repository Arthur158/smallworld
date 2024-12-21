import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';

export default function TurnInfoBlock() {
  const playerNumber = useSelector((state: RootState) => state.application.playerNumber);
  const phase = useSelector((state: RootState) => state.application.phase);
  const turnNumber = useSelector((state: RootState) => state.application.turnNumber);
  const players = useSelector((state: RootState) => state.application.players);

  const currentPlayer = players[playerNumber]?.name || 'Unknown Player';
  const placeholderNumber = 5;

  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6]">
      <h2 className="text-xl font-bold underline mb-2">Turn Information</h2>
      <p><span className="font-semibold">Current Player:</span> {currentPlayer}</p>
      <p><span className="font-semibold">Turn Number:</span> {turnNumber}</p>
      <p><span className="font-semibold">Phase:</span> {phase}</p>
      <p><span className="font-semibold">Number (placeholder):</span> {placeholderNumber}</p>

      <div className="flex space-x-2 mt-4">
        <button className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          Decline
        </button>
        <button className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          Redeploy
        </button>
        <button className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors">
          End Turn
        </button>
      </div>
    </div>
  );
}
