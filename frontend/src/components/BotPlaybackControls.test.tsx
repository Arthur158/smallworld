import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import BotPlaybackControls from './BotPlaybackControls';
import { sendMessageToBackend } from '../services/backendService';
import { Room } from '../types/Board';

jest.mock('../services/backendService', () => ({ sendMessageToBackend: jest.fn() }));
const room: Room = {
  id: 'room-1', name: 'Bot game', creator: 'host', players: ['bot:1', 'bot:2'], capacity: 2,
  mapName: 'map2players', inProgress: true,
  bots: {
    'bot:1': { name: 'Bot 1', modelId: 'random' },
    'bot:2': { name: 'Bot 2', modelId: 'trained-v1' },
  },
  botPlayback: { paused: false, speed: 1 },
};

beforeEach(() => jest.clearAllMocks());

test('pause and resume follow server state without optimistic playback changes', () => {
  const view = render(<BotPlaybackControls room={room} username="host" phase="Conquest" />);
  expect(screen.getByRole('status').textContent).toBe('Playing · 1×');
  fireEvent.click(screen.getByRole('button', { name: 'Pause' }));
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('setBotPlayback', { paused: true });
  expect(screen.queryByRole('button', { name: 'Resume' })).toBeNull();
  expect((screen.getByRole('button', { name: 'Next move' }) as HTMLButtonElement).disabled).toBe(true);

  view.rerender(<BotPlaybackControls room={{ ...room, botPlayback: { paused: true, speed: 1 } }} username="host" phase="Conquest" />);
  expect(screen.getByRole('status').textContent).toBe('Paused · 1×');
  fireEvent.click(screen.getByRole('button', { name: 'Resume' }));
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('setBotPlayback', { paused: false });
});

test('next move is available only when the server reports paused', () => {
  const view = render(<BotPlaybackControls room={room} username="host" phase="Conquest" />);
  fireEvent.click(screen.getByRole('button', { name: 'Next move' }));
  expect(sendMessageToBackend).not.toHaveBeenCalled();

  view.rerender(<BotPlaybackControls room={{ ...room, botPlayback: { paused: true, speed: 2 } }} username="host" phase="Conquest" />);
  fireEvent.click(screen.getByRole('button', { name: 'Next move' }));
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('stepBot', {});
  expect(screen.getByRole('status').textContent).toBe('Paused · 2×');
});

test('the speed selector sends only supported numeric speeds and reflects server updates', () => {
  const view = render(<BotPlaybackControls room={room} username="host" phase="Conquest" />);
  const select = screen.getByRole('combobox', { name: 'Speed' }) as HTMLSelectElement;
  expect(Array.from(select.options).map(option => option.text)).toEqual(['0.25×', '0.5×', '1×', '2×', '3×']);
  for (const speed of [0.25, 0.5, 1, 2, 3]) {
    fireEvent.change(select, { target: { value: speed.toString() } });
    expect(sendMessageToBackend).toHaveBeenLastCalledWith('setBotPlayback', { speed });
  }
  expect(select.value).toBe('1');
  view.rerender(<BotPlaybackControls room={{ ...room, botPlayback: { paused: false, speed: 0.5 } }} username="host" phase="Conquest" />);
  expect(select.value).toBe('0.5');
  expect(screen.getByRole('status').textContent).toBe('Playing · 0.5×');
});

test('other spectators see the playback state without host controls', () => {
  render(<BotPlaybackControls room={{ ...room, botPlayback: { paused: true, speed: 3 } }} username="visitor" phase="Conquest" />);
  expect(screen.getByRole('status').textContent).toBe('Paused · 3×');
  expect(screen.getByText('The host controls playback.')).toBeTruthy();
  expect(screen.queryByRole('button')).toBeNull();
  expect(screen.queryByRole('combobox')).toBeNull();
});

test.each([
  ['no room', undefined],
  ['older server', { ...room, botPlayback: undefined }],
  ['waiting room', { ...room, inProgress: false }],
  ['mixed game', { ...room, players: ['host', 'bot:2'] }],
  ['unfilled game', { ...room, players: ['', 'bot:2'] }],
  ['empty game', { ...room, players: [] }],
] as [string, Room | undefined][])('playback is hidden for %s', (_, currentRoom) => {
  render(<BotPlaybackControls room={currentRoom} username="host" phase="Conquest" />);
  expect(screen.queryByRole('region', { name: 'Bot playback' })).toBeNull();
});

test('finished games have no playback controls', () => {
  render(<BotPlaybackControls room={room} username="host" phase="GameFinished" />);
  expect(screen.queryByRole('region', { name: 'Bot playback' })).toBeNull();
});


test('previous move requires pause and available server history', () => {
  const view = render(<BotPlaybackControls room={{ ...room, botPlayback: { paused: false, speed: 1, canGoBack: true } }} username="host" phase="Conquest" />);
  const previous = screen.getByRole('button', { name: 'Previous move' }) as HTMLButtonElement;
  expect(previous.disabled).toBe(true);
  expect(previous.title).toBe('Pause the game to go back one move');
  fireEvent.click(previous);
  expect(sendMessageToBackend).not.toHaveBeenCalled();

  view.rerender(<BotPlaybackControls room={{ ...room, botPlayback: { paused: true, speed: 1, canGoBack: false } }} username="host" phase="Conquest" />);
  expect(previous.disabled).toBe(true);
  expect(previous.title).toBe('No earlier moves available');
  fireEvent.click(previous);
  expect(sendMessageToBackend).not.toHaveBeenCalled();

  view.rerender(<BotPlaybackControls room={{ ...room, botPlayback: { paused: true, speed: 1, canGoBack: true } }} username="host" phase="Conquest" />);
  expect(previous.disabled).toBe(false);
  fireEvent.click(previous);
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('previousBotMove', {});
  expect(screen.getByRole('status').textContent).toBe('Paused · 1×');
  expect(screen.getAllByRole('button').map(button => button.textContent?.trim())).toEqual(['Previous move', 'Resume', 'Next move']);
});

test('servers without rewind metadata keep previous move disabled', () => {
  render(<BotPlaybackControls room={{ ...room, botPlayback: { paused: true, speed: 1 } }} username="host" phase="Conquest" />);
  const previous = screen.getByRole('button', { name: 'Previous move' }) as HTMLButtonElement;
  expect(previous.disabled).toBe(true);
  fireEvent.click(previous);
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});
