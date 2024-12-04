import React from 'react';
import { useSelector } from 'react-redux';
import { RootState, AppDispatch } from '../../redux/store';

export default function PlayerInfo() {
  const selectedTribe = useSelector((state: RootState) => state.application.selectedTribe);



  if (!selectedTribe) {
    return <div className="p-4">Please select a tribe to start the game.</div>;
  }

  return (
    <div className="p-4">
      <h2 className="text-xl font-bold mb-2">Your Tribe</h2>
      <p>
        <strong>Race:</strong> {selectedTribe}
      </p>
      <p>
        <strong>Power:</strong> {selectedTribe}
      </p>
    </div>
  );
}
