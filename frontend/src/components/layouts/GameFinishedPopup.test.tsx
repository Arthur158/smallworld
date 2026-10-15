import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import GameFinishedPopup from './GameFinishedPopup';
import { sendMessageToBackend } from '../../services/backendService';
import applicationReducer, { websocketMessageReceived } from '../../redux/slices/applicationSlice';
import { Room } from '../../types/Board';
import { ApplicationState } from '../../types/redux';

jest.mock('../../services/backendService', () => ({ sendMessageToBackend: jest.fn() }));

const room: Room = {
  id: 'room-1', name: 'Bot game', creator: 'host', players: ['bot:1', 'bot:2'], capacity: 2,
  mapName: 'map2players', inProgress: true,
  bots: {
    'bot:1': { name: 'Bot 1', modelId: 'random' },
    'bot:2': { name: 'Bot 2', modelId: 'trained-v1' },
  },
  botPlayback: { paused: false, speed: 1, canGoBack: true },
};

function renderPopup(overrides: Partial<ApplicationState> = {}) {
  const state: ApplicationState = {
    ...applicationReducer(undefined, { type: 'test/initialState' }),
    phase: 'GameFinished', name: 'host', roomid: room.id,
    roomsInProgress: [room], scores: [42, 39],
    players: ['Bot 1', 'Bot 2'].map(name => ({ name, activeTribe: null, passiveTribes: [], pieceStacks: [] })),
    ...overrides,
  };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  render(<Provider store={store}><GameFinishedPopup /></Provider>);
  return store;
}

beforeEach(() => jest.clearAllMocks());

test('the host can rewind a finished bot game without pausing first', () => {
  const store = renderPopup();
  fireEvent.click(screen.getByRole('button', { name: 'Previous move' }));
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('previousBotMove', {});
  expect(screen.getByRole('heading', { name: 'Game Results' })).toBeTruthy();

  act(() => store.dispatch(websocketMessageReceived(JSON.stringify({
    type: 'megaUpdate',
    data: {
      turnInfo: { playerNumber: 1, turnNumber: 10, phase: 'Redeployment' },
      players: [], tribeEntries: [], allTiles: [],
    },
  }))));
  expect(screen.queryByRole('heading', { name: 'Game Results' })).toBeNull();
});

test('a completed room outside the progress list still allows rewind', () => {
  renderPopup({ rooms: [{ ...room, inProgress: false }], roomsInProgress: [] });
  expect(screen.getByRole('button', { name: 'Previous move' })).toBeTruthy();
});

test.each([
  ['another spectator', { name: 'visitor' }],
  ['no current room', { roomid: 'other-room' }],
  ['ordinary game', { roomsInProgress: [{ ...room, bots: undefined, botPlayback: undefined }] }],
  ['mixed game', { roomsInProgress: [{ ...room, players: ['host', 'bot:2'] }] }],
  ['no earlier move', { roomsInProgress: [{ ...room, botPlayback: { paused: false, speed: 1, canGoBack: false } }] }],
  ['older playback metadata', { roomsInProgress: [{ ...room, botPlayback: { paused: true, speed: 1 } }] }],
] as [string, Partial<ApplicationState>][])('results omit rewind for %s', (_, overrides) => {
  renderPopup(overrides);
  expect(screen.queryByRole('button', { name: 'Previous move' })).toBeNull();
  expect(screen.getByRole('button', { name: 'Leave Game' })).toBeTruthy();
});

test('the results popup remains hidden before the game finishes', () => {
  renderPopup({ phase: 'Conquest' });
  expect(screen.queryByRole('heading', { name: 'Game Results' })).toBeNull();
});
