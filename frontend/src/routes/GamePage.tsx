// GamePage.js
import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import PlayerInfo from '../components/layouts/PlayerInfo';
import React, { useEffect } from 'react';
import { connectWebSocket } from '../services/backendService';
import { parseAreaFile } from '../utility/MapParser';
import { setTiles, setOpponents } from '../redux/slices/applicationSlice';
import { RootState, AppDispatch } from '../redux/store';
import { useSelector, useDispatch } from 'react-redux';
import { Polygon } from '../types/Board';
import OpponentsList from '../components/layouts/OpponentsList'; // Import the new component


export default function GamePage() {
  const tiles = useSelector((state: RootState) => state.application.tiles);
  const dispatch: AppDispatch = useDispatch();

  useEffect(() => {
    // connect the websocket
    connectWebSocket('ws://yourserver.com');

    // request the tiles' information
    // sendRequest({type:"tiles_request"}) is what we should do, however we don't have a server yet so:
    const loadAreas = async () => {
      try {
        const response = await fetch('/maps/map.txt');
        const text = await response.text();
        console.log("hello")
        console.log(text);
        const polygons = parseAreaFile(text);

        let id = 0;
        const tiles = polygons.map((polygon: Polygon) => ({
          id: id++, // Increment `id` for each element
          pieceStack: [], // Initialize with an empty list
          polygon,
        }));
        dispatch(setTiles(tiles))
      } catch (error) {
        console.log("here")
        console.error('Error loading file:', error);
      }
    };
    loadAreas()

    // sendRequest({type:"opponents_request"})
    dispatch(setOpponents([
    {
      name: "Loris",
      activeTribe: {race: "dwarves", trait: "ugly"},
      passiveTribes: [{race: "ghouls", trait:"flying"}],
      isPlaying: false,
      pieceStacks: [],
    },
    {
      name: "Cristian",
      activeTribe: {race: "amazons", trait: "armed"},
      passiveTribes: [{race: "ghouls", trait:"dragon-riding"}],
      isPlaying: false,
      pieceStacks: [],
    }]))

  }, []);

  return (
  <div className="flex flex-col min-h-screen bg-gray-100">
    <div className="flex flex-row flex-grow">
      <div className="w-1/4 flex flex-col">
        <TribeList />
        <PlayerInfo />
        <OpponentsList />
      </div>
      <div className="w-3/4 flex flex-col">
        <Map />
      </div>
    </div>
  </div>
  );
}
