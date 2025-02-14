import React, { useEffect, useState, useRef } from 'react';
import { setSelectedStack, setIsStackFromBank, setSelectedTile } from '../../redux/slices/applicationSlice';
import { Tile } from '../../types/Board';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService';

function ImageOrBlueSquare({
  imageSrc,
  x,
  y,
  width,
  height,
  isGray,
}: {
  imageSrc: string;
  x: number;
  y: number;
  width: number;
  height: number;
  isGray: boolean;
}) {
  const [hasError, setHasError] = useState(false);

  return hasError ? (
    <rect x={x} y={y} width={width} height={height} fill="blue" />
  ) : (
    <image
      href={imageSrc}
      x={x}
      y={y}
      width={width}
      height={height}
      style={{ filter: isGray ? 'grayscale(100%)' : 'none' }}
      onError={() => setHasError(true)}
    />
  );
}

export default function Map() {
  const dispatch = useDispatch();

  const containerRef = useRef<HTMLDivElement | null>(null);
  const svgRef = useRef<SVGSVGElement | null>(null);

  const tiles: Record<string, Tile> = useSelector((state: RootState) => state.application.tiles);
  const offsetMapTiles: number = useSelector((state: RootState) => state.application.offsetMapTiles);
  const isStackFromBank = useSelector((state: RootState) => state.application.isStackFromBank);
  const selectedStack = useSelector((state: RootState) => state.application.selectedStack);
  const selectedTile = useSelector((state: RootState) => state.application.selectedTile);
  const mapName = useSelector((state: RootState) => state.application.mapName);
  const phase = useSelector((state: RootState) => state.application.phase);
  const offsetStacksX = useSelector((state: RootState) => state.application.offsetStacksX);
  const offsetStacksY = useSelector((state: RootState) => state.application.offsetStacksY);

  const [imageDimensions, setImageDimensions] = useState<{ width: number; height: number }>({
    width: 0,
    height: 0,
  });

  const [scale, setScale] = useState(1);
  const [translateX, setTranslateX] = useState(0);
  const [translateY, setTranslateY] = useState(0);
  const [isPanning, setIsPanning] = useState(false);
  const [lastMousePosition, setLastMousePosition] = useState<{ x: number; y: number } | null>(null);
  const [movedDistance, setMovedDistance] = useState(0);
  const [minScale, setMinScale] = useState(1);

  const maxScale = 5;
  const baseWidth = 1130;

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

  // Load the map image so we know its natural dimensions
  useEffect(() => {
    if (!mapName) return;
    const image = new Image();
    image.src = `/maps/${mapName}.png`;
    image.onload = () => {
      setImageDimensions({ width: image.width, height: image.height });
    };
    image.onerror = () => {
      console.error(`Failed to load map image: /maps/${mapName}.png`);
    };
  }, [mapName]);

  // Once we know the image size and the container size, we can figure out the minScale
  // so that the image is entirely contained, and also center it.
  useEffect(() => {
    if (!containerRef.current) return;
    if (!imageDimensions.width || !imageDimensions.height) return;

    const containerWidth = containerRef.current.clientWidth;
    const containerHeight = containerRef.current.clientHeight;

    if (!containerWidth || !containerHeight) return;

    // Compute the minimal scale that fits the entire image inside the container
    const fitScaleX = containerWidth / imageDimensions.width;
    const fitScaleY = containerHeight / imageDimensions.height;
    const newMinScale = Math.min(fitScaleX, fitScaleY);

    setMinScale(newMinScale);
    setScale(newMinScale);

    // Center the image at first (when scale = minScale)
    const centeredX = (containerWidth - imageDimensions.width * newMinScale) / 2;
    const centeredY = (containerHeight - imageDimensions.height * newMinScale) / 2;

    setTranslateX(centeredX);
    setTranslateY(centeredY);
  }, [imageDimensions]);

  // Listen for 'f' key to reset selection
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key.toLowerCase() === 'f') {
        dispatch(setSelectedStack(null));
        dispatch(setSelectedTile(null));
        dispatch(setIsStackFromBank(false));
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => {
      window.removeEventListener('keydown', handleKeyDown);
    };
  }, [dispatch]);

  if (!mapName || imageDimensions.width === 0 || Object.keys(tiles).length === 0) {
    return <div className="text-center text-[#5F4B32] font-bold">Loading map...</div>;
  }

  const handleTileStackClick = (tileID: string, stackType: string | null) => {
    if (
      (phase === 'Conquest' || phase === 'TileAbandonment' || phase === 'DeclineChoice') &&
      isStackFromBank &&
      selectedStack != null
    ) {
      sendMessageToBackend('Conquest', { tileId: tileID.toString(), attackingStackType: selectedStack.toString() });
    } else if (
      (phase === 'Redeployment' || phase === 'TileAbandonment') &&
      selectedStack === stackType &&
      selectedTile === tileID
    ) {
      dispatch(setSelectedStack(null));
      dispatch(setSelectedTile(null));
      dispatch(setIsStackFromBank(false));
    } else if ((phase === 'TileAbandonment' || phase === 'DeclineChoice') && stackType != null) {
      dispatch(setSelectedStack(stackType));
      dispatch(setSelectedTile(tileID));
    } else if (phase === 'Redeployment' && isStackFromBank && selectedStack != null) {
      sendMessageToBackend('deploymentin', { tileId: tileID.toString(), stackType: selectedStack.toString() });
    } else if (phase === 'Redeployment' && !isStackFromBank && selectedTile != null && selectedStack != null) {
      sendMessageToBackend('deploymentthrough', {
        tileFromId: selectedTile.toString(),
        tileToId: tileID.toString(),
        stackType: selectedStack,
      });
    } else if (phase === 'Redeployment' && selectedStack == null && stackType != null) {
      dispatch(setSelectedStack(stackType));
      dispatch(setSelectedTile(tileID));
      dispatch(setIsStackFromBank(false));
    }
  };

  /**
   * Clamps translateX and translateY so that the image never leaves the viewport entirely.
   * If the image is smaller than the container in one dimension, we keep it centered.
   * Otherwise, we clamp so that the user can't drag it beyond the edges.
   */
  const clampTranslate = (tx: number, ty: number, scl: number) => {
    if (!containerRef.current) {
      return { x: 0, y: 0 };
    }

    const containerWidth = containerRef.current.clientWidth;
    const containerHeight = containerRef.current.clientHeight;
    const scaledWidth = imageDimensions.width * scl;
    const scaledHeight = imageDimensions.height * scl;

    let newTx = tx;
    let newTy = ty;

    // Horizontal clamp/center
    if (scaledWidth <= containerWidth) {
      // If scaled image is narrower than container, center it
      newTx = (containerWidth - scaledWidth) / 2;
    } else {
      // Otherwise, clamp
      const minX = containerWidth - scaledWidth; // negative value
      const maxX = 0;
      newTx = Math.max(minX, Math.min(newTx, maxX));
    }

    // Vertical clamp/center
    if (scaledHeight <= containerHeight) {
      // If scaled image is shorter than container, center it
      newTy = (containerHeight - scaledHeight) / 2;
    } else {
      // Otherwise, clamp
      const minY = containerHeight - scaledHeight; // negative value
      const maxY = 0;
      newTy = Math.max(minY, Math.min(newTy, maxY));
    }

    return { x: newTx, y: newTy };
  };

  /**
   * Convert screen coordinates (clientX, clientY) to SVG coordinates,
   * taking into account the current transform.
   */
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

  /**
   * Zoom in/out on wheel scroll, anchored at the mouse location.
   */
  const handleWheel = (e: React.WheelEvent) => {
    e.preventDefault();
    const { deltaY } = e;
    const scaleFactor = 1.1;
    let newScale = scale;

    if (deltaY < 0) {
      // Scroll up => zoom in
      newScale *= scaleFactor;
    } else {
      // Scroll down => zoom out
      newScale /= scaleFactor;
    }

    // Respect minScale and maxScale
    newScale = Math.max(minScale, Math.min(newScale, maxScale));

    // Convert mouse position to SVG coords
    const { x: svgX, y: svgY } = clientToSvgCoords(e.clientX, e.clientY);

    // Current screen coords of that point
    const prevScreenX = svgX * scale + translateX;
    const prevScreenY = svgY * scale + translateY;

    // Desired new translate so the point under mouse stays in the same place
    let newTranslateX = prevScreenX - svgX * newScale;
    let newTranslateY = prevScreenY - svgY * newScale;

    // Clamp translation
    const clamped = clampTranslate(newTranslateX, newTranslateY, newScale);
    newTranslateX = clamped.x;
    newTranslateY = clamped.y;

    // Update
    setScale(newScale);
    setTranslateX(newTranslateX);
    setTranslateY(newTranslateY);
  };

  /**
   * Begin panning on mousedown
   */
  const handleMouseDown = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsPanning(true);
    setLastMousePosition({ x: e.clientX, y: e.clientY });
    setMovedDistance(0);
  };

  /**
   * Drag the image if panning
   */
  const handleMouseMove = (e: React.MouseEvent) => {
    if (!isPanning || !lastMousePosition) return;

    const dx = e.clientX - lastMousePosition.x;
    const dy = e.clientY - lastMousePosition.y;

    let newTranslateX = translateX + dx;
    let newTranslateY = translateY + dy;

    // Clamp after shifting
    const clamped = clampTranslate(newTranslateX, newTranslateY, scale);
    newTranslateX = clamped.x;
    newTranslateY = clamped.y;

    setTranslateX(newTranslateX);
    setTranslateY(newTranslateY);
    setLastMousePosition({ x: e.clientX, y: e.clientY });
    setMovedDistance(movedDistance + Math.sqrt(dx * dx + dy * dy));
  };

  /**
   * End panning on mouseup
   */
  const handleMouseUp = (e: React.MouseEvent) => {
    e.preventDefault();
    setIsPanning(false);
    setLastMousePosition(null);
  };

  /**
   * If the mouse leaves the container while panning, stop panning
   */
  const handleMouseLeaveContainer = () => {
    if (isPanning) {
      setIsPanning(false);
      setLastMousePosition(null);
    }
  };

  return (
    <div
      ref={containerRef}
      className="relative w-full h-full"
      style={{ overflow: 'hidden' }}
      onMouseLeave={handleMouseLeaveContainer}
    >
      <svg
        ref={svgRef}
        style={{
          userSelect: 'none',
          cursor: isPanning ? 'grabbing' : 'grab',
          width: '100%',
          height: '100%',
          // We make the SVG fill the container so we can handle dynamic resizing
        }}
        onWheel={handleWheel}
        onMouseDown={handleMouseDown}
        onMouseUp={handleMouseUp}
        onMouseMove={handleMouseMove}
      >
        {/* 
          We don't strictly need a viewBox here if we manually transform 
          everything. But if we set a viewBox, let's make sure it matches 
          the native image size. 
        */}
        <g transform={`translate(${translateX},${translateY}) scale(${scale})`}>
          {/* The map image */}
          <image
            x={0}
            y={0}
            width={imageDimensions.width}
            height={imageDimensions.height}
            href={`/maps/${mapName}.png`}
          />

          {/* Invisible polygons for tile click detection */}
          {Object.values(tiles).map((tile) => {
            const scaledCoords = tile.polygon.coords.map(
              (coord: number) =>
                coord *
                (imageDimensions.width / baseWidth) * offsetMapTiles
            );
            const points: string[] = [];
            for (let i = 0; i < scaledCoords.length; i += 2) {
              points.push(`${scaledCoords[i]},${scaledCoords[i + 1]}`);
            }
            const pointsString = points.join(' ');

            const handleMouseEnter = (e: React.MouseEvent<SVGPolygonElement>) => {
              e.currentTarget.setAttribute('stroke', '#8B4513');
              e.currentTarget.setAttribute('fill', 'rgba(139,69,19,0.2)');
              e.currentTarget.setAttribute('stroke-width', '2');
              e.currentTarget.style.cursor = 'pointer';
            };

            const handleMouseLeave = (e: React.MouseEvent<SVGPolygonElement>) => {
              e.currentTarget.setAttribute('stroke', 'transparent');
              e.currentTarget.setAttribute('fill', 'transparent');
              e.currentTarget.setAttribute('stroke-width', '0');
            };

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

          {/* Stacks on each tile */}
          {Object.values(tiles).map((tile) => (
            <g key={`stack-${tile.id}`}>
              {tile.pieceStack
                .slice()
                .reverse()
                .map((stack, index) => {
                  const baseSize = imageDimensions.width * 0.0555;
                  const offset = 0.4 * baseSize;
                  const scaledStackX =
                    (tile.polygon.stackX + index * offset * baseSize * offsetStacksX) *
                    (imageDimensions.width / baseWidth);
                  const scaledStackY =
                    (tile.polygon.stackY - index * offset * baseSize * offsetStacksX ) *
                    (imageDimensions.width / baseWidth);
                  const imageSrc = `/stacks/${stack.type}.png`;
                  const isGray = !stack.isActive;
                  const isSelected =
                    selectedTile === tile.id &&
                    selectedStack === stack.type &&
                    isStackFromBank === false;

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
                            <ImageOrBlueSquare
                              imageSrc={imageSrc}
                              x={pieceX * offsetMapTiles}
                              y={pieceY * offsetMapTiles - baseSize}
                              width={baseSize}
                              height={baseSize}
                              isGray={isGray}
                            />
                            {isTopPiece && (
                              <>
                                {isSelected && (
                                  <rect
                                    x={pieceX * offsetMapTiles}
                                    y={pieceY * offsetMapTiles - baseSize}
                                    width={baseSize}
                                    height={baseSize}
                                    fill="none"
                                    stroke="blue"
                                    strokeWidth={4}
                                    className="flash-border"
                                  />
                                )}
                                <text
                                  x={pieceX * offsetMapTiles + baseSize * 0.97 }
                                  y={pieceY * offsetMapTiles - baseSize * 0.27 }
                                  fill="black"
                                  fontSize={`${baseSize * 0.3}`}
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
