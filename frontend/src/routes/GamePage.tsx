import TribeList from '../components/layouts/TribeList';
import Map from '../components/misc/Map';
import PlayerInfo from '../components/layouts/PlayerInfo';
import React, { useEffect } from 'react';
import { connectWebSocket } from '../services/backendService';
import { parseAreaFile } from '../utility/MapParser';
import { setTiles, setOpponents, updateTileStack, setPlaying } from '../redux/slices/applicationSlice';
import { RootState, AppDispatch } from '../redux/store';
import { useSelector, useDispatch } from 'react-redux';
import { Polygon } from '../types/Board';
import OpponentsList from '../components/layouts/OpponentsList'; 

export default function GamePage() {
  const tiles = useSelector((state: RootState) => state.application.tiles);
  const dispatch: AppDispatch = useDispatch();

  useEffect(() => {
    connectWebSocket('ws://yourserver.com');

    const loadAreas = async () => {
      try {
        const response = await fetch('/maps/map.txt');
        const text = await response.text();
        const polygons = parseAreaFile(text);

        let id = 0;
        const tiles = polygons.map((polygon: Polygon) => ({
          id: id++,
          pieceStack: [],
          polygon,
        }));
        dispatch(setTiles(tiles))
        dispatch(updateTileStack({tile_id:"1", new_stacks: [{type:"elves", amount:3}, {type:"dragon", amount:1}]}))
        dispatch(updateTileStack({tile_id:"5", new_stacks: [{type:"elves", amount:3}]}))
        dispatch(updateTileStack({tile_id:"20", new_stacks: [{type:"dwarves", amount:3}]}))
        dispatch(setPlaying(false))
      } catch (error) {
        console.error('Error loading file:', error);
      }
    };
    loadAreas()

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
        pieceStacks: [{type:"dwarves", amount:3}, {type:"dragon", amount:1}],
      }
    ]))
  }, []);

  return (
    <div className="min-h-screen flex flex-col bg-[#F5F5DC] text-[#5F4B32] font-serif">
      <div className="flex flex-row flex-grow divide-x divide-[#5F4B32]">
        <div className="w-1/4 flex flex-col p-4 space-y-4 border-r border-[#5F4B32]">
          <h1 className="text-2xl mb-4 font-bold">Informations</h1>
          <TribeList />
          <div className="border-t border-[#5F4B32] pt-4">
            <PlayerInfo />
          </div>
          <div className="border-t border-[#5F4B32] pt-4">
            <OpponentsList />
          </div>
        </div>
        <div className="w-3/4 flex flex-col items-center justify-center p-4">
          <div className="border-4 border-[#8B4513] rounded-lg shadow-md bg-white p-2">
            <Map />
          </div>
        </div>
      </div>
    </div>
  );
}
