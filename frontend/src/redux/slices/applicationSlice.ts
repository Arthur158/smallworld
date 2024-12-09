import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { ApplicationState } from '../../types/redux';
import { Language } from '../../types/misc';

const initialState: ApplicationState = {
  language: Language.NL,
  error: null,
  player: {
    name: "Arthur",
    activeTribe: null,
    passiveTribes: [],
    isPlaying: false,
    pieceStacks: [{type:"elves", amount:3}, {type:"show", amount:1}],
  },
  availableTribes: [{race: "elves", trait: "merchant"}, {race: "giants", trait:"fortunate"}], //dummy for now
  tiles: {},
  opponents: [],
};

const applicationSlice = createSlice({
  name: 'application',
  initialState,
  reducers: {
    setLanguage(state, action: PayloadAction<Language>): void {
      state.language = action.payload;
    },
    clearError(state) {
      state.error = null;
    },
    setError(state, action: PayloadAction<string>) {
      state.error = action.payload;
    },
    selectTribe(state, action) {
      state.player.activeTribe = action.payload;
    },
    setTiles(state, action) {
      state.tiles = action.payload
    },
    setPlaying(state, action) {
      state.player.isPlaying = action.payload
    },
    setPlayerStacks(state, action) {
    state.player.pieceStacks = action.payload
    },
    updateTileStack(state, action) {
        const { tile_id, new_stacks } = action.payload
        const tile = state.tiles[tile_id];

        if (!tile) {
          throw new Error(`Tile with ID ${tile_id} does not exist.`);
        }

        tile.pieceStack = new_stacks; // Update the pieceStack for the tile
    },
    setOpponents(state, action) {
      state.opponents = action.payload
    },
    websocketMessageReceived(state, action) {
      const { type, payload } = action.payload;

      switch (type) {
        case 'tilePolygonsResponse':
          state.tiles = payload.tiles
          break

        case 'tilePieceStackChange':
          this.updateTileStack(state, action)
          break;
        case 'OpponentUpdate':
          state.opponents = payload.opponents
          break;

        default:
          // Handle all other messages or log unhandled types
          console.warn('Unhandled WebSocket message type:', type);
          // state.messages.push(action.payload); // Store in general message log
          break;
      }
    },
  },
});

const applicationReducer = applicationSlice.reducer;

export const { setLanguage, setTiles, setPlaying, setPlayerStacks, clearError, setError, selectTribe, setOpponents, websocketMessageReceived, updateTileStack } = applicationSlice.actions;

export default applicationReducer;
