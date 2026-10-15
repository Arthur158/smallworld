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

test('only a waiting-room host on a bot-capable server sees controls', () => {
  const view = render(<RoomBotControls room={room} isHost={false} />);
  expect(screen.queryByRole('region', { name: 'Room bots' })).toBeNull();
  view.rerender(<RoomBotControls room={{ ...room, inProgress: true }} isHost />);
  expect(screen.queryByRole('region', { name: 'Room bots' })).toBeNull();
  view.rerender(<RoomBotControls room={{ ...room, botModels: undefined }} isHost />);
  expect(screen.queryByRole('region', { name: 'Room bots' })).toBeNull();
  view.rerender(<RoomBotControls room={{ ...room, botModels: [] }} isHost />);
  expect(screen.queryByRole('region', { name: 'Room bots' })).toBeNull();
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


test('the host can free their player seat to watch an all-bot game', () => {
  const view = render(<RoomBotControls room={room} isHost />);
  expect(screen.getByRole('button', { name: 'Play' }).getAttribute('aria-pressed')).toBe('true');
  fireEvent.click(screen.getByRole('button', { name: 'Play' }));
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Watch as host' }));
  expect(sendMessageToBackend).toHaveBeenCalledWith('setHostSpectator', { spectator: true });

  view.rerender(<RoomBotControls room={{ ...room, players: ['', ''] }} isHost />);
  expect(screen.getByRole('button', { name: 'Watch as host' }).getAttribute('aria-pressed')).toBe('true');
  expect(screen.getByText(/Fill the open seats with bots/)).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Add bot' }));
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('addBot', { modelId: 'random' });
});

test('a watching host can return to an open seat', () => {
  render(<RoomBotControls room={{ ...room, players: ['', 'bot:123'] }} isHost />);
  fireEvent.click(screen.getByRole('button', { name: 'Watch as host' }));
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Play' }));
  expect(sendMessageToBackend).toHaveBeenCalledWith('setHostSpectator', { spectator: false });
});

test('a watching host must free a seat before returning to play', () => {
  const view = render(<RoomBotControls room={{ ...room, players: ['bot:123', 'bot:456'] }} isHost />);
  const playButton = screen.getByRole('button', { name: 'Play' }) as HTMLButtonElement;
  expect(playButton.disabled).toBe(true);
  expect(screen.getByText(/free a seat to play again/)).toBeTruthy();
  fireEvent.click(playButton);
  expect(sendMessageToBackend).not.toHaveBeenCalled();

  view.rerender(<RoomBotControls room={{ ...room, players: ['', 'bot:456'] }} isHost />);
  expect(playButton.disabled).toBe(false);
  fireEvent.click(playButton);
  expect(sendMessageToBackend).toHaveBeenCalledWith('setHostSpectator', { spectator: false });
});

test('a playing host can watch even when all seats are taken', () => {
  render(<RoomBotControls room={{ ...room, players: ['host', 'bot:123'] }} isHost />);
  fireEvent.click(screen.getByRole('button', { name: 'Watch as host' }));
  expect(sendMessageToBackend).toHaveBeenCalledWith('setHostSpectator', { spectator: true });
});
