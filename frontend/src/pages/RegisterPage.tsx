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

  const { isAuthenticated, error } = useSelector((state: RootState) => state.application);

  const [isLogin, setIsLogin] = useState(true);
  const [username, setUsername] = useState('');
  const [password, setPassword] = useState('');

  useEffect(() => {
    connectWebSocket();
  }, []);

  const handleLoginSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    sendMessageToBackend('login', { username, password });
  };

  const handleRegisterSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    sendMessageToBackend('register', { username, password });
  };

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
    <div
      className="w-screen h-screen bg-cover bg-center font-serif text-[#5F4B32]"
      style={{ 
        backgroundImage: "url('/background2.png')",
    backgroundSize: "100% 100%", // Ensures the entire image is displayed
    backgroundPosition: "center", // Centers the image
    backgroundRepeat: "no-repeat", // Prevents tiling
      }}
    >
      <div className="max-w-md mx-auto mt-10 p-5 bg-[#FDF5E6] border border-[#5F4B32] rounded-lg shadow-lg">
        <div className="flex justify-between mb-6">
          <button
            type="button"
            onClick={() => toggleForm(true)}
            className={`w-1/2 py-2 text-center border-r border-[#5F4B32] ${
              isLogin ? 'bg-[#8B4513] text-white' : 'bg-[#FAF0E6]'
            }`}
          >
            Login
          </button>
          <button
            type="button"
            onClick={() => toggleForm(false)}
            className={`w-1/2 py-2 text-center ${
              !isLogin ? 'bg-[#8B4513] text-white' : 'bg-[#FAF0E6]'
            }`}
          >
            Register
          </button>
        </div>

        {isLogin ? (
          <form onSubmit={handleLoginSubmit}>
            <div className="mb-4">
              <label className="block font-bold">Username</label>
              <input
                type="text"
                className="w-full px-3 py-2 border border-[#5F4B32] rounded-md focus:outline-none"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="mb-4">
              <label className="block font-bold">Password</label>
              <input
                type="password"
                className="w-full px-3 py-2 border border-[#5F4B32] rounded-md focus:outline-none"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            {error && <div className="text-red-600 mb-2 font-semibold">{error}</div>}

            <button
              type="submit"
              className="w-full bg-[#8B4513] text-white py-2 rounded-md hover:bg-[#A0522D]"
            >
              Log In
            </button>
          </form>
        ) : (
          <form onSubmit={handleRegisterSubmit}>
            <div className="mb-4">
              <label className="block font-bold">Username</label>
              <input
                type="text"
                className="w-full px-3 py-2 border border-[#5F4B32] rounded-md focus:outline-none"
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
              />
            </div>
            <div className="mb-4">
              <label className="block font-bold">Password</label>
              <input
                type="password"
                className="w-full px-3 py-2 border border-[#5F4B32] rounded-md focus:outline-none"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
              />
            </div>

            {error && <div className="text-red-600 mb-2 font-semibold">{error}</div>}

            <button
              type="submit"
              className="w-full bg-[#8B4513] text-white py-2 rounded-md hover:bg-[#A0522D]"
            >
              Register
            </button>
          </form>
        )}
      </div>
    </div>
  );
}
