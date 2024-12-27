import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { ApplicationState } from '../../types/redux';
import { Language } from '../../types/misc';
import { Player, TribeEntry, Room } from '../../types/Board';


const initialState: ApplicationState = {
  language: Language.NL,
  error: null,
  availableTribes: [
    { race: 'elves', trait: 'merchant', pieceCount: 0, coinCount: 0 },
    { race: 'giants', trait: 'fortunate', pieceCount: 0, coinCount: 0 },
  ], // dummy
  tiles: {},
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
  gameStarted: false
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
        throw new Error(`Tile with ID ${tile_id} does not exist.`);
      }
      tile.pieceStack = new_stacks;
    },

    websocketMessageReceived(state, action) {
      const { type, data } = JSON.parse(action.payload);
      state.error = null; // clear previous errors on new message

      const parsedData = data;
      switch (type) {
        // ---------------------------------------------------------------------
        // NEW: Rooms Lobby Management
        // ---------------------------------------------------------------------
        case 'gamestarted': {
          state.gameStarted = true
          break;
      }
        case 'roomEntriesUpdate': {
          // Expecting an array of rooms
          // E.g. data = [ { id, name, players: [...], ...}, {...}, ... ]
          state.rooms = parsedData;
          // If user already has a selectedRoom, update it if it changed
          break;
        }

        case 'index':
          console.log('the index:', parsedData);
          state.playerIndex = Number(parsedData.index);
          break;
        case 'roomid':
          state.roomid = parsedData.roomid;
          break;

        case 'error':
          state.error = parsedData.message;
          state.messages.push(JSON.stringify(parsedData.message));
          break;

        // ---------------------------------------------------------------------
        // Game-Related
        // ---------------------------------------------------------------------
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
          const tile = state.tiles[Number(parsedData.tileID)];
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
            const tile = state.tiles[Number(t.tileID)];
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
