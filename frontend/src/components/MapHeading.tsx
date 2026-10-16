import React from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../redux/store';

export default function MapHeading() {
  const name = useSelector((state: RootState) => state.application.mapRegionName)?.trim();
  if (!name) return null;
  return <h2 aria-label={`Continent: ${name}`} className="mb-2 shrink-0 break-words text-center text-lg font-semibold tracking-wide text-[#6b4a2d]">
    {name}
  </h2>;
}
