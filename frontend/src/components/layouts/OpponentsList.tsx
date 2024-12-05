import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { Player } from '../../types/Board';

export default function OpponentsList() {
  const opponents: Player[] = useSelector((state: RootState) => state.application.opponents);
  // Sort so that currently playing players come first
  const sortedOpponents = [...opponents].sort((a, b) => (b.isPlaying ? 1 : 0) - (a.isPlaying ? 1 : 0));

  return (
    <div className="w-full overflow-x-auto">
      <div className="flex space-x-4 py-2">
        {sortedOpponents.map((player) => (
          <div
            key={player.name}
            className={`flex-shrink-0 p-3 border rounded ${
              player.isPlaying ? 'font-bold border-blue-600 bg-blue-50' : 'border-gray-300'
            }`}
          >
            <p className="text-lg">{player.name}</p>
            {player.activeTribe && (
              <p className="text-base mt-1">
                {player.activeTribe.trait} {player.activeTribe.race}
              </p>
            )}
            {player.passiveTribes.length > 0 && (
              <div className="text-sm mt-1 opacity-80">
                {player.passiveTribes.map((tribe, i) => (
                  <p key={i}>
                    {tribe.trait} {tribe.race}
                  </p>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
