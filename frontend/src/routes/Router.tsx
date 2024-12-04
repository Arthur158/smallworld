import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { useSelector } from 'react-redux';
import { RootState } from '../redux/store';
import HomePage from '../pages/HomePage';
import GamePage from './GamePage'; // Import the GamePage component

export default function Router(): JSX.Element {
  const error = useSelector((state: RootState) => state.application.error);

  return (
    <>
      {error && (
        <div className="bg-red-500 text-white text-center py-2">
          {error}
        </div>
      )}
      <Routes>
        <Route path="/" element={<HomePage />} />
        <Route path="/game" element={<GamePage />} /> {/* New Route */}
      </Routes>
    </>
  );
}
