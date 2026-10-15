import React, { useEffect, useRef, useState } from 'react';
import { FiArrowLeft, FiCheck, FiMap, FiRefreshCw, FiShuffle } from 'react-icons/fi';
import { GeneratedMapPreview, GeneratedMapSettings, MapBiome, MapFeature, SeaCorner } from '../types/GeneratedMap';
import './GeneratedMapEditor.css';

export interface GeneratedMapEditorProps {
  defaultSettings: GeneratedMapSettings;
  initialSettings?: GeneratedMapSettings;
  preview?: GeneratedMapPreview;
  playerCount: number;
  loading: boolean;
  selecting?: boolean;
  selectedPreviewId?: string;
  error?: string;
  automatic: boolean;
  onGenerate: (settings: GeneratedMapSettings) => void;
  onSelect: (previewId: string) => void;
  onAutomatic: () => void;
  onClose: () => void;
}

const biomes: [MapBiome, string, string][] = [
  ['field', 'Farmland', '#c7973d'], ['grass', 'Hills', '#8b9e52'], ['forest', 'Forest', '#50764b'],
  ['mountain', 'Mountains', '#8c8b86'], ['swamp', 'Swamp', '#8b7850'],
];
const features: [MapFeature, string][] = [['magic', 'Magic sites'], ['mine', 'Mines'], ['cave', 'Caverns']];
const MIN_TILE_COUNT = 18;
const MAX_TILE_COUNT = 120;
const randomSeed = () => typeof globalThis.crypto?.randomUUID === 'function'
  ? globalThis.crypto.randomUUID() : `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 12)}`;
const corners: [SeaCorner, string][] = [['NW', 'Northwest'], ['NE', 'Northeast'], ['SW', 'Southwest'], ['SE', 'Southeast']];
const clone = (settings: GeneratedMapSettings): GeneratedMapSettings => JSON.parse(JSON.stringify(settings));
// Server JSON object order need not match the order of local edits.
const fingerprint = (value: unknown): string => JSON.stringify(value, (_, item) =>
  item && typeof item === 'object' && !Array.isArray(item)
    ? Object.fromEntries(Object.keys(item).sort().map(key => [key, item[key]])) : item);

function RangeControl({ id, label, value, min = 0, max = 100, step = 1, suffix = '', hint, onChange, disabled = false }: {
  id: string; label: string; value: number; min?: number; max?: number; step?: number; suffix?: string;
  hint: string; onChange: (value: number) => void; disabled?: boolean;
}) {
  const inputId = `map-editor-${id}`;
  return <div className="map-editor-control">
    <label htmlFor={inputId}><span>{label}</span><output htmlFor={inputId}>{value}{suffix}</output></label>
    <input id={inputId} type="range" aria-label={label} min={min} max={max} step={step} value={value}
      aria-valuetext={`${value}${suffix}`} data-tooltip={hint} disabled={disabled}
      onChange={event => onChange(Number(event.target.value))} />
  </div>;
}

function NumberControl({ id, label, value, min = 0, max, step = 1, hint, onChange, clamp = true, invalid, describedBy }: {
  id: string; label: string; value: number; min?: number; max: number; step?: number;
  hint: string; onChange: (value: number) => void; clamp?: boolean; invalid?: boolean; describedBy?: string;
}) {
  const inputId = `map-editor-${id}`;
  return <label className="map-editor-number" htmlFor={inputId}><span>{label}</span>
    <input id={inputId} aria-label={label} type="number" value={Number.isFinite(value) ? value : ''} min={min} max={max} step={step}
      aria-invalid={invalid} aria-describedby={describedBy} data-tooltip={hint} onChange={event => {
        const next = event.target.valueAsNumber;
        if (!clamp) onChange(next);
        else if (Number.isFinite(next)) onChange(Math.min(max, Math.max(min, next)));
      }} />
  </label>;
}

export default function GeneratedMapEditor({ defaultSettings, initialSettings, preview, playerCount, loading,
  selecting = false, selectedPreviewId, error, automatic, onGenerate, onSelect, onAutomatic, onClose }: GeneratedMapEditorProps) {
  const [draft, setDraft] = useState(() => clone(
    preview && preview.playerCount !== playerCount
      ? defaultSettings
      : initialSettings ?? preview?.settings ?? defaultSettings));
  const panel = useRef<HTMLDivElement>(null);
  const lastPlayerCount = useRef(playerCount);
  const pendingDraft = useRef<string | null>(null);
  const lastPreviewId = useRef(preview?.id);
  const draftFingerprint = fingerprint(draft);
  const draftRef = useRef(draftFingerprint);
  draftRef.current = draftFingerprint;
  const previewMatches = !!preview && preview.playerCount === playerCount && fingerprint(preview.settings) === draftFingerprint;
  const previewSelected = !!preview && !automatic && selectedPreviewId === preview.id;
  const tileCountValid = Number.isInteger(draft.tileCount) && draft.tileCount >= MIN_TILE_COUNT && draft.tileCount <= MAX_TILE_COUNT;
  const canGenerate = tileCountValid && !loading && !selecting;
  const canSelect = previewMatches && canGenerate;

  useEffect(() => {
    if (lastPlayerCount.current === playerCount) return;
    lastPlayerCount.current = playerCount;
    pendingDraft.current = null;
    setDraft(clone(defaultSettings));
  }, [defaultSettings, playerCount]);

  useEffect(() => {
    if (!preview || preview.id === lastPreviewId.current) return;
    lastPreviewId.current = preview.id;
    // Adopt server normalization only if the user has not edited the request
    // while it was rendering. A late response must never overwrite newer edits.
    if (preview.playerCount === playerCount && pendingDraft.current === draftRef.current) setDraft(clone(preview.settings));
    pendingDraft.current = null;
  }, [preview, playerCount]);

  useEffect(() => {
    const previous = document.activeElement instanceof HTMLElement ? document.activeElement : null;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    panel.current?.focus();
    return () => {
      document.body.style.overflow = overflow;
      if (previous?.isConnected) previous.focus();
    };
  }, []);

  const set = <K extends keyof GeneratedMapSettings>(key: K, value: GeneratedMapSettings[K]) =>
    setDraft(current => ({ ...current, [key]: value }));
  const generate = (settings: GeneratedMapSettings) => {
    if (!canGenerate) return;
    pendingDraft.current = fingerprint(settings);
    onGenerate(clone(settings));
  };
  const generateRandom = () => {
    if (!canGenerate) return;
    const settings = { ...draft, seed: randomSeed() };
    setDraft(settings);
    generate(settings);
  };
  const handleKey = (event: React.KeyboardEvent) => {
    event.stopPropagation();
    if (event.key === 'Escape' && !event.nativeEvent.isComposing) { event.preventDefault(); onClose(); }
    if (event.key !== 'Tab') return;
    const controls = Array.from(panel.current?.querySelectorAll<HTMLElement>('button:not(:disabled), input:not(:disabled), select:not(:disabled), summary, [tabindex="0"]') ?? [])
      .filter(node => {
        const closed = node.closest('details:not([open])');
        return !node.hidden && !node.closest('[hidden]') && (!closed || (node.tagName === 'SUMMARY' && node.parentElement === closed));
      });
    const first = controls[0], last = controls[controls.length - 1];
    if (!first || !last) { event.preventDefault(); panel.current?.focus(); return; }
    if (document.activeElement === panel.current || (event.shiftKey ? document.activeElement === first : document.activeElement === last)) {
      event.preventDefault(); (event.shiftKey ? last : first).focus();
    }
  };
  const countError = `Choose a whole tile count from ${MIN_TILE_COUNT} to ${MAX_TILE_COUNT}.`;
  const status = !tileCountValid ? countError : selecting ? 'Selecting this map…' : loading ? 'Generating your preview…' : error ? error
    : previewMatches ? (previewSelected ? 'This preview is selected for the room.' : 'Preview ready. Select it to use this exact map.')
    : preview ? 'Settings changed. Generate a new preview before selecting.' : 'Adjust the settings, then generate a preview.';

  return <div className="map-editor-backdrop">
    <div className="map-editor" role="dialog" aria-modal="true" aria-labelledby="map-editor-title" ref={panel} tabIndex={-1} onKeyDown={handleKey}>
      <header className="map-editor-header">
        <button type="button" className="map-editor-back" onClick={onClose} data-tooltip="Return to the lobby. Unselected edits do not change the room’s map. Shortcut: Escape." aria-keyshortcuts="Escape"><FiArrowLeft aria-hidden="true" /> Back</button>
        <div className="map-editor-heading"><span className="map-editor-eyebrow">Map workshop</span><h2 id="map-editor-title">Generated map</h2></div>
        <button type="button" className="map-editor-automatic" aria-pressed={automatic} disabled={selecting} onClick={onAutomatic}
          data-tooltip="Use a fresh automatic map when the game starts, with the current per-player defaults. This replaces any selected preview.">
          {automatic && <FiCheck aria-hidden="true" />}<span>Automatic at game start{automatic && <small>Current choice</small>}</span>
        </button>
      </header>
      <div className="map-editor-layout">
        <aside className="map-editor-settings" aria-label="Map settings">
          <div className="map-editor-settings-intro"><strong>{playerCount} players · {tileCountValid ? draft.tileCount : '…'} regions</strong>
            <p>Explore a layout here before choosing it for your room.</p>
          </div>
          <div className="map-editor-tile-count">
            <NumberControl id="tile-count" label="Tile count" value={draft.tileCount} min={MIN_TILE_COUNT} max={MAX_TILE_COUNT}
              clamp={false} invalid={!tileCountValid} describedBy="map-editor-tile-count-help"
              hint={`Total regions, including water. Choose ${MIN_TILE_COUNT}–${MAX_TILE_COUNT}; the default for ${playerCount} players is ${defaultSettings.tileCount}.`}
              onChange={value => set('tileCount', value)} />
            <p id="map-editor-tile-count-help">Default for {playerCount} players: {defaultSettings.tileCount} tiles. Includes water regions.</p>
          </div>
          <div className="map-editor-seed">
            <label htmlFor="map-editor-seed">Seed</label>
            <div><input id="map-editor-seed" type="text" value={draft.seed} maxLength={128} placeholder="Random seed"
              data-tooltip="The same seed and settings reproduce the same map. Leave empty for a new seed." onChange={event => set('seed', event.target.value)} />
              <button type="button" aria-label="Randomize seed" data-tooltip="Choose a new seed. Generate a preview to see the resulting layout."
                onClick={() => set('seed', randomSeed())}><FiShuffle aria-hidden="true" /></button></div>
          </div>
          <details className="map-editor-group" open>
            <summary>Terrain</summary><div className="map-editor-group-body">
              <p>Weights set the relative share of each land terrain; they are not percentages.</p>
              {biomes.map(([key, label, color]) => <div className="map-editor-biome" key={key}><span style={{ backgroundColor: color }} aria-hidden="true" />
                <RangeControl id={`weight-${key}`} label={`${label} weight`} value={draft.biomeWeights[key]} max={300} step={5}
                  hint={`Relative amount of ${label.toLowerCase()}. Higher values give it more of the land; zero minimizes it.`}
                  onChange={value => set('biomeWeights', { ...draft.biomeWeights, [key]: value })} /></div>)}
              <RangeControl id="spread" label="Terrain clustering" value={draft.spread}
                hint="Low values scatter terrain types. High values gather matching terrain into larger clusters."
                onChange={value => set('spread', value)} />
              <div className="map-editor-scale"><span>Scattered</span><span>Clustered</span></div>
            </div>
          </details>
          <details className="map-editor-group" open>
            <summary>Water</summary><div className="map-editor-group-body">
              <label className="map-editor-check"><input type="checkbox" checked={draft.randomizeSeaCorners}
                data-tooltip="Shuffle which corners contain sea. The number of checked corners sets the count; none checked means two random corners."
                onChange={event => set('randomizeSeaCorners', event.target.checked)} /><span>Randomize sea corners</span></label>
              <fieldset className="map-editor-corners"><legend>{draft.randomizeSeaCorners ? 'Number of sea corners' : 'Sea corners'}</legend>
                {corners.map(([key, label]) => <label key={key} data-tooltip={draft.randomizeSeaCorners ? 'Each checked box adds one randomly chosen sea corner.' : `Place sea in the ${label.toLowerCase()} corner.`}>
                  <input type="checkbox" aria-label={`${label} sea`} checked={draft.seaCorners[key]} onChange={event => set('seaCorners', { ...draft.seaCorners, [key]: event.target.checked })} /> {key}
                </label>)}
              </fieldset>
              {draft.randomizeSeaCorners && <p>The checked count is used; locations are shuffled. With none checked, two corners are chosen.</p>}
              <RangeControl id="sea-size" label="Sea size" value={draft.waterSettings.seaSize} min={40} max={240} step={5} suffix="%"
                hint="Scale the area covered by each sea corner. 100% is its base size."
                onChange={value => set('waterSettings', { ...draft.waterSettings, seaSize: value })} />
              <label className="map-editor-check"><input type="checkbox" checked={draft.centerLake} onChange={event => set('centerLake', event.target.checked)}
                data-tooltip="Add a lake near the center of the map." /><span>Central lake</span></label>
              <RangeControl id="lake-size" label="Lake size" value={draft.waterSettings.centerLakeSize} min={40} max={240} step={5} suffix="%" disabled={!draft.centerLake}
                hint="Scale the central lake. Enable Central lake to apply this setting."
                onChange={value => set('waterSettings', { ...draft.waterSettings, centerLakeSize: value })} />
            </div>
          </details>
          <details className="map-editor-group">
            <summary>Features</summary><div className="map-editor-group-body">
              <p>Magic sites, mines and caverns affect gameplay. Counts are limited by available land regions.</p>
              {features.map(([key, label]) => <div className="map-editor-feature" key={key}>
                <NumberControl id={`${key}-count`} label={`${label} count`} value={draft.featureSettings[key].count} max={40}
                  hint={`Requested number of ${label.toLowerCase()}. A region can carry one feature.`}
                  onChange={value => set('featureSettings', { ...draft.featureSettings, [key]: { ...draft.featureSettings[key], count: Math.round(value) } })} />
                <RangeControl id={`${key}-spread`} label={`${label} clustering`} value={draft.featureSettings[key].spread}
                  hint={`Low values scatter ${label.toLowerCase()}; high values group them together.`}
                  onChange={value => set('featureSettings', { ...draft.featureSettings, [key]: { ...draft.featureSettings[key], spread: value } })} />
              </div>)}
            </div>
          </details>
          <details className="map-editor-group">
            <summary>Scenery & buildings</summary><div className="map-editor-group-body">
              <p>These settings change the artwork, not region rules.</p>
              <RangeControl id="tree-size" label="Tree size" value={draft.scenerySettings.forestTreeSize} min={30} max={220} step={5} suffix="%"
                hint="Larger trees are drawn less densely to keep the forest readable."
                onChange={value => set('scenerySettings', { ...draft.scenerySettings, forestTreeSize: value })} />
              <RangeControl id="mountain-size" label="Mountain size" value={draft.scenerySettings.mountainObjectSize} min={35} max={240} step={5} suffix="%"
                hint="Scale decorative mountain artwork. Larger objects are placed less densely."
                onChange={value => set('scenerySettings', { ...draft.scenerySettings, mountainObjectSize: value })} />
              <NumberControl id="building-average" label="Buildings per region" value={draft.buildingSettings.averagePerTile} max={5} step={0.05}
                hint="Average ordinary buildings per eligible land region. Feature-specific buildings use their separate setting below."
                onChange={value => set('buildingSettings', { ...draft.buildingSettings, averagePerTile: value })} />
              <NumberControl id="building-max" label="Maximum per region" value={draft.buildingSettings.maxPerTile} max={5}
                hint="Maximum ordinary buildings in a region. Set to zero to remove ordinary buildings."
                onChange={value => set('buildingSettings', { ...draft.buildingSettings, maxPerTile: Math.round(value) })} />
              <RangeControl id="building-size" label="Building size" value={draft.buildingSettings.sizeMultiplier} min={0.2} max={3} step={0.05} suffix="×"
                hint="Scale decorative buildings without changing the size or rules of their regions."
                onChange={value => set('buildingSettings', { ...draft.buildingSettings, sizeMultiplier: value })} />
              <RangeControl id="symbol-buildings" label="Feature sites with buildings" value={draft.buildingSettings.symbolBuildingChance} step={5} suffix="%"
                hint="Share of magic, mine and cavern regions with a matching decorative building. Feature rules remain the same."
                onChange={value => set('buildingSettings', { ...draft.buildingSettings, symbolBuildingChance: value })} />
              <h3>Terrain artwork zoom</h3>
              <p>Higher zoom enlarges the texture pattern within each region.</p>
              {[...biomes.map(([key, label]) => [key, label] as const), ['sea', 'Water'] as const].map(([key, label]) =>
                <RangeControl key={key} id={`texture-${key}`} label={`${label} texture zoom`} value={draft.biomeTextureZooms[key]} min={25} max={400} step={5} suffix="%"
                  hint={`Zoom the ${label.toLowerCase()} texture. This changes the artwork only.`}
                  onChange={value => set('biomeTextureZooms', { ...draft.biomeTextureZooms, [key]: value })} />)}
            </div>
          </details>
          <details className="map-editor-group">
            <summary>Advanced shape & borders</summary><div className="map-editor-group-body">
              <label className="map-editor-select" htmlFor="map-editor-shape">Map shape
                <select id="map-editor-shape" value={draft.shapeMode} data-tooltip="Change the distribution of region centers. Island-style creates a less regular arrangement; it does not add the Sky Islands rules."
                  onChange={event => set('shapeMode', event.target.value as GeneratedMapSettings['shapeMode'])}>
                  <option value="mainland">Mainland / coast</option><option value="islands">Island-style layout</option>
                </select>
              </label>
              <RangeControl id="wobble" label="Border irregularity" value={draft.wobble} max={30}
                hint="Add organic bends to region borders. Zero gives straight borders."
                onChange={value => set('wobble', value)} />
              <RangeControl id="chonkiness" label="Region roundness" value={draft.tileShapeSettings.chonkiness} step={5}
                hint="Space region centers more evenly to make regions rounder and easier to place pieces on."
                onChange={value => set('tileShapeSettings', { ...draft.tileShapeSettings, chonkiness: value })} />
              <NumberControl id="shape-passes" label="Shape smoothing passes" value={draft.tileShapeSettings.repulsionSteps} max={12}
                hint="Number of passes used to space region centers. More passes make spacing more even."
                onChange={value => set('tileShapeSettings', { ...draft.tileShapeSettings, repulsionSteps: Math.round(value) })} />
              <p className="map-editor-fixed">Canvas: {defaultSettings.width} × {defaultSettings.height}. Canvas size stays fixed; tile count changes the number of regions within it.</p>
            </div>
          </details>
          <button type="button" className="map-editor-reset" disabled={selecting} data-tooltip="Restore the current defaults for this player count. Generate a new preview to use them."
            onClick={() => { pendingDraft.current = null; setDraft(clone(defaultSettings)); }}><FiRefreshCw aria-hidden="true" /> Reset to current defaults</button>
        </aside>
        <main className="map-editor-preview" aria-label="Map preview" aria-busy={loading}>
          <div className="map-editor-preview-heading"><span>Preview</span><span>{previewMatches ? `${preview!.settings.tileCount} regions · ${preview!.playerCount} players` : 'Your room is unchanged until you select a map'}</span></div>
          <div className={`map-editor-canvas ${preview && !previewMatches ? 'map-editor-canvas-stale' : ''}`}>
            {preview ? <img src={preview.imageUrl} alt={`Generated map preview for ${preview.playerCount} players`} />
              : <div className="map-editor-empty"><FiMap aria-hidden="true" /><h3>A world of your own</h3><p>Choose terrain, water and scenery, then generate a preview. The room keeps its current map choice until you select one.</p></div>}
            {loading && <div className="map-editor-loading" role="status"><FiRefreshCw aria-hidden="true" /><span>Generating map…</span></div>}
            {!loading && preview && !previewMatches && <span className="map-editor-stale-label">Earlier preview · settings have changed</span>}
            {!loading && previewMatches && previewSelected && <span className="map-editor-selected-label"><FiCheck aria-hidden="true" /> Selected for this room</span>}
          </div>
          {preview && <p className="map-editor-preview-caption">Preview seed: <span>{preview.settings.seed || 'Random'}</span></p>}
        </main>
      </div>
      <footer className="map-editor-footer">
        <p role={error || !tileCountValid ? 'alert' : 'status'} className={error || !tileCountValid ? 'map-editor-error' : ''}>{status}</p>
        <div><button type="button" className="map-editor-generate" disabled={!canGenerate} onClick={() => generate(draft)}
            data-tooltip="Generate a preview from these settings. This does not change the room’s selected map."><FiRefreshCw aria-hidden="true" /> {loading ? 'Generating…' : 'Generate preview'}</button>
          <button type="button" className="map-editor-generate" disabled={!canGenerate} onClick={generateRandom}
            data-tooltip="Generate a preview immediately with a fresh seed, keeping your tile count and all other settings."><FiShuffle aria-hidden="true" /> Generate random map</button>
          <button type="button" className="map-editor-select-button" disabled={!canSelect} onClick={() => { if (canSelect && preview) onSelect(preview.id); }}
            data-tooltip={canSelect ? 'Use this exact preview when the game starts.' : 'Generate a preview matching the current settings before selecting it.'}>
            {selecting ? 'Selecting…' : 'Select this map'}<FiCheck aria-hidden="true" /></button></div>
      </footer>
    </div>
  </div>;
}
