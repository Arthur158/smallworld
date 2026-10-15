import React from 'react';
import { act, fireEvent, render, screen } from '@testing-library/react';
import TooltipProvider from './TooltipProvider';
import { useShortcut } from '../../hooks/useShortcut';

afterEach(() => { jest.useRealTimers(); });
test('hover hints appear after exactly 700 ms and cancel when leaving early', () => {
  jest.useFakeTimers();
  render(<TooltipProvider><button data-tooltip="Save the current game. Shift + S.">Save</button></TooltipProvider>);
  const button = screen.getByRole('button');
  fireEvent.pointerOver(button);
  act(() => { jest.advanceTimersByTime(699); });
  expect(screen.queryByRole('tooltip')).toBeNull();
  act(() => { jest.advanceTimersByTime(1); });
  expect(screen.getByRole('tooltip').textContent).toContain('Shift + S');
  expect(button.getAttribute('aria-describedby')).toBe('smallworld-control-hint');
  fireEvent.pointerOut(button);
  expect(screen.queryByRole('tooltip')).toBeNull();
  expect(button.hasAttribute('aria-describedby')).toBe(false);
  fireEvent.pointerOver(button); act(() => { jest.advanceTimersByTime(300); });
  fireEvent.pointerOut(button); act(() => { jest.advanceTimersByTime(1000); });
  expect(screen.queryByRole('tooltip')).toBeNull();
});
test('disabled controls explain their requirements, and Escape dismisses hints', () => {
  jest.useFakeTimers();
  render(<TooltipProvider><button disabled data-tooltip="Pause to step forward. ArrowRight.">Next</button></TooltipProvider>);
  fireEvent.pointerOver(screen.getByRole('button'));
  act(() => { jest.advanceTimersByTime(700); });
  expect(screen.getByRole('tooltip').textContent).toContain('Pause');
  fireEvent.keyDown(document, { key: 'Escape' });
  expect(screen.queryByRole('tooltip')).toBeNull();
});
function Bindings({ action }: { action: () => void }) {
  useShortcut('d', action); useShortcut('Shift+s', action); useShortcut('Space', action); useShortcut('+', action);
  return <><input aria-label="Chat" /><button>Focused action</button></>;
}
test('shortcuts respect typing, modifiers, repeat and modal boundaries, and use fresh callbacks', () => {
  const first = jest.fn(), latest = jest.fn();
  const view = render(<Bindings action={first} />);
  fireEvent.keyDown(window, { key: 'd' }); expect(first).toHaveBeenCalledTimes(1);
  view.rerender(<Bindings action={latest} />);
  fireEvent.keyDown(window, { key: 'S', shiftKey: true }); expect(latest).toHaveBeenCalledTimes(1);
  fireEvent.keyDown(screen.getByRole('textbox'), { key: 'd' });
  fireEvent.keyDown(window, { key: 'd', ctrlKey: true });
  fireEvent.keyDown(window, { key: 'd', metaKey: true });
  fireEvent.keyDown(window, { key: 'd', repeat: true });
  fireEvent.keyDown(window, { key: 'd', isComposing: true });
  fireEvent.keyDown(screen.getByRole('button'), { key: ' ' });
  const modal = document.createElement('div'); modal.setAttribute('role', 'dialog'); modal.setAttribute('aria-modal', 'true'); document.body.append(modal);
  fireEvent.keyDown(window, { key: 'd' }); modal.remove();
  expect(latest).toHaveBeenCalledTimes(1);
});

test('plus works on the main keyboard and numeric keypad', () => {
  const action = jest.fn(); render(<Bindings action={action} />);
  fireEvent.keyDown(window, { key: '+', shiftKey: true });
  fireEvent.keyDown(window, { key: '+', shiftKey: false, code: 'NumpadAdd' });
  expect(action).toHaveBeenCalledTimes(2);
});

test('hints follow updated controls and disappear when their anchor is removed', async () => {
  jest.useFakeTimers();
  const view = render(<TooltipProvider><button data-tooltip="Pause bots">Play</button></TooltipProvider>);
  fireEvent.pointerOver(screen.getByRole('button'));
  view.rerender(<TooltipProvider><button data-tooltip="Resume bots">Play</button></TooltipProvider>);
  act(() => { jest.advanceTimersByTime(700); });
  expect(screen.getByRole('tooltip').textContent).toBe('Resume bots');
  await act(async () => { view.rerender(<TooltipProvider><button data-tooltip="Pause bots">Play</button></TooltipProvider>); });
  expect(screen.getByRole('tooltip').textContent).toBe('Pause bots');
  await act(async () => { view.rerender(<TooltipProvider><p>Room closed</p></TooltipProvider>); });
  expect(screen.queryByRole('tooltip')).toBeNull();
});
