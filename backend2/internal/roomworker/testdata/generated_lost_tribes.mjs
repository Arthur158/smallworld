// Export real terrain for backend-only neutral-placement tests. The generator
// never receives or places the Lost Tribe pieces.
import fs from 'node:fs';
import { makeMap } from '../../../../smallworld-map-generator/src/generation/mapGenerator.js';
import { createMapMetadata } from '../../../../smallworld-map-generator/src/export/mapMetadata.js';

const settings = JSON.parse(fs.readFileSync(0, 'utf8'));
console.log(JSON.stringify(settings.map(config => createMapMetadata(makeMap(config), config).server.tiles)));
