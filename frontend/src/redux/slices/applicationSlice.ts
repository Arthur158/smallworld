import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { ApplicationState } from '../../types/redux';
import { Language } from '../../types/misc';
import { Player, TribeEntry, Room, Tile, SaveGameInfo } from '../../types/Board';
import { mapDatabase, MapData } from '../../data/mapData'; // <-- Import your local map data


const initialState: ApplicationState = {
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
  selectedStack: null,
  isStackFromBank: false,
  selectedTile: null,
  messages: [],
  scores: [],
  rooms: [],
  roomid: "",
  name: "",
  gameStarted: false,
  mapImageUrl: null,
  isAuthenticated: false,
  saveGames: [],
  saveSelectionId: -1,
  mapName: null,
  offsetStacks: 10,
  mapChoices: [],
  playerStatuses: [],
  Xmult: 1,
  Ymult: 1,
  inDisplayRoom: false
};

const applicationSlice = createSlice({
  name: 'application',
  initialState,
  reducers: {
    reset: () => initialState,

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

          state.name = parsedData.name
          break;
        }
        case 'unauth': {
          state.gameStarted = false
          state.isAuthenticated = false
          state.roomid = ""

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
          }
          break;
        }
        case 'index':
          state.playerIndex = Number(parsedData.index);
          break;
        case 'roomid':
          state.roomid = parsedData.roomid;
          break;
        case 'lobby':
          state.roomid = ""
          state.gameStarted = false
          break;
        case 'displayroom':
          state.inDisplayRoom = true
          state.gameStarted = false
          state.saveSelectionId = -1
          break;
        case 'leavedisplayroom':
          state.gameStarted = false
          state.inDisplayRoom = false
          state.saveSelectionId = -1
          break;
        case 'error':
          state.error = parsedData.message;
          state.messages.push(JSON.stringify(parsedData.message));
          break;

        // ---------------------------------------------------------------------
        // Game-Related
        // ---------------------------------------------------------------------
        case 'mapupdate': {
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
          state.mapName = parsedData.MapName
          break;
        }
        case 'smallmapupdate': {
          state.offsetMapTiles = parsedData.offset;
          state.mapName = parsedData.MapName;


          // Load tile definitions from your local map data
          const mapKey = state.mapName || '';
          const tileDataArray = mapDatabase[mapKey] || [];

          state.offsetStacks = tileDataArray.OffsetStacks
          state.Xmult = tileDataArray.Xmult
          state.Ymult = tileDataArray.Ymult

          const newTiles: Record<string, Tile> = {};
          tileDataArray.Tiles.forEach((tileDef) => {
            newTiles[String(tileDef.ID)] = {
              id: String(tileDef.ID),
              polygon: {
                coords: tileDef.Polygon.Coords,
                stackX: tileDef.Polygon.StackX,
                stackY: tileDef.Polygon.StackY,
              },
              pieceStack: [],
            };
          });

          state.tiles = newTiles;
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
              passiveTribes: [],
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
          break;

        case 'message':
          state.messages.push(JSON.stringify(parsedData.message));
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
