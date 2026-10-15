import { useWiki } from '../wiki/WikiContext';
import React, { useState } from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import GameFinishedPopup from './GameFinishedPopup';
import ShortcutsDialog from '../help/ShortcutsDialog';
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

function renderPopup(overrides: Partial<ApplicationState> = {}, extra: React.ReactNode = null) {
  const state: ApplicationState = {
    ...applicationReducer(undefined, { type: 'test/initialState' }),
    phase: 'GameFinished', name: 'host', roomid: room.id,
    roomsInProgress: [room], scores: [42, 39],
    players: ['Bot 1', 'Bot 2'].map(name => ({ name, activeTribe: null, passiveTribes: [], pieceStacks: [] })),
    ...overrides,
  };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  render(<Provider store={store}><GameFinishedPopup />{extra}</Provider>);
  return store;
}

jest.mock('../wiki/WikiContext', () => ({ useWiki: jest.fn() }));
beforeEach(() => {
  jest.clearAllMocks();
  (useWiki as jest.Mock).mockReturnValue({ inspectMode: false, isOpen: false });
});

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

test('the results dialog owns rewind and leave shortcuts and keeps focus inside', () => {
  renderPopup();
  const previous = screen.getByRole('button', { name: 'Previous move' });
  const leave = screen.getByRole('button', { name: 'Leave Game' });
  expect(document.activeElement).toBe(previous);
  fireEvent.keyDown(previous, { key: 'ArrowLeft' });
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('previousBotMove', {});
  fireEvent.keyDown(previous, { key: 'Tab', shiftKey: true });
  expect(document.activeElement).toBe(leave);
  fireEvent.keyDown(leave, { key: 'Tab' });
  expect(document.activeElement).toBe(previous);
  fireEvent.keyDown(previous, { key: 'L', shiftKey: true });
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('leaveroom', {});
  expect(sendMessageToBackend).toHaveBeenCalledTimes(2);
});

test('results shortcut does not invent a previous action when the button is absent', () => {
  renderPopup({ name: 'visitor' });
  fireEvent.keyDown(screen.getByRole('dialog'), { key: 'ArrowLeft' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('results ignore repeated, modified and external shortcuts', () => {
  renderPopup();
  const dialog = screen.getByRole('dialog');
  fireEvent.keyDown(dialog, { key: 'ArrowLeft', repeat: true });
  fireEvent.keyDown(dialog, { key: 'ArrowLeft', altKey: true });
  fireEvent.keyDown(dialog, { key: 'L', shiftKey: true, ctrlKey: true });
  fireEvent.keyDown(window, { key: 'ArrowLeft' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('inspection disables results rewind for both mouse and keyboard', () => {
  (useWiki as jest.Mock).mockReturnValue({ inspectMode: true, isOpen: false });
  renderPopup();
  const previous = screen.getByRole('button', { name: 'Previous move' });
  expect((previous as HTMLButtonElement).disabled).toBe(true);
  fireEvent.click(previous);
  fireEvent.keyDown(screen.getByRole('dialog'), { key: 'ArrowLeft' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});


function GuideHarness() {
  const [open, setOpen] = useState(false);
  return <><button type="button" onClick={() => setOpen(true)}>Open controls</button>
    {open && <ShortcutsDialog onClose={() => setOpen(false)} />}</>;
}

test('finishing under the guide preserves modal focus and blocks background results actions', async () => {
  const store = renderPopup({ phase: 'Conquest' }, <GuideHarness />);
  const opener = screen.getByRole('button', { name: 'Open controls' });
  opener.focus();
  fireEvent.click(opener);
  const guide = screen.getByRole('dialog', { name: 'Controls & shortcuts' });
  expect(document.activeElement).toBe(guide);

  act(() => store.dispatch(websocketMessageReceived(JSON.stringify({ type: 'gamefinished', data: [42, 39] }))));
  const results = screen.getByRole('dialog', { name: 'Game Results' });
  const previous = screen.getByRole('button', { name: 'Previous move' });
  const leave = screen.getByRole('button', { name: 'Leave Game' });
  expect(document.activeElement).toBe(guide);
  fireEvent.keyDown(results, { key: 'ArrowLeft' });
  fireEvent.keyDown(results, { key: 'L', shiftKey: true });
  fireEvent.click(previous);
  fireEvent.click(leave);
  expect(sendMessageToBackend).not.toHaveBeenCalled();

  fireEvent.keyDown(guide, { key: 'Escape' });
  await waitFor(() => expect(document.activeElement).toBe(previous));
  expect(screen.queryByRole('dialog', { name: 'Controls & shortcuts' })).toBeNull();
  fireEvent.keyDown(previous, { key: 'ArrowLeft' });
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('previousBotMove', {});
});

test('closing another modal preserves its restored results-button focus', async () => {
  renderPopup({}, <GuideHarness />);
  const leave = screen.getByRole('button', { name: 'Leave Game' });
  leave.focus();
  fireEvent.click(screen.getByRole('button', { name: 'Open controls' }));
  await act(async () => {});
  expect(document.activeElement).toBe(screen.getByRole('dialog', { name: 'Controls & shortcuts' }));
  fireEvent.click(screen.getByRole('button', { name: 'Close shortcuts' }));
  await act(async () => {});
  expect(document.activeElement).toBe(leave);
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});
