import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';

export default function PlayerInfo() {
  const activeTribe = useSelector((state: RootState) => state.application.player.activeTribe);

  return (
    <div className="p-4">
      {!activeTribe ? (
        <div className="text-red-500">Select your tribe to start the game.</div>
      ) : (
        <>
          <h2 className="text-xl font-bold mb-2">Your Tribe</h2>
          <p>
            <strong>Race:</strong> {activeTribe.race}
          </p>
          <p>
            <strong>Power:</strong> {activeTribe.trait}
          </p>
        </>
      )}
    </div>
  );
}
