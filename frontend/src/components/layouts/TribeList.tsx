import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { TribeEntry } from '../../types/Board';
import { sendMessageToBackend } from '../../services/backendService';
import { useWiki } from '../wiki/WikiContext';
import { useShortcut } from '../../hooks/useShortcut';

export default function TribeList() {
  const { inspect, isOpen, inspectMode } = useWiki();
  const tribes = useSelector((state: RootState) => state.application.availableTribes);
  const selectedStack = useSelector((state: RootState) => state.application.selectedStack);
  const fromBank = useSelector((state: RootState) => state.application.isStackFromBank);

  const { playerIndex, playerNumber, phase, isSpectating, inDisplayRoom } = useSelector((state: RootState) => state.application);
  const canPickWithShortcut = !isOpen && !inspectMode && !isSpectating && !inDisplayRoom &&
    playerIndex >= 0 && playerIndex === playerNumber && phase === 'TribeChoice' && !selectedStack;

  const handleSelectTribe = (i: number) => {
    if (selectedStack != null && fromBank) {
      sendMessageToBackend('entryaction', {pickIndex: i, stack: selectedStack})
    } else {
      sendMessageToBackend('tribepick', { pickIndex: i });
    }
  };

  // This component is mounted only while the market is visible. Number keys
  // choose a tribe; special pieces keep their explicit click-to-target action.
  const pickWithShortcut = (index: number) => {
    if (tribes[index]) sendMessageToBackend('tribepick', { pickIndex: index });
  };
  useShortcut('1', () => pickWithShortcut(0), canPickWithShortcut);
  useShortcut('2', () => pickWithShortcut(1), canPickWithShortcut);
  useShortcut('3', () => pickWithShortcut(2), canPickWithShortcut);
  useShortcut('4', () => pickWithShortcut(3), canPickWithShortcut);
  useShortcut('5', () => pickWithShortcut(4), canPickWithShortcut);
  useShortcut('6', () => pickWithShortcut(5), canPickWithShortcut);
  useShortcut('7', () => pickWithShortcut(6), canPickWithShortcut);
  useShortcut('8', () => pickWithShortcut(7), canPickWithShortcut);
  useShortcut('9', () => pickWithShortcut(8), canPickWithShortcut);

  const getTraitImagePath = (trait?: string) =>
    trait && trait.trim() !== ''
      ? `/traits/${trait}.png`
      : '/traits/Alchemist.png';

  const getRaceImagePath = (race?: string) =>
    race && race.trim() !== ''
      ? `/races/${race}.png`
      : '/races/Trolls.png';

  const renderTribeImages = (trait: string, race: string) => (
    <div className="relative flex items-center h-24">
      {/* Trait Image */}
      <img
        src={getTraitImagePath(trait)}
        alt={trait}
        className="h-full w-auto -mr-6"
      />
      {/* Race Image */}
      <img
        src={getRaceImagePath(race)}
        alt={race}
        className="h-full w-auto"
      />
    </div>
  );

  // A small helper to render the coin "stack"
  const renderCoinStack = (coinCount: number) => {
    if (coinCount <= 0) return null;

    // Limit the physical stack so it doesn't get too tall (e.g., max 6 coins shown)
    // but you can adjust to suit your design or remove if you want all coins displayed.
    const maxCoinsToDisplay = 6;
    const coinsToDisplay = Math.min(coinCount, maxCoinsToDisplay);

    return (
      <div className="relative ml-4 w-10 h-5 flex-shrink-0">
        {Array.from({ length: coinsToDisplay }).map((_, idx) => (
          <img
            key={idx}
            src="/stacks/Coin_No_Num.png"
            alt="Coin"
            className="absolute w-10 bottom-5"
            style={{ bottom: `${idx * 4}px` }}
          />
        ))}
        {/* White number on the top coin */}
        <span
          className="absolute text-white font-bold"
          style={{
            bottom: `${(coinsToDisplay - 1) * 4 + 8}px`,
            left: '50%',
            transform: 'translateX(-50%)',
          }}
        >
          {coinCount}
        </span>
      </div>
    );
  };

  return (
  <div className="p-3 border border-[#5F4B43] rounded bg-[#FDF5E6] h-auto min-h-[450px] flex flex-col">
      {tribes.length === 0 ? (
        <div className="italic text-center flex-1 flex items-center justify-center">
          Loading tribes...
        </div>
      ) : (
        <ul className="space-y-0.85 pr-2 flex-1">
          {tribes.map((tribe: TribeEntry, i: number) => (
            <li key={i}>
              <button
                type="button"
                aria-label={`${tribe.trait} ${tribe.race}`}
                data-tooltip={`${tribe.trait} ${tribe.race} · Click or Enter/Space: ${selectedStack && fromBank ? 'apply selected piece' : 'choose tribe'}${canPickWithShortcut && i < 9 ? ` · ${i + 1}: choose` : ''} · Ctrl/Cmd-click: rules`}
                onClick={(event) => {
                  if (inspect(event, { kind: 'tribe', tribe })) return;
                  handleSelectTribe(i);
                }}
                onKeyDown={(event) => {
                  if (event.key === 'Enter' || event.key === ' ') {
                    if (inspect(event, { kind: 'tribe', tribe })) return;
                  }
                }}
                className="flex items-center z-10 transition-transform hover:scale-110 w-full p-0 border-none bg-transparent hover:z-30"
              >
                {/* Left side: Trait & Race images */}
                {renderTribeImages(tribe.trait, tribe.race)}

                {/* Right side: Coin stack */}
                {renderCoinStack(tribe.coinCount)}
              </button>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
