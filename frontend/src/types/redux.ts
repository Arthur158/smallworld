import { Language } from './misc';
import { TribeEntry, Tile, Player } from './Board'


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
}

export type RootState = {
  application: ApplicationState;
};
