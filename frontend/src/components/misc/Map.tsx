import React, { useEffect, useState, useRef } from 'react';
import { setError } from '../../redux/slices/applicationSlice';
import mapImage from '../../images/mapsw.jpg';
import { Tile } from '../../types/Board';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../redux/store';
import { sendMessageToBackend } from '../../services/backendService'

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
    console.log("entered")
    if ((phase == "Conquest" || phase == "TileAbandonment") && isStackFromBank && selectedStack != null) {
      sendMessageToBackend("Conquest", {tileId: tileID.toString(), attackingStackType: selectedStack})
    } else if (phase == 'Redeployment' && isStackFromBank && selectedStack != null) {
      sendMessageToBackend("deploymentin", {tileId: tileID, stackType: selectedStack})
    } else if (phase == 'Redeployment' && !isStackFromBank && selectedStack != null) {
      sendMessageToBackend("deploymentthrough", {tileFromId: selectedTile, tileToId: tileID, stackType: selectedStack})
    }
  }


  const minScale = 1; 
  const maxScale = 5; 
  const baseWidth = 1130;

  // Convert a screen position (in client coordinates) to SVG coordinates
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

    // Clamp scale
    newScale = Math.max(minScale, Math.min(newScale, maxScale));

    // Get mouse position in SVG coords before scaling
    const { x: svgX, y: svgY } = clientToSvgCoords(e.clientX, e.clientY);

    // After scaling, keep point under cursor stable
    const prevScreenX = (svgX * scale) + translateX;
    const prevScreenY = (svgY * scale) + translateY;

    let newTranslateX = prevScreenX - svgX * newScale;
    let newTranslateY = prevScreenY - svgY * newScale;

    // Clamp translation
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

      // Clamp translation
      const clamped = clampTranslate(newTranslateX, newTranslateY, scale);
      newTranslateX = clamped.x;
      newTranslateY = clamped.y;

      setTranslateX(newTranslateX);
      setTranslateY(newTranslateY);
      setLastMousePosition({ x: e.clientX, y: e.clientY });

      setMovedDistance(movedDistance + Math.sqrt(dx*dx + dy*dy));
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

  const handlePolygonClick = (tileId: string) => {
    // Only trigger click action if map wasn't dragged significantly
    // Define a small threshold to distinguish click from drag, e.g. 5px
    if (movedDistance < 5) {
      console.log(`Polygon ${tileId} clicked`);
      handleTileStackClick(tileId, null)
      dispatch(setError(tileId))
    }
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
          <image x="0" y="0" width={imageDimensions.width} height={imageDimensions.height} href={mapImage} />

          {Object.values(tiles).map((tile) => {
            const scaledCoords = tile.polygon.coords.map((coord: number) => coord * (imageDimensions.width / baseWidth));
            const points = [];
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
                onClick={() => handlePolygonClick(tile.id)}
              />
            );
          })}

          {Object.values(tiles).map((tile) => (
            <g key={`stack-${tile.id}`}>
              {tile.pieceStack
                .slice()
                .reverse()
                .map((stack, index) => {
                  const baseSize = 45;
                  const offset = 0.4 * baseSize;
                  const scaledStackX = (tile.polygon.stackX + index * offset) * (imageDimensions.width / baseWidth);
                  const scaledStackY = (tile.polygon.stackY - index * offset) * (imageDimensions.width / baseWidth);
                  const imageSrc = `/stacks/${stack.type}.png`;

                  return (
                    <g key={`piece-${tile.id}-${index}`}>
                      <rect
                        x={scaledStackX}
                        y={scaledStackY - baseSize}
                        width={baseSize}
                        height={baseSize}
                        fill="blue"
                        stroke="black"
                      />
                      <text
                        x={scaledStackX + baseSize / 2}
                        y={scaledStackY - baseSize / 2}
                        fill="white"
                        fontSize="8"
                        fontWeight="bold"
                        textAnchor="middle"
                        dominantBaseline="middle"
                      >
                        {stack.type}
                      </text>
                      <image
                        href={imageSrc}
                        x={scaledStackX}
                        y={scaledStackY - baseSize}
                        width={baseSize}
                        height={baseSize}
                        onError={(e) => {
                          (e.target as SVGImageElement).style.display = "none";
                        }}
                      />
                      <text
                        x={scaledStackX + baseSize - 3}
                        y={scaledStackY - baseSize + 30}
                        fill="black"
                        fontSize="15"
                        fontWeight="bold"
                        textAnchor="end"
                        dominantBaseline="hanging"
                      >
                        {stack.amount}
                      </text>
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

