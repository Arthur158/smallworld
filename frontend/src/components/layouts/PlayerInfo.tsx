import React, { useEffect, useRef } from 'react';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../redux/store';
import { Player, PieceStack, Tribe } from '../../types/Board';
import { setIsStackFromBank, setSelectedStack, setSelectedTile } from '../../redux/slices/applicationSlice';
import { sendMessageToBackend } from '../../services/backendService';

export default function PlayerInfo() {
  const dispatch = useDispatch();
  const allPlayers = useSelector((state: RootState) => state.application.players);
  const playerIndex = useSelector((state: RootState) => state.application.playerIndex);
  const playerNumber = useSelector((state: RootState) => state.application.playerNumber);
  const isStackFromBank = useSelector((state: RootState) => state.application.isStackFromBank);
  const selectedStack = useSelector((state: RootState) => state.application.selectedStack);
  const selectedTile = useSelector((state: RootState) => state.application.selectedTile);
  const phase = useSelector((state: RootState) => state.application.phase);
  const hasExecutedRef = useRef(false);

  useEffect(() => {
    const handleKeyPress = (event: KeyboardEvent) => {
      if (event.key === 'f') {
        dispatch(setIsStackFromBank(false));
        dispatch(setSelectedTile(null));
        dispatch(setSelectedStack(null));
      }
      if (event.key === 'a' && (phase == "DeclineChoice" || "HandleAbandonment") && selectedTile != null) {
        sendMessageToBackend('abandonment', { tileId: selectedTile.toString() });
      }
      if (event.key === 'a' && phase == "Redeployment" && selectedTile != null && selectedStack != null) {
        sendMessageToBackend('deploymentout', {
          tileId: selectedTile.toString(),
          stackType: selectedStack.toString(),
        });
      }
    };

    window.addEventListener('keydown', handleKeyPress);
    return () => {
      window.removeEventListener('keydown', handleKeyPress);
    };
  }, [dispatch, selectedTile, selectedStack]);

  useEffect(() => {
    if (!allPlayers || allPlayers.length === 0) return;
    const player = allPlayers[playerIndex];
    if (!player) return;

    // Run only once when the phase changes to "DeclineChoice"
    if (phase === "DeclineChoice" && !hasExecutedRef.current) {
      hasExecutedRef.current = true; // Mark as executed

      if (playerIndex === playerNumber && player.pieceStacks.length !== 0) {
        dispatch(setIsStackFromBank(true));
        dispatch(setSelectedTile(null));
        dispatch(setSelectedStack(player.pieceStacks[0].type));
      }
    } else if (phase !== "DeclineChoice") {
      hasExecutedRef.current = false; // Reset when phase changes away
    }
  }, [dispatch, selectedTile, selectedStack, allPlayers, playerIndex, playerNumber, phase]);

  const player = allPlayers?.[playerIndex] || null;

  // Same logic as in TribeList
  const getTraitImagePath = (trait?: string) => {
    return trait && trait.trim() !== ''
      ? `/traits/${trait}.png`
      : '';
  };

  const getRaceImagePath = (race?: string) => {
    return race && race.trim() !== ''
      ? `/races/${race}.png`
      : '';
  };
  const handlePlayerClick = () => {
    if ((phase === 'TileAbandonment' || phase === 'DeclineChoice') && selectedTile != null && selectedStack != null) {
      sendMessageToBackend('abandonment', { tileId: selectedTile.toString() });
    } else if (phase === 'Redeployment' && selectedTile != null && selectedStack != null && !isStackFromBank) {
      sendMessageToBackend('deploymentout', {
        tileId: selectedTile.toString(),
        stackType: selectedStack.toString(),
      });
    } else if (isStackFromBank) {
      dispatch(setSelectedStack(null));
      dispatch(setIsStackFromBank(false));
    }
  }

  const handlePieceStackClick = (stackType: string) => {
    if (isStackFromBank && selectedStack === stackType) {
      dispatch(setIsStackFromBank(false));
      dispatch(setSelectedStack(null));
    } else {
      dispatch(setIsStackFromBank(true));
      dispatch(setSelectedStack(stackType));
    }
  };

  // Helper to render a single tribe's images stuck together
  const renderTribeImages = (tribe: Tribe, customClasses = '') => {
    return (
      <div className={`relative flex items-center ${customClasses}`}>
        <img
          src={getTraitImagePath(tribe.trait)}
          alt={tribe.trait}
          className="h-full w-auto -mr-6 z-20"
        />
        <img
          src={getRaceImagePath(tribe.race)}
          alt={tribe.race}
          className="h-full w-auto z-10"
        />
      </div>
    );
  };

  const renderPassiveTribeImages = (tribe: Tribe, customClasses = '') => {
    return (
      <div className={`relative flex items-center ${customClasses}`}>
        <img
          src={getTraitImagePath(tribe.trait)}
          alt={tribe.trait}
          className="h-full w-auto -mr-3 z-20"
        />
        <img
          src={getRaceImagePath(tribe.race)}
          alt={tribe.race}
          className="h-full w-auto z-10"
        />
      </div>
    );
  };

  const renderPieceStacks = (pieceStacks: PieceStack[]) => {
    return (
      <div className="flex space-x-2 mt-2 relative z-10">
        {pieceStacks.map((stack, index) => {
          const imageSrc = `/stacks/${stack.type}.png`;
          const isFlashy = isStackFromBank && selectedStack === stack.type;

          return (
            <div
              key={index}
              className="relative"
              style={{
                width: 60,
                height: 60,
                cursor: 'pointer',
                border: isFlashy ? '3px solid blue' : '',
                animation: isFlashy ? 'flash 1s infinite' : undefined,
              }}
              onClick={(e) => {
                e.stopPropagation();
                handlePieceStackClick(stack.type);
              }}
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

  return (
    <div className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6] relative h-full" onClick={handlePlayerClick}>
      <h3 className="text-lg font-bold">{player?.name}</h3>

      {/* Active Tribe Display (Only if exists) */}
      {player.activeTribe && (
        <div className="mt-3 flex items-center justify-center" style={{ height: '6rem' }}>
          {renderTribeImages(player.activeTribe, 'h-full')}
        </div>
      )}

      {/* Passive Tribes Display (Only if exists) */}
      {player.passiveTribes.length > 0 && (
        <div className="flex flex-wrap items-center justify-center mt-2 gap-3">
          {player.passiveTribes.map((tribe, i) => (
            <div
              key={i}
              className="opacity-60"
              style={{ height: '3rem', filter: 'grayscale(50%)' }}
            >
              {renderPassiveTribeImages(tribe, 'h-full')}
            </div>
          ))}
        </div>
      )}

      {/* Show the separator line only if there are stacks */}
      {player.pieceStacks.length > 0 && (
        <div className="my-4 border-t-4 border-[#8B4513] w-full"></div>
      )}

      {/* Piece Stacks */}
      {renderPieceStacks(player.pieceStacks)}
    </div>
  );
}

// CSS for the flashing border animation
const styles = document.createElement('style');
styles.innerHTML = `
  @keyframes flash {
    0% { border-color: blue; }
    50% { border-color: lightblue; }
    100% { border-color: blue; }
  }
  .text-shadow {
    text-shadow: 1px 1px 2px black;
  }
`;
document.head.appendChild(styles);
