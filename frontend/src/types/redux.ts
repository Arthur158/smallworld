import { Language } from './misc';
import { Tribe, Tile, Player } from './Board'


export interface ApplicationState {
  language: Language;
  error: string | null;
  player: Player;
  availableTribes: Tribe[];
  tiles: Record<string, Tile>;
  opponents: Player[]
}

export type RootState = {
  application: ApplicationState;
};
