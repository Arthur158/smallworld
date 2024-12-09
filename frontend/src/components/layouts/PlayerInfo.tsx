import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { Player, PieceStack } from '../../types/Board';

export default function PlayerInfo() {
  const player: Player = useSelector((state: RootState) => state.application.player);

  const allPlayers = [player, ...useSelector((state: RootState) => state.application.opponents)];
  const activePlayers = allPlayers.filter((p) => p.isPlaying);
  if (activePlayers.length !== 1) {
    console.warn('There should be exactly one active player at a time.');
  }
  const activePlayer = activePlayers[0];

  const isPlayerActive = player.isPlaying;

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
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6]">
      <h3 className="text-lg font-bold">{player.name}</h3>
      {player.activeTribe && (
        <p className="text-base mt-1 italic">
          {player.activeTribe.trait} {player.activeTribe.race}
        </p>
      )}
      {player.passiveTribes.length > 0 && (
        <div className="text-sm mt-1 opacity-80">
          {player.passiveTribes.map((tribe, i) => (
            <p key={i} className="italic">
              {tribe.trait} {tribe.race}
            </p>
          ))}
        </div>
      )}
      {renderPieceStacks(player.pieceStacks, isPlayerActive)}
    </div>
  );
}
