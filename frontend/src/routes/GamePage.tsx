// GamePage.tsx

import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';
import { connectWebSocket } from '../services/backendService';
import { parseAreaFile } from '../utility/MapParser';
import { setTiles, updateTileStack } from '../redux/slices/applicationSlice';

import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import PlayerInfo from '../components/layouts/PlayerInfo';
import OpponentsList from '../components/layouts/OpponentsList';
import TurnInfoBlock from '../components/layouts/TurnInfoBlock';
import Chat from '../components/inputs/Chat'; // <-- On suppose qu'on l'a mis ici

export default function GamePage() {
  const dispatch: AppDispatch = useDispatch();
  const error = useSelector((state: RootState) => state.application.error);

  const [showTribeList, setShowTribeList] = useState(true);

  useEffect(() => {
    connectWebSocket();

    const loadAreas = async () => {
      try {
        const response = await fetch('/maps/map.txt');
        const text = await response.text();
        const polygons = parseAreaFile(text);

        let id = 0;
        const tileData = polygons.map((polygon) => ({
          id: id++,
          pieceStack: [],
          polygon,
        }));

        dispatch(setTiles(tileData));

      } catch (error) {
        console.error('Error loading file:', error);
      }
    };
    loadAreas();
  }, [dispatch]);

  return (
    <div className="w-screen h-screen overflow-hidden bg-[#F5F5DC] font-serif text-[#5F4B32] relative">
      {/* Bannière d'erreur */}
      {error && (
        <div className="absolute top-0 left-0 w-full bg-red-500 text-white text-center py-2 z-50">
          {error}
        </div>
      )}

      {/* Conteneur principal (enlève la marge haute si une erreur est affichée) */}
      <div
        className="flex flex-row w-full h-full"
        style={{ marginTop: error ? '2.5rem' : 0 }}
      >
        {/* Colonne de gauche */}
        <div className="flex flex-col p-4 space-y-4 w-1/3 h-full">
          {/* Player Info */}
          <div>
            <div className="border border-[#5F4B32] bg-[#FDF5E6] rounded p-2 flex-shrink-0">
              <PlayerInfo />
            </div>
            </div>

          {/* Turn Info */}
          <div className="border border-[#5F4B32] bg-[#FDF5E6] rounded p-2 flex-shrink-0">
            <TurnInfoBlock />
          </div>

          {/* Tribe List */}
          <div className="border border-[#5F4B32] bg-[#FDF5E6] rounded p-2 flex-shrink-0">
            <button
              className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors mb-2"
              onClick={() => setShowTribeList((prev) => !prev)}
            >
              {showTribeList ? 'Hide Tribes' : 'Show Tribes'}
            </button>
            {showTribeList && <TribeList />}
          </div>

          {/* Chat (prend l'espace restant s'il y en a) */}
          <div className="flex-grow">
            <Chat />
          </div>
          {/* OpponentsList en bas */}
          <div className="h-1/4 border border-[#5F4B32] bg-[#FDF5E6] rounded p-2">
            <OpponentsList />
          </div>
        </div>

        {/* Colonne de droite */}
        <div className="flex flex-col w-2/3 h-full p-4">
          {/* Zone principale pour la Map */}
          <div className="flex-grow mb-4">
            <div className="border-4 border-[#8B4513] rounded-lg shadow-md bg-white w-full h-full p-1">
              <Map />
            </div>
          </div>

        </div>
      </div>
    </div>
  );
}
