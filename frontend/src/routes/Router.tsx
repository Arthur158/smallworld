// src/components/Router.tsx (or src/Router.tsx)
import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { useSelector } from 'react-redux';
import { RootState } from '../redux/store';

import LoginRegisterPage from '../pages/RegisterPage';
import LobbyPage from '../pages/LobbyPage';
import GamePage from '../routes/GamePage';

export default function AppRouter(): JSX.Element {
  const { isAuthenticated } = useSelector((state: RootState) => state.application);

  return (
    <Routes>
      {/* Default route: show login/register */}
      <Route path="/" element={<LoginRegisterPage />} />

      {/* The lobby (old homepage). Optionally guard it by checking isAuthenticated. */}
      <Route
        path="/lobby"
        element={isAuthenticated ? <LobbyPage /> : <Navigate to="/" />}
      />

      {/* The game page. Also optionally guard. */}
      <Route
        path="/game"
        element={isAuthenticated ? <GamePage /> : <Navigate to="/" />}
      />

      {/* Catch-all route could redirect to / if path not found */}
      <Route path="*" element={<Navigate to="/" />} />
    </Routes>
  );
}
