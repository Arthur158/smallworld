// Headless adapter to the actual site's Voronoi generator. No canvas, images,
// npm packages, browser or HTTP service are needed for training maps.
import readline from 'node:readline';
import { createHash } from 'node:crypto';
import { makeRng } from '../smallworld-map-generator/src/rng.js';
import { makeMap } from '../smallworld-map-generator/src/generation/mapGenerator.js';
import { createMapMetadata } from '../smallworld-map-generator/src/export/mapMetadata.js';

export function validGraph(tiles, players) {
  const byId = new Map(tiles.map(tile => [String(tile.id), tile]));
  if (byId.size !== tiles.length || tiles.length < 12 || tiles.length > 160) return false;
  const land = tiles.filter(tile => tile.biome !== 'Water');
  if (land.length < players * 4) return false;
  for (const tile of tiles) {
    const neighbors = tile.adjacentIds.map(String);
    if (new Set(neighbors).size !== neighbors.length) return false;
    for (const id of neighbors) {
      const neighbor = byId.get(id);
      if (!neighbor || id === String(tile.id) || !neighbor.adjacentIds.map(String).includes(String(tile.id))) return false;
    }
  }
  const visit = (start, allow) => {
    const seen = new Set([String(start.id)]), queue = [start];
    while (queue.length) {
      for (const id of queue.pop().adjacentIds.map(String)) {
        const next = byId.get(id);
        if (allow(next) && !seen.has(id)) { seen.add(id); queue.push(next); }
      }
    }
    return seen;
  };
  if (visit(tiles[0], () => true).size !== tiles.length) return false;
  // Every land component must have an entry point for a non-flying tribe.
  const checked = new Set();
  for (const tile of land) {
    if (checked.has(String(tile.id))) continue;
    const component = visit(tile, t => t.biome !== 'Water');
    if (![...component].some(id => byId.get(id).isEdge)) return false;
    for (const id of component) checked.add(id);
  }
  return true;
}

export function generate({ seed, players = 2, split = 'train', distribution = 'standard' }) {
  if (!Number.isInteger(players) || players < 2 || players > 6) throw new Error('players must be 2..6');
  if (!['train', 'eval', 'test'].includes(split)) throw new Error('invalid map split');
  if (!['standard', 'stress'].includes(distribution)) throw new Error('invalid map distribution');
  const namespace = `smallworld-v1:${split}:${String(seed)}`;
  for (let attempt = 0; attempt < 30; attempt++) {
    const rng = makeRng(`${namespace}:settings:${attempt}`);
    const between = (a,b) => a+rng()*(b-a);
    const integer = (a,b) => Math.floor(between(a,b+1));
    const stress = distribution === 'stress';
    const base = [0,0,22,29,38,47,56][players];
    const count = Math.max(18, Math.min(110, Math.round(base*between(stress ? .65 : .8, stress ? 1.8 : 1.4))));
    const settings = {
      seed: `${namespace}:geometry:${attempt}`, tileCount: count,
      width: integer(1000,1800), height: integer(750,1250),
      shapeMode: rng() < .3 ? 'islands' : 'mainland', wobble: 0,
      spread: integer(0,100),
      tileShapeSettings: { chonkiness: integer(20,100), repulsionSteps: integer(1,12) },
      randomizeSeaCorners: false,
      seaCorners: Object.fromEntries(['NW','NE','SW','SE'].map(corner => [corner, rng() < .5])),
      centerLake: rng() < .5,
      waterSettings: { seaSize: integer(stress ? 40 : 70, stress ? 220 : 160), centerLakeSize: integer(70,160) },
      biomeWeights: Object.fromEntries(['field','grass','forest','mountain','swamp'].map(biome => [biome, integer(stress ? 5 : 50, stress ? 260 : 180)])),
      featureSettings: Object.fromEntries(['magic','mine','cave'].map(feature => [feature, { count: Math.max(1, Math.round(count*between(.07,.18))), spread: integer(0,100) }])),
      buildingSettings: { averagePerTile: 0, maxPerTile: 0, symbolBuildingChance: 0 },
    };
    const tiles = createMapMetadata(makeMap(settings), settings).server.tiles;
    if (!validGraph(tiles, players)) continue;
    // Arbitrary labels break any tie-order/numbering regularity across maps.
    const labels = tiles.map((_, i) => `region-${i}`);
    for (let i=labels.length-1;i>0;i--) { const j=integer(0,i); [labels[i],labels[j]]=[labels[j],labels[i]]; }
    const ids = new Map(tiles.map((tile,i) => [String(tile.id),labels[i]]));
    const relabeled = tiles.map(tile => ({ ...tile, id: ids.get(String(tile.id)), adjacentIds: tile.adjacentIds.map(id => ids.get(String(id))).sort() })).sort((a,b) => a.id.localeCompare(b.id));
    // Hash the generator's layout before arbitrary label randomization.
    const hash = createHash('sha256').update(JSON.stringify(tiles)).digest('hex');
    // waterPlan is derived/mutated by the generator; omit it from concise logs.
    const { waterPlan, ...loggedSettings } = settings;
    return { tiles: relabeled, info: { seed: String(seed), split, distribution, players, hash, tile_count: tiles.length, attempt, settings: loggedSettings } };
  }
  throw new Error(`Could not generate a connected, enterable map after 30 attempts (${namespace})`);
}

const lines = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });
for await (const line of lines) {
  try { process.stdout.write(JSON.stringify(generate(JSON.parse(line)))+'\n'); }
  catch (error) { process.stdout.write(JSON.stringify({ error: error.message })+'\n'); }
}
