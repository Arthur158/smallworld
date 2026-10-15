import React from 'react';
import { Room } from '../types/Board';
import { sendMessageToBackend } from '../services/backendService';

const playbackSpeeds = [0.25, 0.5, 1, 2, 3];

export default function BotPlaybackControls({ room, username, phase }: {
  room?: Room;
  username: string;
  phase: string;
}) {
  // The server advertises playback only for supported, all-bot rooms.
  if (!room?.botPlayback || !room.inProgress || phase === 'GameFinished' ||
    room.players.length === 0 || !room.players.every(player => room.bots?.[player])) return null;

  const { paused, speed, canGoBack = false } = room.botPlayback;
  const isHost = room.creator === username;

  return (
    <section aria-label="Bot playback" className="mb-3 shrink-0 rounded-lg border border-[#8B4513] bg-[#FDF5E6] px-4 py-3 shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-3">
          <h2 className="font-bold">Bot playback</h2>
          <span className="text-sm" role="status">{paused ? 'Paused' : 'Playing'} · {speed}×</span>
        </div>
        {isHost ? (
          <div className="flex flex-wrap items-center gap-2">
            <button type="button" disabled={!paused || !canGoBack}
              title={!paused ? 'Pause the game to go back one move' : canGoBack ? 'Go back one bot move and stay paused' : 'No earlier moves available'}
              onClick={() => sendMessageToBackend('previousBotMove', {})}
              className="rounded border border-[#8B4513] hover:bg-[#EED5B7] px-3 py-1.5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              Previous move
            </button>
            <button type="button" onClick={() => sendMessageToBackend('setBotPlayback', { paused: !paused })}
              className="rounded bg-[#8B4513] hover:bg-[#A0522D] text-white px-3 py-1.5 transition-colors">
              {paused ? 'Resume' : 'Pause'}
            </button>
            <button type="button" disabled={!paused}
              title={paused ? 'Advance one bot move, then stay paused' : 'Pause the game to advance one move'}
              onClick={() => sendMessageToBackend('stepBot', {})}
              className="rounded border border-[#8B4513] hover:bg-[#EED5B7] px-3 py-1.5 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
              Next move
            </button>
            <label htmlFor="bot-playback-speed" className="text-sm font-semibold ml-1">Speed</label>
            <select id="bot-playback-speed" value={speed}
              onChange={event => sendMessageToBackend('setBotPlayback', { speed: Number(event.target.value) })}
              className="rounded border border-[#8B4513] bg-white py-1.5 pl-2 pr-7">
              {playbackSpeeds.map(value => <option key={value} value={value}>{value}×</option>)}
            </select>
          </div>
        ) : <p className="text-sm">The host controls playback.</p>}
      </div>
    </section>
  );
}
