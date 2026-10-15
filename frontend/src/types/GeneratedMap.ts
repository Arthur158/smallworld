export type MapBiome = 'field' | 'grass' | 'forest' | 'mountain' | 'swamp';
export type MapFeature = 'magic' | 'mine' | 'cave';
export type SeaCorner = 'NW' | 'NE' | 'SW' | 'SE';

export interface GeneratedMapSettings {
  seed: string;
  width: number;
  height: number;
  tileCount: number;
  preserveRegionCount?: boolean;
  shapeMode: 'mainland' | 'islands';
  wobble: number;
  spread: number;
  tileShapeSettings: { chonkiness: number; repulsionSteps: number };
  waterSettings: { centerLakeSize: number; seaSize: number };
  scenerySettings: { forestTreeSize: number; mountainObjectSize: number };
  biomeWeights: Record<MapBiome, number>;
  biomeTextureZooms: Record<MapBiome | 'sea', number>;
  buildingSettings: { averagePerTile: number; maxPerTile: number; sizeMultiplier: number; symbolBuildingChance: number };
  featureSettings: Record<MapFeature, { count: number; spread: number }>;
  seaCorners: Record<SeaCorner, boolean>;
  randomizeSeaCorners: boolean;
  centerLake: boolean;
}

export interface GeneratedMapPreview {
  id: string;
  imageUrl: string;
  settings: GeneratedMapSettings;
  playerCount: number;
}
