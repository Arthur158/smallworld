import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import RoomBotControls, { BotModelSelect } from './RoomBotControls';
import { sendMessageToBackend } from '../services/backendService';
import { Room } from '../types/Board';

jest.mock('../services/backendService', () => ({ sendMessageToBackend: jest.fn() }));
const room: Room = {
  id: 'room-1', name: 'Room', creator: 'host', players: ['host', ''], capacity: 2, mapName: 'map2players',
  botModels: [
    { id: 'random', name: 'Random bot', description: 'Random choices' },
    { id: 'trained-v1', name: 'Trained v1', description: 'A trained policy' },
  ],
};

beforeEach(() => jest.clearAllMocks());

test('only a waiting-room host sees add controls', () => {
  const view = render(<RoomBotControls room={room} isHost={false} />);
  expect(screen.queryByRole('button', { name: 'Add bot' })).toBeNull();
  view.rerender(<RoomBotControls room={{ ...room, inProgress: true }} isHost />);
  expect(screen.queryByRole('button', { name: 'Add bot' })).toBeNull();
  view.rerender(<RoomBotControls room={room} isHost />);
  fireEvent.change(screen.getByLabelText('Bot model'), { target: { value: 'trained-v1' } });
  fireEvent.click(screen.getByRole('button', { name: 'Add bot' }));
  expect(sendMessageToBackend).toHaveBeenCalledWith('addBot', { modelId: 'trained-v1' });
});

test('full rooms cannot add another bot', () => {
  render(<RoomBotControls room={{ ...room, players: ['host', 'other'] }} isHost />);
  fireEvent.click(screen.getByRole('button', { name: 'Room is full' }));
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('bot model changes refer to the stable seat identity', () => {
  render(<BotModelSelect room={{ ...room, bots: { 'bot:123': { name: 'Bot 2', modelId: 'random' } } }} username="bot:123" />);
  fireEvent.change(screen.getByLabelText('Model for Bot 2'), { target: { value: 'trained-v1' } });
  expect(sendMessageToBackend).toHaveBeenCalledWith('setBotModel', { username: 'bot:123', modelId: 'trained-v1' });
});
