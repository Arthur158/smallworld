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
  const hasBots = Object.keys(room.bots ?? {}).length > 0;
  return (
    <section className="mb-5 rounded-lg border border-[#8B4513] bg-[#FDF5E6] p-4" aria-label="Room bots">
      <h3 className="font-bold mb-2">Play with bots</h3>
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
          className="rounded bg-[#8B4513] text-white px-4 py-2 disabled:opacity-50 disabled:cursor-not-allowed">
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
