import React from 'react';
import { useNavigate } from 'react-router-dom';

export default function HomePage() {
  const navigate = useNavigate();

  const handleStartGame = () => {
    navigate('/game');
  };

  return (
    <div className="flex justify-center items-center min-h-screen pt-5">
      <button type="button" onClick={handleStartGame} className="btn btn-primary">
        Start Game
      </button>
    </div>
  );
}
