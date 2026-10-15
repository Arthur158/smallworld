import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import applicationReducer, { websocketMessageReceived } from '../redux/slices/applicationSlice';
import { ApplicationState } from '../types/redux';
import { Player, Tile } from '../types/Board';
import { sendMessageToBackend } from '../services/backendService';
import { WikiProvider, useWiki } from './wiki/WikiContext';
import PlayerInfo from './layouts/PlayerInfo';
import TribeList from './layouts/TribeList';
import Map from './misc/Map';

jest.mock('../services/backendService', () => ({ sendMessageToBackend: jest.fn() }));
jest.mock('./wiki/WikiDialog', () => ({
  __esModule: true,
  default: () => <aside role="dialog" aria-modal="true">Rules open</aside>,
}));

const players: Player[] = [
  { name: 'Me', activeTribe: { race: 'Elves', trait: 'Merchant' }, passiveTribes: [], pieceStacks: [
    { type: 'Elves', amount: 4, isActive: true },
    { type: 'Staff', amount: 0, isActive: true },
    { type: 'Dragon', amount: 1, isActive: true },
  ] },
  { name: 'Rival', activeTribe: { race: 'Trolls', trait: 'Flying' }, passiveTribes: [], pieceStacks: [] },
];
const tile: Tile = { id: '1', polygon: { coords: [0, 0, 100, 0, 100, 100], stackX: 30, stackY: 30 }, pieceStack: [players[0].pieceStacks[0]] };
function WikiControls() {
  const { openWiki, setInspectMode } = useWiki();
  return <><button onClick={() => openWiki()}>Open rules</button><button onClick={() => setInspectMode(true)}>Inspect</button></>;
}
function setup(content: React.ReactNode, overrides: Partial<ApplicationState> = {}) {
  const state: ApplicationState = {
    ...applicationReducer(undefined, { type: 'test/init' }), players, playerIndex: 0, playerNumber: 0,
    phase: 'DeclineChoice', selectedTile: '1', selectedStack: 'Elves', isStackFromBank: false,
    mapName: 'map2players', tiles: { '1': tile }, availableTribes: [
      { race: 'Humans', trait: 'Mounted', pieceCount: 7, coinCount: 0 },
      { race: 'Giants', trait: 'Forest', pieceCount: 9, coinCount: 1 },
    ], ...overrides,
  };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  const dispatch = jest.spyOn(store, 'dispatch');
  const image = { width: 1130, height: 700, src: '', onload: undefined as (() => void) | undefined };
  jest.spyOn(window, 'Image').mockImplementation(() => image as unknown as HTMLImageElement);
  const view = render(<Provider store={store}><WikiProvider><WikiControls />{content}</WikiProvider></Provider>);
  act(() => image.onload?.());
  return { store, dispatch, ...view };
}
const key = (value: string, options: Record<string, unknown> = {}) => fireEvent.keyDown(window, { key: value, ...options });
beforeEach(() => jest.clearAllMocks());
afterEach(() => jest.restoreAllMocks());

test.each(['f', 'Escape'])('%s clears the complete selection once when map and bank are both mounted', shortcut => {
  const { store, dispatch } = setup(<><PlayerInfo /><Map /></>);
  dispatch.mockClear();
  key(shortcut);
  expect(store.getState().application).toEqual(expect.objectContaining({ selectedStack: null, selectedTile: null, isStackFromBank: false }));
  expect(dispatch.mock.calls.map(([action]) => (action as any).type)).toEqual([
    'application/setSelectedStack', 'application/setSelectedTile', 'application/setIsStackFromBank',
  ]);
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('map clearing works in spectator view without a player bank', () => {
  const { store } = setup(<Map />, { playerIndex: -1, isSpectating: true, isStackFromBank: true });
  key('Escape');
  expect(store.getState().application.selectedStack).toBeNull();
  expect(store.getState().application.isStackFromBank).toBe(false);
});

test('G cycles usable bank pieces, clears the board origin and deselects after the last piece', () => {
  const { store } = setup(<PlayerInfo />);
  key('g');
  expect(store.getState().application).toEqual(expect.objectContaining({ selectedStack: 'Elves', selectedTile: null, isStackFromBank: true }));
  key('g');
  expect(store.getState().application.selectedStack).toBe('Dragon');
  key('g');
  expect(store.getState().application).toEqual(expect.objectContaining({ selectedStack: null, selectedTile: null, isStackFromBank: false }));
  key('g');
  expect(store.getState().application.selectedStack).toBe('Elves');
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('A uses current phase after an update and sends one command for the selected board piece', () => {
  const { store } = setup(<PlayerInfo />);
  key('a');
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('abandonment', { tileId: '1', stackType: 'Elves' });
  act(() => { store.dispatch(websocketMessageReceived(JSON.stringify({ type: 'turnupdate', data: { playerNumber: 0, turnNumber: 2, phase: 'Redeployment' } }))); });
  key('a');
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('deploymentout', { tileId: '1', stackType: 'Elves' });
  expect(sendMessageToBackend).toHaveBeenCalledTimes(2);
  expect(store.getState().application.selectedStack).toBeNull();
});

test.each([
  { playerNumber: 1 }, { isSpectating: true }, { inDisplayRoom: true }, { phase: 'GameFinished' },
])('A and G are disabled outside a playable own turn: %j', state => {
  const { store } = setup(<PlayerInfo />, state);
  key('a'); key('g');
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application).toEqual(expect.objectContaining({ selectedStack: 'Elves', selectedTile: '1', isStackFromBank: false }));
});

test.each([{ isStackFromBank: true }, { phase: 'Conquest' }, { selectedTile: null }])('A does not return pieces for an ineligible selection: %j', state => {
  setup(<PlayerInfo />, state);
  key('a');
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test.each([{ ctrlKey: true }, { metaKey: true }, { altKey: true }, { repeat: true }, { isComposing: true }])('board shortcuts leave modified/repeated/composing events alone: %j', options => {
  const { store } = setup(<><PlayerInfo /><Map /></>);
  for (const shortcut of ['a', 'g', 'f', 'Escape']) key(shortcut, options);
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application.selectedTile).toBe('1');
});

test('typing or an open dialog cannot trigger board shortcuts', () => {
  const { store } = setup(<><PlayerInfo /><Map /><input aria-label="Message" /><div role="textbox" contentEditable suppressContentEditableWarning>Notes</div></>);
  for (const target of screen.getAllByRole('textbox')) for (const shortcut of ['a', 'g', 'f', 'Escape']) fireEvent.keyDown(target, { key: shortcut });
  fireEvent.click(screen.getByRole('button', { name: 'Open rules' }));
  for (const shortcut of ['a', 'g', 'f', 'Escape']) key(shortcut);
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application.selectedTile).toBe('1');
});

test('Inspect mode blocks board actions and consumes its own Escape without clearing the selection', () => {
  const { store } = setup(<><PlayerInfo /><Map /></>);
  fireEvent.click(screen.getByRole('button', { name: 'Inspect' }));
  for (const shortcut of ['a', 'g', 'f', 'Escape']) key(shortcut);
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application.selectedTile).toBe('1');
});

test('market number keys choose only existing visible entries on an own tribe-choice turn', () => {
  setup(<TribeList />, { phase: 'TribeChoice', selectedStack: null, selectedTile: null });
  key('2'); key('9');
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('tribepick', { pickIndex: 1 });
});

test.each([{ playerNumber: 1 }, { phase: 'Conquest' }, { selectedStack: 'Mirror', isStackFromBank: true }, { isSpectating: true }])('market shortcuts do not pick outside a plain own tribe choice: %j', state => {
  setup(<TribeList />, { phase: 'TribeChoice', selectedStack: null, selectedTile: null, ...state });
  key('1');
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('market number keys ignore typing and repeated presses', () => {
  setup(<><TribeList /><input aria-label="Chat" /></>, { phase: 'TribeChoice', selectedStack: null });
  fireEvent.keyDown(screen.getByRole('textbox'), { key: '1' });
  key('1', { repeat: true });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('pieces and regions expose custom hints without native hover titles', () => {
  const { container } = setup(<><PlayerInfo /><TribeList /><Map /></>);
  expect(screen.getByRole('button', { name: 'Elves pieces in your bank, 4' }).getAttribute('data-tooltip')).toContain('G: next bank piece');
  expect(screen.getByRole('button', { name: 'Mounted Humans' }).getAttribute('data-tooltip')).toContain('Enter/Space');
  expect(screen.getByRole('button', { name: 'Zone 1' }).getAttribute('data-tooltip')).toContain('Ctrl/Cmd-click');
  expect(container.querySelector('[title], svg title')).toBeNull();
});
