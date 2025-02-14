import React, { useEffect } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';
import { reset } from '../redux/slices/applicationSlice';

import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import PlayerInfo from '../components/layouts/PlayerInfo';
import OpponentsList from '../components/layouts/OpponentsList';
import TurnInfoBlock from '../components/layouts/TurnInfoBlock';
import Chat from '../components/inputs/Chat';
import GameFinishedPopup from '../components/layouts/GameFinishedPopup';

export default function GamePage() {
  const dispatch: AppDispatch = useDispatch();
  const error = useSelector((state: RootState) => state.application.error);

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
      <div className="flex w-full h-full">
        {/* Left column (1/3 width) */}
        <div className="w-1/3 h-full flex flex-col space-y-2 p-2">
          
          {/* Tribe List (Now Larger) */}
          <div className="flex-[1.3] w-3/5 border border-[#5F4B32] bg-[#FDF5E6] p-2 overflow-y-auto">
            <TribeList />
          </div>

          {/* Player Info + Turn Info Block (Stacked) */}
          <div className="flex-1 border border-[#5F4B32] bg-[#FDF5E6] p-2">
            <PlayerInfo />
            <TurnInfoBlock />
          </div>

          {/* Chat + OpponentsList (Still 50/50 but resized) */}
          <div className="flex-1 flex space-x-2">
            <div className="w-1/2 border border-[#5F4B32] bg-[#FDF5E6] p-2">
              <Chat />
            </div>
            <div className="w-1/2 border border-[#5F4B32] bg-[#FDF5E6] p-2">
              <OpponentsList />
            </div>
          </div>
        </div>

        {/* Right column (2/3 width) */}
        <div className="w-2/3 h-full p-4">
          <div className="w-full h-full border-4 border-[#8B4513] rounded-lg bg-white">
            <Map />
          </div>
        </div>
      </div>

      {/* Game Finished Popup */}
      <GameFinishedPopup />

      {/* Error Banner */}
      {error && (
        <div className="fixed bottom-0 left-0 w-full bg-red-500 text-white text-center py-2 z-50">
          {error}
        </div>
      )}
    </div>
  );
}
