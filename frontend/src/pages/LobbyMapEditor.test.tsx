import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { MemoryRouter } from 'react-router-dom';
import { configureStore } from '@reduxjs/toolkit';
import LobbyPage from './LobbyPage';
import applicationReducer, { websocketMessageReceived } from '../redux/slices/applicationSlice';
import { ApplicationState } from '../types/redux';
import { GeneratedMapSettings } from '../types/GeneratedMap';
import { Room } from '../types/Board';
import { sendMessageToBackend } from '../services/backendService';

jest.mock('../services/backendService', () => ({ sendMessageToBackend: jest.fn(), getAuthConfig: jest.fn() }));
jest.mock('../components/wiki/WikiToolbar', () => () => null);
jest.mock('../components/RoomBotControls', () => ({ __esModule: true, default: () => null, BotModelSelect: () => null }));
const send = sendMessageToBackend as jest.Mock;
const room: Room = { id: 'custom-room', name: 'Map room', creator: 'Host', players: ['Host', 'Bot'], capacity: 6, mapName: 'generatedmap' };
const settings: GeneratedMapSettings = {
  seed: 'current-default', width: 1400, height: 900, tileCount: 22, preserveRegionCount: true,
  shapeMode: 'mainland', wobble: 30, spread: 15, tileShapeSettings: { chonkiness: 80, repulsionSteps: 12 },
  waterSettings: { centerLakeSize: 140, seaSize: 160 }, scenerySettings: { forestTreeSize: 95, mountainObjectSize: 240 },
  biomeWeights: { field: 100, grass: 100, forest: 100, mountain: 100, swamp: 100 },
  biomeTextureZooms: { field: 60, grass: 60, forest: 40, mountain: 60, swamp: 60, sea: 60 },
  buildingSettings: { averagePerTile: .28, maxPerTile: 1, sizeMultiplier: 3, symbolBuildingChance: 100 },
  featureSettings: { magic: { count: 3, spread: 10 }, mine: { count: 3, spread: 10 }, cave: { count: 4, spread: 10 } },
  seaCorners: { NW: true, NE: false, SW: false, SE: true }, randomizeSeaCorners: false, centerLake: true,
};
function setup(overrides: Partial<ApplicationState> = {}) {
  const state = { ...applicationReducer(undefined, { type: 'init' }), name: 'Host', isAuthenticated: true,
    roomid: room.id, rooms: [room], mapChoices: ['generatedmap', 'map2players'], ...overrides };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  render(<Provider store={store}><MemoryRouter future={{ v7_startTransition: true, v7_relativeSplatPath: true }}><LobbyPage /></MemoryRouter></Provider>);
  const receive = (type: string, data: unknown) => act(() => { store.dispatch(websocketMessageReceived(JSON.stringify({ type, data }))); });
  return { store, receive };
}
function request(type: string) { return send.mock.calls.filter(([name]) => name === type).slice(-1)[0]?.[1]; }
function open(receive: ReturnType<typeof setup>['receive'], selectedPreviewId?: string) {
  fireEvent.click(screen.getByRole('button', { name: 'Customize map' }));
  expect(screen.getByRole('dialog')).toBeTruthy();
  receive('generatedmapsettings', { roomId: room.id, requestId: request('getGeneratedMapSettings').requestId,
    playerCount: 2, settings, selectedPreviewId });
}
function generate(receive: ReturnType<typeof setup>['receive'], previewId = 'preview-one') {
  fireEvent.click(screen.getByRole('button', { name: 'Generate preview' }));
  const req = request('previewGeneratedMap');
  receive('generatedmappreview', { roomId: room.id, requestId: req.requestId, previewId, playerCount: 2,
    settings: req.settings, imageBase64: 'aW1hZ2U=' });
}
beforeEach(() => {
  jest.clearAllMocks(); send.mockReturnValue(true);
  URL.createObjectURL = jest.fn(() => 'blob:preview'); URL.revokeObjectURL = jest.fn();
});

test.each([
  { name: 'Guest' }, { rooms: [{ ...room, mapName: 'map2players' }] }, { saveSelectionId: 8 },
])('customizer is available only to a waiting generated-map host without a save', overrides => {
  setup(overrides); expect(screen.queryByRole('button', { name: 'Customize map' })).toBeNull();
});

test('automatic remains selected until a matching generated preview is explicitly selected and acknowledged', () => {
  const { store, receive } = setup(); open(receive);
  expect(screen.getByRole('button', { name: /Automatic at game start/ }).getAttribute('aria-pressed')).toBe('true');
  expect((screen.getByRole('button', { name: 'Select this map' }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.change(screen.getByLabelText('Seed'), { target: { value: 'chosen-seed' } });
  generate(receive);
  expect(request('previewGeneratedMap').settings.seed).toBe('chosen-seed');
  expect(request('selectGeneratedMap')).toBeUndefined();
  expect(store.getState().application.mapName).toBeNull();
  expect(store.getState().application.mapImageUrl).toBeNull();
  const select = screen.getByRole('button', { name: 'Select this map' }) as HTMLButtonElement;
  expect(select.disabled).toBe(false);
  fireEvent.change(screen.getByRole('slider', { name: 'Sea size' }), { target: { value: '180' } });
  expect(select.disabled).toBe(true);
  fireEvent.change(screen.getByRole('slider', { name: 'Sea size' }), { target: { value: '160' } });
  fireEvent.click(select);
  expect(request('selectGeneratedMap')).toEqual({ previewId: 'preview-one' });
  expect(screen.getByRole('dialog')).toBeTruthy();
  receive('generatedmapselection', { roomId: room.id, previewId: 'preview-one', playerCount: 2 });
  expect(screen.queryByRole('dialog')).toBeNull();
  expect(URL.revokeObjectURL).toHaveBeenCalledWith('blob:preview');
});

test('closed and outdated requests cannot replace a preview or game state', () => {
  const { store, receive } = setup(); open(receive);
  fireEvent.click(screen.getByRole('button', { name: 'Generate preview' }));
  const req = request('previewGeneratedMap');
  receive('generatedmappreview', { roomId: room.id, requestId: 'old-request', previewId: 'old', settings, imageBase64: 'aW1hZ2U=', playerCount: 2 });
  expect(URL.createObjectURL).not.toHaveBeenCalled();
  fireEvent.click(screen.getByRole('button', { name: 'Back' }));
  receive('generatedmappreview', { roomId: room.id, requestId: req.requestId, previewId: 'late', settings, imageBase64: 'aW1hZ2U=', playerCount: 2 });
  expect(store.getState().application.generatedMapEditor).toBeNull();
  expect(URL.createObjectURL).not.toHaveBeenCalled();
});

test('automatic reset waits for server acknowledgement and clears the custom choice', () => {
  const { receive } = setup({ rooms: [{ ...room, generatedMapSelection: { previewId: 'selected', playerCount: 2, seed: 'seed' } }] });
  open(receive, 'selected');
  fireEvent.click(screen.getByRole('button', { name: /Automatic at game start/ }));
  expect(request('resetGeneratedMap')).toEqual({});
  expect(screen.getByRole('dialog')).toBeTruthy();
  receive('generatedmapselection', { roomId: room.id });
  expect(screen.queryByRole('dialog')).toBeNull();
});

test('errors are visible inside the editor and a disconnected request can be retried', () => {
  const { receive } = setup(); send.mockReturnValue(false);
  fireEvent.click(screen.getByRole('button', { name: 'Customize map' }));
  expect(screen.getByRole('alert').textContent).toContain('Not connected');
  send.mockReturnValue(true); fireEvent.click(screen.getByRole('button', { name: 'Try again' }));
  const req = request('getGeneratedMapSettings');
  receive('generatedmaperror', { roomId: room.id, requestId: req.requestId, message: 'Generator unavailable' });
  expect(screen.getByRole('alert').textContent).toContain('Generator unavailable');
});

test('editor blocks lobby shortcuts and player-count changes prevent starting a stale selected map', () => {
  const { receive } = setup({ rooms: [{ ...room, generatedMapSelection: { previewId: 'selected', playerCount: 3, seed: 'seed' } }] });
  expect((screen.getByRole('button', { name: 'Start Game' }) as HTMLButtonElement).disabled).toBe(true);
  fireEvent.keyDown(window, { key: 'p' }); expect(request('startGame')).toBeUndefined();
  fireEvent.keyDown(window, { key: 'm' });
  expect(request('getGeneratedMapSettings')).toBeTruthy();
  fireEvent.keyDown(window, { key: 'p' }); expect(request('startGame')).toBeUndefined();
  receive('generatedmapsettings', { roomId: room.id, requestId: request('getGeneratedMapSettings').requestId, playerCount: 2, settings });
  expect(screen.getByRole('dialog')).toBeTruthy();
});
