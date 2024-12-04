import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { selectTribe } from '../../redux/slices/applicationSlice';
import { RootState, AppDispatch } from '../../redux/store';


export default function TribeList() {
  const tribes = useSelector((state: RootState) => state.application.availableTribes);
  const dispatch: AppDispatch = useDispatch();

  const handleSelectTribe = (tribe: string) => {
    console.log(`Selected tribe: ${tribe}`);
    dispatch(selectTribe(tribe));
  };

  return (
    <div className="p-4">
      <h2 className="text-xl font-bold mb-4">Choose Your Tribe</h2>
      {tribes.length === 0 ? (
        <div>Loading tribes...</div>
      ) : (
        <ul>
          {tribes.map((tribe: string) => (
            <li className="mb-2">
              <button
                onClick={() => handleSelectTribe(tribe)}
                className="bg-blue-500 text-white py-2 px-4 rounded"
              >
                {tribe}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
