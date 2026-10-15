import WikiToolbar from '../components/wiki/WikiToolbar';
import { useShortcut } from '../hooks/useShortcut';
import React, { useEffect, useState } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState, AppDispatch } from '../redux/store';
import { sendMessageToBackend } from '../services/backendService';
import { reset, clearError } from '../redux/slices/applicationSlice';

import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import OpponentsListAll from '../components/layouts/OpponentsListAll';
import TurnInfoBlockDisplay from '../components/layouts/TurnInfoBlockDisplay';

export default function DisplayPage() {
  const dispatch: AppDispatch = useDispatch();
  const error = useSelector((state: RootState) => state.application.error);
  const saveGames = useSelector((state: RootState) => state.application.saveGames);
  const mapChoices = useSelector((state: RootState) => state.application.mapChoices);
  const saveSelectionId = useSelector((state: RootState) => state.application.saveSelectionId);
  const mapName = useSelector((state: RootState) => state.application.mapName);

  const [showPlayers, setShowPlayers] = useState(true);
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

  useShortcut('s', () => setShowPlayers(previous => !previous));
  useShortcut('c', () => dispatch(clearError()), !!error);

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
  className={`flex items-start p-4 bg-white rounded border-2 transition-colors ${
    gameSave.saveId === saveSelectionId ? 'border-[#8B4513]' : 'border-transparent'
  }`}
>
  <button type="button" className="flex-1 min-w-0 text-left"
    aria-label={`Load saved game ${gameSave.saveId}`}
    data-tooltip="Load this saved game for inspection · Enter / Space"
    onClick={() => handleGameIdClick(gameSave.saveId)}>
    <span className="block font-bold">Game ID: {gameSave.saveId}</span>
    <span className="block text-sm text-gray-600">{gameSave.summary}</span>
  </button>
  <button type="button" aria-label={`Delete saved game ${gameSave.saveId}`}
    data-tooltip="Delete this saved game"
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
                      className={`relative bg-white rounded border-2 transition-colors ${
                        mapName === otherMapName ? 'border-[#8B4513]' : 'border-transparent'
                      }`}
                    >
                      <button type="button" className="w-full p-4 text-left"
                        aria-label={`Load map ${otherMapName}`}
                        data-tooltip="Load this map for inspection · Enter / Space"
                        onClick={() => handleMapClick(otherMapName)}>
                        <span className="font-bold">Map name: {otherMapName}</span>
                      </button>
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
                type="button"
                onClick={() => setShowPlayers(previous => !previous)}
                aria-keyshortcuts="S"
                data-tooltip={showPlayers ? 'Show available race and trait choices · S' : 'Show players and their tribes · S'}
                className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white font-bold py-1 px-2 rounded mt-2 mb-2"
              >
                {showPlayers ? 'Show Tribes' : 'Show Players'}
              </button>
              <div className="flex-1 overflow-auto min-h-0">
                {showPlayers ? <OpponentsListAll /> : <TribeList />}
              </div>
            </div>
          </div>
        </div>

        {/* RIGHT SECTION (2/3 width): Map */}
        <div className="w-2/3 h-full p-4 flex flex-col min-h-0">
          <WikiToolbar />
          <div className="w-full flex-1 min-h-0 border-4 border-[#8B4513] rounded-lg bg-white">
            <Map />
          </div>
        </div>
      </div>
      {error && (
        <div role="alert" className="fixed bottom-0 left-0 w-full bg-red-500 text-white py-2 px-4 z-50 flex items-center justify-center gap-3">
          <span>{error}</span>
          <button type="button" onClick={() => dispatch(clearError())} aria-label="Dismiss error"
            aria-keyshortcuts="C" data-tooltip="Dismiss this message · C"
            className="shrink-0 rounded border border-white/70 px-2 py-1 text-sm hover:bg-white/15">
            Dismiss <kbd className="ml-1 text-xs">C</kbd>
          </button>
        </div>
      )}
    </div>
  );
}
