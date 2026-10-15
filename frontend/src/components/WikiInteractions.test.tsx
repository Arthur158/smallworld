import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import applicationReducer from '../redux/slices/applicationSlice';
import { ApplicationState } from '../types/redux';
import { Player, Tile } from '../types/Board';
import { sendMessageToBackend } from '../services/backendService';
import { WikiProvider, useWiki } from './wiki/WikiContext';
import TribeList from './layouts/TribeList';
import PlayerInfo from './layouts/PlayerInfo';
import OpponentsList from './layouts/OpponentsList';
import OpponentsListAll from './layouts/OpponentsListAll';
import Map from './misc/Map';

jest.mock('../services/backendService', () => ({ sendMessageToBackend: jest.fn() }));
// Exercise the real provider's event interception without coupling these game
// interaction tests to the dialog's article presentation or search implementation.
jest.mock('./wiki/WikiDialog', () => ({
  __esModule: true,
  default: ({ initialTarget, onClose }: any) => {
    const React = require('react');
    return React.createElement('aside', null,
      React.createElement('output', { 'data-testid': 'wiki-target' }, JSON.stringify(initialTarget)),
      React.createElement('button', { onClick: onClose }, 'Close wiki'));
  },
}));

const players: Player[] = [
  { name: 'Host', activeTribe: { race: 'Elves', trait: 'Merchant' },
    passiveTribes: [{ race: 'Dwarves', trait: 'Fortified' }],
    pieceStacks: [{ type: 'Elves', amount: 4, isActive: true }] },
  { name: 'Opponent', activeTribe: { race: 'Sorcerers', trait: 'Flying' },
    passiveTribes: [{ race: 'Trolls', trait: 'Alchemist' }],
    pieceStacks: [{ type: 'Sorcerers', amount: 3, isActive: true }, { type: 'Trolls', amount: 1, isActive: false }] },
];
const tile: Tile = {
  id: '1', polygon: { coords: [0, 0, 100, 0, 100, 100], stackX: 30, stackY: 30 },
  pieceStack: [{ type: 'Sorcerers', amount: 2, isActive: true }],
};
const marketTribe = { race: 'Humans', trait: 'Mounted', pieceCount: 7, coinCount: 0 };

function InspectToggle() {
  const { inspectMode, setInspectMode } = useWiki();
  return <button onClick={() => setInspectMode(!inspectMode)}>Toggle inspection</button>;
}

function renderGame(content: React.ReactNode, overrides: Partial<ApplicationState> = {}) {
  const state: ApplicationState = {
    ...applicationReducer(undefined, { type: 'test/init' }),
    players, playerIndex: 0, playerNumber: 1, phase: 'Conquest',
    availableTribes: [marketTribe], mapName: 'map2players', tiles: { '1': tile },
    ...overrides,
  };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  const view = render(<Provider store={store}><WikiProvider><InspectToggle />{content}</WikiProvider></Provider>);
  return { store, ...view };
}

function renderMap(overrides: Partial<ApplicationState> = {}) {
  const image = { width: 1130, height: 700, src: '', onload: undefined as (() => void) | undefined };
  jest.spyOn(window, 'Image').mockImplementation(() => image as unknown as HTMLImageElement);
  const view = renderGame(<Map />, overrides);
  act(() => image.onload?.());
  return view;
}

function wikiTarget() {
  return JSON.parse(screen.getByTestId('wiki-target').textContent || '{}');
}

beforeEach(() => jest.clearAllMocks());
afterEach(() => jest.restoreAllMocks());

test('market inspection does not pick a tribe or apply a selected piece action', () => {
  const { store } = renderGame(<TribeList />, { selectedStack: 'Dragon', isStackFromBank: true });
  fireEvent.click(screen.getByRole('button', { name: 'Mounted Humans' }), { ctrlKey: true });
  expect(wikiTarget()).toEqual({ kind: 'tribe', tribe: marketTribe });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application.selectedStack).toBe('Dragon');
});

test('normal market clicks retain tribe selection and special-piece actions', () => {
  const view = renderGame(<TribeList />);
  fireEvent.click(screen.getByRole('button', { name: 'Mounted Humans' }));
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('tribepick', { pickIndex: 0 });
  view.unmount();
  renderGame(<TribeList />, { selectedStack: 'Dragon', isStackFromBank: true });
  fireEvent.click(screen.getByRole('button', { name: 'Mounted Humans' }));
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('entryaction', { pickIndex: 0, stack: 'Dragon' });
});

test('market keyboard and inspect-mode clicks open the same paired tribe', () => {
  renderGame(<TribeList />);
  const card = screen.getByRole('button', { name: 'Mounted Humans' });
  fireEvent.keyDown(card, { key: 'Enter', metaKey: true });
  expect(wikiTarget()).toEqual({ kind: 'tribe', tribe: marketTribe });
  fireEvent.click(screen.getByRole('button', { name: 'Close wiki' }));
  fireEvent.click(screen.getByRole('button', { name: 'Toggle inspection' }));
  fireEvent.click(card);
  expect(wikiTarget()).toEqual({ kind: 'tribe', tribe: marketTribe });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('own bank inspection preserves pending selection and uses its tribe pair', () => {
  const { store } = renderGame(<PlayerInfo />, { selectedStack: 'Dragon', selectedTile: '1', phase: 'Redeployment' });
  fireEvent.click(screen.getByRole('button', { name: 'Elves pieces in your bank, 4' }), { metaKey: true });
  expect(wikiTarget()).toEqual({ kind: 'piece', stack: players[0].pieceStacks[0], tribe: players[0].activeTribe });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application.selectedStack).toBe('Dragon');
  expect(store.getState().application.selectedTile).toBe('1');
});

test('own card inspection cannot bubble into abandonment or deployment', () => {
  renderGame(<PlayerInfo />, { selectedStack: 'Elves', selectedTile: '1', phase: 'TileAbandonment' });
  fireEvent.click(screen.getByRole('button', { name: 'Fortified Dwarves' }), { ctrlKey: true });
  expect(wikiTarget()).toEqual({ kind: 'tribe', tribe: players[0].passiveTribes[0] });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('normal own bank clicks still select and deselect pieces', () => {
  const { store } = renderGame(<PlayerInfo />);
  const bank = screen.getByRole('button', { name: 'Elves pieces in your bank, 4' });
  fireEvent.click(bank);
  expect(store.getState().application.selectedStack).toBe('Elves');
  expect(store.getState().application.isStackFromBank).toBe(true);
  fireEvent.click(bank);
  expect(store.getState().application.selectedStack).toBeNull();
  expect(store.getState().application.isStackFromBank).toBe(false);
});

test.each([['opponent list', OpponentsList], ['spectator list', OpponentsListAll]])('%s inspects active and declined bank pieces with their own pair', (_, Component) => {
  renderGame(<Component />, { selectedStack: 'Dragon', isStackFromBank: true });
  fireEvent.click(screen.getByRole('button', { name: "Sorcerers pieces in Opponent's bank, 3" }), { ctrlKey: true });
  expect(wikiTarget()).toEqual({ kind: 'piece', stack: players[1].pieceStacks[0], tribe: players[1].activeTribe });
  fireEvent.click(screen.getByRole('button', { name: 'Close wiki' }));
  fireEvent.keyDown(screen.getByRole('button', { name: "Trolls pieces in Opponent's bank, 1" }), { key: 'Enter', metaKey: true });
  expect(wikiTarget()).toEqual({ kind: 'piece', stack: players[1].pieceStacks[1], tribe: players[1].passiveTribes[0] });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('opponent card inspection consumes the click while normal clicks retain opponent actions', () => {
  renderGame(<OpponentsList />, { selectedStack: 'Dragon', isStackFromBank: true });
  const card = screen.getByRole('button', { name: 'Flying Sorcerers' });
  fireEvent.click(card, { ctrlKey: true });
  expect(wikiTarget()).toEqual({ kind: 'tribe', tribe: players[1].activeTribe });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Close wiki' }));
  fireEvent.click(card);
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith('opponentaction', { opponentName: 'Opponent', stackType: 'Dragon' });
});

test('map zone inspection prevents conquest and preserves selected pieces', () => {
  const { store } = renderMap({ selectedStack: 'Elves', isStackFromBank: true });
  const zone = screen.getByRole('button', { name: 'Zone 1' });
  fireEvent.click(zone, { ctrlKey: true });
  expect(wikiTarget()).toEqual({ kind: 'zone', tile });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application.selectedStack).toBe('Elves');
  fireEvent.click(screen.getByRole('button', { name: 'Close wiki' }));
  fireEvent.click(zone);
  expect(sendMessageToBackend).toHaveBeenCalledWith('Conquest', { tileId: '1', attackingStackType: 'Elves' });
});

test('map piece inspection supplies the stack and zone without moving or selecting it', () => {
  const { store } = renderMap({ selectedStack: 'Elves', selectedTile: '2' });
  fireEvent.click(screen.getByRole('button', { name: 'Sorcerers pieces in zone 1, 2 active' }), { metaKey: true });
  expect(wikiTarget()).toEqual({ kind: 'piece', stack: tile.pieceStack[0], tileId: '1' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application.selectedTile).toBe('2');
});

test('normal map piece clicks still select a stack', () => {
  const { store } = renderMap();
  fireEvent.click(screen.getByRole('button', { name: 'Sorcerers pieces in zone 1, 2 active' }));
  expect(store.getState().application.selectedStack).toBe('Sorcerers');
  expect(store.getState().application.selectedTile).toBe('1');
  expect(store.getState().application.isStackFromBank).toBe(false);
});

test('map keyboard inspection and inspect-mode space consume game actions', () => {
  renderMap({ selectedStack: 'Elves', isStackFromBank: true });
  fireEvent.keyDown(screen.getByRole('button', { name: 'Zone 1' }), { key: 'Enter', ctrlKey: true });
  expect(wikiTarget()).toEqual({ kind: 'zone', tile });
  fireEvent.click(screen.getByRole('button', { name: 'Close wiki' }));
  fireEvent.click(screen.getByRole('button', { name: 'Toggle inspection' }));
  fireEvent.keyDown(screen.getByRole('button', { name: 'Sorcerers pieces in zone 1, 2 active' }), { key: ' ' });
  expect(wikiTarget()).toEqual({ kind: 'piece', stack: tile.pieceStack[0], tileId: '1' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test.each(['Zone 1', 'Sorcerers pieces in zone 1, 2 active'])('dragging across %s does not inspect or issue a game command', targetName => {
  const { container } = renderMap({ selectedStack: 'Elves', isStackFromBank: true });
  const target = screen.getByRole('button', { name: targetName });
  const svg = container.querySelector('svg')!;
  fireEvent.mouseDown(target, { clientX: 0, clientY: 0 });
  fireEvent.mouseMove(svg, { clientX: 20, clientY: 0 });
  fireEvent.mouseUp(svg, { clientX: 20, clientY: 0 });
  fireEvent.click(target, { ctrlKey: true });
  fireEvent.click(target);
  expect(screen.queryByTestId('wiki-target')).toBeNull();
  expect(sendMessageToBackend).not.toHaveBeenCalled();

  fireEvent.mouseDown(target, { clientX: 20, clientY: 0 });
  fireEvent.mouseUp(target, { clientX: 20, clientY: 0 });
  fireEvent.click(target, { ctrlKey: true });
  expect(screen.getByTestId('wiki-target')).toBeTruthy();
});

test('game shortcuts cannot clear a selection while inspecting or typing in a field', () => {
  const { store } = renderGame(<><PlayerInfo /><input aria-label="Search rules" /></>, {
    selectedStack: 'Elves', selectedTile: '1', phase: 'Redeployment',
  });
  fireEvent.keyDown(screen.getByRole('textbox', { name: 'Search rules' }), { key: 'a' });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Merchant Elves' }), { ctrlKey: true });
  fireEvent.keyDown(window, { key: 'f' });
  fireEvent.keyDown(window, { key: 'a' });
  expect(store.getState().application.selectedStack).toBe('Elves');
  expect(store.getState().application.selectedTile).toBe('1');
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});


test('map shortcuts preserve selected pieces while the wiki is open', () => {
  const { store } = renderMap({ selectedStack: 'Elves', isStackFromBank: true });
  fireEvent.click(screen.getByRole('button', { name: 'Zone 1' }), { ctrlKey: true });
  fireEvent.keyDown(window, { key: 'f' });
  expect(store.getState().application.selectedStack).toBe('Elves');
  expect(store.getState().application.isStackFromBank).toBe(true);
});

test.each([['opponent list', OpponentsList], ['spectator list', OpponentsListAll]])('%s can inspect a declined race and trait together', (_, Component) => {
  renderGame(<Component />, { selectedStack: 'Dragon', isStackFromBank: true });
  fireEvent.click(screen.getByRole('button', { name: 'Toggle inspection' }));
  fireEvent.keyDown(screen.getByRole('button', { name: 'Alchemist Trolls' }), { key: ' ' });
  expect(wikiTarget()).toEqual({ kind: 'tribe', tribe: players[1].passiveTribes[0] });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});


const traitArtworkCases = [
  { name: 'market', Component: TribeList, tribe: marketTribe },
  { name: 'own active card', Component: PlayerInfo, tribe: players[0].activeTribe! },
  { name: 'own declined card', Component: PlayerInfo, tribe: players[0].passiveTribes[0] },
  { name: 'opponent active card', Component: OpponentsList, tribe: players[1].activeTribe! },
  { name: 'opponent declined card', Component: OpponentsList, tribe: players[1].passiveTribes[0] },
  { name: 'spectator active card', Component: OpponentsListAll, tribe: players[1].activeTribe! },
  { name: 'spectator declined card', Component: OpponentsListAll, tribe: players[1].passiveTribes[0] },
];

test.each(traitArtworkCases)('$name trait artwork supports Ctrl, Cmd and Inspect mode without a game action', ({ Component, tribe }) => {
  const { store } = renderGame(<Component />, {
    selectedStack: 'Dragon', selectedTile: '1', isStackFromBank: true, phase: 'Redeployment',
  });
  const artwork = screen.getByRole('img', { name: tribe.trait });
  for (const modifiers of [{ ctrlKey: true }, { metaKey: true }]) {
    fireEvent.click(artwork, modifiers);
    expect(wikiTarget()).toEqual({ kind: 'tribe', tribe });
    fireEvent.click(screen.getByRole('button', { name: 'Close wiki' }));
  }
  fireEvent.click(screen.getByRole('button', { name: 'Toggle inspection' }));
  fireEvent.click(artwork);
  expect(wikiTarget()).toEqual({ kind: 'tribe', tribe });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(store.getState().application).toEqual(expect.objectContaining({
    selectedStack: 'Dragon', selectedTile: '1', isStackFromBank: true,
  }));
});
