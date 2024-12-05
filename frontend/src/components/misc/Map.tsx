import React, { useEffect, useState } from 'react';
import { setError } from '../../redux/slices/applicationSlice';
import mapImage from '../../images/mapsw.jpg';
import { Tile } from '../../types/Board';
import { useSelector, useDispatch } from 'react-redux';
import { RootState } from '../../redux/store';

export default function Map() {
  const tiles: Record<string, Tile> = useSelector((state: RootState) => state.application.tiles);
  const [imageDimensions, setImageDimensions] = useState<{ width: number; height: number }>({ width: 0, height: 0 });
  const dispatch = useDispatch();

  // Load the map areas from the external file
  // Load the image and get its dimensions
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
    return <div>Loading...</div>;
  }

  // Calculate scale factor
  const baseWidth = 1130; // The base width that the coordinates are based on
  const scaleFactor = imageDimensions.width / baseWidth;

  return (
    <div className="flex justify-center items-center min-h-screen bg-gray-100">
      <svg
        width={imageDimensions.width}
        height={imageDimensions.height}
        viewBox={`0 0 ${imageDimensions.width} ${imageDimensions.height}`}
        xmlns="http://www.w3.org/2000/svg"
      >
        <image x="0" y="0" width={imageDimensions.width} height={imageDimensions.height} href={mapImage} />

        {Object.values(tiles).map((tile => {
          const scaledCoords = tile.polygon.coords.map((coord: number) => coord * scaleFactor);
          const points = [];
          for (let i = 0; i < scaledCoords.length; i += 2) {
            points.push(`${scaledCoords[i]},${scaledCoords[i + 1]}`);
          }
          const pointsString = points.join(' ');
          return (
            <polygon
              key={tile.id}
              points={pointsString}
              fill="rgba(255, 0, 0, 0.5)"
              stroke="black"
              onClick={() => {
                console.log(`Polygon ${tile.id} clicked`);
                dispatch(setError(`Polygon ${tile.id} clicked`)); // Optional Redux action
              }}
            />
          );
        }))}
      </svg>
    </div>
  );
}
