import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { Provider } from 'react-redux';
import { configureStore } from '@reduxjs/toolkit';
import TurnInfoBlock from './TurnInfoBlock';
import TurnInfoBlockSpectate from './TurnInfoBlockSpectate';
import TurnInfoBlockDisplay from './TurnInfoBlockDisplay';
import { sendMessageToBackend } from '../../services/backendService';
import applicationReducer from '../../redux/slices/applicationSlice';
import { ApplicationState } from '../../types/redux';
import { useWiki } from '../wiki/WikiContext';

jest.mock('../../services/backendService', () => ({ sendMessageToBackend: jest.fn() }));
jest.mock('../wiki/WikiContext', () => ({ useWiki: jest.fn() }));
const wiki = useWiki as jest.Mock;

function show(component = <TurnInfoBlock />, overrides: Partial<ApplicationState> = {}) {
  const state: ApplicationState = {
    ...applicationReducer(undefined, { type: 'test/initialState' }),
    phase: 'DeclineChoice', playerIndex: 0, playerNumber: 0, isSpectating: false, inDisplayRoom: false,
    players: ['Alice', 'Bob'].map(name => ({ name, activeTribe: { race: 'Humans', trait: 'Stout' }, passiveTribes: [], pieceStacks: [] })),
    ...overrides,
  };
  const store = configureStore({ reducer: { application: applicationReducer }, preloadedState: { application: state } });
  return render(<Provider store={store}>{component}</Provider>);
}

beforeEach(() => {
  jest.clearAllMocks();
  wiki.mockReturnValue({ inspectMode: false, isOpen: false });
});

test('each turn shortcut invokes the same command as its enabled button', () => {
  show();
  const actions = [
    ['d', false, /Decline/, 'decline'], ['r', false, /Redeploy/, 'startredeployment'],
    ['e', false, /End Turn/, 'finishturn'], ['S', true, /Save Game/, 'savegame'],
    ['R', true, /Rollback/, 'rollback'], ['L', true, /Leave Game/, 'leaveroom'],
  ] as const;
  for (const [key, shiftKey, name, command] of actions) {
    fireEvent.keyDown(window, { key, shiftKey });
    expect(sendMessageToBackend).toHaveBeenLastCalledWith(command, {});
    fireEvent.click(screen.getByRole('button', { name }));
    expect(sendMessageToBackend).toHaveBeenLastCalledWith(command, {});
  }
  expect(sendMessageToBackend).toHaveBeenCalledTimes(12);
  for (const button of screen.getAllByRole('button')) {
    expect(button.getAttribute('data-tooltip')).toBeTruthy();
    expect(button.getAttribute('aria-keyshortcuts')).toBeTruthy();
  }
});

test.each([
  ['another player turn', { playerNumber: 1 }],
  ['spectator', { isSpectating: true }],
  ['display', { inDisplayRoom: true }],
  ['game finished', { phase: 'GameFinished' }],
] as [string, Partial<ApplicationState>][])('turn commands cannot fire during %s', (_, overrides) => {
  show(<TurnInfoBlock />, overrides);
  for (const key of ['d', 'r', 'e']) fireEvent.keyDown(window, { key });
  fireEvent.keyDown(window, { key: 'R', shiftKey: true });
  for (const name of [/Decline/, /Redeploy/, /End Turn/, /Rollback/]) {
    const button = screen.getByRole('button', { name }) as HTMLButtonElement;
    expect(button.disabled).toBe(true);
    fireEvent.click(button);
  }
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test.each([
  ['TribeSelection', []], ['DeclineChoice', ['decline', 'startredeployment', 'finishturn']],
  ['TileAbandonment', ['startredeployment', 'finishturn']], ['Conquest', ['startredeployment', 'finishturn']],
  ['Redeployment', ['decline', 'finishturn']],
] as [string, string[]][])('phase %s only enables commands supported by the server phase guard', (phase, expected) => {
  show(<TurnInfoBlock />, { phase });
  for (const key of ['d', 'r', 'e']) fireEvent.keyDown(window, { key });
  expect((sendMessageToBackend as jest.Mock).mock.calls.map(call => call[0])).toEqual(expected);
});

test('decline requires an active race', () => {
  show(<TurnInfoBlock />, { players: [{ name: 'Alice', activeTribe: null, passiveTribes: [], pieceStacks: [] }] });
  fireEvent.keyDown(window, { key: 'd' });
  expect((screen.getByRole('button', { name: /Decline/ }) as HTMLButtonElement).disabled).toBe(true);
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test('inspection blocks gameplay while keeping save and leave available', () => {
  wiki.mockReturnValue({ inspectMode: true, isOpen: false });
  show();
  for (const key of ['d', 'r', 'e']) fireEvent.keyDown(window, { key });
  fireEvent.keyDown(window, { key: 'R', shiftKey: true });
  fireEvent.click(screen.getByRole('button', { name: /Decline/ }));
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  fireEvent.keyDown(window, { key: 'S', shiftKey: true });
  fireEvent.keyDown(window, { key: 'L', shiftKey: true });
  expect((sendMessageToBackend as jest.Mock).mock.calls.map(call => call[0])).toEqual(['savegame', 'leaveroom']);
});

test('open wiki blocks buttons and shortcuts', () => {
  wiki.mockReturnValue({ inspectMode: false, isOpen: true });
  show();
  for (const key of ['d', 'r', 'e']) fireEvent.keyDown(window, { key });
  for (const key of ['S', 'R', 'L']) fireEvent.keyDown(window, { key, shiftKey: true });
  for (const button of screen.getAllByRole('button')) {
    expect((button as HTMLButtonElement).disabled).toBe(true);
    fireEvent.click(button);
  }
  expect(sendMessageToBackend).not.toHaveBeenCalled();
});

test.each([
  ['spectator', <TurnInfoBlockSpectate />, 'leaveroom'],
  ['display', <TurnInfoBlockDisplay />, 'leavedisplayroom'],
] as const)('%s exposes only its leave action', (_, component, command) => {
  show(component);
  for (const key of ['d', 'r', 'e']) fireEvent.keyDown(window, { key });
  for (const key of ['S', 'R']) fireEvent.keyDown(window, { key, shiftKey: true });
  expect(sendMessageToBackend).not.toHaveBeenCalled();
  expect(screen.getAllByRole('button')).toHaveLength(1);
  fireEvent.keyDown(window, { key: 'L', shiftKey: true });
  expect(sendMessageToBackend).toHaveBeenCalledTimes(1);
  expect(sendMessageToBackend).toHaveBeenCalledWith(command, {});
});
