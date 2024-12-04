import React, { useEffect, useState } from 'react';
import { useDispatch } from 'react-redux';
import { setError } from '../../redux/slices/applicationSlice';
import mapImage from '../../images/mapsw.jpg';
import { parseAreaFile } from '../../utility/MapParser';
import { Polygon } from '../../types/Board';

export default function Map() {
  const [areas, setAreas] = useState<Polygon[]>([]);
  const [imageDimensions, setImageDimensions] = useState<{ width: number; height: number }>({ width: 0, height: 0 });
  const dispatch = useDispatch();

  // Load the map areas from the external file
  useEffect(() => {
    const loadAreas = async () => {
      try {
        const response = await fetch('/maps/map.txt');
        const text = await response.text();
        console.log(text);
        setAreas(parseAreaFile(text));
      } catch (error) {
        console.error('Error loading file:', error);
      }
    };

    loadAreas();
  }, []);

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

  if (imageDimensions.width === 0 || areas.length === 0) {
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

        {areas.map((area, index) => {
          const scaledCoords = area.coords.map((coord) => coord * scaleFactor);
          const points = [];
          for (let i = 0; i < scaledCoords.length; i += 2) {
            points.push(`${scaledCoords[i]},${scaledCoords[i + 1]}`);
          }
          const pointsString = points.join(' ');
          return (
            <polygon
              key={index}
              points={pointsString}
              fill="rgba(255, 0, 0, 0.5)"
              stroke="black"
              onClick={() => {
                console.log(`Polygon ${index + 1} clicked`);
                dispatch(setError(`Polygon ${index + 1} clicked`)); // Optional Redux action
              }}
            />
          );
        })}
      </svg>
    </div>
  );
}
