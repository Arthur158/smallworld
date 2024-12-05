export interface Tile {
  id: string;
  polygon: Polygon;
  pieceStack: PieceStack[];
}

export interface Polygon {
  // id: number
  coords: number[];
}

export interface PieceStack {
  tribe: Tribe;
  amount: number;
}

export interface Tribe {
  race: string;
  trait: string;
}

export interface Player {
  name: string;
  activeTribe: Tribe | null;
  passiveTribes: Tribe[];
  isPlaying: boolean;
  pieceStacks: PieceStack[]
}

