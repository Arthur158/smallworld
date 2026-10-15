import React, { useState } from 'react';
import { useShortcut } from '../../hooks/useShortcut';
import ShortcutsDialog from '../help/ShortcutsDialog';
import { FiBookOpen, FiCrosshair } from 'react-icons/fi';
import { useWiki } from './WikiContext';

export default function WikiToolbar({ compact = false }: { compact?: boolean }) {
  const { openWiki, inspectMode, setInspectMode } = useWiki();
  const [showHelp, setShowHelp] = useState(false);
  useShortcut('w', () => openWiki());
  useShortcut('i', () => setInspectMode(!inspectMode), !compact);
  useShortcut('h', () => setShowHelp(true));
  useShortcut('?', () => setShowHelp(true));
  return <div className={`wiki-toolbar flex flex-wrap items-center gap-2 ${compact ? '' : 'mb-3 shrink-0'}`}>
    <button type="button" aria-keyshortcuts="W" data-tooltip="Browse races, traits, powers and pieces in the game wiki. Shortcut: W." onClick={() => openWiki()} className="inline-flex items-center gap-2 rounded border border-[#8B4513] bg-[#FDF5E6] px-3 py-1.5 font-semibold hover:bg-[#EED5B7]">
      <FiBookOpen aria-hidden="true" /> Game wiki
    </button>
    {!compact && <>
      <button type="button" aria-pressed={inspectMode} onClick={() => setInspectMode(!inspectMode)} aria-keyshortcuts="I" data-tooltip="Toggle Inspect mode, then click a race, trait, piece or region to read its rules. Shortcut: I. Ctrl/⌘ + click also inspects; Escape exits."
        className={`inline-flex items-center gap-2 rounded border border-[#8B4513] px-3 py-1.5 ${inspectMode ? 'bg-[#8B4513] text-white' : 'bg-[#FDF5E6] hover:bg-[#EED5B7]'}`}>
        <FiCrosshair aria-hidden="true" /> {inspectMode ? 'Inspecting' : 'Inspect'}
      </button>
      <span className="text-xs text-[#75634b]" role="status">{inspectMode ? 'Click a tribe, piece or region · Esc to exit' : 'Ctrl / ⌘ + click to inspect'}</span>
    </>}
    <button type="button" aria-label="Controls and shortcuts" aria-keyshortcuts="H Shift+?" data-tooltip="Show mouse controls and keyboard shortcuts. Shortcut: ? or H." onClick={() => setShowHelp(true)} className="inline-flex items-center justify-center rounded border border-[#8B4513] bg-[#FDF5E6] w-8 h-8 font-bold hover:bg-[#EED5B7]">?</button>
    {showHelp && <ShortcutsDialog onClose={() => setShowHelp(false)} />}
  </div>;
}
