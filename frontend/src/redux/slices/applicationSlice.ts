import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { ApplicationState } from '../../types/redux';
import { Language } from '../../types/misc';
import { Player, TribeEntry, Room, Tile, SaveGameInfo } from '../../types/Board';
import wikiMapMetadata from '../../data/wikiMapMetadata.json';
import { mapDatabase, MapData } from '../../data/mapData'; // <-- Import your local map data

const generatedMapNames = new Set<string>();

function base64PngToObjectUrl(base64: string): string {
  const byteString = atob(base64);
  const array = new Uint8Array(byteString.length);

  for (let i = 0; i < byteString.length; i++) {
    array[i] = byteString.charCodeAt(i);
  }

  const imageBlob = new Blob([array], { type: 'image/png' });
  return URL.createObjectURL(imageBlob);
}

function applyMapDataToState(state: any, mapName: string, mapData: MapData) {
  state.mapName = mapName;
  state.offsetStacks = mapData.OffsetStacks;
  state.Xmult = mapData.Xmult;
  state.Ymult = mapData.Ymult;

  const newTiles: Record<string, Tile> = {};

  mapData.Tiles.forEach((tileDef) => {
    newTiles[String(tileDef.ID)] = {
      id: String(tileDef.ID),
      polygon: {
        coords: tileDef.Polygon.Coords,
        stackX: tileDef.Polygon.StackX,
        stackY: tileDef.Polygon.StackY,
      },
      pieceStack: [],
      terrain: (wikiMapMetadata as Record<string, Record<string, NonNullable<Tile['terrain']>>>)[mapName]?.[String(tileDef.ID)],
    };
  });

  state.tiles = newTiles;
}

function regionMetadata(data: { regionName?: unknown; regionNameVersion?: unknown }) {
  const regionName = typeof data.regionName === 'string' ? data.regionName.trim() || undefined : undefined;
  const regionNameVersion = regionName && Number.isInteger(data.regionNameVersion) && Number(data.regionNameVersion) > 0
    ? Number(data.regionNameVersion) : undefined;
  return { regionName, regionNameVersion };
}

function clearMapRegionName(state: ApplicationState) {
  state.mapRegionName = null;
  state.mapRegionNameVersion = null;
}

function reconcileSelectedStack(state: ApplicationState) {
  if (state.selectedStack == null) return;
  const stacks = state.isStackFromBank
    ? state.players[state.playerIndex]?.pieceStacks
    : state.selectedTile == null ? undefined : state.tiles[state.selectedTile]?.pieceStack;
  if (stacks?.some(stack => stack.type === state.selectedStack)) return;
  state.selectedStack = null;
  state.selectedTile = null;
  state.isStackFromBank = false;
}

const initialState: ApplicationState = {
  generatedMapEditor: null,
  language: Language.NL,
  error: null,
  availableTribes: [
    { race: 'elves', trait: 'merchant', pieceCount: 0, coinCount: 0 },
    { race: 'giants', trait: 'fortunate', pieceCount: 0, coinCount: 0 },
  ], // dummy
  tiles: {},
  offsetMapTiles: 1,
  players: [],
  playerIndex: 1,
  turnNumber: 1,
  playerNumber: 1,
  phase: 'tribechoice',
  coins: 5,
  selectedStack: null,
  isStackFromBank: false,
  selectedTile: null,
  messages: [],
  scores: [],
  rooms: [],
  roomsInProgress: [],
  roomid: "",
  name: "",
  gameStarted: false,
  mapImageUrl: null,
  isAuthenticated: false,
  saveGames: [],
  saveSelectionId: -1,
  mapName: null,
  mapRegionName: null,
  mapRegionNameVersion: null,
  offsetStacks: 10,
  mapChoices: [],
  playerStatuses: [],
  Xmult: 1,
  Ymult: 1,
  inDisplayRoom: false,
  isSpectating: false,
  extensionChoices: [],
  globalToggle: true,
};

const applicationSlice = createSlice({
  name: 'application',
  initialState,
  reducers: {
    reset: (state) => {
      if (state.generatedMapEditor?.preview?.imageUrl) URL.revokeObjectURL(state.generatedMapEditor.preview.imageUrl);
      return initialState;
    },
    mapEditorRequested(state, action: PayloadAction<{ roomId: string; requestId: string; clear?: boolean }>) {
      const { roomId, requestId, clear } = action.payload;
      if (clear || state.generatedMapEditor?.roomId !== roomId) {
        if (state.generatedMapEditor?.preview?.imageUrl) URL.revokeObjectURL(state.generatedMapEditor.preview.imageUrl);
        state.generatedMapEditor = { roomId, requestId, playerCount: 2, defaults: null,
          preview: null, loading: true, selecting: false, error: null };
      } else if (state.generatedMapEditor) {
        state.generatedMapEditor.requestId = requestId;
        state.generatedMapEditor.loading = true;
        state.generatedMapEditor.error = null;
      }
    },
    mapEditorSelecting(state) {
      if (state.generatedMapEditor) {
        state.generatedMapEditor.selecting = true;
        state.generatedMapEditor.error = null;
      }
    },
    mapEditorFailed(state, action: PayloadAction<string>) {
      if (state.generatedMapEditor) {
        state.generatedMapEditor.loading = false;
        state.generatedMapEditor.selecting = false;
        state.generatedMapEditor.error = action.payload;
      }
    },
    mapEditorClosed(state) {
      if (state.generatedMapEditor?.preview?.imageUrl) URL.revokeObjectURL(state.generatedMapEditor.preview.imageUrl);
      state.generatedMapEditor = null;
    },

    setLanguage(state, action: PayloadAction<Language>) {
      state.language = action.payload;
    },
    setName(state, action) {
      state.name = action.payload
    },

    setSelectedStack(state, action) {
      state.selectedStack = action.payload;
    },

    setSelectedTile(state, action) {
      state.selectedTile = action.payload;
    },

    setIsStackFromBank(state, action) {
      state.isStackFromBank = action.payload;
    },

    clearError(state) {
      state.error = null;
    },
    setError(state, action: PayloadAction<string>) {
      state.error = action.payload;
    },

    setPlayers(state, action) {
      state.players = action.payload;
    },

    setScores(state, action) {
      state.scores = action.payload;
    },

    setTiles(state, action) {
      state.tiles = action.payload;
    },

    updateTileStack(state, action) {
      const { tile_id, new_stacks } = action.payload;
      const tile = state.tiles[tile_id];
      if (!tile) {
        throw new Error(`Tile with ID ${tile_id} does not exist.${state.tiles}`);
      }
      tile.pieceStack = new_stacks;
    },

    websocketMessageReceived(state, action) {
      const { type, data } = JSON.parse(action.payload);

      const parsedData = data;
      switch (type) {
        case 'generatedmapsettings':
        case 'generatedmappreview': {
          const editor = state.generatedMapEditor;
          if (!editor || editor.roomId !== parsedData.roomId || editor.requestId !== parsedData.requestId) break;
          editor.loading = false;
          editor.error = null;
          editor.playerCount = parsedData.playerCount;
          if (type === 'generatedmapsettings') {
            editor.defaults = parsedData.settings;
            editor.selectedPreviewId = parsedData.selectedPreviewId || undefined;
          }
          const preview = type === 'generatedmappreview' ? parsedData : parsedData.preview;
          if (editor.preview?.imageUrl) URL.revokeObjectURL(editor.preview.imageUrl);
          editor.preview = preview ? { id: preview.previewId, imageUrl: base64PngToObjectUrl(preview.imageBase64),
            settings: preview.settings, playerCount: preview.playerCount, ...regionMetadata(preview) } : null;
          break;
        }
        case 'generatedmapselection': {
          const editor = state.generatedMapEditor;
          const preview = editor && editor.roomId === parsedData.roomId && editor.preview?.id === parsedData.previewId ? editor.preview : undefined;
          for (const room of [...state.rooms, ...state.roomsInProgress]) {
            if (room.id !== parsedData.roomId || room.mapName !== 'generatedmap') continue;
            if (!parsedData.previewId) {
              delete room.generatedMapSelection;
              continue;
            }
            const previous = room.generatedMapSelection?.previewId === parsedData.previewId ? room.generatedMapSelection : undefined;
            const seed = parsedData.seed ?? preview?.settings.seed ?? previous?.seed;
            const playerCount = parsedData.playerCount ?? preview?.playerCount ?? previous?.playerCount;
            if (typeof seed === 'string' && Number.isInteger(playerCount)) {
              room.generatedMapSelection = { previewId: parsedData.previewId, playerCount, seed,
                ...regionMetadata({ ...previous, ...preview, ...parsedData }) };
            }
          }
          if (editor && editor.roomId === parsedData.roomId) {
            editor.selecting = false;
            editor.error = null;
            editor.selectedPreviewId = parsedData.previewId || undefined;
          }
          break;
        }
        case 'generatedmaperror': {
          const editor = state.generatedMapEditor;
          if (!editor || editor.roomId !== parsedData.roomId ||
              (parsedData.requestId && editor.requestId !== parsedData.requestId)) break;
          editor.loading = false;
          editor.selecting = false;
          editor.error = parsedData.message;
          break;
        }
        // ---------------------------------------------------------------------
        // NEW: Rooms Lobby Management
        // ---------------------------------------------------------------------
        // case 'gamestarted': {
        //   state.gameStarted = true
        //   break;
        // }
        case 'auth': {
          state.gameStarted = false
          state.isAuthenticated = true
          state.error = null

          state.name = parsedData.name
          break;
        }
        case 'unauth': {
          clearMapRegionName(state);
          state.gameStarted = false
          state.isAuthenticated = false
          state.roomid = ""
          state.error = null

          state.name = ""
          break;
        }
        case 'loadSaves' : 
          state.saveGames = parsedData.saves
          break;
        case 'playerStatuses' :
          state.playerStatuses = parsedData
          break;
        case 'saveSelection' : 
          state.saveSelectionId = parsedData.index
          break;
        case 'roomEntriesUpdate': {
          if (parsedData != null) {
            state.rooms = parsedData;
          } else {
            state.rooms = []
          }
          break;
        }
        case 'roomsInProgress': {
          if (parsedData != null) {
            state.roomsInProgress = parsedData;
          } else {
            state.roomsInProgress = []
          }
          break;
        }
        case 'index':
          state.playerIndex = Number(parsedData.index);
          break;
        case 'roomid':
          if (state.roomid !== parsedData.roomid) clearMapRegionName(state);
          state.roomid = parsedData.roomid;
          break;
        case 'lobby':
          clearMapRegionName(state);
          state.roomid = ""
          state.gameStarted = false
          state.isSpectating = false
          state.error = null
          break;
        case 'displayroom':
          clearMapRegionName(state);
          state.inDisplayRoom = true
          state.gameStarted = false
          state.isSpectating = false
          state.saveSelectionId = -1
          state.error = null
          break;
        case 'spectate':
          state.inDisplayRoom = false
          state.gameStarted = true
          state.isSpectating = true
          state.error = null
          break;
        case 'leavedisplayroom':
          clearMapRegionName(state);
          state.gameStarted = false
          state.inDisplayRoom = false
          state.saveSelectionId = -1
          state.error = null
          break;
        case 'error':
          if (state.generatedMapEditor?.loading || state.generatedMapEditor?.selecting) {
            state.generatedMapEditor.loading = false;
            state.generatedMapEditor.selecting = false;
            state.generatedMapEditor.error = parsedData.message;
          }
          state.error = parsedData.message;
          state.messages.push(JSON.stringify(parsedData.message));
          break;

        // ---------------------------------------------------------------------
        // Game-Related
        // ---------------------------------------------------------------------
        case 'mapupdate': {
          // Legacy image refreshes have no map identity, so retain its name.
          if (data.picture) {
            const byteString = atob(data.picture);
            const array = new Uint8Array(byteString.length);

            for (let i = 0; i < byteString.length; i++) {
              array[i] = byteString.charCodeAt(i);
            }

            const imageBlob = new Blob([array], { type: 'image/png' });
            const imgUrl = URL.createObjectURL(imageBlob);

            state.mapImageUrl = imgUrl;
          }
          state.offsetMapTiles = parsedData.offset
          if (!Array.isArray(data.zones)) break;
          const newTiles: Record<string, Tile> = {};
          data.zones.forEach((tileData: any, index: number) => {
            const tileId = String(tileData.id);
            newTiles[tileId] = {
              id: tileId,
              polygon: {
                coords: tileData.polygon.coords,
                stackX: tileData.polygon.stackX,
                stackY: tileData.polygon.stackY,
              },
              pieceStack: [],
            };
          });
          state.tiles = newTiles;
          break;
        }
        case 'mapChoices': {
          state.mapChoices = parsedData.mapChoices
          break;
        }
        case 'choices': {
          state.extensionChoices = parsedData.extensionChoices
          state.globalToggle = parsedData.globalToggle
          break;
        }
        // case 'smallmapupdate': {
        //   state.offsetMapTiles = parsedData.offset;
        //   state.mapName = parsedData.MapName;
        //
        //
        //   // Load tile definitions from your local map data
        //   const mapKey = state.mapName || '';
        //   const tileDataArray = mapDatabase[mapKey] || [];
        //
        //   state.offsetStacks = tileDataArray.OffsetStacks
        //   state.Xmult = tileDataArray.Xmult
        //   state.Ymult = tileDataArray.Ymult
        //
        //   const newTiles: Record<string, Tile> = {};
        //   tileDataArray.Tiles.forEach((tileDef) => {
        //     newTiles[String(tileDef.ID)] = {
        //       id: String(tileDef.ID),
        //       polygon: {
        //         coords: tileDef.Polygon.Coords,
        //         stackX: tileDef.Polygon.StackX,
        //         stackY: tileDef.Polygon.StackY,
        //       },
        //       pieceStack: [],
        //     };
        //   });
        //
        //   state.tiles = newTiles;
        //   break;
        // }
        case 'smallmapupdate': {
          const nextMapName = parsedData.mapName ?? parsedData.MapName;
          const sameGeneratedMap = state.mapName === nextMapName && generatedMapNames.has(nextMapName);
          state.offsetMapTiles = parsedData.offset;
          // A spectator joining can refresh this map without resending visuals
          // to existing players. Preserve its image, name, terrain and stacks.
          if (sameGeneratedMap) break;
          clearMapRegionName(state);
          state.mapImageUrl = null;
          state.mapName = nextMapName;

          const mapKey = state.mapName || '';
          const mapData = mapDatabase[mapKey];

          if (!mapData) {
            // This is expected for runtime generated maps.
            // The generatedmapvisuals message will arrive next and register the map.
            state.tiles = {};
            break;
          }

          // Builtin maps use static files from /maps.
          state.mapImageUrl = null;

          applyMapDataToState(state, mapKey, mapData);
          break;
        }
        case 'generatedmapvisuals': {
          const { regionName, regionNameVersion } = regionMetadata(parsedData);
          state.mapRegionName = regionName ?? null;
          state.mapRegionNameVersion = regionNameVersion ?? null;
          const mapName = parsedData.mapName;
          const graphics = parsedData.graphics as MapData;

          // Register this generated map exactly like a builtin map.
          generatedMapNames.add(mapName);
          mapDatabase[mapName] = graphics;

          // Store the generated PNG as an object URL.
          state.mapImageUrl = base64PngToObjectUrl(parsedData.imageBase64);

          // Generated maps should always use the backend-provided offset.
          state.offsetMapTiles = parsedData.offset ?? 0.808;

          applyMapDataToState(state, mapName, graphics);
          for (const tile of parsedData.serverTiles ?? []) {
            const region = state.tiles[String(tile.id)];
            if (region) region.terrain = {
              biome: tile.biome, attributes: tile.attributes ?? [],
              isEdge: tile.isEdge, isSkyIsland: tile.isSkyIsland,
            };
          }

          console.log('Registered generated map:', mapName);
          console.log('Generated server tile metadata:', parsedData.serverTiles);

          break;
        }
        case 'playerupdate': {
          const players: Player[] = [];
          for (let i = 0; i < parsedData.length; i++) {
            const playerData = parsedData[i];
            const player: Player = {
              name: playerData.name,
              activeTribe: {
                race: playerData.activeTribe.race,
                trait: playerData.activeTribe.trait,
              },
              passiveTribes: playerData.passiveTribes ?? [],
              pieceStacks: [],
            };
            if (playerData.pieceStacks && Array.isArray(playerData.pieceStacks)) {
              for (const stack of playerData.pieceStacks) {
                player.pieceStacks.push({
                  type: stack.type,
                  amount: stack.amount,
                  isActive: stack.isActive,
                });
              }
            }
            players.push(player);
          }
          state.players = players;
          reconcileSelectedStack(state);
          break;
        }
        case 'tribeentries':
          console.log(parsedData)
          state.availableTribes = parsedData;
          break;

        case 'tileupdate': {
          const tile = state.tiles[parsedData.tileID];
          if (!tile) {
            console.error(`Tile with ID ${parsedData.tileID} does not exist.`);
            return;
          }
          const stacks = [];
          for (const stack of parsedData.stacks) {
            stacks.push({
              type: stack.type,
              amount: stack.amount,
              isActive: stack.isActive,
            });
          }
          tile.pieceStack = stacks;
          break;
        }
        case 'alltileupdate': {
          for (let i = 0; i < parsedData.length; i++) {
            const t = parsedData[i];
            const tile = state.tiles[t.tileID];
            if (!tile) {
              console.error(`Tile with ID ${t.tileID} does not exist.`);
              continue;
            }
            const stacks = [];
            if (t.stacks && Array.isArray(t.stacks)) {
              for (const stack of t.stacks) {
                stacks.push({
                  type: stack.type,
                  amount: stack.amount,
                  isActive: stack.isActive,
                });
              }
            }
            tile.pieceStack = stacks;
          }
          break;
        }
        case 'turnupdate':
          state.playerNumber = parsedData.playerNumber;
          state.turnNumber = parsedData.turnNumber;
          state.phase = parsedData.phase;
          break;

        case 'tribeentries': {
          const tribeEntries: TribeEntry[] = [];
          for (let i = 0; i < parsedData.length; i++) {
            const tribeData = parsedData[i];
            tribeEntries.push({
              race: tribeData.Race,
              trait: tribeData.Trait,
              pieceCount: tribeData.piecepile,
              coinCount: tribeData.coinpile,
            });
          }
          state.availableTribes = tribeEntries;
          break;
        }
        case 'gamefinished':
          state.scores = parsedData;
          state.phase = 'GameFinished';
          state.error = null
          break;

        case 'message':
          state.messages.push(JSON.stringify(parsedData.message));
          break;
        case 'coins':
          state.coins = parsedData.coins
          break;
       case 'megaUpdate': {
          // This is your combined big update.
          // Pull out the relevant fields one by one.

          // 1) Turn info
          state.playerNumber = parsedData.turnInfo.playerNumber;
          state.turnNumber   = parsedData.turnInfo.turnNumber;
          state.phase        = parsedData.turnInfo.phase;

          // 2) Players
          const players: Player[] = [];
          if (parsedData.players != null) {
            for (let i = 0; i < parsedData.players.length; i++) {
              const pData = parsedData.players[i];
              const tempPlayer: Player = {
                name: pData.name,
                activeTribe: {
                  race: pData.activeTribe.race,
                  trait: pData.activeTribe.trait,
                },
                passiveTribes: [],
                pieceStacks: [],
              };

              // Passive tribes
              if (pData.passiveTribes && Array.isArray(pData.passiveTribes)) {
                for (const t of pData.passiveTribes) {
                  tempPlayer.passiveTribes.push({
                    race: t.race,
                    trait: t.trait,
                  });
                }
              }

              // Piece stacks
              if (pData.pieceStacks && Array.isArray(pData.pieceStacks)) {
                for (const stack of pData.pieceStacks) {
                  tempPlayer.pieceStacks.push({
                    type: stack.type,
                    amount: stack.amount,
                    isActive: stack.isActive,
                  });
                }
              }

              players.push(tempPlayer);
            }
          }
          state.players = players;

          // 3) Tribe entries
          const newTribeEntries: TribeEntry[] = [];
          if (Array.isArray(parsedData.tribeEntries)) {
            parsedData.tribeEntries.forEach((entry: any) => {
              newTribeEntries.push({
                race: entry.race,
                trait: entry.trait,
                pieceCount: entry.pieceCount || entry.piecePile,  // depending on naming
                coinCount: entry.coinCount || entry.coinPile,      // depending on naming
              });
            });
          }
          state.availableTribes = newTribeEntries;

          // 4) All tiles
          if (Array.isArray(parsedData.allTiles)) {
            for (const t of parsedData.allTiles) {
              const tileId = t.tileID;
              const tileObj = state.tiles[tileId];
              if (!tileObj) {
                console.error('Tile does not exist:', tileId);
                continue;
              }
              const stacks = [];
              for (const stack of t.stacks) {
                stacks.push({
                  type: stack.type,
                  amount: stack.amount,
                  isActive: stack.isActive,
                });
              }
              tileObj.pieceStack = stacks;
            }
          }
          reconcileSelectedStack(state);

          // 5) Next player index (if needed)
          // e.g., you might store it or do something else with it:
          // state.playerNumber = parsedData.nextPlayerIndex;

          // 6) Optionally mark the game as started, etc.
          state.gameStarted = true;

          break;
        }

        default:
          console.warn('Unhandled WebSocket message type:', data);
          break;
      }
    },
  },
});

export const applicationReducer = applicationSlice.reducer;

// Export the auto-generated actions
export const {
  reset,
  mapEditorRequested,
  mapEditorSelecting,
  mapEditorFailed,
  mapEditorClosed,
  setLanguage,
  setName,
  setSelectedTile,
  setSelectedStack,
  setIsStackFromBank,
  setTiles,
  clearError,
  setError,
  websocketMessageReceived,
  updateTileStack,
  setPlayers,
} = applicationSlice.actions;

export default applicationReducer;
