// GamePage.js
import React from 'react';
import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import PlayerInfo from '../components/layouts/PlayerInfo';

export default function GamePage() {
  return (
    <div className="flex flex-col min-h-screen bg-gray-100">
      <div className="flex flex-row flex-grow">
        {/* Left Sidebar: TribeList */}
        <div className="w-1/4">
          <TribeList />
        </div>
        {/* Main Content: Map */}
        <div className="w-3/4">
          <Map />
        </div>
      </div>
      {/* Footer: PlayerInfo */}
      <div className="mt-4">
        <PlayerInfo />
      </div>
    </div>
  );
}
