import { Language } from './misc';
import { TribeEntry, Tile, Player, Room, SaveGameInfo } from './Board'


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
  offsetStacksX: number
  offsetStacksY: number
  mapChoices: string[]
}

export type RootState = {
  application: ApplicationState;
};
