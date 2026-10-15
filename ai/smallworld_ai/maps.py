"""Generate fresh game maps with the repository's actual Voronoi generator."""
import json
import os
from pathlib import Path
import selectors
import shutil
import subprocess

SCRIPT = Path(__file__).resolve().parents[1] / "generate-map.mjs"
BIOMES = {name: i for i, name in enumerate(['Forest', 'Hill', 'Field', 'Swamp', 'Water', 'River', 'Mountain', 'MineArea', 'CrystalArea', 'AbyssalChasm'])}
ATTRIBUTES = {name: i for i, name in enumerate(['Magic', 'Mine', 'Cave'])}


class MapGenerator:
    def __init__(self):
        node = os.environ.get("SMALLWORLD_NODE") or shutil.which("node")
        if not node:
            raise RuntimeError("Generated training maps require Node.js 18+. Install Node or set SMALLWORLD_NODE.")
        source = SCRIPT.parent.parent / "smallworld-map-generator/src/generation/mapGenerator.js"
        if not source.is_file():
            raise RuntimeError("Initialize the map generator: git submodule update --init --recursive")
        self.process = subprocess.Popen([node, str(SCRIPT)], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, bufsize=1)

    def generate(self, seed, players, split, distribution):
        self.process.stdin.write(json.dumps(dict(seed=str(seed), players=players, split=split, distribution=distribution))+"\n")
        self.process.stdin.flush()
        with selectors.DefaultSelector() as selector:
            selector.register(self.process.stdout, selectors.EVENT_READ)
            if not selector.select(30):
                raise RuntimeError("map generator timed out")
        line = self.process.stdout.readline()
        if not line:
            raise RuntimeError("map generator exited; see stderr")
        result = json.loads(line)
        if result.get("error"):
            raise RuntimeError(result["error"])
        # RuntimeTileSpec uses numeric Go enums; the exported site metadata has names.
        tiles = [dict(ID=t['id'], Biome=BIOMES[t['biome']], Attributes=[ATTRIBUTES[a] for a in t['attributes']],
                      IsEdge=t['isEdge'], IsSkyIsland=t.get('isSkyIsland', False), AdjacentIDs=t['adjacentIds']) for t in result['tiles']]
        return tiles, result['info']

    def close(self):
        self.process.stdin.close()
        try:
            self.process.wait(timeout=3)
        except subprocess.TimeoutExpired:
            self.process.kill()
            self.process.wait()
        self.process.stdout.close()
