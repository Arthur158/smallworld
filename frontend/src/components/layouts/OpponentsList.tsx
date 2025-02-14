import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { Player, Tribe, PieceStack } from '../../types/Board';

/**
 * OpponentsList:
 * Renders all other players except the current user (playerIndex).
 * The active player (activeIndex) is displayed with a distinct style.
 * Shows each opponent's:
 *   - Name
 *   - Active tribe (images stuck together, if any)
 *   - Passive tribes (smaller/grayed-out images, if any)
 *   - A brown separator line if they have piece stacks
 *   - Their piece stacks (slightly bigger icons with amounts/types)
 * The list is scrollable (overflow-y-auto).
 */

// Helper: get PNG path for traits / races
const getTraitImagePath = (trait?: string) => {
  return trait && trait.trim() !== '' ? `/traits/${trait}.png` : '';
};

const getRaceImagePath = (race?: string) => {
  return race && race.trim() !== '' ? `/races/${race}.png` : '';
};

// Render the stuck-together images for a single tribe
const renderTribeImages = (tribe: Tribe, customClasses = '') => {
  return (
    <div className={`relative flex items-center ${customClasses}`}>
      <img
        src={getTraitImagePath(tribe.trait)}
        alt={tribe.trait}
        className="h-full w-auto -mr-1 z-20"
      />
      <img
        src={getRaceImagePath(tribe.race)}
        alt={tribe.race}
        className="h-full w-auto z-10"
      />
    </div>
  );
};

// Render a player's piece stacks in the same style as PlayerInfo
const renderPieceStacks = (pieceStacks: PieceStack[]) => {
  return (
    <div className="flex space-x-2 mt-2 relative z-10">
      {pieceStacks.map((stack, index) => {
        const imageSrc = `/stacks/${stack.type}.png`;
        return (
          <div
            key={index}
            className="relative"
            style={{
              width: 45,
              height: 45,
              cursor: 'pointer',
            }}
            onClick={() => console.log(`Stack ${stack.type} clicked`)}
          >
            <img
              src={imageSrc}
              onError={(e) => {
                (e.currentTarget as HTMLImageElement).style.display = 'none';
              }}
              className="absolute w-full h-full top-0 left-0"
            />
            {/* Amount */}
            <span className="absolute top-2 right-2 text-white text-xs font-bold text-shadow">
              {stack.amount}
            </span>
            {/* Stack Type */}
            <span className="absolute left-1/2 top-1/2 transform -translate-x-1/2 -translate-y-1/2 text-white text-xs font-bold text-center text-shadow">
              {stack.type}
            </span>
          </div>
        );
      })}
    </div>
  );
};

// Render a single opponent (or active player), reusing the same style as PlayerInfo
const renderOneOpponent = (opponent: Player, isActive: boolean) => {
  return (
    <div
      key={opponent.name}
      className={`p-3 mb-4 border rounded ${
        isActive
          ? 'border-[#8B4513] bg-[#FAEBD7]' // Distinct style for active player
          : 'border-[#5F4B32] bg-[#FAF0E6]'
      }`}
    >
      {/* Name */}
      <p className="text-lg font-bold">{opponent.name}</p>

      {/* Active Tribe */}
      {opponent.activeTribe && (
        <div className="mt-3 flex items-center justify-center" style={{ height: '6rem' }}>
          {renderTribeImages(opponent.activeTribe, 'h-full')}
        </div>
      )}

      {/* Passive Tribes */}
      {opponent.passiveTribes.length > 0 && (
        <div className="flex flex-wrap items-center justify-center mt-2 gap-3">
          {opponent.passiveTribes.map((tribe, i) => (
            <div
              key={i}
              className="opacity-60"
              style={{ height: '3rem', filter: 'grayscale(50%)' }}
            >
              {renderTribeImages(tribe, 'h-full')}
            </div>
          ))}
        </div>
      )}

      {/* Brown separator line if they have piece stacks */}
      {opponent.pieceStacks.length > 0 && (
        <div className="my-4 border-t-4 border-[#8B4513] w-full"></div>
      )}

      {/* Piece stacks */}
      {renderPieceStacks(opponent.pieceStacks)}
    </div>
  );
};

export default function OpponentsList() {
  const allPlayers = useSelector((state: RootState) => state.application.players);
  const playerIndex = useSelector((state: RootState) => state.application.playerIndex);
  const activeIndex = useSelector((state: RootState) => state.application.playerNumber);

  if (!allPlayers || allPlayers.length === 0) {
    return (
      <div className="flex flex-col w-full h-full overflow-hidden border border-[#5F4B32] rounded bg-[#FDF5E6] p-4">
        <h3 className="text-lg font-bold underline mb-2">Opponents</h3>
        <p>Aucun joueur trouvé</p>
      </div>
    );
  }

  // The current user
  const currentUser = allPlayers[playerIndex];
  // The active player (could be the same as currentUser, or different)
  const activePlayer = allPlayers[activeIndex];

  // Filter out the current user from the list
  const opponents = allPlayers.filter((_, i) => i !== playerIndex);

  return (
    <div className="flex flex-col w-full h-full overflow-hidden border border-[#5F4B32] rounded bg-[#FDF5E6] p-4">
      <h3 className="text-lg font-bold underline mb-2">Opponents</h3>

      {/* Scrollable container */}
      <div className="flex-1 overflow-y-auto pr-2">
        {/* We map through all "opponents" (anyone not the current user) */}
        {opponents.map((opp, index) => {
          const isThisActive = activePlayer && activePlayer.name === opp.name;
          return renderOneOpponent(opp, isThisActive);
        })}
      </div>
    </div>
  );
}
