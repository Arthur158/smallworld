import { Language } from './misc';
import { TribeEntry, Tile, Player, Room, SaveGameInfo, ChoiceEntry } from './Board'


export interface ApplicationState {
  language: Language;
  error: string | null;
  availableTribes: TribeEntry[];
  tiles: Record<string, Tile>;
  offsetMapTiles: number;
  players: Player[];
  playerIndex: number;
  turnNumber: number;
  playerNumber: number;
  phase: string;
  coins: number;
  selectedStack: string | null;
  isStackFromBank: boolean
  selectedTile: string | null;
  messages: string[]
  scores: number[]
  rooms: Room[]
  roomid: string
  name: string
  gameStarted: boolean
  mapImageUrl: string | null;
  isAuthenticated: boolean;
  saveGames: SaveGameInfo[]
  saveSelectionId: number
  mapName: string | null;
  offsetStacks: number
  mapChoices: string[]
  playerStatuses: string[]
  Xmult: number
  Ymult: number
  inDisplayRoom: boolean
  isSpectating: boolean
  extensionChoices: {
    extensionName: string;
    isChecked: boolean;
    raceChoices: { choice: string; isChecked: boolean }[];
    traitChoices: { choice: string; isChecked: boolean }[];
  }[];
  globalToggle: boolean
}

export type RootState = {
  application: ApplicationState;
};
