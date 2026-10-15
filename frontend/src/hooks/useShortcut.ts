import { useEffect, useRef } from 'react';

/** Screen shortcuts never take keystrokes from editors, modal dialogs or the browser. */
export function canUseShortcut(event: KeyboardEvent): boolean {
  if (event.defaultPrevented || event.repeat || event.isComposing || event.ctrlKey || event.metaKey || event.altKey) return false;
  const target = event.target instanceof Element ? event.target : document.activeElement;
  if (target?.closest('input, textarea, select, [contenteditable]:not([contenteditable="false"]), [role="textbox"], [role="dialog"]')) return false;
  if (document.querySelector('[role="dialog"][aria-modal="true"]')) return false;
  // Preserve native keyboard activation on focused controls.
  if ((event.key === ' ' || event.key === 'Enter') && target?.closest('button, a, [role="button"]')) return false;
  return true;
}

export function useShortcut(key: string, action: () => void, enabled = true): void {
  const latest = useRef(action);
  latest.current = action;
  useEffect(() => {
    if (!enabled) return;
    const explicitShift = key.startsWith('Shift+');
    const wanted = explicitShift ? key.slice(6) : key;
    const acceptsEitherShift = !explicitShift && ['+', '?'].includes(wanted);
    const normalized = wanted === 'Space' ? ' ' : wanted.toLowerCase();
    const onKey = (event: KeyboardEvent) => {
      if (!canUseShortcut(event) || (!acceptsEitherShift && event.shiftKey !== explicitShift) || event.key.toLowerCase() !== normalized) return;
      event.preventDefault();
      latest.current();
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [key, enabled]);
}
