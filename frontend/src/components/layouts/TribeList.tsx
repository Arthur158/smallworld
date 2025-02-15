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
    return trait && trait.trim() !== '' ? `/traits/${trait}.png` : '/traits/Alchemist.png';
  };

  const getRaceImagePath = (race?: string) => {
    return race && race.trim() !== '' ? `/races/${race}.png` : '/races/Trolls.png';
  };

  const renderTribeImages = (trait: string, race: string) => {
    return (
      <div className="relative flex items-center h-24"> {/* Same height as in OpponentsList */}
        <img src={getTraitImagePath(trait)} alt={trait} className="h-full w-auto -mr-1 z-20" />
        <img src={getRaceImagePath(race)} alt={race} className="h-full w-auto z-10" />
      </div>
    );
  };

  return (
    <div className="p-4 border border-[#5F4B43] rounded bg-[#FDF5E6] h-full min-h-[450px] flex flex-col">
      {tribes.length === 0 ? (
        <div className="italic text-center flex-1 flex items-center justify-center">
          Loading tribes...
        </div>
      ) : (
        <ul className="space-y-3 overflow-y-auto pr-2 flex-1">
          {tribes.map((tribe: TribeEntry, i: number) => (
            <li key={i}>
              <button
                onClick={() => handleSelectTribe(i)}
                className="flex items-center transition-transform hover:scale-105 w-full p-0 border-none bg-transparent"
              >
                {renderTribeImages(tribe.trait, tribe.race)}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
