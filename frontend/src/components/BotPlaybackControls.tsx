import React from 'react';
import { Room } from '../types/Board';
import { sendMessageToBackend } from '../services/backendService';
import { useShortcut } from '../hooks/useShortcut';
import { useWiki } from './wiki/WikiContext';

const playbackSpeeds = [0.25, 0.5, 1, 2, 3];

export default function BotPlaybackControls({ room, username, phase }: {
  room?: Room;
  username: string;
  phase: string;
}) {
  const { inspectMode, isOpen } = useWiki();
  // The server advertises playback only for supported, all-bot rooms.
  const visible = !!room?.botPlayback && room.inProgress && phase !== 'GameFinished' &&
    room.players.length > 0 && room.players.every(player => room.bots?.[player]);
  const { paused = false, speed = 1, canGoBack = false } = room?.botPlayback ?? {};
  const isHost = room?.creator === username;
  const canControl = visible && isHost && !inspectMode && !isOpen;
  const canPrevious = canControl && paused && canGoBack;
  const canNext = canControl && paused;
  const speedIndex = playbackSpeeds.indexOf(speed);
  const setSpeed = (value: number) => {
    if (canControl && playbackSpeeds.includes(value)) sendMessageToBackend('setBotPlayback', { speed: value });
  };
  const previous = () => {
    if (canPrevious) sendMessageToBackend('previousBotMove', {});
  };
  const togglePause = () => {
    if (canControl) sendMessageToBackend('setBotPlayback', { paused: !paused });
  };
  const next = () => {
    if (canNext) sendMessageToBackend('stepBot', {});
  };
  const slower = () => setSpeed(playbackSpeeds[speedIndex - 1]);
  const faster = () => setSpeed(playbackSpeeds[speedIndex + 1]);
  useShortcut('Space', togglePause, canControl);
  useShortcut('ArrowLeft', previous, canPrevious);
  useShortcut('ArrowRight', next, canNext);
  useShortcut('-', slower, canControl && speedIndex > 0);
  useShortcut('=', faster, canControl && speedIndex >= 0 && speedIndex < playbackSpeeds.length - 1);
  useShortcut('+', faster, canControl && speedIndex >= 0 && speedIndex < playbackSpeeds.length - 1);

  if (!visible) return null;
  const previousTitle = !paused ? 'Pause the game to go back one move' : canGoBack ? 'Go back one bot move and stay paused' : 'No earlier moves available';
  const nextTitle = paused ? 'Advance one bot move, then stay paused' : 'Pause the game to advance one move';

  return (
    <section aria-label="Bot playback" className="mb-3 shrink-0 rounded-lg border border-[#8B4513] bg-[#FDF5E6] px-4 py-3 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <h2 className="font-bold">Bot playback</h2>
          <span className="text-sm" role="status">{paused ? 'Paused' : 'Playing'} · {speed}×</span>
        </div>
        {isHost ? (
          <div className="flex flex-wrap items-center gap-2">
            <button type="button" disabled={!canPrevious}
              data-tooltip={`${previousTitle}. Shortcut: Left arrow.`} aria-keyshortcuts="ArrowLeft" onClick={previous}
              className="rounded border border-[#8B4513] hover:bg-[#EED5B7] px-3 py-1.5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              Previous move
            </button>
            <button type="button" onClick={togglePause} disabled={!canControl}
              data-tooltip={`${paused ? 'Resume automatic bot moves' : 'Pause automatic bot moves'}. Shortcut: Space.`} aria-keyshortcuts="Space"
              className="rounded bg-[#8B4513] hover:bg-[#A0522D] text-white px-3 py-1.5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              {paused ? 'Resume' : 'Pause'}
            </button>
            <button type="button" disabled={!canNext}
              data-tooltip={`${nextTitle}. Shortcut: Right arrow.`} aria-keyshortcuts="ArrowRight" onClick={next}
              className="rounded border border-[#8B4513] hover:bg-[#EED5B7] px-3 py-1.5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              Next move
            </button>
            <label htmlFor="bot-playback-speed" className="text-sm font-semibold ml-1">Speed</label>
            <select id="bot-playback-speed" value={speed} disabled={!canControl}
              data-tooltip="Change the delay between bot moves. Shortcuts: − slower, = or + faster." aria-keyshortcuts="- = Shift+="
              onChange={event => setSpeed(Number(event.target.value))}
              className="rounded border border-[#8B4513] bg-white py-1.5 pl-2 pr-7 disabled:opacity-50 disabled:cursor-not-allowed">
              {playbackSpeeds.map(value => <option key={value} value={value}>{value}×</option>)}
            </select>
          </div>
        ) : <p className="text-sm">The host controls playback.</p>}
      </div>
    </section>
  );
}
