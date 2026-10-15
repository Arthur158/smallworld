export interface Tile {
  id: string;
  polygon: Polygon;
  pieceStack: PieceStack[];
}

export interface ChoiceEntry {
    choice: string; 
    isChecked: boolean; 
}

export interface Polygon {
  // id: number
  coords: number[];
  stackX: number, 
  stackY: number, 
}

export interface PieceStack {
  type: string; // Not a race or tribe, since player will dispose of other type of stacks from powers, such as heroes or behemoths
  amount: number;
  isActive: boolean
}

export interface SaveGameInfo {
  saveId: number;
  summary: string;
}

export interface Tribe {
  race: string;
  trait: string;
}

export interface TribeEntry {
  race: string;
  trait: string;
  pieceCount: number;
  coinCount: number;
}


export interface Player {
  name: string;
  activeTribe: Tribe | null;
  passiveTribes: Tribe[];
  pieceStacks: PieceStack[]
}

export interface BotModel { id: string; name: string; description: string; }
export interface BotSeat { modelId: string; name: string; }
export interface BotPlayback { paused: boolean; speed: number; canGoBack?: boolean; }

export interface Room {
  botPlayback?: BotPlayback;
  bots?: Record<string, BotSeat>;
  botModels?: BotModel[];
  inProgress?: boolean;
  id: string;
  name: string;
  creator: string;
  players: string[];
  capacity: number
  mapName: string
}


