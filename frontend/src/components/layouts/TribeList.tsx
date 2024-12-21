import React from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../../redux/store';
import { TribeEntry } from '../../types/Board';
import { sendMessageToBackend } from '../../services/backendService'

export default function TribeList() {
  const tribes = useSelector((state: RootState) => state.application.availableTribes);

  const handleSelectTribe = (i: number) => {
    sendMessageToBackend("tribepick", {pickIndex: i})
  };

  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6]">
      <h2 className="text-xl font-bold mb-4 underline">Choose your tribe</h2>
      {tribes.length === 0 ? (
        <div className="italic">Loading tribes...</div>
      ) : (
        <ul className="space-y-2">
          {tribes.map((tribe: TribeEntry, i: number) => (
            <li key={i}>
              <button
                onClick={() => handleSelectTribe(i)}
                className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors"
              >
                {tribe.trait} {tribe.race} - Coins: {tribe.coinCount} - Pieces: {tribe.pieceCount}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
