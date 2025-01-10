// src/pages/LoginRegisterPage.tsx
import React, { useState, useEffect } from 'react';
import { useDispatch, useSelector } from 'react-redux';
import { useNavigate } from 'react-router-dom';
import { connectWebSocket, sendMessageToBackend } from '../services/backendService';
import { clearError, setName } from '../redux/slices/applicationSlice';
import { AppDispatch, RootState } from '../redux/store';

export default function LoginRegisterPage() {
  const dispatch: AppDispatch = useDispatch();
  const navigate = useNavigate();

  // Pull any state you need from Redux
  const { isAuthenticated, error } = useSelector((state: RootState) => state.application);

  // Track form toggle: true => “Login” form, false => “Register” form
  const [isLogin, setIsLogin] = useState(true);

  // Input fields
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  useEffect(() => {
    connectWebSocket();
  }, []);

  // LOGIN FORM SUBMIT
  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    sendMessageToBackend('login', {username: username, password: password});

  };

  // REGISTER FORM SUBMIT
  const handleRegisterSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    sendMessageToBackend('register', {username: username, password: password});
  };

  // If already authenticated, go straight to lobby
  useEffect(() => {
    if (isAuthenticated) {
      navigate('/lobby');
    }
  }, [isAuthenticated, navigate]);

  const toggleForm = (toLogin: boolean) => {
    setIsLogin(toLogin);
    dispatch(clearError());
  };

  return (
    <div className="max-w-md mx-auto mt-10 p-5 bg-white rounded-lg shadow-md">
      <div className="flex justify-between mb-6">
        <button
          type="button"
          onClick={() => toggleForm(true)}
          className={`w-1/2 py-2 text-center ${
            isLogin ? 'bg-blue-600 text-white' : 'bg-gray-200'
          }`}
        >
          Login
        </button>
        <button
          type="button"
          onClick={() => toggleForm(false)}
          className={`w-1/2 py-2 text-center ${
            !isLogin ? 'bg-blue-600 text-white' : 'bg-gray-200'
          }`}
        >
          Register
        </button>
      </div>

      {isLogin ? (
        <form onSubmit={handleLoginSubmit}>
          <div className="mb-4">
            <label className="block text-gray-700">Username</label>
            <input
              type="text"
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-600"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
          </div>
          <div className="mb-4">
            <label className="block text-gray-700">Password</label>
            <input
              type="password"
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-600"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          {error && <div className="text-red-500 mb-2">{error}</div>}

          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-2 rounded-md hover:bg-blue-700"
          >
            Log In
          </button>
        </form>
      ) : (
        <form onSubmit={handleRegisterSubmit}>
          <div className="mb-4">
            <label className="block text-gray-700">Username</label>
            <input
              type="text"
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-600"
              value={username}
              onChange={(e) => setUsername(e.target.value)}
              required
            />
          </div>
          <div className="mb-4">
            <label className="block text-gray-700">Password</label>
            <input
              type="password"
              className="w-full px-3 py-2 border rounded-md focus:outline-none focus:ring-2 focus:ring-blue-600"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              required
            />
          </div>

          {error && <div className="text-red-500 mb-2">{error}</div>}

          <button
            type="submit"
            className="w-full bg-blue-600 text-white py-2 rounded-md hover:bg-blue-700"
          >
            Register
          </button>
        </form>
      )}
    </div>
  );
}
