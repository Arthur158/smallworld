import React, { useEffect, useState } from 'react';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService';
import { setIsStackFromBank, setSelectedStack, setSelectedTile } from '../../redux/slices/applicationSlice';
import { useSelector, useDispatch } from 'react-redux';

export default function TurnInfoBlockDisplay() {
  const playerNumber = useSelector((state: RootState) => state.application.playerNumber);
  const phase = useSelector((state: RootState) => state.application.phase);
  const turnNumber = useSelector((state: RootState) => state.application.turnNumber);
  const players = useSelector((state: RootState) => state.application.players);
  const dispatch = useDispatch();

  const currentPlayer = players[playerNumber]?.name || 'Unknown Player';

  const handleLeave = () => {
    sendMessageToBackend("leavedisplayroom", {});
  };

  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6] shadow-md">
      <h2 className="text-xl font-bold underline mb-2">Turn Information</h2>
      <p><span className="font-semibold">Current Player:</span> {currentPlayer}</p>
      <p><span className="font-semibold">Turn Number:</span> {turnNumber}</p>
      <p><span className="font-semibold">Phase:</span> {phase}</p>

      {/* Buttons Layout */}
      <div className="grid grid-cols-2 gap-2 mt-4">
        <button onClick={handleLeave} className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-3 rounded transition-colors">
          Leave Display Room <span className="text-sm text-gray-300">(D)</span>
        </button>
      </div>
    </div>
  );
}
