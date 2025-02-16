import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';
import { sendMessageToBackend } from '../services/backendService';
import { reset, clearError } from '../redux/slices/applicationSlice';

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
  const phase = useSelector((state: RootState) => state.application.phase);
  const playerNumber = useSelector((state: RootState) => state.application.playerNumber);
  const playerIndex = useSelector((state: RootState) => state.application.playerIndex);
  const players = useSelector((state: RootState) => state.application.players);

  const currentPlayer = players[playerNumber]?.name || 'Unknown Player';
  const [showTurnInfo, setShowTurnInfo] = useState(false);

  // Auto-toggle the middle section based on phase
  useEffect(() => {
    if (phase === 'Conquest') {
      setShowTurnInfo(true);
    } else if (playerNumber === playerIndex && phase === 'TribeChoice') {
      setShowTurnInfo(false);
    }
  }, [phase, playerNumber, playerIndex]);

  // Keyboard shortcuts
  useEffect(() => {
    const handleKeyPress = (event: KeyboardEvent) => {
      switch (event.key) {
        case 's':
          setShowTurnInfo((prev) => !prev);
          break;
        case 'c':
          dispatch(clearError());
          break;
      }
    };
    window.addEventListener('keydown', handleKeyPress);
    return () => {
      window.removeEventListener('keydown', handleKeyPress);
    };
  }, [dispatch]);

  // Handle page refresh
  useEffect(() => {
    const handlePageRefresh = () => {
      dispatch(reset());
      sendMessageToBackend('requestrefresh', {});
    };
    window.addEventListener('beforeunload', handlePageRefresh);
    return () => {
      window.removeEventListener('beforeunload', handlePageRefresh);
    };
  }, [dispatch]);

  return (
    <div className="w-screen h-screen overflow-hidden bg-[#F5F5DC] font-serif text-[#5F4B32] relative">
      <div className="flex w-full h-full">
        {/* LEFT SECTION (1/3 width) */}
        <div className="w-1/3 h-full flex p-2 min-h-0">
          {/* SUBCOLUMN A (3/5) */}
          <div className="h-full w-3/5 p-0 flex flex-col min-h-0">
            {/* Top: PlayerInfo (fixed height: h-1/3) */}
            <div className="h-1/3 min-h-[150px]">
              <PlayerInfo />
            </div>
            {/* Middle/Bottom: Toggle + OpponentsList/TribeList */}
            <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
              <button
                onClick={() => setShowTurnInfo(!showTurnInfo)}
                className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white font-bold py-1 px-2 rounded mt-2 mb-2"
              >
                {showTurnInfo ? 'Show Tribe List' : 'Show Turn Info'}
              </button>
              <div className="flex-1 overflow-auto min-h-0">
                {showTurnInfo ? <OpponentsList /> : <TribeList />}
              </div>
            </div>
          </div>

          {/* SUBCOLUMN B (2/5) */}
          <div className="w-2/5 h-full flex flex-col p-1 min-h-0">
            {/* TurnInfoBlock on top (auto height) */}
            <div className="flex-none mb-2">
              <TurnInfoBlock />
            </div>
            {/* Chat below (scrollable if large) */}
            <div className="flex-1 overflow-auto min-h-0">
              <Chat />
            </div>
          </div>
        </div>

        {/* RIGHT SECTION (2/3 width): Map */}
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
