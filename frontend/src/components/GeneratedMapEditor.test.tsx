import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import GeneratedMapEditor, { GeneratedMapEditorProps } from './GeneratedMapEditor';
import { GeneratedMapPreview, GeneratedMapSettings } from '../types/GeneratedMap';
import { useShortcut } from '../hooks/useShortcut';

const defaults: GeneratedMapSettings = {
  seed: 'smallworld-seed', width: 1400, height: 900, tileCount: 22, preserveRegionCount: true,
  shapeMode: 'mainland', wobble: 30, spread: 0,
  tileShapeSettings: { chonkiness: 80, repulsionSteps: 12 },
  waterSettings: { centerLakeSize: 140, seaSize: 160 },
  scenerySettings: { forestTreeSize: 95, mountainObjectSize: 240 },
  biomeWeights: { field: 100, grass: 100, forest: 100, mountain: 100, swamp: 100 },
  biomeTextureZooms: { field: 60, grass: 60, forest: 40, mountain: 60, swamp: 60, sea: 60 },
  buildingSettings: { averagePerTile: 0.28, maxPerTile: 1, sizeMultiplier: 3, symbolBuildingChance: 100 },
  featureSettings: { magic: { count: 3, spread: 0 }, mine: { count: 3, spread: 0 }, cave: { count: 4, spread: 0 } },
  seaCorners: { NW: true, NE: false, SW: false, SE: true }, randomizeSeaCorners: false, centerLake: true,
};
const makePreview = (settings = defaults, id = 'preview-1'): GeneratedMapPreview => ({
  id, settings, imageUrl: 'data:image/png;base64,cHJldmlldw==', playerCount: 2,
});
function setup(extra: Partial<GeneratedMapEditorProps> = {}) {
  const props: GeneratedMapEditorProps = {
    defaultSettings: defaults, playerCount: 2, loading: false, automatic: true,
    onGenerate: jest.fn(), onSelect: jest.fn(), onAutomatic: jest.fn(), onClose: jest.fn(), ...extra,
  };
  const view = render(<GeneratedMapEditor {...props} />);
  return { props, ...view, update: (patch: Partial<GeneratedMapEditorProps>) => {
    Object.assign(props, patch);
    view.rerender(<GeneratedMapEditor {...props} />);
  } };
}
const select = () => screen.getByRole('button', { name: 'Select this map' }) as HTMLButtonElement;
const generate = () => screen.getByRole('button', { name: 'Generate preview' });
const randomMap = () => screen.getByRole('button', { name: 'Generate random map' }) as HTMLButtonElement;
const tileCount = () => screen.getByRole('spinbutton', { name: 'Tile count' }) as HTMLInputElement;

test('automatic remains the current choice until a preview is explicitly selected', () => {
  const { props, update } = setup();
  expect(screen.getByRole('button', { name: /Automatic at game start/ }).getAttribute('aria-pressed')).toBe('true');
  expect(screen.getByText('A world of your own')).toBeTruthy();
  expect(select().disabled).toBe(true);
  fireEvent.click(generate());
  expect(props.onGenerate).toHaveBeenCalledWith(defaults);
  expect(props.onSelect).not.toHaveBeenCalled();
  expect(props.onAutomatic).not.toHaveBeenCalled();
  update({ preview: makePreview() });
  expect(select().disabled).toBe(false);
  expect(screen.getByRole('button', { name: /Automatic at game start/ }).getAttribute('aria-pressed')).toBe('true');
  fireEvent.click(select());
  expect(props.onSelect).toHaveBeenCalledWith('preview-1');
  expect(props.onClose).not.toHaveBeenCalled();
});

test('edits invalidate selection until a matching preview arrives', () => {
  const { props, update } = setup({ preview: makePreview() });
  fireEvent.change(screen.getByRole('slider', { name: 'Forest weight' }), { target: { value: '185' } });
  expect(select().disabled).toBe(true);
  expect(screen.getByText('Earlier preview · settings have changed')).toBeTruthy();
  fireEvent.click(select());
  expect(props.onSelect).not.toHaveBeenCalled();
  fireEvent.click(generate());
  const requested = (props.onGenerate as jest.Mock).mock.calls[0][0];
  expect(requested.biomeWeights.forest).toBe(185);
  expect(requested.tileCount).toBe(22);
  expect(requested.width).toBe(1400);
  expect(requested.height).toBe(900);
  update({ preview: makePreview(requested, 'preview-2') });
  expect(select().disabled).toBe(false);
  fireEvent.click(select());
  expect(props.onSelect).toHaveBeenCalledWith('preview-2');
  expect(defaults.biomeWeights.forest).toBe(100);
});

test('tile count starts at the current player default and only the latest matching preview can be selected', () => {
  const { props, update } = setup({ preview: makePreview() });
  expect(tileCount().value).toBe('22');
  expect(screen.getByText('Default for 2 players: 22 tiles. Includes water regions.')).toBeTruthy();
  fireEvent.change(tileCount(), { target: { value: '80' } });
  expect(select().disabled).toBe(true);
  fireEvent.click(generate());
  const firstRequest = (props.onGenerate as jest.Mock).mock.calls[0][0];
  expect(firstRequest.tileCount).toBe(80);
  update({ loading: true });
  fireEvent.change(tileCount(), { target: { value: '90' } });
  update({ loading: false, preview: makePreview(firstRequest, 'preview-80') });
  expect(tileCount().value).toBe('90');
  expect(select().disabled).toBe(true);
  fireEvent.click(select());
  expect(props.onSelect).not.toHaveBeenCalled();
  fireEvent.click(generate());
  const latestRequest = (props.onGenerate as jest.Mock).mock.calls[1][0];
  expect(latestRequest.tileCount).toBe(90);
  update({ preview: makePreview(latestRequest, 'preview-90') });
  expect(select().disabled).toBe(false);
  fireEvent.click(select());
  expect(props.onSelect).toHaveBeenCalledTimes(1);
  expect(props.onSelect).toHaveBeenCalledWith('preview-90');
});

test('tile count allows empty and partial editing but prevents invalid generation or selection', () => {
  const { props } = setup({ preview: makePreview() });
  for (const value of ['', '1', '17', '121', '40.5']) {
    fireEvent.change(tileCount(), { target: { value } });
    expect(tileCount().value).toBe(value);
    expect(tileCount().getAttribute('aria-invalid')).toBe('true');
    expect(screen.getByRole('alert').textContent).toBe('Choose a whole tile count from 18 to 120.');
    expect((generate() as HTMLButtonElement).disabled).toBe(true);
    expect(randomMap().disabled).toBe(true);
    expect(select().disabled).toBe(true);
    fireEvent.click(generate());
    fireEvent.click(randomMap());
  }
  expect(props.onGenerate).not.toHaveBeenCalled();
  for (const value of ['18', '120']) {
    fireEvent.change(tileCount(), { target: { value } });
    expect((generate() as HTMLButtonElement).disabled).toBe(false);
    fireEvent.click(generate());
    expect(props.onGenerate).toHaveBeenLastCalledWith({ ...defaults, tileCount: Number(value) });
  }
});

test('Generate random map submits one fresh seed immediately and preserves the other draft settings', () => {
  const { props, update } = setup({ preview: makePreview() });
  fireEvent.change(tileCount(), { target: { value: '72' } });
  fireEvent.change(screen.getByRole('slider', { name: 'Forest weight' }), { target: { value: '185' } });
  fireEvent.change(screen.getByLabelText('Seed'), { target: { value: 'previous-seed' } });
  fireEvent.click(randomMap());
  expect(props.onGenerate).toHaveBeenCalledTimes(1);
  const requested = (props.onGenerate as jest.Mock).mock.calls[0][0];
  expect(requested.seed).toEqual(expect.any(String));
  expect(requested.seed.length).toBeGreaterThan(0);
  expect(requested.seed).not.toBe('previous-seed');
  expect(requested).toEqual({ ...defaults, seed: requested.seed, tileCount: 72,
    biomeWeights: { ...defaults.biomeWeights, forest: 185 } });
  expect((screen.getByLabelText('Seed') as HTMLInputElement).value).toBe(requested.seed);
  expect(select().disabled).toBe(true);
  expect(props.onSelect).not.toHaveBeenCalled();
  expect(props.onAutomatic).not.toHaveBeenCalled();
  update({ preview: makePreview(requested, 'random-preview') });
  expect(select().disabled).toBe(false);
  expect(defaults.tileCount).toBe(22);
  expect(defaults.biomeWeights.forest).toBe(100);
});

test('server-normalized settings are adopted for an unchanged submitted draft', () => {
  const { update } = setup();
  fireEvent.change(screen.getByLabelText('Seed'), { target: { value: '' } });
  fireEvent.click(generate());
  const normalized = { ...defaults, seed: 'server-seed' };
  update({ preview: makePreview(normalized) });
  expect((screen.getByLabelText('Seed') as HTMLInputElement).value).toBe('server-seed');
  expect(select().disabled).toBe(false);
});

test('a late response never overwrites edits made while generation is running', () => {
  const { update } = setup();
  fireEvent.click(generate());
  update({ loading: true });
  fireEvent.change(screen.getByLabelText('Seed'), { target: { value: 'my-next-map' } });
  update({ loading: false, preview: makePreview({ ...defaults, seed: 'server-normalized' }) });
  expect((screen.getByLabelText('Seed') as HTMLInputElement).value).toBe('my-next-map');
  expect(select().disabled).toBe(true);
});

test('settings equality ignores JSON property order', () => {
  const reordered = Object.fromEntries(Object.entries(defaults).reverse()) as unknown as GeneratedMapSettings;
  setup({ initialSettings: defaults, preview: makePreview(reordered) });
  expect(select().disabled).toBe(false);
});

test('reset restores current defaults without selecting or generating a map', () => {
  const { props } = setup({ preview: makePreview() });
  fireEvent.change(screen.getByRole('slider', { name: 'Terrain clustering' }), { target: { value: '70' } });
  fireEvent.change(screen.getByLabelText('Seed'), { target: { value: 'edited' } });
  fireEvent.change(tileCount(), { target: { value: '85' } });
  fireEvent.click(screen.getByRole('button', { name: 'Reset to current defaults' }));
  expect(tileCount().value).toBe('22');
  expect((screen.getByLabelText('Seed') as HTMLInputElement).value).toBe(defaults.seed);
  expect((screen.getByRole('slider', { name: 'Terrain clustering' }) as HTMLInputElement).value).toBe('0');
  expect(select().disabled).toBe(false);
  expect(props.onGenerate).not.toHaveBeenCalled();
  expect(props.onSelect).not.toHaveBeenCalled();
});

test('roster changes use the new safe region count and invalidate older previews', () => {
  const { props, update } = setup({ preview: makePreview() });
  const threePlayerDefaults = { ...defaults, tileCount: 29 };
  update({ playerCount: 3, defaultSettings: threePlayerDefaults });
  expect(screen.getByText('3 players · 29 regions')).toBeTruthy();
  expect(select().disabled).toBe(true);
  fireEvent.click(generate());
  expect(props.onGenerate).toHaveBeenCalledWith(threePlayerDefaults);
});

test('opening with a preview from an older roster starts from the current defaults', () => {
  const currentDefaults = { ...defaults, seed: 'three-player-default', tileCount: 29 };
  const previousSettings = { ...defaults, seed: 'old-custom-map', biomeWeights: { ...defaults.biomeWeights, forest: 240 } };
  const { props } = setup({
    defaultSettings: currentDefaults, playerCount: 3,
    initialSettings: previousSettings, preview: makePreview(previousSettings),
  });
  expect((screen.getByLabelText('Seed') as HTMLInputElement).value).toBe('three-player-default');
  expect((screen.getByRole('slider', { name: 'Forest weight' }) as HTMLInputElement).value).toBe('100');
  expect(select().disabled).toBe(true);
  fireEvent.click(generate());
  expect(props.onGenerate).toHaveBeenCalledWith(currentDefaults);
});

test('the host can restore automatic generation without selecting the preview', () => {
  const { props } = setup({ automatic: false, preview: makePreview(), selectedPreviewId: 'preview-1' });
  expect(screen.getByText('Selected for this room')).toBeTruthy();
  fireEvent.click(screen.getByRole('button', { name: 'Automatic at game start' }));
  expect(props.onAutomatic).toHaveBeenCalledTimes(1);
  expect(props.onSelect).not.toHaveBeenCalled();
  expect(props.onClose).not.toHaveBeenCalled();
});

test('pending generation and selection prevent duplicate actions', () => {
  const { props, update } = setup({ preview: makePreview(), loading: true });
  expect(select().disabled).toBe(true);
  const generating = screen.getByRole('button', { name: 'Generating…' }) as HTMLButtonElement;
  expect(generating.disabled).toBe(true);
  expect(randomMap().disabled).toBe(true);
  fireEvent.click(randomMap());
  fireEvent.click(generating);
  update({ loading: false, selecting: true });
  const selecting = screen.getByRole('button', { name: 'Selecting…' }) as HTMLButtonElement;
  expect(selecting.disabled).toBe(true);
  expect(randomMap().disabled).toBe(true);
  fireEvent.click(randomMap());
  fireEvent.click(selecting);
  expect((screen.getByRole('button', { name: /Automatic at game start/ }) as HTMLButtonElement).disabled).toBe(true);
  expect(props.onGenerate).not.toHaveBeenCalled();
  expect(props.onSelect).not.toHaveBeenCalled();
});

test('server errors remain visible and allow another preview attempt', () => {
  const { props } = setup({ error: 'The generator is unavailable.' });
  expect(screen.getByRole('alert').textContent).toBe('The generator is unavailable.');
  fireEvent.click(generate());
  expect(props.onGenerate).toHaveBeenCalledTimes(1);
});

test('Back and Escape close without selecting, and focus stays inside until closing', () => {
  const opener = document.createElement('button');
  document.body.appendChild(opener);
  opener.focus();
  const { props, unmount } = setup();
  const dialog = screen.getByRole('dialog', { name: 'Generated map' });
  const back = screen.getByRole('button', { name: 'Back' });
  expect(document.activeElement).toBe(dialog);
  fireEvent.keyDown(dialog, { key: 'Tab' });
  expect(document.activeElement).toBe(back);
  fireEvent.keyDown(back, { key: 'Tab', shiftKey: true });
  expect(document.activeElement).toBe(randomMap());
  fireEvent.keyDown(randomMap(), { key: 'Tab' });
  expect(document.activeElement).toBe(back);
  fireEvent.keyDown(screen.getByLabelText('Seed'), { key: 'Escape' });
  fireEvent.click(back);
  expect(props.onClose).toHaveBeenCalledTimes(2);
  expect(props.onSelect).not.toHaveBeenCalled();
  unmount();
  expect(document.activeElement).toBe(opener);
  opener.remove();
});

test('an open editor blocks the lobby shortcuts even when dispatched outside it', () => {
  const start = jest.fn();
  function LobbyShortcut() { useShortcut('p', start); return null; }
  const view = setup();
  render(<LobbyShortcut />);
  fireEvent.keyDown(window, { key: 'p' });
  fireEvent.keyDown(screen.getByLabelText('Seed'), { key: 'p' });
  expect(start).not.toHaveBeenCalled();
  view.unmount();
  fireEvent.keyDown(window, { key: 'p' });
  expect(start).toHaveBeenCalledTimes(1);
});
