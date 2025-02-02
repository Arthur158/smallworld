// src/components/Router.tsx (or src/Router.tsx)
import { Routes, Route, Navigate, useNavigate } from 'react-router-dom';
import { useSelector } from 'react-redux';
import React, { useEffect} from 'react';
import { RootState } from '../redux/store';

import LoginRegisterPage from '../pages/RegisterPage';
import LobbyPage from '../pages/LobbyPage';
import GamePage from '../routes/GamePage';

export default function AppRouter(): JSX.Element {
  const { isAuthenticated, gameStarted } = useSelector((state: RootState) => state.application);
  const navigate = useNavigate()

  // Detect changes in isAuthenticated and redirect if false
  useEffect(() => {
    if (gameStarted) {
      navigate("/game", { replace: true})
    } else if (isAuthenticated) {
      navigate("/lobby", { replace: true });
    } else {
      navigate("/", { replace: true });
    }
  }, [isAuthenticated, navigate]);

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
