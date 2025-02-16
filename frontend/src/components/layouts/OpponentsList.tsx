import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { Player, Tribe, PieceStack } from '../../types/Board';

const getTraitImagePath = (trait?: string) => {
  return trait && trait.trim() !== '' ? `/traits/${trait}.png` : '';
};

const getRaceImagePath = (race?: string) => {
  return race && race.trim() !== '' ? `/races/${race}.png` : '';
};

const renderTribeImages = (tribe: Tribe, customClasses = '') => {
  return (
    <div className={`relative flex items-center ${customClasses} h-24 xl:h-20 lg:h-20 md:h-16 sm:h-14 overflow-hidden`}>
      <img
        src={getTraitImagePath(tribe.trait)}
        alt={tribe.trait}
        className="max-h-full w-auto -mr-6 z-20"
      />
      <img
        src={getRaceImagePath(tribe.race)}
        alt={tribe.race}
        className="max-h-full w-auto z-10"
      />
    </div>
  );
};

const renderPassiveTribeImages = (tribe: Tribe, customClasses = '') => {
  return (
    <div className={`relative flex items-center ${customClasses} h-16 xl:h-14 lg:h-14 md:h-12 sm:h-10 overflow-hidden`}>
      <img
        src={getTraitImagePath(tribe.trait)}
        alt={tribe.trait}
        className="max-h-full w-auto -mr-3 z-20"
      />
      <img
        src={getRaceImagePath(tribe.race)}
        alt={tribe.race}
        className="max-h-full w-auto z-10"
      />
    </div>
  );
};

const renderPieceStacks = (pieceStacks: PieceStack[]) => {
  return (
    <div className="flex flex-wrap space-x-2 mt-2 relative z-10">
      {pieceStacks.map((stack, index) => {
        const imageSrc = `/stacks/${stack.type}.png`;
        return (
          <div
            key={index}
            className="relative m-1"
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
            <span className="absolute top-2 right-2 text-white text-xs font-bold text-shadow">
              {stack.amount}
            </span>
          </div>
        );
      })}
    </div>
  );
};

const renderOneOpponent = (opponent: Player, isActive: boolean) => {
  return (
    <div
      key={opponent.name}
      className={`p-3 mb-4 border rounded ${
        isActive
          ? 'border-[#8B4513] bg-[#FAEBD7]' 
          : 'border-[#5F4B32] bg-[#FAF0E6]'
      }`}
    >
      <p className="text-lg font-bold">{opponent.name}</p>

      {opponent.activeTribe && (
        <div className="mt-3 flex items-center justify-center">
          {renderTribeImages(opponent.activeTribe)}
        </div>
      )}

      {opponent.passiveTribes.length > 0 && (
        <div className="flex flex-wrap items-center justify-center mt-2 gap-3">
          {opponent.passiveTribes.map((tribe, i) => (
            <div
              key={i}
              className="opacity-60"
              style={{ filter: 'grayscale(50%)' }}
            >
              {renderPassiveTribeImages(tribe)}
            </div>
          ))}
        </div>
      )}

      {opponent.pieceStacks.length > 0 && (
        <div className="my-4 border-t-4 border-[#8B4513] w-full"></div>
      )}
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
        <p>No player found</p>
      </div>
    );
  }

  const currentUser = allPlayers[playerIndex];
  const activePlayer = allPlayers[activeIndex];
  const opponents = allPlayers.filter((_, i) => i !== playerIndex);

  return (
    <div className="flex flex-col w-full h-full overflow-hidden border border-[#5F4B32] rounded bg-[#FDF5E6] p-4">
      <div className="flex-1 overflow-y-auto pr-2">
        {opponents.map((opp) => {
          const isThisActive = activePlayer && activePlayer.name === opp.name;
          return renderOneOpponent(opp, isThisActive);
        })}
      </div>
    </div>
  );
}
