import React from 'react';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService';
import { setIsStackFromBank, setSelectedStack, setSelectedTile } from '../../redux/slices/applicationSlice';
import { useSelector, useDispatch } from 'react-redux';
import { useShortcut } from '../../hooks/useShortcut';
import { useWiki } from '../wiki/WikiContext';

export default function TurnInfoBlockSpectate() {
  const playerNumber = useSelector((state: RootState) => state.application.playerNumber);
  const phase = useSelector((state: RootState) => state.application.phase);
  const turnNumber = useSelector((state: RootState) => state.application.turnNumber);
  const players = useSelector((state: RootState) => state.application.players);
  const { isOpen } = useWiki();
  const dispatch = useDispatch();
  const currentPlayer = players[playerNumber]?.name || 'Unknown Player';
  const handleLeaveGame = () => {
    if (isOpen) return;
    sendMessageToBackend('leaveroom', {});
    dispatch(setIsStackFromBank(false));
    dispatch(setSelectedTile(null));
    dispatch(setSelectedStack(null));
  };
  useShortcut('Shift+l', handleLeaveGame, !isOpen);

  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6] shadow-md">
      <h2 className="text-xl font-bold underline mb-2">Turn Information</h2>
      <p><span className="font-semibold">Current Player:</span> {currentPlayer}</p>
      <p><span className="font-semibold">Turn Number:</span> {turnNumber}</p>
      <p><span className="font-semibold">Phase:</span> {phase}</p>
      <div className="mt-4">
        <button type="button" onClick={handleLeaveGame} disabled={isOpen}
          data-tooltip="Stop spectating and return to the lobby. Shortcut: Shift+L." aria-keyshortcuts="Shift+L"
          className="w-full bg-red-600 hover:bg-red-700 text-white py-2 px-3 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          Leave Game
        </button>
      </div>
    </div>
  );
}
