import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';
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
  const dynamicPlaceholder = "Game Event Placeholder"; // Can be dynamic later

  const [showTurnInfo, setShowTurnInfo] = useState(false);

  useEffect(() => {
    if (phase == "Conquest") {
      setShowTurnInfo(true)
    } else if (playerNumber == playerIndex && phase == "TribeChoice") {
      setShowTurnInfo(false)
    }
  }, [playerIndex, playerNumber, phase, setShowTurnInfo])

  useEffect(() => {
    const handleKeyPress = (event: KeyboardEvent) => {
      switch (event.key) {
        case 's':
          if (showTurnInfo) {
            setShowTurnInfo(false)
          } else {
            setShowTurnInfo(true)
          }
          break;
        case 'c':
          dispatch(clearError())
          break;
      }
    };

    window.addEventListener('keydown', handleKeyPress);
    return () => {
      window.removeEventListener('keydown', handleKeyPress);
    };
  }, [setShowTurnInfo, showTurnInfo]);

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
        <div className="w-1/3 h-full flex p-2">
          
          {/* Toggleable Section - TribeList OR Player/Opponents List */}
          <div className="h-full w-3/5 p-0">
            <div className='h-1/3'>
            <PlayerInfo />
            </div>
            <div className="flex-1 flex flex-col min-h-[600px]"> 
            <button
              onClick={() => setShowTurnInfo(!showTurnInfo)}
              className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white font-bold py-1 px-2 rounded mb-1 mt-1"
            >
              {showTurnInfo ? 'Show Tribe List' : 'Show Turn Info'}
            </button>
              {showTurnInfo ? (
                <div className="flex flex-col flex-grow"> 
                  <div className="h-2/3 ">
                    <OpponentsList />
                  </div>
                </div>
              ) : (
                <div className="h-2/3 "> 
                  <div className="h-2/3 ">
                    <TribeList />
                  </div>
                </div>
              )}
            </div>
          </div>

          {/* Right section of the left column */}
          <div className="w-2/5 h-full flex flex-col p-1">
            {/* Turn Info Block (Above Chat) */}
            <div className="p-1">
              <TurnInfoBlock />
            </div>
            
            {/* Chat Component */}
            <div className="h-3/5 p-1">
              <Chat />
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
