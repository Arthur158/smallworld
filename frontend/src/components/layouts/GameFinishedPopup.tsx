import React, { useEffect, useRef } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService';
import { setIsStackFromBank, setSelectedStack, setSelectedTile } from '../../redux/slices/applicationSlice';
import { useWiki } from '../wiki/WikiContext';

const hasOtherModal = (ownDialog: Element | null) =>
  Array.from(document.querySelectorAll<HTMLElement>('[role="dialog"][aria-modal="true"]')).some(dialog =>
    dialog !== ownDialog && !dialog.closest('[hidden], [aria-hidden="true"]') &&
    window.getComputedStyle(dialog).display !== 'none' && window.getComputedStyle(dialog).visibility !== 'hidden');

export default function GameFinishedPopup() {
  const phase = useSelector((state: RootState) => state.application.phase);
  const players = useSelector((state: RootState) => state.application.players);
  const scores = useSelector((state: RootState) => state.application.scores);
  const username = useSelector((state: RootState) => state.application.name);
  const currentRoom = useSelector((state: RootState) => {
    const { roomid, rooms, roomsInProgress } = state.application;
    return roomsInProgress?.find(room => room.id === roomid) ?? rooms?.find(room => room.id === roomid);
  });
  const { inspectMode, isOpen } = useWiki();
  const dispatch = useDispatch();
  const dialogRef = useRef<HTMLDivElement>(null);
  const finished = phase === 'GameFinished';
  const hasHistory = currentRoom?.creator === username &&
    currentRoom.botPlayback?.canGoBack && currentRoom.players.length > 0 &&
    currentRoom.players.every(player => currentRoom.bots?.[player]);
  const canGoBack = finished && !!hasHistory && !inspectMode && !isOpen;

  useEffect(() => {
    if (!finished) return;
    const previousFocus = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const dialog = dialogRef.current;
    let waitingForFocus = true;
    let ownedFocus = false;
    const updateFocus = () => {
      if (hasOtherModal(dialog)) {
        waitingForFocus = true;
        return;
      }
      if (!waitingForFocus) return;
      waitingForFocus = false;
      ownedFocus = true;
      // A closing guide may already have restored a particular results button.
      if (!dialog?.contains(document.activeElement)) {
        (dialog?.querySelector<HTMLButtonElement>('button:not(:disabled)') ?? dialog)?.focus();
      }
    };
    updateFocus();
    // Games can finish beneath the guide or wiki. Resume only after that modal
    // closes, after its own focus restoration has completed.
    const observer = new MutationObserver(updateFocus);
    observer.observe(document.body, {
      childList: true, subtree: true, attributes: true,
      attributeFilter: ['role', 'aria-modal', 'hidden', 'aria-hidden', 'style'],
    });
    return () => {
      observer.disconnect();
      if (ownedFocus && !hasOtherModal(dialog) && previousFocus?.isConnected) previousFocus.focus();
    };
  }, [finished]);

  const previous = () => {
    if (canGoBack && !hasOtherModal(dialogRef.current)) sendMessageToBackend('previousBotMove', {});
  };
  const handleLeaveGame = () => {
    if (!finished || isOpen || hasOtherModal(dialogRef.current)) return;
    sendMessageToBackend('leaveroom', {});
    dispatch(setIsStackFromBank(false));
    dispatch(setSelectedTile(null));
    dispatch(setSelectedStack(null));
  };
  const handleDialogKey = (event: React.KeyboardEvent<HTMLDivElement>) => {
    if (isOpen || hasOtherModal(dialogRef.current) || event.defaultPrevented || event.repeat || event.nativeEvent.isComposing || event.ctrlKey || event.metaKey || event.altKey) return;
    if (event.target instanceof Element && event.target.closest('input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="textbox"]')) return;
    if (event.key === 'Tab') {
      const buttons = Array.from(dialogRef.current?.querySelectorAll<HTMLButtonElement>('button:not(:disabled)') ?? []);
      const first = buttons[0];
      const last = buttons[buttons.length - 1];
      if (first && last && (document.activeElement === dialogRef.current || (event.shiftKey ? document.activeElement === first : document.activeElement === last))) {
        event.preventDefault();
        (event.shiftKey ? last : first).focus();
      }
    } else if (event.key === 'ArrowLeft' && !event.shiftKey && canGoBack) {
      event.preventDefault();
      previous();
    } else if (event.key.toLowerCase() === 'l' && event.shiftKey) {
      event.preventDefault();
      handleLeaveGame();
    }
  };

  if (!finished) return null;
  const combined = players.map((player, index) => ({ player: player.name, score: scores[index] }));
  combined.sort((a, b) => b.score - a.score);
  const rankLabel = (index: number) => index === 0 ? 'Winner' : index === 1 ? '2nd' : index === 2 ? '3rd' : `${index + 1}th`;

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
      <div ref={dialogRef} role="dialog" aria-modal="true" aria-labelledby="game-results-title" tabIndex={-1}
        onKeyDown={handleDialogKey} className="bg-white p-4 rounded shadow-md">
        <h2 id="game-results-title" className="text-xl font-bold mb-2">Game Results</h2>
        {combined.map((item, index) => (
          <div key={`${item.player}-${index}`} className="mb-1">
            <strong>{rankLabel(index)}:</strong> {item.player} - {item.score}
          </div>
        ))}
        <div className="flex flex-col gap-2 mt-3">
          {hasHistory && (
            <button type="button" onClick={previous} disabled={!canGoBack}
              data-tooltip="Return to the move before the game finished, with playback paused. Shortcut: Left arrow." aria-keyshortcuts="ArrowLeft"
              className="w-full border border-[#8B4513] hover:bg-[#EED5B7] py-2 px-3 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              Previous move
            </button>
          )}
          <button type="button" onClick={handleLeaveGame} disabled={isOpen}
            data-tooltip="Leave the results and return to the lobby. Shortcut: Shift+L." aria-keyshortcuts="Shift+L"
            className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-3 rounded transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
            Leave Game
          </button>
        </div>
      </div>
    </div>
  );
}
