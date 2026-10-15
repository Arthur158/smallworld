import React from 'react';
import '@testing-library/jest-dom';
import { fireEvent, render, screen } from '@testing-library/react';
import AppErrorBoundary from './AppErrorBoundary';

test('offers recovery when a screen throws instead of leaving a blank root', () => {
  const recover = jest.fn();
  const log = jest.spyOn(console, 'error').mockImplementation(() => {});
  const BrokenScreen = (): JSX.Element => { throw new Error('stale game state'); };
  try {
    render(<AppErrorBoundary onRecover={recover}><BrokenScreen /></AppErrorBoundary>);
    expect(screen.getByRole('alert')).toHaveTextContent('The page couldn’t be displayed');
    fireEvent.click(screen.getByRole('button', { name: 'Reopen Smallworld' }));
    expect(recover).toHaveBeenCalledTimes(1);
  } finally {
    log.mockRestore();
  }
});

test('renders a healthy screen normally', () => {
  render(<AppErrorBoundary onRecover={jest.fn()}><h1>Your game</h1></AppErrorBoundary>);
  expect(screen.getByRole('heading', { name: 'Your game' })).toBeInTheDocument();
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
});
