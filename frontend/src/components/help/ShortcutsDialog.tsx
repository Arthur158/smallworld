import React, { useEffect, useRef } from 'react';
import './help.css';

const groups: { title: string; rows: [string, string][] }[] = [
  { title: 'Explore & navigate', rows: [['Game wiki', 'W'], ['Toggle Inspect mode', 'I'], ['Inspect a race, trait, piece or region', 'Ctrl / ⌘ + click'], ['Switch players / tribe choices', 'S'], ['Clear selected pieces', 'Esc or F'], ['Exit Inspect mode', 'Esc'], ['Dismiss error message', 'C'], ['This shortcut guide', '? or H']] },
  { title: 'Your turn', rows: [['Choose the numbered tribe in the market', '1–9'], ['Cycle through pieces in your reserve', 'G'], ['Abandon selected region / return a piece during redeployment', 'A'], ['Decline', 'D'], ['Start redeployment', 'R'], ['Finish turn', 'E'], ['Save game', 'Shift + S'], ['Rollback', 'Shift + R'], ['Leave game or room', 'Shift + L']] },
  { title: 'Bot playback · host only', rows: [['Pause / resume', 'Space'], ['Previous move · while paused', '←'], ['Next move · while paused', '→'], ['Slower / faster', '− / +']] },
  { title: 'Lobby', rows: [['Switch saves / game choices', 'S'], ['Start game · room host', 'P'], ['Browse saved games and maps', 'V'], ['Standard / Underground preset · choices tab', 'N / U'], ['Play / watch as host', 'O'], ['Add a bot', 'B'], ['Apply bot-compatible rules', 'Shift + B']] },
  { title: 'Mouse & focused controls', rows: [['Select a piece or region', 'Click'], ['Move around the map', 'Drag'], ['Zoom map', 'Scroll'], ['Move between controls', 'Tab / Shift + Tab'], ['Activate the focused button', 'Enter / Space'], ['Read an action hint', 'Hover for 0.7 seconds']] },
];
export default function ShortcutsDialog({ onClose }: { onClose: () => void }) {
  const panel = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    const overflow = document.body.style.overflow;
    document.body.style.overflow = 'hidden';
    panel.current?.focus();
    return () => { document.body.style.overflow = overflow; previous?.isConnected && previous.focus(); };
  }, []);
  return <div className="shortcut-backdrop" onMouseDown={event => { if (event.target === event.currentTarget) onClose(); }}>
    <div className="shortcut-dialog" role="dialog" aria-modal="true" aria-labelledby="shortcuts-title" ref={panel} tabIndex={-1} onKeyDown={event => {
      event.stopPropagation();
      if (event.key === 'Escape') { event.preventDefault(); onClose(); }
      if (event.key === 'Tab') { event.preventDefault(); panel.current?.querySelector<HTMLButtonElement>('button')?.focus(); }
    }}>
      <header><h2 id="shortcuts-title">Controls & shortcuts</h2><button type="button" aria-label="Close shortcuts" data-tooltip="Close this guide. Shortcut: Escape." onClick={onClose}>×</button></header>
      <p>Shortcuts work on the current screen, outside text fields and dialogs. Game actions are available only on your turn and in the appropriate phase. Inspect mode lets you read without making a move.</p>
      {groups.map(group => <section key={group.title}><h3>{group.title}</h3><dl>{group.rows.map(([action, key]) => <div key={action}><dt>{action}</dt><dd><kbd>{key}</kbd></dd></div>)}</dl></section>)}
    </div>
  </div>;
}
