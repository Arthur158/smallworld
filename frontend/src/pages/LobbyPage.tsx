import WikiToolbar from '../components/wiki/WikiToolbar';
// src/pages/LobbyPage.tsx

import React, { useEffect, useRef, useState } from 'react';
import { useSelector, useDispatch, shallowEqual } from 'react-redux';
import { RootState } from '../redux/store';
import { useNavigate } from 'react-router-dom';
import { getAuthConfig, sendMessageToBackend } from '../services/backendService';
import { reset, clearError, mapEditorRequested, mapEditorSelecting, mapEditorFailed, mapEditorClosed } from '../redux/slices/applicationSlice';
import { Room, SaveGameInfo } from '../types/Board';
import RoomBotControls, { BotModelSelect } from '../components/RoomBotControls';
import { useShortcut } from '../hooks/useShortcut';
import GeneratedMapEditor from '../components/GeneratedMapEditor';
import type { GeneratedMapSettings } from '../types/GeneratedMap';
import { createRandomRoomName } from '../utility/roomNames';

const mapLabel = (name: string) => name === 'generatedmap' ? 'Generated map · random layout'
  : name === 'underground2players' ? 'Underground · 2 players'
  : /^[0-9a-f]{8}-(?:[0-9a-f]{4}-){3}[0-9a-f]{12}$/i.test(name) ? 'Generated map' : name;

export default function LobbyPage() {
  const navigate = useNavigate();
  const dispatch = useDispatch();

  const {
    name: username,
    isAuthenticated,
    rooms,
    roomsInProgress,
    roomid,
    gameStarted,
    saveGames,
    error,
    saveSelectionId,
    mapChoices,
    playerStatuses,
    generatedMapEditor,
  } = useSelector((state: RootState) => ({
    name: state.application.name,
    isAuthenticated: state.application.isAuthenticated,
    rooms: state.application.rooms,
    roomsInProgress: state.application.roomsInProgress,
    roomid: state.application.roomid,
    gameStarted: state.application.gameStarted,
    saveGames: state.application.saveGames as SaveGameInfo[],
    error: state.application.error,
    saveSelectionId: state.application.saveSelectionId,
    mapChoices: state.application.mapChoices,
    playerStatuses: state.application.playerStatuses,
    generatedMapEditor: state.application.generatedMapEditor,
  }), shallowEqual);

  const [roomName, setRoomName] = useState('');
  const [mapEditorOpen, setMapEditorOpen] = useState(false);
  const customizeMapButton = useRef<HTMLButtonElement>(null);
  const editorWasOpen = useRef(false);
  const [pendingMapChoice, setPendingMapChoice] = useState<string | null>(null);
  const [activeTab, setActiveTab] = useState<'saves' | 'dlc'>('saves');

  let currentRoom: Room | null = null;
  if (rooms) {
    currentRoom = rooms.find((r) => r.id === roomid) || null;
  }

  // Redirect if not authenticated or if game started
  useEffect(() => {
    if (!isAuthenticated) {
      navigate('/');
      return;
    }
  }, [isAuthenticated, gameStarted, navigate]);

  // Handle page refresh
  useEffect(() => {
    const handlePageRefresh = () => {
      dispatch(reset());
      sendMessageToBackend('requestrefresh', {});
    };

    window.addEventListener('beforeunload', handlePageRefresh);
    return () => {
      window.removeEventListener('beforeunload', handlePageRefresh);
    };
  }, [dispatch]);

  // Create / Join / Start / Leave handlers
  const handleCreateRoom = () => {
    if (!username.trim()) return;
    sendMessageToBackend('createRoom', {
      roomName: roomName.trim() ? roomName : createRandomRoomName(),
      maxPlayers: 2,
    });
  };

  const handleJoinRoom = (selectedRoomId: string) => {
    if (!username.trim()) return;
    sendMessageToBackend('joinRoom', {
      roomId: selectedRoomId,
    });
  };

  const handleSpectateRoom = (selectedRoomId: string) => {
    if (!username.trim()) return;
    sendMessageToBackend('spectateRoom', {
      roomId: selectedRoomId,
    });
  };

  const handleEnterDisplayRoom = () => {
    if (!username.trim()) return;
    sendMessageToBackend('enterdisplayroom', {});
  };

  const handleStartGame = () => {
    if (!currentRoom) return;
    sendMessageToBackend('startGame', {
      roomId: currentRoom.id,
    });
  };

  const handleLeaveRoom = () => {
    sendMessageToBackend('leaveroom', {});
  };

  const handleGameIdClick = (gameId: number) => {
    sendMessageToBackend('loadgame', { saveId: gameId });
  };

  const handleDeleteGame = (gameId: number) => {
    sendMessageToBackend('deletesave', { saveId: gameId });
  };

  const handleMapChange = (event: React.ChangeEvent<HTMLSelectElement>) => {
    if (!currentRoom || !event.target.value) return;
    const newMap = event.target.value;
    sendMessageToBackend('changeRoomMap', { roomId: currentRoom.id, newMap });
  };

  const handleMoveUp = (playerName: string) => {
    if (!currentRoom) return;
    sendMessageToBackend('moveUp', { roomId: currentRoom.id, username: playerName });
  };

  const handleMoveDown = (playerName: string) => {
    if (!currentRoom) return;
    sendMessageToBackend('moveDown', { roomId: currentRoom.id, username: playerName });
  };

  const handleKickPlayer = (playerName: string) => {
    if (!currentRoom) return;
    sendMessageToBackend('kickPlayer', { roomId: currentRoom.id, username: playerName });
  };

  const handleLogout = async () => {
    const authConfig = await getAuthConfig().catch(() => ({ enabled: false }));
    if (authConfig.enabled) {
      dispatch(reset());
      window.location.assign('/auth/logout');
      return;
    }
    sendMessageToBackend('logout', {});
  };

  const userInRoom = !!currentRoom;
  const isWaitingHost = !!currentRoom && currentRoom.creator === username && !currentRoom.inProgress;
  const occupiedPlayers = currentRoom?.players.filter(player => player?.trim()).length ?? 0;
  const canCustomizeMap = isWaitingHost && currentRoom?.mapName === 'generatedmap' && saveSelectionId < 0;
  const mapSelection = currentRoom?.mapName === 'generatedmap' ? currentRoom.generatedMapSelection : undefined;
  const selectedRegionName = mapSelection?.regionName?.trim();
  const selectedCountChanged = !!mapSelection && mapSelection.playerCount !== Math.max(2, occupiedPlayers);
  const editor = generatedMapEditor?.roomId === currentRoom?.id ? generatedMapEditor : null;

  const sendEditorCommand = (type: string, data: Record<string, unknown>) => {
    if (sendMessageToBackend(type, data) === false) {
      dispatch(mapEditorFailed('Not connected. Reconnect and reopen the map editor to continue.'));
    }
  };
  const requestMapSettings = () => {
    if (!currentRoom) return;
    const requestId = `settings-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    dispatch(mapEditorRequested({ roomId: currentRoom.id, requestId, clear: true }));
    sendEditorCommand('getGeneratedMapSettings', { requestId });
  };
  const closeMapEditor = () => {
    setMapEditorOpen(false);
    setPendingMapChoice(null);
    dispatch(mapEditorClosed());
  };
  const generatePreview = (settings: GeneratedMapSettings) => {
    if (!currentRoom) return;
    const requestId = `preview-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    dispatch(mapEditorRequested({ roomId: currentRoom.id, requestId }));
    sendEditorCommand('previewGeneratedMap', { requestId, settings });
  };
  const selectPreview = (previewId: string) => {
    setPendingMapChoice(previewId);
    dispatch(mapEditorSelecting());
    sendEditorCommand('selectGeneratedMap', { previewId });
  };
  const selectAutomatic = () => {
    setPendingMapChoice('');
    dispatch(mapEditorSelecting());
    sendEditorCommand('resetGeneratedMap', {});
  };

  useEffect(() => {
    if (!mapEditorOpen) return;
    if (!canCustomizeMap) {
      setMapEditorOpen(false);
      setPendingMapChoice(null);
      dispatch(mapEditorClosed());
      return;
    }
    // Settings come from the backend preset for the actual occupied seats.
    requestMapSettings();
    // Request again only when this editor opens or the roster/map changes.
  }, [mapEditorOpen, canCustomizeMap, currentRoom?.id, occupiedPlayers, dispatch]);
  useEffect(() => {
    if (!mapEditorOpen || pendingMapChoice === null || !editor || editor.selecting) return;
    if (editor.error) { setPendingMapChoice(null); return; }
    if ((editor.selectedPreviewId ?? '') === pendingMapChoice) {
      setMapEditorOpen(false);
      setPendingMapChoice(null);
      dispatch(mapEditorClosed());
    }
  }, [mapEditorOpen, pendingMapChoice, editor, dispatch]);
  useEffect(() => {
    if (editorWasOpen.current && !mapEditorOpen) customizeMapButton.current?.focus();
    editorWasOpen.current = mapEditorOpen;
  }, [mapEditorOpen]);
  useEffect(() => () => { dispatch(mapEditorClosed()); }, [dispatch]);
  useEffect(() => {
    if (!mapEditorOpen || (!editor?.loading && !editor?.selecting)) return;
    const timeout = window.setTimeout(() => dispatch(mapEditorFailed('The map request timed out. Please try again.')), 75000);
    return () => window.clearTimeout(timeout);
  }, [mapEditorOpen, editor?.requestId, editor?.loading, editor?.selecting, dispatch]);
  useShortcut('m', () => setMapEditorOpen(true), canCustomizeMap);
  useShortcut('n', handleCreateRoom, !userInRoom && !!username.trim());
  useShortcut('s', () => setActiveTab(previous => previous === 'saves' ? 'dlc' : 'saves'));
  useShortcut('p', handleStartGame, isWaitingHost && !selectedCountChanged);
  useShortcut('v', handleEnterDisplayRoom, activeTab === 'saves' && !!username.trim());
  useShortcut('Shift+L', handleLeaveRoom, userInRoom);
  useShortcut('c', () => dispatch(clearError()), !!error);

  return (
    <div className="w-screen h-screen overflow-hidden bg-[#F5F5DC] font-serif text-[#5F4B32] relative">
      {mapEditorOpen && (editor?.defaults ? <GeneratedMapEditor
        defaultSettings={editor.defaults}
        initialSettings={editor.preview?.settings}
        preview={editor.preview ?? undefined}
        playerCount={Math.max(2, occupiedPlayers)}
        loading={editor.loading}
        selecting={editor.selecting}
        selectedPreviewId={editor.selectedPreviewId}
        automatic={!editor.selectedPreviewId}
        error={editor.error ?? undefined}
        onGenerate={generatePreview}
        onSelect={selectPreview}
        onAutomatic={selectAutomatic}
        onClose={closeMapEditor}
      /> : <div className="fixed inset-0 z-[80] flex items-center justify-center bg-black/60 p-4"
        role="dialog" aria-modal="true" aria-labelledby="map-editor-loading-title"
        onKeyDown={event => {
          event.stopPropagation();
          if (event.key === 'Escape') { event.preventDefault(); closeMapEditor(); }
          if (event.key === 'Tab') {
            const buttons = Array.from(event.currentTarget.querySelectorAll<HTMLButtonElement>('button'));
            const first = buttons[0], last = buttons[buttons.length - 1];
            if (event.shiftKey ? document.activeElement === first : document.activeElement === last) {
              event.preventDefault(); (event.shiftKey ? last : first)?.focus();
            }
          }
        }}>
        <div className="w-full max-w-lg rounded-xl border border-[#8B4513] bg-[#FDF5E6] p-6 shadow-2xl">
          <h2 id="map-editor-loading-title" className="text-2xl font-bold">Customize generated map</h2>
          <p role={editor?.error ? 'alert' : 'status'} className="my-4">{editor?.error ?? 'Loading the current map settings…'}</p>
          <div className="flex gap-3">
            {editor?.error && <button type="button" onClick={requestMapSettings} className="rounded bg-[#8B4513] px-4 py-2 text-white">Try again</button>}
            <button type="button" autoFocus onClick={closeMapEditor} data-tooltip="Return to the lobby · Esc"
              className="rounded border border-[#8B4513] px-4 py-2">Back to lobby</button>
          </div>
        </div>
      </div>)}
      <div className="flex w-full h-full">
        {/* Left column */}
        <div 
          className="w-2/3 h-full flex flex-col border-r border-[#5F4B32] relative"
          style={{
            backgroundImage: "url('/background.png')",
            backgroundSize: "cover",
            backgroundPosition: "center",
            backgroundRepeat: "no-repeat",
          }}
        >
          <div className="absolute inset-0 z-0"></div>
          <div className="relative z-10 flex-1 overflow-y-auto p-6 bg-white/0">
            {/* Header */}
            <div className="mb-6 flex items-center justify-between">
              <span className="text-lg font-semibold text-white">Logged in as: {username}</span>
              <button
                type="button" onClick={handleLogout} data-tooltip="Log out of Small World"
                className="bg-red-600 hover:bg-red-700 text-white py-1 px-4 rounded shadow-md transition-colors"
              >
                Log Out
              </button>
            </div>

            {/* Not in room: Create Room and Room List */}
{/* Not in room: Create Room and Room List */}
{!userInRoom && (
  <div className="space-y-8">
    <form onSubmit={event => { event.preventDefault(); handleCreateRoom(); }} className="p-6 bg-white rounded-lg shadow-md border border-[#5F4B32]">
      <h2 className="text-xl font-bold mb-4 underline">Create a Room</h2>
      <div className="mb-4">
        <label htmlFor="new-room-name" className="block font-semibold mb-1">Room Name:</label>
        <input
          id="new-room-name" type="text"
          className="w-full px-3 py-2 border rounded focus:outline-none focus:ring-2 focus:ring-[#8B4513]"
          value={roomName}
          onChange={(e) => setRoomName(e.target.value)}
          placeholder="Leave blank for a random room name"
          data-tooltip="Name your room, or leave blank for a random name like The Wobbly Tavern"
        />
      </div>
      <button
        type="submit" aria-keyshortcuts="N"
        data-tooltip="Create a room. Leave its name blank for a random name. Shortcut: N, or Enter from the name field."
        className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 rounded transition-colors shadow-md"
      >
        Create Room
      </button>
    </form>

    {/* Show Available Rooms ONLY if there are rooms */}
    {rooms?.length > 0 && (
      <div className="p-6 bg-white rounded-lg shadow-md border border-[#5F4B32]">
        <h2 className="text-xl font-bold mb-4 underline">Available Rooms</h2>
        <ul className="space-y-3">
          {rooms.map((rm) => (
            <li
              key={rm.id}
              className="flex items-center justify-between p-3 bg-gray-50 rounded border border-transparent hover:border-[#8B4513] transition-colors"
            >
              <div className="text-md font-medium">
                <strong>{rm.name}</strong>{' '}
                <span className="text-sm text-gray-600">
                  ({rm.players?.length || 0})
                </span>
              </div>
              <div className="flex space-x-2">
                <button
                  type="button"
                  onClick={() => handleSpectateRoom(rm.id)} data-tooltip={`Watch ${rm.name} as a spectator`}
                  className="bg-blue-500 hover:bg-blue-600 text-white py-1 px-3 rounded transition-colors"
                >
                  Spectate
                </button>
                <button
                  type="button"
                  onClick={() => handleJoinRoom(rm.id)} data-tooltip={`Join ${rm.name} as a player`}
                  className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded transition-colors"
                >
                  Join
                </button>
              </div>
            </li>
          ))}
        </ul>
      </div>
    )}

    {/* Show Ongoing Games ONLY if there are rooms in progress */}
    {roomsInProgress?.length > 0 && (
      <div className="p-6 bg-white rounded-lg shadow-md border border-[#5F4B32]">
        <h2 className="text-xl font-bold mb-4 underline">Ongoing Games</h2>
        <ul className="space-y-3">
          {roomsInProgress.map((rm) => (
            <li
              key={rm.id}
              className="flex items-center justify-between p-3 bg-gray-50 rounded border border-transparent hover:border-[#8B4513] transition-colors"
            >
              <div className="text-md font-medium">
                <strong>{rm.name}</strong>{' '}
                <span className="text-sm text-gray-600">
                  ({rm.players?.length || 0})
                </span>
              </div>
              <div className="flex space-x-2">
                <button
                  type="button"
                  onClick={() => handleSpectateRoom(rm.id)} data-tooltip={`Watch ${rm.name} as a spectator`}
                  className="bg-blue-500 hover:bg-blue-600 text-white py-1 px-3 rounded transition-colors"
                >
                  Spectate
                </button>
              </div>
            </li>
          ))}
        </ul>
      </div>
    )}
  </div>
)}

            {/* In room: Room Details */}
            {userInRoom && currentRoom && (
              <div className="p-6 bg-white rounded-lg shadow-md border border-[#5F4B32]">
                <h2 className="text-xl font-bold mb-4 underline">Room: {currentRoom.name}</h2>
                <div className="mb-4">
                  <p className="font-semibold">Selected Map: {currentRoom.mapName === 'generatedmap' && mapSelection ? selectedRegionName || 'Generated map · custom preview selected' : mapLabel(currentRoom.mapName)}</p>
                  {currentRoom.mapName === 'generatedmap' && <p className="mt-1 text-sm text-[#78674f]">
                    {mapSelection ? `${selectedRegionName ? 'Custom preview selected · ' : ''}${mapSelection.playerCount} players · Seed: ${mapSelection.seed}` : 'A fresh, balanced layout is generated when the game starts.'}
                  </p>}
                  {selectedCountChanged && <p role="status" className="mt-2 text-sm text-amber-800">The player count changed. Open Customize map and generate a new preview, or use automatic generation.</p>}
                  {canCustomizeMap && <button ref={customizeMapButton} type="button" onClick={() => setMapEditorOpen(true)} aria-keyshortcuts="M"
                    data-tooltip="Adjust settings, generate a preview, and select the map to play · M"
                    className="mt-3 rounded border border-[#8B4513] bg-[#FDF5E6] px-4 py-2 font-semibold text-[#8B4513] hover:bg-[#EED5B7] transition-colors">
                    Customize map
                  </button>}
                </div>
                {currentRoom.creator === username && mapChoices && mapChoices.length > 0 && (
                  <div className="mb-4">
                    <label htmlFor="lobby-room-map" className="font-semibold mr-2">Choose Map:</label>
                    <select id="lobby-room-map" value={currentRoom.mapName} onChange={handleMapChange}
                      data-tooltip="Choose the room's map; generated maps can be automatic or customized with a preview"
                      className="border p-2 rounded focus:outline-none">
                      <option value="">-- Select a Map --</option>
                      {mapChoices.map((map) => (
                        <option key={map} value={map}>
                          {mapLabel(map)}
                        </option>
                      ))}
                    </select>
                  </div>
                )}
                <RoomBotControls room={currentRoom} isHost={currentRoom.creator === username} />
                <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
                  <p className="font-semibold">Players in this room:</p>
                  {!currentRoom.players.includes(currentRoom.creator) && (
                    <span className="text-sm">{currentRoom.creator} · Host watching</span>
                  )}
                </div>
                <div className="space-y-3 mb-4">
                  {currentRoom.players?.map((p, idx) => {
                    if (!p || !p.trim().length) return (
                      <div key={`empty-${idx}`} className="p-3 rounded-lg border border-dashed border-[#8B4513]/40 bg-[#FDF5E6] text-sm">
                        <span className="font-semibold">{idx + 1}: Open seat</span>
                      </div>
                    );
                    const botSeat = currentRoom?.bots?.[p];
                    const modelName = currentRoom?.botModels?.find(model => model.id === botSeat?.modelId)?.name ?? 'Bot';
                    return (
                      <div
                        key={p}
                        className="flex items-center justify-between p-3 bg-[#EED5B7] rounded-lg shadow-sm"
                      >
                        <span className="font-semibold">
                          {idx + 1}: {currentRoom?.bots?.[p]?.name ?? p}
                          {botSeat && !botSeat.name.includes(modelName) && <span className="text-sm font-normal"> · {modelName}</span>}
                          {(playerStatuses?.[idx] ?? "") && playerStatuses[idx].trim() !== '' && (
                            <span className="text-sm font-normal"> | {playerStatuses?.[idx] ?? ""}</span>
                          )}
                        </span>
                        {currentRoom?.creator === username && (
                          <div className="flex flex-wrap gap-2">
                            <BotModelSelect room={currentRoom} username={p} />
                            <button
                              type="button" onClick={() => handleMoveUp(p)}
                              aria-label={`Move ${currentRoom?.bots?.[p]?.name ?? p} up`}
                              data-tooltip={`Move ${currentRoom?.bots?.[p]?.name ?? p} up in turn order`}
                              className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded"
                            >
                              ↑
                            </button>
                            <button
                              type="button" onClick={() => handleMoveDown(p)}
                              aria-label={`Move ${currentRoom?.bots?.[p]?.name ?? p} down`}
                              data-tooltip={`Move ${currentRoom?.bots?.[p]?.name ?? p} down in turn order`}
                              className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-1 px-3 rounded"
                            >
                              ↓
                            </button>
                            <button
                              type="button" onClick={() => handleKickPlayer(p)}
                              aria-label={`Remove ${currentRoom?.bots?.[p]?.name ?? p} from room`}
                              data-tooltip={`Remove ${currentRoom?.bots?.[p]?.name ?? p} from this room`}
                              className="bg-red-600 hover:bg-red-700 text-white py-1 px-3 rounded"
                            >
                              ✕
                            </button>
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
                <div className="flex space-x-4">
                  {currentRoom.creator === username && (
                    <button
                      type="button"
                      onClick={handleStartGame} disabled={selectedCountChanged} aria-keyshortcuts="P" data-tooltip="Start the game with the current map and rules · P"
                      className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-4 rounded transition-colors shadow-md"
                    >
                      {currentRoom.players.includes(currentRoom.creator) ? 'Start Game' : 'Start & Watch'}
                    </button>
                  )}
                  <button
                    type="button"
                    onClick={handleLeaveRoom} aria-keyshortcuts="Shift+L" data-tooltip="Leave this room · Shift + L"
                    className="bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 px-4 rounded transition-colors shadow-md"
                  >
                    Leave Room
                  </button>
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Right column with togglable menu */}
        <div className="w-1/3 h-full flex flex-col bg-[#FDF5E6] border-l border-[#5F4B32]">
          <div className="p-6 border-b border-[#5F4B32] flex items-center justify-between">
            <div className="text-2xl font-bold underline">Menu</div>
            <div className="space-x-4">
              <button
                type="button" onClick={() => setActiveTab('saves')} aria-pressed={activeTab === 'saves'}
                aria-keyshortcuts="S" data-tooltip="Show saved games · S switches tabs"
                className={`py-1 px-3 rounded ${
                  activeTab === 'saves'
                    ? 'bg-[#8B4513] text-white'
                    : 'bg-white text-[#5F4B32] border border-[#8B4513]'
                }`}
              >
                Saves
              </button>
              <button
                type="button" onClick={() => setActiveTab('dlc')} aria-pressed={activeTab === 'dlc'}
                aria-keyshortcuts="S" data-tooltip="Choose presets, races and traits · S switches tabs"
                className={`py-1 px-3 rounded ${
                  activeTab === 'dlc'
                    ? 'bg-[#8B4513] text-white'
                    : 'bg-white text-[#5F4B32] border border-[#8B4513]'
                }`}
              >
                Races & Traits
              </button>
            </div>
          </div>

          <div className="px-6 pt-4"><WikiToolbar compact /></div>

          {/* Content area of the right column */}
          <div className="flex-1 overflow-y-auto p-6">
            {activeTab === 'saves' && (
              <>
                <button
                  type="button"
                  onClick={handleEnterDisplayRoom} aria-keyshortcuts="V" data-tooltip="Open the map and saved-game viewer · V"
                  className="w-full bg-[#8B4513] hover:bg-[#A0522D] text-white py-2 rounded transition-colors shadow-md mb-6"
                >
                  Enter Display Room
                </button>
                {userInRoom && currentRoom && currentRoom.creator === username ? (
                  <>
                    {saveGames && saveGames.length > 0 ? (
                      <ul className="space-y-4">
                        {saveGames.map((gameSave) => (
                          <li
                            key={gameSave.saveId}
                            className={`flex items-center gap-2 p-4 bg-white rounded border-2 transition-colors ${
                              gameSave.saveId === saveSelectionId
                                ? 'border-[#8B4513]'
                                : 'border-transparent'
                            }`}
                          >
                            <button type="button" className="flex-1 min-w-0 text-left"
                              aria-label={`Load saved game ${gameSave.saveId}`}
                              data-tooltip="Load this saved game into the room · Enter / Space"
                              onClick={() => handleGameIdClick(gameSave.saveId)}>
                              <span className="block font-bold">ID: {gameSave.saveId}</span>
                              <span className="block text-sm text-gray-600">{gameSave.summary}</span>
                            </button>
                            <button type="button" aria-label={`Delete saved game ${gameSave.saveId}`}
                              data-tooltip="Delete this saved game"
                              onClick={(e) => {
                                e.stopPropagation();
                                handleDeleteGame(gameSave.saveId);
                              }}
                              className="shrink-0 bg-red-600 hover:bg-red-700 text-white py-1 px-3 rounded transition-colors"
                            >
                              Delete
                            </button>
                          </li>
                        ))}
                      </ul>
                    ) : (
                      <p className="text-gray-600">No saved games available.</p>
                    )}
                  </>
                ) : (
                  <p className="text-gray-600">You are not the owner</p>
                )}
              </>
            )}

            {activeTab === 'dlc' && (
              <>
                {userInRoom && currentRoom && currentRoom.creator === username ? (
                  <ExtraChoices room={currentRoom} saveSelectionId={saveSelectionId} />
                ) : (
                  <p className="text-gray-600">You are not the owner</p>
                )}
              </>
            )}
          </div>
        </div>
      </div>

      {/* Error Banner at the Bottom */}
      {error && (
        <div role="alert" className="fixed bottom-0 left-0 w-full bg-red-500 text-white py-2 px-4 z-50 flex items-center justify-center gap-3">
          <span>{error}</span>
          <button type="button" onClick={() => dispatch(clearError())} aria-label="Dismiss error"
            aria-keyshortcuts="C" data-tooltip="Dismiss this message · C"
            className="shrink-0 rounded border border-white/70 px-2 py-1 text-sm hover:bg-white/15">
            Dismiss <kbd className="ml-1 text-xs">C</kbd>
          </button>
        </div>
      )}
    </div>
  );
}

function ExtraChoices({ room, saveSelectionId }: { room: Room; saveSelectionId: number }) {
  const { extensionChoices, globalToggle } = useSelector((state: RootState) => ({
    extensionChoices: state.application.extensionChoices,
    globalToggle: state.application.globalToggle,
  }), shallowEqual);

  const matchesPreset = (preset: 'standard' | 'underground') => extensionChoices.length > 0 &&
    extensionChoices.every(extension => {
      const underground = ['Underground', 'Powers'].includes(extension.extensionName);
      const checked = preset === 'underground' ? underground : !underground;
      return extension.isChecked === checked &&
        [...extension.raceChoices, ...extension.traitChoices].every(choice => choice.isChecked === checked);
    });
  const activePreset = room.mapName === 'generatedmap' && matchesPreset('standard') ? 'standard'
    : room.mapName === 'underground2players' && matchesPreset('underground') ? 'underground' : null;
  const applyPreset = (preset: 'standard' | 'underground') => sendMessageToBackend('setRulesPreset', { preset });
  useShortcut('n', () => applyPreset('standard'), !room.inProgress);
  useShortcut('u', () => applyPreset('underground'), !room.inProgress);

  const handleToggleAll = (checked: boolean) => {
    sendMessageToBackend('toggleAll', { checked });
  };

  const handleToggleExtension = (extensionName: string, checked: boolean) => {
    sendMessageToBackend('toggleExtension', {
      extensionName,
      checked,
    });
  };

  const handleToggleRace = (
    extensionName: string,
    raceChoice: string,
    checked: boolean
  ) => {
    sendMessageToBackend('toggleRace', {
      extensionName,
      raceChoice,
      checked,
    });
  };

  const handleToggleTrait = (
    extensionName: string,
    traitChoice: string,
    checked: boolean
  ) => {
    sendMessageToBackend('toggleTrait', {
      extensionName,
      traitChoice,
      checked,
    });
  };

  return (
    <div className="mt-4 p-4 bg-white rounded border border-[#8B4513]">
      <section aria-label="Game preset" className="mb-5 rounded border border-[#8B4513]/40 bg-[#FDF5E6] p-3">
        <div className="flex flex-wrap items-center justify-between gap-2 mb-3">
          <h3 className="font-semibold">Game preset</h3>
          <span className="text-xs" role="status">{activePreset === 'standard' ? 'Standard selected' : activePreset === 'underground' ? 'Underground selected' : 'Custom setup'}</span>
        </div>
        <div className="flex rounded border border-[#8B4513] overflow-hidden" role="group" aria-label="Choose a game preset">
          <button type="button" disabled={room.inProgress} aria-pressed={activePreset === 'standard'}
            aria-keyshortcuts="N" data-tooltip="Select Standard expansions and a generated map · N"
            onClick={() => applyPreset('standard')}
            className={`flex-1 px-3 py-2 text-sm disabled:opacity-50 ${activePreset === 'standard' ? 'bg-[#8B4513] text-white' : 'hover:bg-[#EED5B7]'}`}>
            Standard
          </button>
          <button type="button" disabled={room.inProgress} aria-pressed={activePreset === 'underground'}
            aria-keyshortcuts="U" data-tooltip="Select Underground, relics and places, and the Underground map · U"
            onClick={() => applyPreset('underground')}
            className={`flex-1 border-l border-[#8B4513] px-3 py-2 text-sm disabled:opacity-50 ${activePreset === 'underground' ? 'bg-[#8B4513] text-white' : 'hover:bg-[#EED5B7]'}`}>
            Underground
          </button>
        </div>
        <p className="text-xs leading-relaxed mt-3">Standard uses a random map and all expansions except Underground and Powers. Underground uses its two-player map, races, traits, relics and places.</p>
        <p className="text-xs mt-2">You can customize the selections below after choosing a preset.</p>
        {Object.keys(room.bots ?? {}).length > 0 && <p className="text-xs mt-2">These presets include rules bots may not support. Use bot rules before starting a bot game.</p>}
        {saveSelectionId >= 0 && <p className="text-xs font-semibold mt-2">Choosing a preset clears the selected save and sets up a new game.</p>}
      </section>
      <div className="mb-4">
        <label className="flex items-center space-x-2" data-tooltip="Select or exclude every expansion, race and trait · Space toggles the focused checkbox">
          <input
            type="checkbox"
            checked={globalToggle} data-tooltip="Select or exclude every expansion, race and trait · Space"
            onChange={(e) => handleToggleAll(e.target.checked)}
            className="appearance-none w-5 h-5 border-2 border-[#8B4513] rounded-md checked:bg-[#8B4513] checked:border-[#5F4B32] focus:outline-none focus:ring-2 focus:ring-[#A0522D]"
          />
          <span>Toggle All</span>
        </label>
      </div>

      <div className="space-y-6">
        {extensionChoices.map((ext, index) => (
          <div key={index} className="p-4 border rounded border-[#8B4513]">
            <label className="flex items-center space-x-2 mb-4" data-tooltip={`Select or exclude all choices in ${ext.extensionName} · Space toggles the focused checkbox`}>
              <input
                type="checkbox"
                checked={ext.isChecked} data-tooltip={`Select or exclude all choices in ${ext.extensionName} · Space`}
                onChange={(e) =>
                  handleToggleExtension(ext.extensionName, e.target.checked)
                }
                className="appearance-none w-5 h-5 border-2 border-[#8B4513] rounded-md checked:bg-[#8B4513] checked:border-[#5F4B32] focus:outline-none focus:ring-2 focus:ring-[#A0522D]"
              />
              <span>{ext.extensionName}</span>
            </label>
            <div className="flex space-x-8">
              <div className="w-1/2">
                <h4 className="font-bold text-md mb-2">Races</h4>
                <div className="space-y-2">
                  {ext.raceChoices.map((race, rIdx) => (
                    <label key={rIdx} className="flex items-center space-x-2" data-tooltip={`Select or exclude ${race.choice} · Space toggles the focused checkbox`}>
                      <input
                        type="checkbox"
                        checked={race.isChecked} data-tooltip={`Select or exclude ${race.choice} from this game's races · Space`}
                        onChange={(e) =>
                          handleToggleRace(
                            ext.extensionName,
                            race.choice,
                            e.target.checked
                          )
                        }
                        className="appearance-none w-5 h-5 border-2 border-[#8B4513] rounded-md checked:bg-[#8B4513] checked:border-[#5F4B32] focus:outline-none focus:ring-2 focus:ring-[#A0522D]"
                      />
                      <span>{race.choice}</span>
                    </label>
                  ))}
                </div>
              </div>
              <div className="w-1/2">
                <h4 className="font-bold text-md mb-2">Traits</h4>
                <div className="space-y-2">
                  {ext.traitChoices.map((trait, tIdx) => (
                    <label key={tIdx} className="flex items-center space-x-2" data-tooltip={`Select or exclude ${trait.choice} · Space toggles the focused checkbox`}>
                      <input
                        type="checkbox"
                        checked={trait.isChecked} data-tooltip={`Select or exclude ${trait.choice} from this game's traits · Space`}
                        onChange={(e) =>
                          handleToggleTrait(
                            ext.extensionName,
                            trait.choice,
                            e.target.checked
                          )
                        }
                        className="appearance-none w-5 h-5 border-2 border-[#8B4513] rounded-md checked:bg-[#8B4513] checked:border-[#5F4B32] focus:outline-none focus:ring-2 focus:ring-[#A0522D]"
                      />
                      <span>{trait.choice}</span>
                    </label>
                  ))}
                </div>
              </div>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
