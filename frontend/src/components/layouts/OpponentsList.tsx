import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { Player, PieceStack } from '../../types/Board';

export default function OpponentsList() {
  const allPlayers: Player[] = useSelector((state: RootState) => state.application.players);
  const playerIndex: number = useSelector((state: RootState) => state.application.playerIndex);
  const activeIndex: number = useSelector((state: RootState) => state.application.playerNumber);

  const player = allPlayers[playerIndex]

  const activePlayer = allPlayers[activeIndex]

  const inactiveOpponents = allPlayers.filter((_, index) => {
    return index !== playerIndex && index !== activeIndex;
  });

  const baseSize = 45;
  const renderPieceStacks = (pieceStacks: PieceStack[], isActive: boolean) => {
    return (
      <div className="flex space-x-2 mt-2">
        {pieceStacks.map((stack, index) => {
          const imageSrc = `/stacks/${stack.type}.png`;
          const isClickable = isActive;
          return (
            <div
              key={index}
              className="relative"
              style={{
                width: baseSize,
                height: baseSize,
                cursor: isClickable ? 'pointer' : 'default',
                opacity: isActive ? 1 : 0.5,
              }}
              onClick={isClickable ? () => console.log(`Stack ${stack.type} clicked`) : undefined}
            >
              <div
                style={{
                  position: 'absolute',
                  width: baseSize,
                  height: baseSize,
                  backgroundColor: 'blue',
                  border: '1px solid black',
                }}
              />
              <img
                src={imageSrc}
                onError={(e) => {
                  (e.currentTarget as HTMLImageElement).style.display = 'none';
                }}
                style={{
                  position: 'absolute',
                  width: baseSize,
                  height: baseSize,
                  top: 0,
                  left: 0,
                }}
              />
              <span
                style={{
                  position: 'absolute',
                  top: 2,
                  right: 2,
                  color: 'white',
                  fontSize: '10px',
                  fontWeight: 'bold',
                  textShadow: '1px 1px 2px black',
                }}
              >
                {stack.amount}
              </span>
              <span
                style={{
                  position: 'absolute',
                  left: '50%',
                  top: '50%',
                  transform: 'translate(-50%, -50%)',
                  color: 'white',
                  fontSize: '8px',
                  fontWeight: 'bold',
                  textAlign: 'center',
                  textShadow: '1px 1px 2px black',
                }}
              >
                {stack.type}
              </span>
            </div>
          );
        })}
      </div>
    );
  };

  return (
    <div className="w-full overflow-x-auto border border-[#5F4B32] rounded bg-[#FDF5E6] p-4 mt-4">
      <h3 className="text-lg font-bold underline mb-2">Adversaires</h3>
      <div className="flex space-x-4 py-2">
        {/* Render active opponent if activeIndex is not the current player */}
        {activeIndex !== playerIndex && activePlayer && (
          <div
            key={activePlayer.name}
            className="flex-shrink-0 p-3 border rounded font-bold border-[#8B4513] bg-[#FAEBD7]"
          >
            <p className="text-lg">{activePlayer.name}</p>
            {activePlayer.activeTribe && (
              <p className="text-base mt-1 italic">
                {activePlayer.activeTribe.trait} {activePlayer.activeTribe.race}
              </p>
            )}
            {activePlayer.passiveTribes.length > 0 && (
              <div className="text-sm mt-1 opacity-80 italic">
                {activePlayer.passiveTribes.map((tribe, i) => (
                  <p key={i}>
                    {tribe.trait} {tribe.race}
                  </p>
                ))}
              </div>
            )}
          </div>
        )}

        {/* Render all inactive opponents */}
        {inactiveOpponents.map((opponent) => (
          <div
            key={opponent.name}
            className="flex-shrink-0 p-3 border rounded border-[#5F4B32] bg-[#FAF0E6]"
          >
            <p className="text-lg">{opponent.name}</p>
            {opponent.activeTribe && (
              <p className="text-base mt-1 italic">
                {opponent.activeTribe.trait} {opponent.activeTribe.race}
              </p>
            )}
            {opponent.passiveTribes.length > 0 && (
              <div className="text-sm mt-1 opacity-80 italic">
                {opponent.passiveTribes.map((tribe, i) => (
                  <p key={i}>
                    {tribe.trait} {tribe.race}
                  </p>
                ))}
              </div>
            )}
          </div>
        ))}
      </div>

      {/* Render active player's piece stacks */}
      {activePlayer &&
        activePlayer.name !== player.name &&
        activePlayer.pieceStacks && (
          <div className="flex space-x-4 mt-4">
            {renderPieceStacks(activePlayer.pieceStacks, activeIndex === playerIndex)}
          </div>
        )}
    </div>
  );
}
