import React from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { selectTribe } from '../../redux/slices/applicationSlice';
import { RootState, AppDispatch } from '../../redux/store';
import { Tribe } from '../../types/Board';

export default function TribeList() {
  const tribes = useSelector((state: RootState) => state.application.availableTribes);
  const dispatch: AppDispatch = useDispatch();

  const handleSelectTribe = (tribe: Tribe) => {
    console.log(`Selected tribe: ${tribe}`);
    dispatch(selectTribe(tribe));
  };

  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6]">
      <h2 className="text-xl font-bold mb-4 underline">Choisissez votre tribu</h2>
      {tribes.length === 0 ? (
        <div className="italic">Chargement des tribus...</div>
      ) : (
        <ul className="space-y-2">
          {tribes.map((tribe: Tribe, i) => (
            <li key={i}>
              <button
                onClick={() => handleSelectTribe(tribe)}
                className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors"
              >
                {tribe.trait} {tribe.race}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
