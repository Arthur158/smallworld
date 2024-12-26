// GamePage.tsx

import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';
import { connectWebSocket } from '../services/backendService';
import { parseAreaFile } from '../utility/MapParser';
import { setTiles, reset } from '../redux/slices/applicationSlice';

import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import PlayerInfo from '../components/layouts/PlayerInfo';
import OpponentsList from '../components/layouts/OpponentsList';
import TurnInfoBlock from '../components/layouts/TurnInfoBlock';
import Chat from '../components/inputs/Chat';
import GameFinishedPopup from '../components/layouts/GameFinishedPopup'; // <--- import popup

export default function GamePage() {
  const dispatch: AppDispatch = useDispatch();
  const error = useSelector((state: RootState) => state.application.error);
  const phase = useSelector((state: RootState) => state.application.phase);
  const players = useSelector((state: RootState) => state.application.players); 
  const scores = useSelector((state: RootState) => state.application.scores);

  const [showTribeList, setShowTribeList] = useState(true);

  useEffect(() => {
    const handlePageRefresh = () => {
      dispatch(reset());
    };

    window.addEventListener('beforeunload', handlePageRefresh);

    return () => {
      window.removeEventListener('beforeunload', handlePageRefresh);
    };
  }, [dispatch]);

  return (
<div className="w-screen h-screen overflow-hidden bg-[#F5F5DC] font-serif text-[#5F4B32] relative">
  {/* Main Content */}
  <div className="flex w-full h-full">
    {/* Left column: 1/3 width, full height */}
    <div className="w-1/3 h-full flex flex-col">
      <div className="flex h-1/2">
        <div className="w-1/2 flex flex-col border border-[#5F4B32] bg-[#FDF5E6]">
          <div className="flex-1 overflow-y-auto p-2">
            <PlayerInfo />
            <TurnInfoBlock />
          </div>
        </div>
        <div className="w-1/2 flex flex-col border border-[#5F4B32] bg-[#FDF5E6]">
          <div className="flex-1 overflow-y-auto p-2">
            <Chat />
          </div>
        </div>
      </div>

      <div className="h-1/2 border border-[#5F4B32] bg-[#FDF5E6] flex flex-col">
        <div className="p-2">
          <button
            className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded"
            onClick={() => setShowTribeList((prev) => !prev)}
          >
            {showTribeList ? 'Show Opponents' : 'Show Tribes'}
          </button>
        </div>
        <div className="flex-1 overflow-y-auto p-2">
          {showTribeList ? <TribeList /> : <OpponentsList />}
        </div>
      </div>
    </div>

    {/* Right column: 2/3 width, full height for the Map */}
    <div className="w-2/3 h-full p-4">
      <div className="w-full h-full border-4 border-[#8B4513] rounded-lg bg-white">
        <Map />
      </div>
    </div>
  </div>

  {/* Popup for GameFinished */}
  <GameFinishedPopup />

  {/* Error Banner at the Bottom */}
  {error && (
    <div className="fixed bottom-0 left-0 w-full bg-red-500 text-white text-center py-2 z-50">
      {error}
    </div>
  )}
</div>
  );
}
