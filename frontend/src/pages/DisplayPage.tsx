import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';
import { sendMessageToBackend } from '../services/backendService';
import { reset, clearError } from '../redux/slices/applicationSlice';

import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import PlayerInfo from '../components/layouts/PlayerInfo';
import OpponentsListAll from '../components/layouts/OpponentsListAll';
import TurnInfoBlockDisplay from '../components/layouts/TurnInfoBlockDisplay';
import Chat from '../components/inputs/Chat';

export default function DisplayPage() {
  const dispatch: AppDispatch = useDispatch();
  const saveGames = useSelector((state: RootState) => state.application.saveGames);
  const mapChoices = useSelector((state: RootState) => state.application.mapChoices);
  const saveSelectionId = useSelector((state: RootState) => state.application.saveSelectionId);
  const mapName = useSelector((state: RootState) => state.application.mapName);

  const [showTurnInfo, setShowTurnInfo] = useState(false);
  // Callback when a saved game is clicked
  const handleMapClick = (mapName: string) => {
    sendMessageToBackend('loadmapdisplay', { mapName: mapName });
  };

  const handleGameIdClick = (gameId: number) => {
    sendMessageToBackend('loadgamedisplay', { saveId: gameId });
  };

  const handleDeleteGame = (gameId: number) => {
    sendMessageToBackend('deletesave', { saveId: gameId });
  };

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
          <div className="h-full w-1/2 p-0 flex flex-col min-h-0">
            {/* Top: PlayerInfo (fixed height: h-1/3) */}
            <div className="space-y-6 overflow-y-auto border border-[#5F4B32] bg-[#FDF5E6] p-4">
              <h2 className="text-xl font-bold underline mb-2">Saved Games</h2>
              {saveGames && saveGames.length > 0 ? (
                <ul className="space-y-4">
                  {saveGames.map((gameSave) => (
<li
  key={gameSave.saveId}
  className={`flex items-start p-4 cursor-pointer bg-white rounded border-2 transition-colors ${
    gameSave.saveId === saveSelectionId ? 'border-[#8B4513]' : 'border-transparent'
  }`}
  onClick={() => handleGameIdClick(gameSave.saveId)}
>
  <div className="flex-1">
    <div className="font-bold">Game ID: {gameSave.saveId}</div>
    <div className="text-sm text-gray-600">{gameSave.summary}</div>
  </div>
  <button
    onClick={(e) => {
      e.stopPropagation();
      handleDeleteGame(gameSave.saveId);
    }}
    className="ml-auto bg-red-600 hover:bg-red-700 text-white py-1 px-3 rounded transition-colors"
  >
    Delete
  </button>
</li>
                  ))}
                </ul>
              ) : (
                <p className="text-gray-600">No saved games available.</p>
              )}

              <h2 className="text-xl font-bold underline mb-2">Map Choices</h2>
              {mapChoices && mapChoices.length > 0 ? (
                <ul className="space-y-4">
                  {mapChoices.map((otherMapName) => (
                    <li
                      key={otherMapName} // correct unique key
                      className={`relative flex items-center p-4 cursor-pointer bg-white rounded border-2 transition-colors ${
                        mapName === otherMapName ? 'border-[#8B4513]' : 'border-transparent'
                      }`}
                      onClick={() => handleMapClick(otherMapName)}
                    >
                      <div className="flex-1">
                        <div className="font-bold">Map name: {otherMapName}</div>
                      </div>
                    </li>
                  ))}
                </ul>
              ) : (
                <p className="text-gray-600">No maps available.</p>
              )}
            </div>
          </div>

          {/* SUBCOLUMN B (2/5) */}
          <div className="w-1/2 h-full flex flex-col p-1 min-h-0">
            {/* TurnInfoBlock on top (auto height) */}
            <div className="flex-none mb-2">
              <TurnInfoBlockDisplay />
            </div>
            {/* Chat below (scrollable if large) */}
            <div className="flex-1 flex flex-col min-h-0 overflow-hidden">
              <button
                onClick={() => setShowTurnInfo(!showTurnInfo)}
                className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white font-bold py-1 px-2 rounded mt-2 mb-2"
              >
                {showTurnInfo ? 'Show Tribe List' : 'Show Turn Info'}
              </button>
              <div className="flex-1 overflow-auto min-h-0">
                {showTurnInfo ? <OpponentsListAll /> : <TribeList />}
              </div>
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
    </div>
  );
}
