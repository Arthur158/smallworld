import React, { useEffect, useState } from 'react';
import { Room } from '../types/Board';
import { sendMessageToBackend } from '../services/backendService';

export default function RoomBotControls({ room, isHost }: { room: Room; isHost: boolean }) {
  const models = room.botModels ?? [];
  const [modelId, setModelId] = useState('random');
  useEffect(() => {
    if (!models.some(model => model.id === modelId)) setModelId(models[0]?.id ?? '');
  }, [models, modelId]);
  // Older servers do not advertise a catalog and cannot accept bot commands.
  if (!isHost || room.inProgress || models.length === 0) return null;
  const full = !room.players.some(player => !player);
  const watching = !room.players.includes(room.creator);
  const hasBots = Object.keys(room.bots ?? {}).length > 0;
  return (
    <section className="mb-5 rounded-lg border border-[#8B4513] bg-[#FDF5E6] p-4" aria-label="Room bots">
      <div className="flex flex-wrap items-center justify-between gap-3 mb-2">
        <h3 className="font-bold">Bot players</h3>
        <div className="flex flex-wrap items-center gap-2" role="group" aria-label="Your role">
          <span className="text-sm font-semibold">Your role</span>
          <div className="inline-flex rounded border border-[#8B4513] overflow-hidden">
            <button type="button" aria-pressed={!watching} disabled={watching && full}
              aria-describedby="host-role-description"
              onClick={() => watching && sendMessageToBackend('setHostSpectator', { spectator: false })}
              className={`px-3 py-1 text-sm transition-colors disabled:opacity-50 disabled:cursor-not-allowed ${!watching ? 'bg-[#8B4513] text-white' : 'hover:bg-[#EED5B7]'}`}>
              Play
            </button>
            <button type="button" aria-pressed={watching} aria-describedby="host-role-description"
              onClick={() => !watching && sendMessageToBackend('setHostSpectator', { spectator: true })}
              className={`border-l border-[#8B4513] px-3 py-1 text-sm transition-colors ${watching ? 'bg-[#8B4513] text-white' : 'hover:bg-[#EED5B7]'}`}>
              Watch as host
            </button>
          </div>
        </div>
      </div>
      <p id="host-role-description" className="text-sm mb-3">
        {watching
          ? full
            ? 'You are watching as host. Start the game to watch, or free a seat to play again.'
            : 'You are watching as host. Fill the open seats with bots, then start the game to watch.'
          : 'Play alongside bots, or watch as host to make room for an all-bot game.'}
      </p>
      <p className="text-sm mb-3">
        Adding the first bot applies the bot rules preset: 12 races, 12 traits, and no powers.
        Bots use the same game rules as the human players in this room.
      </p>
      <div className="flex flex-wrap items-center gap-2">
        <label htmlFor="new-bot-model" className="font-semibold">Bot model</label>
        <select id="new-bot-model" value={modelId} onChange={event => setModelId(event.target.value)}
          className="border rounded p-2 bg-white">
          {models.map(model => <option key={model.id} value={model.id}>{model.name}</option>)}
        </select>
        <button type="button" disabled={full || !modelId}
          onClick={() => sendMessageToBackend('addBot', { modelId })}
          className="rounded bg-[#8B4513] hover:bg-[#A0522D] text-white px-4 py-2 transition-colors disabled:opacity-50 disabled:cursor-not-allowed">
          {full ? 'Room is full' : 'Add bot'}
        </button>
        {hasBots && <button type="button" onClick={() => sendMessageToBackend('useBotRules', {})}
          className="rounded border border-[#8B4513] px-3 py-2">Use bot rules</button>}
      </div>
      <p className="text-sm mt-2" role="status">{models.find(model => model.id === modelId)?.description}</p>
      {hasBots && <p className="text-sm mt-2">Change models or remove bots before starting. Loaded saves must also use compatible races and traits.</p>}
    </section>
  );
}

export function BotModelSelect({ room, username }: { room: Room; username: string }) {
  const seat = room.bots?.[username];
  if (!seat) return null;
  return <select aria-label={`Model for ${seat.name}`} value={seat.modelId}
    disabled={room.inProgress}
    className="border rounded p-1 max-w-[12rem] bg-white"
    onChange={event => sendMessageToBackend('setBotModel', { username, modelId: event.target.value })}>
    {(room.botModels ?? []).map(model => <option key={model.id} value={model.id}>{model.name}</option>)}
  </select>;
}
