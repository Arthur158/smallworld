import React, { useEffect, useState, useRef } from 'react';
import { setError, setSelectedStack, setIsStackFromBank, setSelectedTile } from '../../redux/slices/applicationSlice';
import mapImage from '../../images/mapsw.jpg';
import { Tile } from '../../types/Board';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService';

export default function Map() {
  const tiles: Record<string, Tile> = useSelector((state: RootState) => state.application.tiles);
  const isStackFromBank = useSelector((state: RootState) => state.application.isStackFromBank);
  const selectedStack = useSelector((state: RootState) => state.application.selectedStack);
  const selectedTile = useSelector((state: RootState) => state.application.selectedTile);
  const phase = useSelector((state: RootState) => state.application.phase);
  const [imageDimensions, setImageDimensions] = useState<{ width: number; height: number }>({ width: 0, height: 0 });
  const dispatch = useDispatch();

  // Pan & Zoom state
  const [scale, setScale] = useState(1);
  const [translateX, setTranslateX] = useState(0);
  const [translateY, setTranslateY] = useState(0);
  const [isPanning, setIsPanning] = useState(false);
  const [lastMousePosition, setLastMousePosition] = useState<{ x: number; y: number } | null>(null);

  // Track movement to distinguish click vs drag
  const [movedDistance, setMovedDistance] = useState(0);

  const svgRef = useRef<SVGSVGElement | null>(null);

  // Inject the keyframes into the document once
  useEffect(() => {
    const styles = document.createElement('style');
    styles.innerHTML = `
      @keyframes flash {
        0% { stroke: blue; }
        50% { stroke: lightblue; }
        100% { stroke: blue; }
      }
      .flash-border {
        animation: flash 1s infinite;
      }
    `;
    document.head.appendChild(styles);
  }, []);

  useEffect(() => {
    const image = new Image();
    image.src = mapImage;
    image.onload = () => {
      setImageDimensions({ width: image.width, height: image.height });
    };
    image.onerror = () => {
      console.error('Failed to load image');
    };
  }, []);

  if (imageDimensions.width === 0 || Object.keys(tiles).length === 0) {
    return <div className="text-center text-[#5F4B32] font-bold">Loading...</div>;
  }

  const handleTileStackClick = (tileID: string, stackType: string | null) => {
    console.log("getting here")
    console.log(tileID)
    if (
      (phase === "Conquest" || phase === "TileAbandonment" || phase === "DeclineChoice") &&
      isStackFromBank &&
      selectedStack != null
    ) {
      console.log("trying here")
      sendMessageToBackend("Conquest", { tileId: tileID.toString(), attackingStackType: selectedStack.toString() });
    } else if (
      (phase === 'Redeployment' || phase === 'TileAbandonment') &&
      selectedStack === stackType &&
      selectedTile === tileID
    ) {
      dispatch(setSelectedStack(null));
      dispatch(setSelectedTile(null));
      dispatch(setIsStackFromBank(false));
    } else if ((phase === "TileAbandonment" || phase === "DeclineChoice") && stackType != null) {
      dispatch(setSelectedStack(stackType));
      dispatch(setSelectedTile(tileID));
    } else if (phase === 'Redeployment' && isStackFromBank && selectedStack != null) {
      sendMessageToBackend("deploymentin", { tileId: tileID.toString(), stackType: selectedStack.toString() });
    } else if (phase === 'Redeployment' && !isStackFromBank && selectedTile != null && selectedStack != null) {
      sendMessageToBackend("deploymentthrough", {
        tileFromId: selectedTile.toString(),
        tileToId: tileID.toString(),
        stackType: selectedStack
      });
    } else if (phase === 'Redeployment' && selectedStack == null && stackType != null) {
      dispatch(setSelectedStack(stackType));
      dispatch(setSelectedTile(tileID));
      dispatch(setIsStackFromBank(false));
    }
  };

  const minScale = 1;
  const maxScale = 5;
  const baseWidth = 1130;

  const clientToSvgCoords = (clientX: number, clientY: number) => {
    if (!svgRef.current) return { x: 0, y: 0 };
    const pt = svgRef.current.createSVGPoint();
    pt.x = clientX;
    pt.y = clientY;
    const ctm = svgRef.current.getScreenCTM();
    if (ctm) {
      return pt.matrixTransform(ctm.inverse());
    }
    return { x: 0, y: 0 };
  };

  const handleMouseEnter = (e: React.MouseEvent<SVGPolygonElement>) => {
    const target = e.currentTarget;
    target.setAttribute('stroke', '#8B4513');
    target.setAttribute('fill', 'rgba(139,69,19,0.2)');
    target.setAttribute('stroke-width', '2');
    target.style.cursor = 'pointer';
  };

  const handleMouseLeave = (e: React.MouseEvent<SVGPolygonElement>) => {
    const target = e.currentTarget;
    target.setAttribute('stroke', 'transparent');
    target.setAttribute('fill', 'transparent');
    target.setAttribute('stroke-width', '0');
  };

  const clampTranslate = (tx: number, ty: number, scl: number) => {
    if (scl <= 1) {
      return { x: 0, y: 0 };
    }
    const maxX = 0;
    const maxY = 0;
    const minX = -(imageDimensions.width * (scl - 1));
    const minY = -(imageDimensions.height * (scl - 1));
    const clampedX = Math.min(maxX, Math.max(tx, minX));
    const clampedY = Math.min(maxY, Math.max(ty, minY));
    return { x: clampedX, y: clampedY };
  };

  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();

    const { deltaY } = e;
    const scaleAmount = 1.1;
    let newScale = scale;

    if (deltaY < 0) {
      newScale *= scaleAmount;
    } else {
      newScale /= scaleAmount;
    }

    newScale = Math.max(minScale, Math.min(newScale, maxScale));

    const { x: svgX, y: svgY } = clientToSvgCoords(e.clientX, e.clientY);
    const prevScreenX = svgX * scale + translateX;
    const prevScreenY = svgY * scale + translateY;
    let newTranslateX = prevScreenX - svgX * newScale;
    let newTranslateY = prevScreenY - svgY * newScale;

    const clamped = clampTranslate(newTranslateX, newTranslateY, newScale);
    newTranslateX = clamped.x;
    newTranslateY = clamped.y;

    setScale(newScale);
    setTranslateX(newTranslateX);
    setTranslateY(newTranslateY);
  };

  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsPanning(true);
    setLastMousePosition({ x: e.clientX, y: e.clientY });
    setMovedDistance(0);
  };

  const handleMouseMove = (e: React.MouseEvent) => {
    if (isPanning && lastMousePosition) {
      const dx = e.clientX - lastMousePosition.x;
      const dy = e.clientY - lastMousePosition.y;

      let newTranslateX = translateX + dx;
      let newTranslateY = translateY + dy;

      const clamped = clampTranslate(newTranslateX, newTranslateY, scale);
      newTranslateX = clamped.x;
      newTranslateY = clamped.y;

      setTranslateX(newTranslateX);
      setTranslateY(newTranslateY);
      setLastMousePosition({ x: e.clientX, y: e.clientY });

      setMovedDistance(movedDistance + Math.sqrt(dx * dx + dy * dy));
    }
  };

  const handleMouseUp = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsPanning(false);
    setLastMousePosition(null);
  };

  const handleMouseLeaveContainer = () => {
    if (isPanning) {
      setIsPanning(false);
      setLastMousePosition(null);
    }
  };

  const handleMouseOver = () => {
    document.body.style.overflow = 'hidden';
  };

  const handleMouseOut = () => {
    document.body.style.overflow = '';
  };

  return (
    <div
      className="flex justify-center items-center overflow-hidden relative"
      style={{ width: '100%', height: '100%' }}
      onMouseLeave={handleMouseLeaveContainer}
      onMouseOver={handleMouseOver}
      onMouseOut={handleMouseOut}
    >
      <svg
        ref={svgRef}
        width={imageDimensions.width}
        height={imageDimensions.height}
        viewBox={`0 0 ${imageDimensions.width} ${imageDimensions.height}`}
        xmlns="http://www.w3.org/2000/svg"
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        onMouseMove={handleMouseMove}
        style={{ userSelect: 'none', cursor: isPanning ? 'grabbing' : 'grab' }}
      >
        <g transform={`translate(${translateX},${translateY}) scale(${scale})`}>
          <image
            x="0"
            y="0"
            width={imageDimensions.width}
            height={imageDimensions.height}
            href={mapImage}
          />

          {Object.values(tiles).map((tile) => {
            const scaledCoords = tile.polygon.coords.map(
              (coord: number) => coord * (imageDimensions.width / baseWidth)
            );
            const points: string[] = [];
            for (let i = 0; i < scaledCoords.length; i += 2) {
              points.push(`${scaledCoords[i]},${scaledCoords[i + 1]}`);
            }
            const pointsString = points.join(' ');

            return (
              <polygon
                key={tile.id}
                points={pointsString}
                fill="transparent"
                stroke="transparent"
                strokeWidth={0}
                onMouseEnter={handleMouseEnter}
                onMouseLeave={handleMouseLeave}
                onClick={() => handleTileStackClick(tile.id, null)}
              />
            );
          })}

          {Object.values(tiles).map((tile) => (
            <g key={`stack-${tile.id}`}>
              {tile.pieceStack
                .slice()
                .reverse()
                .map((stack, index) => {
                  const baseSize = 61;
                  const offset = 0.4 * baseSize;
                  const scaledStackX =
                    (tile.polygon.stackX + index * offset) *
                    (imageDimensions.width / baseWidth);
                  const scaledStackY =
                    (tile.polygon.stackY - index * offset) *
                    (imageDimensions.width / baseWidth);
                  const imageSrc = `/stacks/${stack.type}.png`;

                  const isGray = !stack.isActive;
                  const isSelected =
                    selectedTile === tile.id && selectedStack === stack.type && isStackFromBank === false;

                  // We create multiple images (or "layers") to visually stack them
                  // Each piece moves slightly bottom-right, so we see the "pile".
                  return (
                    <g
                      key={`piece-${tile.id}-${index}`}
                      onClick={() => handleTileStackClick(tile.id, stack.type)}
                    >
                      {[...Array(stack.amount)].map((_, i) => {
                        const pieceX = scaledStackX + i * 3;
                        const pieceY = scaledStackY + i * 3;
                        const isTopPiece = i === stack.amount - 1;

                        return (
                          <g key={`piece-layer-${i}`}>
                            <image
                              href={imageSrc}
                              x={pieceX}
                              y={pieceY - baseSize}
                              width={baseSize}
                              height={baseSize}
                              style={{
                                filter: isGray ? 'grayscale(100%)' : 'none',
                              }}
                              onError={(e) => {
                                (e.target as SVGImageElement).style.display = 'none';
                              }}
                            />
                            {/* If it's the top piece, show type, amount and (optionally) a flashing border */}
                            {isTopPiece && (
                              <>
                                {/* A stroke behind the image to show selection (flash) */}
                                {isSelected && (
                                  <rect
                                    x={pieceX}
                                    y={pieceY - baseSize}
                                    width={baseSize}
                                    height={baseSize}
                                    fill="none"
                                    stroke="blue"
                                    strokeWidth={4}
                                    className="flash-border"
                                  />
                                )}
                                <text
                                  x={pieceX + baseSize / 2}
                                  y={pieceY - baseSize / 2}
                                  fill="white"
                                  fontSize="8"
                                  fontWeight="bold"
                                  textAnchor="middle"
                                  dominantBaseline="middle"
                                >
                                  {stack.type}
                                </text>
                                <text
                                  x={pieceX + baseSize - 3}
                                  y={pieceY - baseSize + 45}
                                  fill="black"
                                  fontSize="18"
                                  fontWeight="bold"
                                  textAnchor="end"
                                  dominantBaseline="hanging"
                                >
                                  {stack.amount}
                                </text>
                              </>
                            )}
                          </g>
                        );
                      })}
                    </g>
                  );
                })}
            </g>
          ))}
        </g>
      </svg>
    </div>
  );
}
