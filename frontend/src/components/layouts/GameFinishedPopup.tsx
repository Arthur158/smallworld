import React from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService';
import { setIsStackFromBank, setSelectedStack, setSelectedTile } from '../../redux/slices/applicationSlice';

export default function GameFinishedPopup() {
  const phase = useSelector((state: RootState) => state.application.phase);
  const players = useSelector((state: RootState) => state.application.players);
  const scores = useSelector((state: RootState) => state.application.scores);
  const dispatch = useDispatch();

  if (phase !== 'GameFinished') {
    return null;
  }

  // Combine players and scores
  const combined = players.map((player, index) => ({
    player: player.name, // Ensure player is a string
    score: scores[index],
  }));

  const resetSelections = () => {
    dispatch(setIsStackFromBank(false));
    dispatch(setSelectedTile(null));
    dispatch(setSelectedStack(null));
  };

  const handleLeaveGame = () => {
    sendMessageToBackend("leaveroom", {});
    resetSelections();
  };


  // Sort descending by score
  combined.sort((a, b) => b.score - a.score);

  // Helper to label ranks
  const rankLabel = (index: number) => {
    if (index === 0) return 'Winner';
    if (index === 1) return '2nd';
    if (index === 2) return '3rd';
    return `${index + 1}th`;
  };

  return (
    <div className="fixed inset-0 flex items-center justify-center bg-black bg-opacity-50 z-50">
      <div className="bg-white p-4 rounded shadow-md">
        <h2 className="text-xl font-bold mb-2">Game Results</h2>
        {combined.map((item, index) => (
          <div key={`${item.player}-${index}`} className="mb-1">
            {/* Add unique key */}
            <strong>{rankLabel(index)}:</strong> {item.player} - {item.score}
          </div>
        ))}
        <button onClick={handleLeaveGame} className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-3 rounded transition-colors">
          Leave Game <span className="text-sm text-gray-300"></span>
        </button>
      </div>
    </div>
  );
}
