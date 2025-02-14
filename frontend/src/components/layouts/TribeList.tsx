import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { TribeEntry } from '../../types/Board';
import { sendMessageToBackend } from '../../services/backendService';

export default function TribeList() {
  const tribes = useSelector((state: RootState) => state.application.availableTribes);

  const handleSelectTribe = (i: number) => {
    sendMessageToBackend('tribepick', { pickIndex: i });
  };

  const getTraitImagePath = (trait?: string) => {
    return trait && trait.trim() !== ''
      ? `/traits/${trait}.png`
      : '/traits/Alchemist.png';
  };

  const getRaceImagePath = (race?: string) => {
    return race && race.trim() !== ''
      ? `/races/${race}.png`
      : `/races/Trolls.png`;
  };

  return (
    <div className="p-4 border border-[#5F4B43] rounded bg-[#FDF5E6]">
      {tribes.length === 0 ? (
        <div className="italic text-center">Loading tribes...</div>
      ) : (
        <ul className="space-y-3">
          {tribes.map((tribe: TribeEntry, i: number) => (
            <li key={i}>
              <button
                onClick={() => handleSelectTribe(i)}
                className="flex items-center transition-transform hover:scale-105 w-full p-0 border-none bg-transparent"
              >
                <div className="relative flex items-center w-full">
                  <img
                    src={getTraitImagePath(tribe.trait)}
                    alt={tribe.trait}
                    className="h-24 w-auto -mr-1 z-20"
                  />
                  <img
                    src={getRaceImagePath(tribe.race)}
                    alt={tribe.race}
                    className="h-24 w-auto z-10"
                  />
                </div>
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
