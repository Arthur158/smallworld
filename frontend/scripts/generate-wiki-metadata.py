"""Regenerate frontend reference metadata from existing game definitions and artwork.
Run from any directory: python3 frontend/scripts/generate-wiki-metadata.py
No game rules or backend protocol are modified.
"""
import json,re
from pathlib import Path
root=Path(__file__).resolve().parents[2]
source=(root/'backend2/internal/gamestate/Maps.go').read_text()
blocks={m[1]:m[2] for m in re.finditer(r'func (\w+)\([^\n]*\) [^{]*\{(.*?)(?=\nfunc |\nvar MapRegistry|\Z)',source,re.S)}
def metadata(fn):
 block=blocks[fn]
 merged=re.search(r'mergeMaps\((\w+)\(gs\), (\w+)\(gs\)\)',block)
 if merged:return {**metadata(merged[1]),**metadata(merged[2])}
 result={}
 for m in re.finditer(r'"([^"\n]+)":\s*\{Id:\s*"[^"]+",\s*Biome:\s*(\w+),\s*Attributes:\s*\[\]Attribute\{([^}]+)?\}([^\n]*)',block):
  result[m[1]]={'biome':m[2],'attributes':[s.strip() for s in (m[3] or '').split(',') if s.strip()], 'isEdge':bool(re.search(r'IsEdge:\s*true',m[4])), 'isSkyIsland':fn in ('MapIsles2','MapIsles3')}
 assert result,fn
 return result
registry=source.split('var MapRegistry =',1)[1]
output={m[1]:metadata(m[2]) for m in re.finditer(r'"([^"]+)":\s*(\w+),',registry)}
(root/'frontend/src/data/wikiMapMetadata.json').write_text(json.dumps(output,indent=2)+'\n')
assets={kind:sorted(p.stem for p in (root/'frontend/public'/kind).glob('*.png')) for kind in ('races','traits','stacks')}
(root/'frontend/src/data/wikiAssets.json').write_text(json.dumps(assets,indent=2)+'\n')
print(f'Generated {len(output)} maps and {sum(map(len,assets.values()))} artwork references')
