import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import GamePage from '../routes/GamePage';
import SpectatorPage from './SpectatorPage';
import DisplayPage from './DisplayPage';
import applicationReducer, { setError, websocketMessageReceived } from '../redux/slices/applicationSlice';
import { ApplicationState } from '../types/redux';
import { sendMessageToBackend } from '../services/backendService';

jest.mock('../services/backendService', () => ({ sendMessageToBackend: jest.fn() }));
jest.mock('../components/layouts/TribeList', () => () => 'Tribe choices');
jest.mock('../components/layouts/OpponentsList', () => () => 'Player summary');
jest.mock('../components/layouts/OpponentsListAll', () => () => 'Player summary');
jest.mock('../components/layouts/PlayerInfo', () => () => null);
jest.mock('../components/layouts/TurnInfoBlock', () => () => null);
jest.mock('../components/layouts/TurnInfoBlockSpectate', () => () => null);
jest.mock('../components/layouts/TurnInfoBlockDisplay', () => () => null);
jest.mock('../components/layouts/GameFinishedPopup', () => () => null);
jest.mock('../components/misc/Map', () => () => null);
jest.mock('../components/misc/CustomCursor', () => () => null);
jest.mock('../components/wiki/WikiToolbar', () => () => null);
jest.mock('../components/inputs/Chat', () => () => null);
jest.mock('../components/BotPlaybackControls', () => () => null);

function renderPage(Page: React.ComponentType, overrides: Partial<ApplicationState> = {}, extra?: React.ReactNode) {
  const state: ApplicationState = {
    ...applicationReducer(undefined, { type: 'test/init' }),
    name: 'Host', playerIndex: 0, playerNumber: 1, phase: 'Conquest',
    players: ['Host', 'Opponent'].map(name => ({ name, activeTribe: null, passiveTribes: [], pieceStacks: [] })),
    ...overrides,
  };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  const view = render(<Provider store={store}><Page />{extra}</Provider>);
  return { store, ...view };
}

function updateTurn(store: ReturnType<typeof renderPage>['store'], phase: string, playerNumber: number, turnNumber = 1) {
  act(() => store.dispatch(websocketMessageReceived(JSON.stringify({
    type: 'turnupdate', data: { phase, playerNumber, turnNumber },
  }))));
}

beforeEach(() => jest.clearAllMocks());

test('the player summary is the default while another player chooses a tribe', () => {
  renderPage(GamePage, { phase: 'TribeChoice', playerNumber: 1 });
  expect(screen.getByText('Player summary')).toBeTruthy();
  expect(screen.queryByText('Tribe choices')).toBeNull();
});

test('joining during your own tribe-choice phase opens the tribe choices', () => {
  renderPage(GamePage, { phase: 'TribeChoice', playerNumber: 0 });
  expect(screen.getByText('Tribe choices')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Show Players' })).toBeTruthy();
});

test('only entering your own tribe-choice phase overrides the manually selected panel', () => {
  const { store } = renderPage(GamePage);
  updateTurn(store, 'TribeChoice', 1);
  expect(screen.getByText('Player summary')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Show Tribes' }));
  updateTurn(store, 'Conquest', 1);
  expect(screen.getByText('Tribe choices')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Show Players' }));
  updateTurn(store, 'TribeChoice', 0);
  expect(screen.getByText('Tribe choices')).toBeTruthy();

  fireEvent.click(screen.getByRole('button', { name: 'Show Players' }));
  updateTurn(store, 'TribeChoice', 0, 2);
  expect(screen.getByText('Player summary')).toBeTruthy();
  updateTurn(store, 'Conquest', 0, 2);
  updateTurn(store, 'TribeChoice', 1, 2);
  expect(screen.getByText('Player summary')).toBeTruthy();
  updateTurn(store, 'TribeChoice', 0, 3);
  expect(screen.getByText('Tribe choices')).toBeTruthy();
});

test.each([
  ['negative index', { playerIndex: -1, playerNumber: -1 }],
  ['out-of-range index', { playerIndex: 2, playerNumber: 2 }],
  ['stale index belonging to someone else', { playerIndex: 1, playerNumber: 1 }],
  ['missing logged-in identity', { name: '', playerIndex: 0, playerNumber: 0 }],
] as [string, Partial<ApplicationState>][])('a %s cannot trigger your tribe-choice panel', (_, overrides) => {
  renderPage(GamePage, { phase: 'TribeChoice', ...overrides });
  expect(screen.getByText('Player summary')).toBeTruthy();
});

test.each([['spectator', SpectatorPage], ['display', DisplayPage]])('%s panels preserve manual choices through every player phase', (_, Page) => {
  const { store } = renderPage(Page, { phase: 'TribeChoice', playerNumber: 0 });
  expect(screen.getByText('Player summary')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Show Tribes' }));
  updateTurn(store, 'Conquest', 0);
  updateTurn(store, 'TribeChoice', 1);
  expect(screen.getByText('Tribe choices')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Show Players' }));
  updateTurn(store, 'TribeChoice', 0);
  expect(screen.getByText('Player summary')).toBeTruthy();
});

test.each([['game', GamePage], ['spectator', SpectatorPage], ['display', DisplayPage]])('%s panel shortcuts are advertised and ignore typing, modifiers, and repeats', (_, Page) => {
  renderPage(Page, {}, <input aria-label="Message" />);
  const toggle = screen.getByRole('button', { name: 'Show Tribes' });
  expect(toggle.getAttribute('aria-keyshortcuts')).toBe('S');
  expect(toggle.getAttribute('data-tooltip')).toContain('· S');
  fireEvent.keyDown(screen.getByRole('textbox', { name: 'Message' }), { key: 's' });
  fireEvent.keyDown(window, { key: 's', ctrlKey: true });
  fireEvent.keyDown(window, { key: 's', metaKey: true });
  fireEvent.keyDown(window, { key: 's', repeat: true });
  expect(screen.getByText('Player summary')).toBeTruthy();
  fireEvent.keyDown(window, { key: 's' });
  expect(screen.getByText('Tribe choices')).toBeTruthy();
  fireEvent.keyDown(window, { key: 's' });
  expect(screen.getByText('Player summary')).toBeTruthy();
});

test.each([['game', GamePage], ['spectator', SpectatorPage], ['display', DisplayPage]])('%s page errors can be dismissed by button or C', (_, Page) => {
  const { store } = renderPage(Page, { error: 'Cannot make that move.' });
  const dismiss = screen.getByRole('button', { name: 'Dismiss error' });
  expect(dismiss.getAttribute('aria-keyshortcuts')).toBe('C');
  expect(dismiss.getAttribute('data-tooltip')).toContain('· C');
  fireEvent.click(dismiss);
  expect(screen.queryByRole('alert')).toBeNull();
  act(() => store.dispatch(setError('Try again.')));
  fireEvent.keyDown(window, { key: 'c' });
  expect(screen.queryByRole('alert')).toBeNull();
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('an open modal blocks the panel and error shortcuts', () => {
  renderPage(GamePage, { error: 'Cannot make that move.' }, <div role="dialog" aria-modal="true">Rules</div>);
  fireEvent.keyDown(window, { key: 's' });
  fireEvent.keyDown(window, { key: 'c' });
  expect(screen.getByText('Player summary')).toBeTruthy();
  expect(screen.getByRole('alert')).toBeTruthy();
});

test('display load and delete are independent native buttons with helpful tooltips', () => {
  renderPage(DisplayPage, { saveGames: [{ saveId: 7, summary: 'Saved match' }], mapChoices: ['map2players'] });
  const load = screen.getByRole('button', { name: 'Load saved game 7' });
  const remove = screen.getByRole('button', { name: 'Delete saved game 7' });
  const map = screen.getByRole('button', { name: 'Load map map2players' });
  for (const button of [load, remove, map]) {
    expect(button.tagName).toBe('BUTTON');
    expect(button.tabIndex).toBe(0);
    expect(button.getAttribute('data-tooltip')).toBeTruthy();
  }
  load.focus();
  expect(document.activeElement).toBe(load);
  fireEvent.click(load);
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('loadgamedisplay', { saveId: 7 });
  fireEvent.click(map);
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('loadmapdisplay', { mapName: 'map2players' });
  jest.clearAllMocks();
  fireEvent.click(remove);
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('deletesave', { saveId: 7 });
  jest.clearAllMocks();
  fireEvent.keyDown(window, { key: 'Delete' });
  fireEvent.keyDown(window, { key: 'Backspace' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});
