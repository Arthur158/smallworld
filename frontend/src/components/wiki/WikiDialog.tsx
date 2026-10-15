import React, { useEffect, useMemo, useRef, useState } from 'react';
import { useSelector } from 'react-redux';
import { FiArrowLeft, FiBookOpen, FiSearch, FiX, FiExternalLink } from 'react-icons/fi';
import { RootState } from '../../redux/store';
import { PieceStack, Tile, Tribe } from '../../types/Board';
import { WikiArticle } from '../../data/wikiTypes';
import { artwork, findArticle, pieceArticle, relatedArticles, terrainNames, tribePieces, tribesForPiece, wikiArticles } from '../../data/wikiCatalog';
import mapMetadata from '../../data/wikiMapMetadata.json';
import type { WikiTarget } from './WikiContext';
import './wiki.css';

type Navigate = (target?: WikiTarget) => void;
const kinds: [string, string][] = [['all', 'All'], ['race', 'Races'], ['trait', 'Traits'], ['power', 'Powers'], ['piece', 'Pieces'], ['terrain', 'Terrain']];
const kindName = (kind: string) => kinds.find(([key]) => key === kind)?.[1] ?? kind;

function Art({ kind, name, className = '' }: { kind: 'race' | 'trait' | 'power' | 'piece'; name: string; className?: string }) {
  const src = artwork(kind, name);
  const [failed, setFailed] = useState(false);
  useEffect(() => setFailed(false), [src]);
  return src && !failed ? <img src={src} alt={name} className={className} onError={() => setFailed(true)} />
    : <div className={`wiki-art-placeholder ${className}`}><FiBookOpen aria-hidden="true" /><span>{name}</span></div>;
}
function Sources({ sources }: { sources: WikiArticle['sources'] }) {
  const unique = sources.filter((s, i) => sources.findIndex(other => other.url === s.url) === i);
  return unique.length ? <div className="wiki-sources"><span>Read the rules</span>{unique.map(source => <a key={source.url} href={source.url} target="_blank" rel="noreferrer">{source.label}<FiExternalLink aria-hidden="true" /></a>)}</div> : null;
}
function ArticleCard({ article }: { article?: WikiArticle }) {
  if (!article) return <p className="wiki-muted">No verified description is available for this entry yet.</p>;
  return <article className="wiki-article">
    {article.kind !== 'terrain' && <div className="wiki-banner-art"><Art kind={article.kind} name={article.kind === 'piece' ? article.pieces[0] : article.name} /></div>}
    <div className="wiki-article-body"><div className="wiki-eyebrow">{kindName(article.kind)}</div><h3>{article.name}</h3>
      <p className="wiki-summary">{article.summary}</p>
      {article.details.map((detail, i) => <p key={i}>{detail}</p>)}
      {article.note && <aside className="wiki-note"><strong>In this game</strong><p>{article.note}</p></aside>}
      <Sources sources={article.sources} />
    </div>
  </article>;
}
function Pieces({ pieces, navigate, tribe }: { pieces: string[]; navigate: Navigate; tribe?: Tribe }) {
  const unique = Array.from(new Set(pieces));
  return unique.length ? <section className="wiki-section"><h3>Related pieces</h3><div className="wiki-piece-grid">{unique.map(name => <button type="button" key={name} aria-label={name} onClick={() => navigate({ kind: 'piece', stack: { type: name, amount: 1, isActive: true }, tribe })} className="wiki-piece-card"><Art kind="piece" name={name} /><span>{name}</span></button>)}</div></section> : null;
}
function TribeView({ tribe, navigate }: { tribe: Tribe; navigate: Navigate }) {
  return <><div className="wiki-detail-heading"><div className="wiki-eyebrow">Race & trait</div><h2>{tribe.trait} {tribe.race}</h2><p>The race and trait work together. Read both abilities and their decline exceptions.</p></div>
    <div className="wiki-pair"><ArticleCard article={findArticle(`race:${tribe.race}`)} /><ArticleCard article={findArticle(`trait:${tribe.trait}`)} /></div>
    <Pieces pieces={tribePieces(tribe)} navigate={navigate} tribe={tribe} />
  </>;
}
function ZonePreview({ tile }: { tile: Tile }) {
  const { mapName, mapImageUrl, offsetMapTiles } = useSelector((state: RootState) => state.application);
  const src = mapImageUrl ?? (mapName ? `/maps/${mapName}.png` : undefined);
  const [size, setSize] = useState({ width: 0, height: 0 });
  useEffect(() => {
    let disposed = false;
    setSize({ width: 0, height: 0 });
    if (src) { const img = new Image(); img.onload = () => { if (!disposed) setSize({ width: img.width, height: img.height }); }; img.src = src; }
    return () => { disposed = true; };
  }, [src]);
  if (!src || !size.width || !tile.polygon.coords.length) return null;
  // Match the map renderer's coordinate system, then enlarge this region.
  const factor = size.width * offsetMapTiles * offsetMapTiles / 1130;
  const xs = tile.polygon.coords.filter((_, i) => i % 2 === 0).map(x => x * factor);
  const ys = tile.polygon.coords.filter((_, i) => i % 2 === 1).map(y => y * factor);
  const minX = Math.min(...xs), minY = Math.min(...ys);
  const width = Math.max(...xs) - minX, height = Math.max(...ys) - minY;
  const pad = Math.max(width, height) * 0.15;
  return <svg className="wiki-zone-preview" role="img" aria-label={`Enlarged map region ${tile.id}`} viewBox={`${minX-pad} ${minY-pad} ${width+pad*2} ${height+pad*2}`}>
    <image href={src} width={size.width * offsetMapTiles} height={size.height * offsetMapTiles} />
    <polygon points={xs.map((x, i) => `${x},${ys[i]}`).join(' ')} fill="rgba(255,230,154,.12)" stroke="#f9dd8e" strokeWidth="3" vectorEffect="non-scaling-stroke" />
  </svg>;
}
function StackLink({ stack, tileId, navigate }: { stack: PieceStack; tileId?: string; navigate: Navigate }) {
  return <button type="button" className="wiki-stack-link" onClick={() => navigate({ kind: 'piece', stack, tileId })}><Art kind="piece" name={stack.type} /><span><strong>{stack.type}</strong><small>{stack.amount} {stack.amount === 1 ? 'piece' : 'pieces'} · {stack.isActive ? 'Active' : 'Inactive / in decline'}</small></span><span aria-hidden="true">→</span></button>;
}
function TargetView({ target, navigate }: { target?: WikiTarget; navigate: Navigate }) {
  const { players: roomPlayers, availableTribes: roomTribes, mapName, gameStarted, inDisplayRoom } = useSelector((state: RootState) => state.application);
  // Room data survives a return to the lobby; only show pairings for a visible board.
  const players = gameStarted || inDisplayRoom ? roomPlayers : [];
  const availableTribes = gameStarted || inDisplayRoom ? roomTribes : [];
  const currentTribes = players.flatMap(player => [player.activeTribe, ...(player.passiveTribes ?? [])]).filter((tribe): tribe is Tribe => !!tribe?.race);
  if (!target) return <>
    <div className="wiki-welcome"><div className="wiki-eyebrow">Your companion at the table</div><h2>A small world.<br />A lot to discover.</h2><p>Explore races, traits, powers and the pieces that bring them to life.</p><div className="wiki-welcome-art"><Art kind="trait" name="Dragon Master" /><Art kind="race" name="Trolls" /><Art kind="piece" name="Dragon" /></div></div>
    <section className="wiki-section"><h3>Inspect as you play</h3><p>Hold <kbd>Ctrl</kbd> or <kbd>⌘</kbd> and click a race, trait, piece stack or region. You can also turn on <strong>Inspect</strong> above the board. Inspecting does not select a troop or send a game action.</p></section>
    {(currentTribes.length > 0 || availableTribes.length > 0) && <section className="wiki-section"><h3>In your game</h3><div className="wiki-combos">{[...currentTribes, ...availableTribes].filter((tribe, i, all) => all.findIndex(t => t.race === tribe.race && t.trait === tribe.trait) === i).map(tribe => <button type="button" key={`${tribe.race}:${tribe.trait}`} onClick={() => navigate({ kind: 'tribe', tribe })}><Art kind="race" name={tribe.race} /><span>{tribe.trait}<strong>{tribe.race}</strong></span><span aria-hidden="true">→</span></button>)}</div></section>}
    <p className="wiki-footer-note">Original summaries of the published rules, checked against this game. Differences are marked “In this game”. Artwork belongs to its respective creators.</p>
  </>;
  if (target.kind === 'tribe') return <TribeView tribe={target.tribe} navigate={navigate} />;
  if (target.kind === 'zone') {
    const { tile } = target;
    const terrain = tile.terrain ?? (mapMetadata as Record<string, Record<string, Tile['terrain']>>)[mapName ?? '']?.[tile.id];
    const entries = terrain ? [terrain.biome, ...terrain.attributes] : [];
    return <><div className="wiki-detail-heading"><div className="wiki-eyebrow">Region · board snapshot</div><h2>Region {tile.id}</h2><p>{terrain ? terrainNames[terrain.biome] ?? terrain.biome : 'Explore the terrain and the pieces in this region.'}{terrain?.isEdge ? ' · Board edge' : ''}{terrain?.isSkyIsland ? ' · Sky island' : ''}</p></div>
      <ZonePreview tile={tile} />
      <section className="wiki-section"><h3>Pieces in this region</h3>{tile.pieceStack.length ? tile.pieceStack.map((stack, i) => <StackLink key={`${stack.type}:${i}`} stack={stack} tileId={tile.id} navigate={navigate} />) : <p className="wiki-muted">No pieces occupy this region.</p>}</section>
      <div className="wiki-pair">{entries.map(id => <ArticleCard key={id} article={findArticle(`terrain:${id}`)} />)}</div>
      {!terrain && <p className="wiki-note">Terrain information is unavailable for this map. The enlarged region and its visible pieces are shown above.</p>}
      <p className="wiki-footer-note">Conquest costs can change with race abilities, traits and protective markers. Inspect those pieces before attacking.</p>
    </>;
  }
  if (target.kind === 'piece') {
    const { stack } = target;
    const rosterTribes = tribesForPiece(stack, players);
    const marketTribes = availableTribes.filter(tribe => tribePieces(tribe).includes(stack.type));
    const isIndependentPower = relatedArticles(stack.type).some(article => article.kind === 'power');
    const tribes = isIndependentPower ? [] : target.tribe ? [target.tribe] : rosterTribes.length ? rosterTribes : marketTribes;
    const related = relatedArticles(stack.type);
    const article = pieceArticle(stack.type);
    return <><div className="wiki-piece-hero"><Art kind="piece" name={stack.type} /><div><div className="wiki-eyebrow">Piece reference{target.tileId ? ` · Region ${target.tileId}` : ''}</div><h2>{stack.type}</h2><p>{article.summary}</p><span className="wiki-badge">{stack.amount} {stack.amount === 1 ? 'piece' : 'pieces'} · {stack.isActive ? 'Active' : 'Inactive / in decline'}</span></div></div>
      <div className="wiki-section">{article.details.map((line, i) => <p key={i}>{line}</p>)}<Sources sources={article.sources} /></div>
      {tribes.map(tribe => <TribeView key={`${tribe.race}:${tribe.trait}`} tribe={tribe} navigate={navigate} />)}
      {!tribes.length && related.length > 0 && <><div className="wiki-section"><h3>{isIndependentPower ? 'Related power' : 'Related race & trait'}</h3><p className="wiki-muted">{isIndependentPower ? 'Relics and places grant abilities separately from a race’s trait.' : 'These abilities use this piece. Its current race–trait pairing is not available.'}</p></div><div className="wiki-pair">{related.map(entry => <ArticleCard key={entry.id} article={entry} />)}</div><Pieces pieces={related.flatMap(entry => entry.pieces)} navigate={navigate} /></>}
      {stack.type === 'Balrog' && <button className="wiki-inline-link" onClick={() => navigate({kind:'article',id:'piece:Wickedest Pentacle'})}>Read about the Wickedest Pentacle →</button>}
      {stack.type === 'Tomb-raider' && <button className="wiki-inline-link" onClick={() => navigate({kind:'article',id:'piece:Crypt of the Tomb-raider'})}>Read about the Crypt of the Tomb-raider →</button>}
    </>;
  }
  const article = findArticle(target.id);
  if (!article) return <p className="wiki-muted">This reference is not available.</p>;
  if (article.kind === 'piece') return <TargetView target={{ kind: 'piece', stack: { type: article.pieces[0], amount: 1, isActive: true } }} navigate={navigate} />;
  const matching = (article.kind === 'race' || article.kind === 'trait' ? [...currentTribes, ...availableTribes] : []).filter(tribe => (article.kind === 'race' ? tribe.race : tribe.trait) === article.name)
    .filter((tribe, i, all) => all.findIndex(other => other.race === tribe.race && other.trait === tribe.trait) === i);
  return <><ArticleCard article={article} /><Pieces pieces={article.pieces} navigate={navigate} />{matching.length > 0 && <section className="wiki-section"><h3>Paired in this game</h3>{matching.map(tribe => <button type="button" className="wiki-inline-link" key={`${tribe.race}:${tribe.trait}`} onClick={() => navigate({kind:'tribe',tribe})}>{tribe.trait} {tribe.race} →</button>)}</section>}</>;
}

export default function WikiDialog({ initialTarget, onClose }: { initialTarget?: WikiTarget; onClose: () => void }) {
  const [history, setHistory] = useState<(WikiTarget | undefined)[]>([initialTarget]);
  const [query, setQuery] = useState('');
  const [filter, setFilter] = useState('all');
  const panelRef = useRef<HTMLDivElement>(null);
  const contentRef = useRef<HTMLDivElement>(null);
  const searchRef = useRef<HTMLInputElement>(null);
  const target = history[history.length - 1];
  const navigate: Navigate = next => { setHistory(previous => [...previous, next]); contentRef.current?.scrollTo?.(0, 0); };
  const matches = useMemo(() => {
    const terms = query.toLocaleLowerCase().trim().split(/\s+/).filter(Boolean);
    return wikiArticles.filter(article => (filter === 'all' || article.kind === filter) && terms.every(term => `${article.name} ${article.summary} ${article.pieces.join(' ')}`.toLocaleLowerCase().includes(term))).sort((a, b) => a.name.localeCompare(b.name));
  }, [query, filter]);
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    // Keep contextual entries at their heading; opening the library focuses search.
    if (initialTarget) panelRef.current?.focus(); else searchRef.current?.focus();
    return () => { document.body.style.overflow = overflow; if (previous?.isConnected) previous.focus(); };
  }, [initialTarget]);
  const handleKey = (event: React.KeyboardEvent) => {
    event.stopPropagation();
    if (event.key === 'Escape') { event.preventDefault(); onClose(); }
    if (event.key === 'Tab') {
      const nodes = Array.from(panelRef.current?.querySelectorAll<HTMLElement>('button:not([disabled]), input, select, a[href], [tabindex="0"]') ?? []).filter(node => !node.hidden);
      const first = nodes[0], last = nodes[nodes.length - 1];
      if (!first) return;
      if (event.shiftKey && (document.activeElement === first || document.activeElement === panelRef.current)) { event.preventDefault(); last.focus(); }
      else if (!event.shiftKey && (document.activeElement === last || document.activeElement === panelRef.current)) { event.preventDefault(); first.focus(); }
    }
  };
  return <div className="wiki-backdrop" onMouseDown={event => { if (event.target === event.currentTarget) onClose(); }}>
    <div ref={panelRef} role="dialog" aria-modal="true" aria-labelledby="wiki-title" tabIndex={-1} className="wiki-dialog" onKeyDown={handleKey}>
      <header className="wiki-header"><button type="button" className="wiki-home" onClick={() => navigate()} aria-label="Wiki home" data-tooltip="Return to the wiki overview. Click, or focus and press Enter."><FiBookOpen aria-hidden="true" /><span><span className="wiki-eyebrow">Field guide</span><strong id="wiki-title">Small World wiki</strong></span></button><button type="button" aria-label="Close wiki" data-tooltip="Close the wiki and return to the board. Shortcut: Escape." className="wiki-icon-button" onClick={onClose}><FiX aria-hidden="true" /></button></header>
      <div className="wiki-layout"><aside className="wiki-sidebar" aria-label="Wiki navigation">
        <label className="wiki-search"><FiSearch aria-hidden="true" /><input ref={searchRef} value={query} onChange={event => setQuery(event.target.value)} aria-label="Search the wiki" placeholder="Search races, traits, powers…" />{query && <button type="button" aria-label="Clear search" data-tooltip="Clear the wiki search and show all matching entries." onClick={() => setQuery('')}><FiX /></button>}</label>
        <div className="wiki-filters" aria-label="Reference categories">{kinds.map(([key, label]) => <button type="button" key={key} data-tooltip={`Show ${label.toLowerCase()} in the wiki. Click, or focus and press Enter.`} aria-pressed={filter === key} onClick={() => setFilter(key)}>{label}</button>)}</div>
        <div className="wiki-result-count" role="status">{matches.length} {matches.length === 1 ? 'entry' : 'entries'}</div>
        <nav className="wiki-results" aria-label="Wiki entries">{matches.map(article => <button type="button" key={article.id} aria-current={target?.kind === 'article' && target.id === article.id ? 'page' : undefined} onClick={() => navigate({kind:'article',id:article.id})}><span>{article.name}</span><small>{kindName(article.kind)}</small></button>)}{!matches.length && <p>No entries found. Try a race, trait, power or piece name.</p>}</nav>
      </aside><main ref={contentRef} className="wiki-content"><div className="wiki-breadcrumb"><button type="button" data-tooltip="Return to the previous wiki page. Click, or focus and press Enter." disabled={history.length <= 1} onClick={() => { setHistory(previous => previous.slice(0,-1)); contentRef.current?.scrollTo?.(0,0); }}><FiArrowLeft aria-hidden="true" /> Back</button><span>Explore · Learn · Play</span></div><TargetView target={target} navigate={navigate} /></main></div>
    </div>
  </div>;
}
