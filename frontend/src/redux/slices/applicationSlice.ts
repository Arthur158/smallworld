import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { ApplicationState } from '../../types/redux';
import { Language } from '../../types/misc';
import { Player, TribeEntry } from '../../types/Board'
import { json } from 'react-router-dom';
import { useDispatch } from 'react-redux';

const initialState: ApplicationState = {
  language: Language.NL,
  error: null,
  availableTribes: [{race: "elves", trait: "merchant", pieceCount: 0, coinCount: 0}, {race: "giants", trait:"fortunate", pieceCount: 0, coinCount: 0}], //dummy for now
  tiles: {},
  players: [],
  playerIndex: 1,
  turnNumber: 1,
  playerNumber: 1,
  phase: "tribechoice",
  selectedStack: null,
  isStackFromBank: false,
  selectedTile: null,
  messages: [],
};

const applicationSlice = createSlice({
  name: 'application',
  initialState,
  reducers: {
    setLanguage(state, action: PayloadAction<Language>): void {
      state.language = action.payload;
    },
    setSelectedStack(state, action) {
      state.selectedStack = action.payload
    },
    setSelectedTile(state, action) {
      state.selectedTile = action.payload
    },
    setIsStackFromBank(state, action) {
      state.isStackFromBank = action.payload
    },
    clearError(state) {
      state.error = null;
    },
    setError(state, action: PayloadAction<string>) {
      state.error = action.payload;
    },
    setPlayers(state, action) {
      state.players = action.payload
    },
    setTiles(state, action) {
      state.tiles = action.payload
    },
    updateTileStack(state, action) {
        const { tile_id, new_stacks } = action.payload
        const tile = state.tiles[tile_id];

        if (!tile) {
          throw new Error(`Tile with ID ${tile_id} does not exist.`);
        }

        tile.pieceStack = new_stacks; // Update the pieceStack for the tile
    },
    websocketMessageReceived(state, action) {
      const { type, data } = JSON.parse(action.payload);
      state.error = null

      const parsedData = data
      switch (type) {

        case 'index':
          console.log("the index:")
          console.log(parsedData)
          state.playerIndex = Number(parsedData.index)
          break;
        case 'error':
          state.error = parsedData.message
          break;
        case 'playerupdate':
          const players: Player[] = [];

          // Use a for loop to construct Player objects
          for (let i = 0; i < parsedData.length; i++) {
            const playerData = parsedData[i];
            const player: Player = {
              name: playerData.name,
              activeTribe: {race: playerData.activeTribe.race, trait: playerData.activeTribe.trait},
              passiveTribes: [],
              pieceStacks: [],
            };
            if (parsedData[i].pieceStacks && Array.isArray(playerData.pieceStacks)) {
              for (const stack of playerData.pieceStacks) {
                player.pieceStacks.push({
                  type: stack.type,
                  amount: stack.amount,
                  isActive: stack.isActive,
                })
              }
            }
            console.log("updating...")
            console.log(player)
            players.push(player);
          }

            state.players = players; // Update the state
            
            console.log("Updated players:", players);
          break;
        case 'entriesupdate':
          state.availableTribes = data
          break;
        case 'tileupdate': {

          // Safely check if the tile exists
          const tile = state.tiles[Number(parsedData.tileID)];
          if (!tile) {
            console.error(`Tile with ID ${parsedData.tileID} does not exist.`);
            return;
          }
          const stacks = []
          for (const stack of parsedData.stacks) {
              stacks.push({
                type: stack.type,
                amount: stack.amount,
                isActive: stack.isActive
              })
          }

          // Update the pieceStack for the existing tile
          tile.pieceStack = stacks;
          break;
        }
        case 'alltileupdate': {
          const players: Player[] = [];

          // Use a for loop to construct Player objects
          for (let i = 0; i < parsedData.length; i++) {
            const tile = state.tiles[Number(parsedData[i].tileID)]
            if (!tile) {
              console.error(`Tile with ID ${parsedData.tileID} does not exist.`);
              return;
            }
            const stacks = []
            if (parsedData[i].stacks && Array.isArray(parsedData[i].stacks)) {
              for (const stack of parsedData[i].stacks) {
                stacks.push({
                  type: stack.type,
                  amount: stack.amount,
                  isActive: stack.isActive,
                });
              }
            } 

            // Update the pieceStack for the existing tile
            tile.pieceStack = stacks;
          }
          break;
        }
        case 'turnupdate':
          state.playerNumber = parsedData.playerNumber
          state.turnNumber = parsedData.turnNumber
          state.phase = parsedData.Phase
          break;

        case 'tribeentries':
          const tribeEntries: TribeEntry[] = [];

          for (let i = 0; i < parsedData.length; i++) {
            const tribeData = parsedData[i];
            const tribeEntry: TribeEntry = {
              race: tribeData.Race,
              trait: tribeData.Trait,
              pieceCount: tribeData.piecepile,
              coinCount: tribeData.coinpile,
            };
            tribeEntries.push(tribeEntry);
          }

          state.availableTribes = tribeEntries; 
          console.log("Updated tribe entries:", tribeEntries);
          break;

        default:
          console.warn('Unhandled WebSocket message type:', data);
          console.log(type);
          // On ajoute le message brut (ou formaté) au tableau messages
          state.messages.push(
            `Type: ${type}, Content: ${JSON.stringify(parsedData)}`
          );
          break;
      }
    },
  },
});

const applicationReducer = applicationSlice.reducer;

export const { setLanguage, setSelectedTile, setSelectedStack, setIsStackFromBank, setTiles, clearError, setError, websocketMessageReceived, updateTileStack, setPlayers } = applicationSlice.actions;

export default applicationReducer;
