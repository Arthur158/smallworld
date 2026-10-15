import React from 'react';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService';
import { setIsStackFromBank, setSelectedStack, setSelectedTile } from '../../redux/slices/applicationSlice';
import { useSelector, useDispatch } from 'react-redux';
import { useShortcut } from '../../hooks/useShortcut';
import { useWiki } from '../wiki/WikiContext';

const buttonClass = 'w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-3 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed';
const destructiveClass = 'w-full bg-red-600 hover:bg-red-700 text-white py-2 px-3 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed';

export default function TurnInfoBlock() {
  const playerNumber = useSelector((state: RootState) => state.application.playerNumber);
  const playerIndex = useSelector((state: RootState) => state.application.playerIndex);
  const phase = useSelector((state: RootState) => state.application.phase);
  const turnNumber = useSelector((state: RootState) => state.application.turnNumber);
  const players = useSelector((state: RootState) => state.application.players);
  const coins = useSelector((state: RootState) => state.application.coins);
  const isSpectating = useSelector((state: RootState) => state.application.isSpectating);
  const inDisplayRoom = useSelector((state: RootState) => state.application.inDisplayRoom);
  const { inspectMode, isOpen } = useWiki();
  const dispatch = useDispatch();
  const currentPlayer = players[playerNumber]?.name || 'Unknown Player';
  const isPlayer = !!players[playerIndex] && !isSpectating && !inDisplayRoom && phase !== 'GameFinished';
  const ownTurn = isPlayer && playerIndex === playerNumber && !inspectMode && !isOpen;
  // Borrowed powers are not exposed in the board payload. The server checks
  // Stout and race-specific end-turn requirements after these phase guards.
  const canDecline = ownTurn && !!players[playerIndex]?.activeTribe && ['DeclineChoice', 'Redeployment'].includes(phase);
  const canRedeploy = ownTurn && ['DeclineChoice', 'TileAbandonment', 'Conquest'].includes(phase);
  const canEndTurn = ownTurn && ['DeclineChoice', 'TileAbandonment', 'Conquest', 'Redeployment'].includes(phase);
  const canSave = isPlayer && !isOpen;
  const canLeave = !isOpen;

  const resetSelections = () => {
    dispatch(setIsStackFromBank(false));
    dispatch(setSelectedTile(null));
    dispatch(setSelectedStack(null));
  };
  const handleDecline = () => {
    if (!canDecline) return;
    sendMessageToBackend('decline', {});
    resetSelections();
  };
  const handleRedeploy = () => {
    if (!canRedeploy) return;
    sendMessageToBackend('startredeployment', {});
    resetSelections();
  };
  const handleEndTurn = () => {
    if (!canEndTurn) return;
    sendMessageToBackend('finishturn', {});
    resetSelections();
  };
  const handleLeaveGame = () => {
    if (!canLeave) return;
    sendMessageToBackend('leaveroom', {});
    resetSelections();
  };
  const handleSaveGame = () => {
    if (canSave) sendMessageToBackend('savegame', {});
  };
  const handleRollBack = () => {
    if (ownTurn) sendMessageToBackend('rollback', {});
  };

  useShortcut('d', handleDecline, canDecline);
  useShortcut('r', handleRedeploy, canRedeploy);
  useShortcut('e', handleEndTurn, canEndTurn);
  useShortcut('Shift+s', handleSaveGame, canSave);
  useShortcut('Shift+r', handleRollBack, ownTurn);
  useShortcut('Shift+l', handleLeaveGame, canLeave);

  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6] shadow-md">
      <h2 className="text-xl font-bold underline mb-2">Turn Information</h2>
      <p><span className="font-semibold">Current Player:</span> {currentPlayer}</p>
      <p><span className="font-semibold">Turn Number:</span> {turnNumber}</p>
      <p><span className="font-semibold">Phase:</span> {phase}</p>
      <p><span className="font-semibold">Coin Pile:</span> {coins}</p>
      <div className="grid gap-2 mt-4" style={{ gridTemplateColumns: 'repeat(auto-fit, minmax(min(100%, 7rem), 1fr))' }}>
        <button type="button" onClick={handleDecline} disabled={!canDecline} className={buttonClass}
          data-tooltip="Put your active race in decline. During redeployment, requires Stout. Shortcut: D." aria-keyshortcuts="D">
          Decline <span className="text-sm text-gray-300">(D)</span>
        </button>
        <button type="button" onClick={handleRedeploy} disabled={!canRedeploy} className={buttonClass}
          data-tooltip="Finish conquests and redistribute your active pieces. Shortcut: R." aria-keyshortcuts="R">
          Redeploy <span className="text-sm text-gray-300">(R)</span>
        </button>
        <button type="button" onClick={handleEndTurn} disabled={!canEndTurn} className={buttonClass}
          data-tooltip="Complete your turn and score your regions. Shortcut: E." aria-keyshortcuts="E">
          End Turn <span className="text-sm text-gray-300">(E)</span>
        </button>
        <button type="button" onClick={handleSaveGame} disabled={!canSave} className={buttonClass}
          data-tooltip="Save the current game so it can be loaded later. Shortcut: Shift+S." aria-keyshortcuts="Shift+S">
          Save Game
        </button>
        <button type="button" onClick={handleRollBack} disabled={!ownTurn} className={destructiveClass}
          data-tooltip="Restore the latest turn autosave during your turn. Shortcut: Shift+R." aria-keyshortcuts="Shift+R">
          Rollback
        </button>
        <button type="button" onClick={handleLeaveGame} disabled={!canLeave} className={destructiveClass}
          data-tooltip="Leave this game and return to the lobby. Shortcut: Shift+L." aria-keyshortcuts="Shift+L">
          Leave Game
        </button>
      </div>
    </div>
  );
}
