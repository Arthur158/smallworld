import { Language } from './misc';
import { TribeEntry, Tile, Player, Room } from './Board'


export interface ApplicationState {
  language: Language;
  error: string | null;
  availableTribes: TribeEntry[];
  tiles: Record<string, Tile>;
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
  room: Room | null
  name: string
  gameStarted: boolean
}

export type RootState = {
  application: ApplicationState;
};
