import React from 'react';
import { RootState } from '../../redux/store';
import { Player, PieceStack } from '../../types/Board';
import { setIsStackFromBank, setSelectedStack } from '../../redux/slices/applicationSlice'
import { useSelector, useDispatch } from 'react-redux';
import { sendMessageToBackend } from '../../services/backendService';

export default function PlayerInfo() {
  const dispatch = useDispatch();
  const allPlayers: Player[] = useSelector((state: RootState) => state.application.players);
  const playerIndex: number = useSelector((state: RootState) => state.application.playerIndex);
  const ActiveIndex: number = useSelector((state: RootState) => state.application.playerNumber);
  const isStackFromBank = useSelector((state: RootState) => state.application.isStackFromBank);
  const selectedStack = useSelector((state: RootState) => state.application.selectedStack);
  const selectedTile = useSelector((state: RootState) => state.application.selectedTile);
  const phase = useSelector((state: RootState) => state.application.phase);

  const player = allPlayers[playerIndex];
  const activePlayer = allPlayers[ActiveIndex];

  const baseSize = 45;

  const handlePieceStackClick = (stackType: string) => {
    if (isStackFromBank && selectedStack == stackType) {
      dispatch(setIsStackFromBank(false))
      dispatch(setSelectedStack(null))
    } else {
    dispatch(setIsStackFromBank(true))
    dispatch(setSelectedStack(stackType))
    }
  }

  const handlePlayerClick = () => {
    if (phase == "TileAbandonment" && selectedTile != null && selectedStack != null) {
      sendMessageToBackend("abandonment", {tileId: selectedTile.toString()})
    } else if (phase == "Redeployment" && selectedTile != null && selectedStack != null) {
      sendMessageToBackend("deploymentout", {tileId: selectedTile.toString(), stackType: selectedStack.toString()})
    }
  };

  const renderPieceStacks = (pieceStacks: PieceStack[], isActive: boolean) => {
    return (
      <div className="flex space-x-2 mt-2 relative z-10">
        {pieceStacks.map((stack, index) => {
          const imageSrc = `/stacks/${stack.type}.png`;
          const isClickable = isActive;

          // Determine if the stack should be flashy
          const isFlashy =
            isStackFromBank && selectedStack === stack.type;

          return (
            <div
              key={index}
              className="relative"
              style={{
                width: baseSize,
                height: baseSize,
                cursor: isClickable ? 'pointer' : 'default',
                opacity: isActive ? 1 : 0.5,
                border: isFlashy ? '3px solid blue' : '1px solid black',
                animation: isFlashy ? 'flash 1s infinite' : undefined,
              }}
              onClick={(e) => {
                e.stopPropagation(); // Prevents the click event from bubbling up
                if (isClickable) {
                  handlePieceStackClick(stack.type);
                }
              }}
            >
              <div
                style={{
                  position: 'absolute',
                  width: baseSize,
                  height: baseSize,
                  backgroundColor: 'blue',
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
    <div
      className="p-4 border border-[#5F4B32] rounded bg-[#FDF5E6] relative"
      onClick={handlePlayerClick}
      style={{ cursor: 'pointer' }}
    >
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
      {renderPieceStacks(player.pieceStacks, playerIndex === ActiveIndex)}
    </div>
  );
}

// CSS for the flashing animation
const styles = document.createElement('style');
styles.innerHTML = `
  @keyframes flash {
    0% { border-color: blue; }
    50% { border-color: lightblue; }
    100% { border-color: blue; }
  }
`;
document.head.appendChild(styles);
