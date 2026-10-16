// Uses the real generator and metadata exporter without canvas dependencies.
import assert from 'node:assert/strict';
import fs from 'node:fs';
import { createHash } from 'node:crypto';
import { makeMap } from '../../../../smallworld-map-generator/src/generation/mapGenerator.js';
import { createMapMetadata } from '../../../../smallworld-map-generator/src/export/mapMetadata.js';

const cases = JSON.parse(fs.readFileSync(0, 'utf8'));
const legacy = JSON.parse(fs.readFileSync(new URL('./map_legacy_hashes.json', import.meta.url), 'utf8'));
const histogram = (items, key) => Object.fromEntries([...new Set(items.map(key))].sort().map(value => [value, items.filter(item => key(item) === value).length]));
function landComponents(tiles) {
  const unseen = new Set(tiles.filter(tile => tile.biome !== 'Water').map(tile => tile.id));
  const byID = new Map(tiles.map(tile => [tile.id, tile]));
  let components = 0;
  while (unseen.size) {
    components++;
    const todo = [unseen.values().next().value];
    while (todo.length) {
      const id = todo.pop();
      if (!unseen.delete(id)) continue;
      for (const neighbor of byID.get(id).adjacentIds.map(String)) if (unseen.has(neighbor)) todo.push(neighbor);
    }
  }
  return components;
}
function generate(settings) {
  const config = structuredClone(settings);
  const cells = makeMap(config);
  return { cells, metadata: createMapMetadata(cells, config) };
}
// Recorded before the opt-in generator changes: full cell state and metadata,
// including geometry, biomes, features and RNG-driven decorative objects.
for (const fixture of legacy) {
  const settings = { ...cases.find(item => item.players === fixture.players).base, seed: fixture.seed };
  for (const optIn of [undefined, false]) {
    if (optIn !== undefined) settings.preserveRegionCount = optIn;
    const output = generate(settings);
    // Region names are additive display metadata, not terrain or decorative
    // state. Keep the original golden hashes for every pre-existing field.
    delete output.metadata.regionName;
    delete output.metadata.regionNameVersion;
    assert.equal(createHash('sha256').update(JSON.stringify(output)).digest('hex'), fixture.hash, `legacy output changed: ${fixture.players}/${fixture.seed}/${optIn}`);
  }
}
const actualCounts = {}, seaCounts = new Set();
for (const { players, base, varied } of cases) {
  // Match the API's unchanged border clamp; no border-style variation.
  const after = generate({ ...varied, wobble: Math.min(30, varied.wobble) });
  const label = `${players} players / ${base.seed}`;
  const tiles = after.metadata.server.tiles;
  const byID = new Map(tiles.map(tile => [tile.id, tile]));
  const seas = Object.values(varied.seaCorners).filter(Boolean).length;
  seaCounts.add(seas);
  assert.equal(varied.wobble, base.wobble, `${label}: border style changed`);
  assert.equal(tiles.length, base.tileCount, `${label}: exact total region count`);
  assert.equal(byID.size, tiles.length, `${label}: duplicate regions`);
  assert.equal(after.cells.filter(cell => cell.waterKind === 'sea').length, seas, `${label}: sea count`);
  assert.equal(after.cells.filter(cell => cell.waterKind === 'lake').length, 1, `${label}: lake count`);
  const land = tiles.filter(tile => tile.biome !== 'Water');
  assert.equal(land.length, base.tileCount - seas - 1, `${label}: land count`);
  const counts = Object.values(histogram(land, tile => tile.biome));
  assert.equal(counts.length, 5, `${label}: all land biomes present`);
  assert(Math.max(...counts) - Math.min(...counts) <= 1, `${label}: land biome weights`);
  assert.equal(landComponents(tiles), 1, `${label}: connected land`);
  for (const tile of tiles) {
    assert(tile.adjacentIds.length > 0, `${label}: orphan region ${tile.id}`);
    assert.equal(new Set(tile.adjacentIds.map(String)).size, tile.adjacentIds.length, `${label}: duplicate neighbors`);
    for (const id of tile.adjacentIds.map(String)) {
      assert.notEqual(id, tile.id, `${label}: self adjacency`);
      assert(byID.get(id)?.adjacentIds.map(String).includes(tile.id), `${label}: reciprocal adjacency ${tile.id}/${id}`);
    }
  }
  for (const feature of ['magic', 'mine', 'cave']) {
    assert.equal(after.cells.filter(cell => cell.feature === feature).length, base.featureSettings[feature].count, `${label}: ${feature} supply`);
  }
  (actualCounts[players] ??= new Set()).add(tiles.length);
}
assert.equal(seaCounts.size, 3, 'All sea counts exercised');
console.log(JSON.stringify({ maps: cases.length, legacyParityChecks: legacy.length * 2, seaCounts: [...seaCounts].sort(), exactRegionCounts: Object.fromEntries(Object.entries(actualCounts).map(([players, counts]) => [players, [...counts]])) }));
