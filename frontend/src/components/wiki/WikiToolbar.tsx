import React from 'react';
import { FiBookOpen, FiCrosshair } from 'react-icons/fi';
import { useWiki } from './WikiContext';

export default function WikiToolbar({ compact = false }: { compact?: boolean }) {
  const { openWiki, inspectMode, setInspectMode } = useWiki();
  return <div className={`wiki-toolbar flex flex-wrap items-center gap-2 ${compact ? '' : 'mb-3 shrink-0'}`}>
    <button type="button" onClick={() => openWiki()} className="inline-flex items-center gap-2 rounded border border-[#8B4513] bg-[#FDF5E6] px-3 py-1.5 font-semibold hover:bg-[#EED5B7]">
      <FiBookOpen aria-hidden="true" /> Game wiki
    </button>
    {!compact && <>
      <button type="button" aria-pressed={inspectMode} onClick={() => setInspectMode(!inspectMode)} title="Click a tribe, piece or region to read about it without making a move"
        className={`inline-flex items-center gap-2 rounded border border-[#8B4513] px-3 py-1.5 ${inspectMode ? 'bg-[#8B4513] text-white' : 'bg-[#FDF5E6] hover:bg-[#EED5B7]'}`}>
        <FiCrosshair aria-hidden="true" /> {inspectMode ? 'Inspecting' : 'Inspect'}
      </button>
      <span className="text-xs text-[#75634b]" role="status">{inspectMode ? 'Click a tribe, piece or region · Esc to exit' : 'Ctrl / ⌘ + click to inspect'}</span>
    </>}
  </div>;
}
