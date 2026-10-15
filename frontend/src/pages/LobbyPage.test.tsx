import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { MemoryRouter } from 'react-router-dom';
import { configureStore } from '@reduxjs/toolkit';
import LobbyPage from './LobbyPage';
import applicationReducer, { websocketMessageReceived } from '../redux/slices/applicationSlice';
import { ApplicationState } from '../types/redux';
import { Room } from '../types/Board';
import { sendMessageToBackend } from '../services/backendService';

jest.mock('../services/backendService', () => ({ sendMessageToBackend: jest.fn(), getAuthConfig: jest.fn() }));
jest.mock('../components/wiki/WikiToolbar', () => () => null);
jest.mock('../components/RoomBotControls', () => ({ __esModule: true, default: () => null, BotModelSelect: () => null }));

const room: Room = { id: 'room-1', name: 'Room', creator: 'Host', players: ['Host', 'Guest'], capacity: 6, mapName: 'generatedmap' };
const standardChoices: ApplicationState['extensionChoices'] = [
  { extensionName: 'Base', isChecked: true, raceChoices: [{ choice: 'Humans', isChecked: true }], traitChoices: [{ choice: 'Mounted', isChecked: true }] },
  { extensionName: 'Sky Islands', isChecked: true, raceChoices: [{ choice: 'Storm Giants', isChecked: true }], traitChoices: [{ choice: 'Zeppelin', isChecked: true }] },
  { extensionName: 'Underground', isChecked: false, raceChoices: [{ choice: 'Drow', isChecked: false }], traitChoices: [{ choice: 'Muddy', isChecked: false }] },
  { extensionName: 'Powers', isChecked: false, raceChoices: [], traitChoices: [{ choice: 'Scepter of Avarice', isChecked: false }] },
];
const undergroundChoices = standardChoices.map(extension => ({
  ...extension, isChecked: !extension.isChecked,
  raceChoices: extension.raceChoices.map(choice => ({ ...choice, isChecked: !choice.isChecked })),
  traitChoices: extension.traitChoices.map(choice => ({ ...choice, isChecked: !choice.isChecked })),
}));

function renderLobby(overrides: Partial<ApplicationState> = {}, extra?: React.ReactNode) {
  const state: ApplicationState = {
    ...applicationReducer(undefined, { type: 'test/init' }),
    name: 'Host', isAuthenticated: true, roomid: room.id, rooms: [room],
    extensionChoices: standardChoices, globalToggle: false,
    mapChoices: ['generatedmap', 'underground2players', 'map2players'],
    ...overrides,
  };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  render(<Provider store={store}><MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><LobbyPage />{extra}</MemoryRouter></Provider>);
  return store;
}

function broadcast(store: ReturnType<typeof renderLobby>, type: string, data: unknown) {
  act(() => store.dispatch(websocketMessageReceived(JSON.stringify({ type, data }))));
}

function openChoices() {
  fireEvent.click(screen.getByRole('button', { name: 'Races & Traits' }));
}

function checkbox(name: string) {
  return screen.getByRole('checkbox', { name }) as HTMLInputElement;
}

beforeEach(() => jest.clearAllMocks());

test('presets request one atomic change and reflect only the server map and selections', () => {
  const store = renderLobby();
  openChoices();
  const standard = screen.getByRole('button', { name: 'Standard' });
  const underground = screen.getByRole('button', { name: 'Underground' });
  const map = screen.getByRole('combobox', { name: 'Choose Map:' }) as HTMLSelectElement;
  expect(standard.getAttribute('aria-pressed')).toBe('true');
  expect(underground.getAttribute('aria-pressed')).toBe('false');
  fireEvent.click(underground);
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('setRulesPreset', { preset: 'underground' });
  expect(map.value).toBe('generatedmap');
  expect(checkbox('Humans').checked).toBe(true);
  expect(checkbox('Drow').checked).toBe(false);
  expect(standard.getAttribute('aria-pressed')).toBe('true');

  broadcast(store, 'choices', { extensionChoices: undergroundChoices, globalToggle: false });
  expect(screen.getByText('Custom setup')).toBeTruthy();
  broadcast(store, 'roomEntriesUpdate', [{ ...room, mapName: 'underground2players', capacity: 2 }]);
  expect(standard.getAttribute('aria-pressed')).toBe('false');
  expect(underground.getAttribute('aria-pressed')).toBe('true');
  expect(map.value).toBe('underground2players');
  for (const name of ['Base', 'Sky Islands', 'Humans', 'Mounted', 'Storm Giants', 'Zeppelin']) expect(checkbox(name).checked).toBe(false);
  for (const name of ['Underground', 'Powers', 'Drow', 'Muddy', 'Scepter of Avarice']) expect(checkbox(name).checked).toBe(true);

  jest.clearAllMocks();
  fireEvent.click(standard);
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('setRulesPreset', { preset: 'standard' });
  expect(map.value).toBe('underground2players');
  broadcast(store, 'choices', { extensionChoices: standardChoices, globalToggle: false });
  broadcast(store, 'roomEntriesUpdate', [room]);
  expect(standard.getAttribute('aria-pressed')).toBe('true');
  expect(underground.getAttribute('aria-pressed')).toBe('false');
  expect(map.value).toBe('generatedmap');
  expect(checkbox('Base').checked).toBe(true);
  expect(checkbox('Sky Islands').checked).toBe(true);
  expect(checkbox('Underground').checked).toBe(false);
  expect(checkbox('Powers').checked).toBe(false);
});

test.each([
  ['a different map', { rooms: [{ ...room, mapName: 'map2players' }] }],
  ['a customized race', { extensionChoices: standardChoices.map(extension => extension.extensionName === 'Base' ? { ...extension, raceChoices: [{ choice: 'Humans', isChecked: false }] } : extension) }],
  ['a customized trait', { extensionChoices: standardChoices.map(extension => extension.extensionName === 'Underground' ? { ...extension, traitChoices: [{ choice: 'Muddy', isChecked: true }] } : extension) }],
  ['no choice catalog', { extensionChoices: [] }],
] as [string, Partial<ApplicationState>][])('presets show Custom setup for %s', (_, overrides) => {
  renderLobby(overrides);
  openChoices();
  expect(screen.getByText('Custom setup')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Standard' }).getAttribute('aria-pressed')).toBe('false');
  expect(screen.getByRole('button', { name: 'Underground' }).getAttribute('aria-pressed')).toBe('false');
});

test('the host can customize individual selections after applying a preset', () => {
  renderLobby();
  openChoices();
  fireEvent.click(checkbox('Mounted'));
  expect(sendMessageToBackend).toHaveBeenCalledWith('toggleTrait', { extensionName: 'Base', traitChoice: 'Mounted', checked: false });
  expect(checkbox('Mounted').checked).toBe(true);
  expect(checkbox('Mounted').closest('label')?.getAttribute('data-tooltip')).toContain('Space');
});

test('a selected save is clearly described as replaced when applying a preset', () => {
  renderLobby({ saveSelectionId: 9 });
  openChoices();
  expect(screen.getByText('Choosing a preset clears the selected save and sets up a new game.')).toBeTruthy();
});

test('U and N shortcuts work only while host choices are visible', () => {
  renderLobby();
  fireEvent.keyDown(window, { key: 'u' });
  fireEvent.keyDown(window, { key: 'n' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.keyDown(window, { key: 's' });
  fireEvent.keyDown(window, { key: 'u' });
  fireEvent.keyDown(window, { key: 'n' });
  expect(sendMessageToBackend).toHaveBeenNthCalledWith(1, 'setRulesPreset', { preset: 'underground' });
  expect(sendMessageToBackend).toHaveBeenNthCalledWith(2, 'setRulesPreset', { preset: 'standard' });
  jest.clearAllMocks();
  const map = screen.getByRole('combobox', { name: 'Choose Map:' });
  fireEvent.keyDown(map, { key: 'n' });
  fireEvent.keyDown(window, { key: 'u', ctrlKey: true });
  fireEvent.keyDown(window, { key: 'n', repeat: true });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.keyDown(window, { key: 's' });
  fireEvent.keyDown(window, { key: 'u' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('nonhosts cannot access or invoke presets or start games', () => {
  renderLobby({ name: 'Guest' });
  openChoices();
  expect(screen.queryByRole('button', { name: 'Standard' })).toBeNull();
  fireEvent.keyDown(window, { key: 'u' });
  fireEvent.keyDown(window, { key: 'n' });
  fireEvent.keyDown(window, { key: 'p' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('started rooms disable preset buttons and shortcuts', () => {
  renderLobby({ rooms: [{ ...room, inProgress: true }] });
  openChoices();
  const standard = screen.getByRole('button', { name: 'Standard' }) as HTMLButtonElement;
  expect(standard.disabled).toBe(true);
  fireEvent.click(standard);
  fireEvent.keyDown(window, { key: 'n' });
  fireEvent.keyDown(window, { key: 'p' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('lobby shortcuts respect fields and expose their corresponding actions', () => {
  const store = renderLobby({ error: 'Cannot start yet.' });
  const map = screen.getByRole('combobox', { name: 'Choose Map:' });
  fireEvent.keyDown(map, { key: 's' });
  expect(screen.getByRole('button', { name: 'Enter Display Room' })).toBeTruthy();
  fireEvent.keyDown(map, { key: 'p' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.keyDown(window, { key: 'p' });
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('startGame', { roomId: room.id });
  fireEvent.keyDown(window, { key: 'v' });
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('enterdisplayroom', {});
  fireEvent.keyDown(window, { key: 'L', shiftKey: true });
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('leaveroom', {});
  fireEvent.keyDown(window, { key: 'c' });
  expect(store.getState().application.error).toBeNull();
  expect(screen.getByRole('button', { name: 'Start Game' }).getAttribute('aria-keyshortcuts')).toBe('P');
  expect(screen.getByRole('button', { name: 'Leave Room' }).getAttribute('aria-keyshortcuts')).toBe('Shift+L');
  expect(screen.getByRole('button', { name: 'Move Guest up' }).getAttribute('data-tooltip')).toContain('turn order');
  expect(screen.getByRole('button', { name: 'Remove Guest from room' })).toBeTruthy();
});

test('modal dialogs block preset and lobby shortcuts', () => {
  renderLobby({}, <div role="dialog" aria-modal="true">Wiki</div>);
  openChoices();
  for (const key of ['u', 'n', 'p', 's']) fireEvent.keyDown(window, { key });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(screen.getByRole('button', { name: 'Standard' })).toBeTruthy();
});

test('deleting a saved game does not also load it or add a global delete shortcut', () => {
  renderLobby({ saveGames: [{ saveId: 9, summary: 'Saved game' }] });
  const load = screen.getByRole('button', { name: 'Load saved game 9' });
  expect(load.tabIndex).toBe(0);
  expect(load.getAttribute('data-tooltip')).toContain('Enter / Space');
  fireEvent.click(screen.getByRole('button', { name: 'Delete saved game 9' }));
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('deletesave', { saveId: 9 });
  jest.clearAllMocks();
  fireEvent.keyDown(window, { key: 'Delete' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});


test('rooms with bots explain how to keep bot-compatible rules', () => {
  renderLobby({ rooms: [{ ...room, players: ['Host', 'bot:1'], bots: { 'bot:1': { name: 'Bot 1', modelId: 'random' } } }] });
  openChoices();
  expect(screen.getByText('These presets include rules bots may not support. Use bot rules before starting a bot game.')).toBeTruthy();
});
