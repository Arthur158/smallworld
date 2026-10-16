import React from 'react';
import { act, fireEvent, render, screen, waitFor } from '@testing-library/react';
import { Provider } from 'react-redux';
import { MemoryRouter } from 'react-router-dom';
import { configureStore } from '@reduxjs/toolkit';
import { createTransform, persistReducer, persistStore } from 'redux-persist';
import Router from './Router';
import GamePage from './GamePage';
import { WikiProvider } from '../components/wiki/WikiContext';
import TooltipProvider from '../components/help/TooltipProvider';
import applicationReducer, { websocketMessageReceived } from '../redux/slices/applicationSlice';
import rootReducer from '../redux/reducer';
import { persistApplicationPreferences, restoreApplicationPreferences } from '../redux/persistence';
import { connectWebSocket, getAuthConfig, sendMessageToBackend } from '../services/backendService';
import { ApplicationState } from '../types/redux';
import { Language } from '../types/misc';

jest.mock('../services/backendService', () => ({
  sendMessageToBackend: jest.fn(() => true), connectWebSocket: jest.fn(), getAuthConfig: jest.fn(),
}));

const originalImage = global.Image;
const originalCreateURL = URL.createObjectURL;
const originalRevokeURL = URL.revokeObjectURL;
const mapName = 'generated-integration-recovery';
const serverPlayers = [
  { name: 'Host', activeTribe: { race: 'Humans', trait: 'Mounted' }, passiveTribes: [], pieceStacks: [{ type: 'Humans', amount: 3, isActive: true }] },
  { name: 'Guest', activeTribe: { race: '', trait: '' }, passiveTribes: [], pieceStacks: [] },
];
const oldState = () => ({
  ...applicationReducer(undefined, { type: 'test/init' }), language: Language.EN,
  generatedMapEditor: undefined, name: 'Host', isAuthenticated: true, gameStarted: true,
  roomid: 'old-room', mapName: 'generated-old', mapImageUrl: 'blob:http://localhost/expired',
  phase: 'Conquest', playerIndex: 0, playerNumber: 0, players: serverPlayers,
  selectedTile: 'removed-region', selectedStack: 'Humans', isStackFromBank: false,
});

beforeEach(() => {
  jest.clearAllMocks();
  (getAuthConfig as jest.Mock).mockResolvedValue({ enabled: false });
  URL.createObjectURL = jest.fn(() => 'blob:http://localhost/fresh-map');
  URL.revokeObjectURL = jest.fn();
  global.Image = class {
    width = 1400; height = 900; onload: (() => void) | null = null; onerror: (() => void) | null = null;
    set src(_value: string) { Promise.resolve().then(() => this.onload?.()); }
  } as unknown as typeof Image;
  jest.spyOn(console, 'log').mockImplementation(() => {});
});
afterEach(() => {
  global.Image = originalImage;
  URL.createObjectURL = originalCreateURL;
  URL.revokeObjectURL = originalRevokeURL;
  jest.restoreAllMocks();
});
function wrap(store: any, content: React.ReactNode) {
  return <Provider store={store}><TooltipProvider><WikiProvider>{content}</WikiProvider></TooltipProvider></Provider>;
}
function broadcast(store: any, type: string, data: unknown) {
  act(() => store.dispatch(websocketMessageReceived(JSON.stringify({ type, data }))));
}
async function startGeneratedGame(store: any, metadata: { regionName?: string; regionNameVersion?: number } = {}) {
  broadcast(store, 'smallmapupdate', { mapName, offset: 0.808 });
  broadcast(store, 'generatedmapvisuals', {
    mapName, imageBase64: 'cG5n', offset: 0.808, ...metadata,
    graphics: { OffsetStacks: 0.002, Xmult: 1, Ymult: 1, Tiles: [
      { ID: '1', Polygon: { Coords: [0, 0, 200, 0, 200, 200, 0, 200], StackX: 100, StackY: 100 } },
    ] }, serverTiles: [{ id: '1', biome: 'field', attributes: [] }],
  });
  broadcast(store, 'index', { index: '0' });
  broadcast(store, 'megaUpdate', {
    turnInfo: { playerNumber: 0, turnNumber: 1, phase: 'Conquest' }, players: serverPlayers,
    tribeEntries: [{ race: 'Elves', trait: 'Merchant', pieceCount: 8, coinCount: 0 }],
    allTiles: [{ tileID: '1', stacks: [{ type: 'Humans', amount: 2, isActive: true }] }],
  });
  await waitFor(() => expect(screen.getByRole('button', { name: 'Zone 1' })).toBeTruthy());
}

test('old persisted active games are discarded before routing, then real server hydration renders the full game', async () => {
  const legacy = { ...oldState(), players: serverPlayers.map(player => ({ ...player, passiveTribes: null, pieceStacks: null })) };
  const persisted = JSON.stringify({ application: JSON.stringify(legacy), _persist: JSON.stringify({ version: -1, rehydrated: true }) });
  const storage = { getItem: jest.fn(async () => persisted), setItem: jest.fn(async (_key: string, _value: string) => {}), removeItem: jest.fn(async () => {}) };
  const reducer = persistReducer({ key: 'recovery-test', storage, whitelist: ['application'], transforms: [
    createTransform(persistApplicationPreferences, restoreApplicationPreferences, { whitelist: ['application'] }),
  ] }, rootReducer);
  const store = configureStore({ reducer, middleware: getDefault => getDefault({ serializableCheck: false }) });
  let persistor: ReturnType<typeof persistStore>;
  await new Promise<void>(resolve => { persistor = persistStore(store, undefined, resolve); });
  expect(store.getState().application.isAuthenticated).toBe(false);
  expect(store.getState().application.gameStarted).toBe(false);
  expect(store.getState().application.players).toEqual([]);
  expect(store.getState().application.mapImageUrl).toBeNull();
  expect(store.getState().application.language).toBe(Language.EN);
  render(wrap(store, <MemoryRouter initialEntries={['/game']} future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><Router /></MemoryRouter>));
  await screen.findByRole('button', { name: 'Log In' });
  expect(connectWebSocket).toHaveBeenCalledTimes(1);
  expect(screen.queryByText('Turn Information')).toBeNull();
  broadcast(store, 'auth', { name: 'Host' });
  await startGeneratedGame(store);
  expect(screen.getByText('Turn Information')).toBeTruthy();
  expect(screen.getByRole('button', { name: 'Humans pieces in zone 1, 2 active' })).toBeTruthy();
  await persistor!.flush();
  const written = JSON.parse(storage.setItem.mock.calls[storage.setItem.mock.calls.length - 1][1]);
  expect(JSON.parse(written.application)).toEqual({ language: 'en' });
  persistor!.pause();
});

test('a replaced map clears a stale selected region while rendering the complete live start sequence', async () => {
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: oldState() as unknown as ApplicationState } });
  render(wrap(store, <GamePage />));
  await startGeneratedGame(store);
  expect(store.getState().application.selectedStack).toBeNull();
  expect(store.getState().application.selectedTile).toBeNull();
  expect(screen.getByText('Turn Information')).toBeTruthy();
});

test.each(['playerupdate', 'megaUpdate'])('%s clears a bank selection whose player index is no longer available', type => {
  const previous = { ...oldState(), playerIndex: 8, isStackFromBank: true } as unknown as ApplicationState;
  const data = type === 'playerupdate' ? serverPlayers : {
    turnInfo: { playerNumber: 0, turnNumber: 1, phase: 'Conquest' }, players: serverPlayers, tribeEntries: [], allTiles: [],
  };
  const next = applicationReducer(previous, websocketMessageReceived(JSON.stringify({ type, data })));
  expect(next.selectedStack).toBeNull();
  expect(next.selectedTile).toBeNull();
  expect(next.isStackFromBank).toBe(false);
});

test('persisted preferences never include sessions, snapshots or blob URLs, and malformed preferences reset safely', () => {
  expect(persistApplicationPreferences(oldState() as unknown as ApplicationState)).toEqual({ language: 'en' });
  for (const old of [null, {}, { language: 'unknown', isAuthenticated: true }]) {
    const restored = restoreApplicationPreferences(old);
    expect(restored.isAuthenticated).toBe(false);
    expect(restored.gameStarted).toBe(false);
    expect(restored.roomid).toBe('');
    expect(restored.mapImageUrl).toBeNull();
    expect(restored.generatedMapEditor).toBeNull();
    expect(restored.language).toBe(Language.NL);
  }
});


test('generated visuals show a continent heading and older visuals replace it with no name', async () => {
  const store = configureStore({ reducer: { application: applicationReducer } });
  render(wrap(store, <GamePage />));
  await startGeneratedGame(store, { regionName: 'The Golden Coast', regionNameVersion: 1 });
  expect(screen.getByRole('heading', { name: 'Continent: The Golden Coast' })).toBeTruthy();
  expect(store.getState().application.mapName).toBe(mapName);
  expect(store.getState().application.mapRegionNameVersion).toBe(1);
  await startGeneratedGame(store);
  expect(screen.queryByRole('heading', { name: /Continent:/ })).toBeNull();
  expect(store.getState().application.mapRegionName).toBeNull();
  expect(store.getState().application.mapRegionNameVersion).toBeNull();
});

test.each(['lobby', 'unauth', 'leavedisplayroom', 'displayroom', 'roomid'])('%s clears any previous continent name', event => {
  const old = { ...oldState(), mapRegionName: 'Previous Continent', mapRegionNameVersion: 1 } as unknown as ApplicationState;
  const next = applicationReducer(old, websocketMessageReceived(JSON.stringify({ type: event, data: { roomid: 'different-room' } })));
  expect(next.mapRegionName).toBeNull();
  expect(next.mapRegionNameVersion).toBeNull();
});


test('saving and a same-map spectator refresh keep the continent heading, image and terrain', async () => {
  const store = configureStore({ reducer: { application: applicationReducer } });
  render(wrap(store, <GamePage />));
  broadcast(store, 'roomid', { roomid: 'current-room' });
  await startGeneratedGame(store, { regionName: 'Celunal', regionNameVersion: 1 });
  const imageUrl = store.getState().application.mapImageUrl;
  const terrain = store.getState().application.tiles['1'].terrain;
  fireEvent.click(screen.getByRole('button', { name: 'Save Game' }));
  expect(sendMessageToBackend).toHaveBeenLastCalledWith('savegame', {});
  broadcast(store, 'message', { message: 'Game successfully saved' });
  broadcast(store, 'loadSaves', { saves: [{ saveId: 5, summary: 'Celunal' }] });
  // Joining spectators trigger this room-wide refresh; existing players do not
  // receive generatedmapvisuals again.
  broadcast(store, 'smallmapupdate', { mapName, offset: 0.808 });
  broadcast(store, 'roomid', { roomid: 'current-room' });
  expect(screen.getByRole('heading', { name: 'Continent: Celunal' })).toBeTruthy();
  expect(store.getState().application.mapRegionNameVersion).toBe(1);
  expect(store.getState().application.mapImageUrl).toBe(imageUrl);
  expect(store.getState().application.tiles['1'].terrain).toEqual(terrain);
  expect(screen.getByRole('button', { name: 'Humans pieces in zone 1, 2 active' })).toBeTruthy();
});

test('a legacy identity-free image refresh preserves the continent name and region metadata', async () => {
  const store = configureStore({ reducer: { application: applicationReducer } });
  render(wrap(store, <GamePage />));
  await startGeneratedGame(store, { regionName: 'Celunal', regionNameVersion: 1 });
  await act(async () => broadcast(store, 'mapupdate', { picture: 'cG5n', offset: 0.808 }));
  expect(screen.getByRole('heading', { name: 'Continent: Celunal' })).toBeTruthy();
  expect(store.getState().application.tiles['1'].terrain?.biome).toBe('field');
  expect(screen.getByRole('button', { name: 'Zone 1' })).toBeTruthy();
});

test.each(['map2players', 'different-generated-map'])('a switch to %s clears the old continent name', async nextMap => {
  const store = configureStore({ reducer: { application: applicationReducer } });
  render(wrap(store, <GamePage />));
  await startGeneratedGame(store, { regionName: 'Celunal', regionNameVersion: 1 });
  await act(async () => broadcast(store, 'smallmapupdate', { mapName: nextMap, offset: 0.808 }));
  expect(screen.queryByRole('heading', { name: /Continent:/ })).toBeNull();
  expect(store.getState().application.mapRegionName).toBeNull();
  expect(store.getState().application.mapRegionNameVersion).toBeNull();
});
