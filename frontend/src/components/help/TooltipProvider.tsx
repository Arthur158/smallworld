import React, { useEffect, useLayoutEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';
import './help.css';

export const TOOLTIP_DELAY = 700;
const tooltipId = 'smallworld-control-hint';
type Hint = { anchor: Element; text: string };

/** Delegated hints also cover SVG stacks and disabled controls without layout wrappers. */
export default function TooltipProvider({ children }: { children: React.ReactNode }) {
  const [hint, setHint] = useState<Hint | null>(null);
  const [position, setPosition] = useState({ left: 8, top: 8 });
  const bubble = useRef<HTMLDivElement>(null);
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout> | undefined;
    let anchor: Element | null = null;
    const cancel = () => { if (timer !== undefined) clearTimeout(timer); timer = undefined; };
    const hide = () => { cancel(); anchor = null; setHint(null); };
    const find = (target: EventTarget | null) => target instanceof Element ? target.closest('[data-tooltip], button, [role="button"]') : null;
    const hintText = (element: Element) => {
      const label = element.getAttribute('aria-label') || element.textContent?.trim().replace(/\s+/g, ' ');
      return element.getAttribute('data-tooltip') || (label ? `Click to ${label.charAt(0).toLowerCase()}${label.slice(1)}. Keyboard: Tab to focus, then Enter or Space.` : 'Click to activate. Keyboard: Tab to focus, then Enter or Space.');
    };
    const begin = (target: EventTarget | null, delay: number) => {
      const next = find(target);
      if (!next || next === anchor || bubble.current?.contains(next)) return;
      hide();
      anchor = next;
      timer = setTimeout(() => { if (next.isConnected) setHint({ anchor: next, text: hintText(next) }); }, delay);
    };
    const over = (event: PointerEvent) => { if (event.pointerType !== 'touch') begin(event.target, TOOLTIP_DELAY); };
    const out = (event: PointerEvent) => {
      const related = event.relatedTarget as Node | null;
      if (related && (anchor?.contains(related) || bubble.current?.contains(related))) return;
      hide();
    };
    const focus = (event: FocusEvent) => begin(event.target, 0);
    const blur = (event: FocusEvent) => { if (!bubble.current?.contains(event.relatedTarget as Node | null)) hide(); };
    const key = () => hide();
    const observer = new MutationObserver(() => {
      if (!anchor) return;
      if (!anchor.isConnected) { hide(); return; }
      const text = hintText(anchor);
      setHint(current => current && current.anchor === anchor && current.text !== text ? { ...current, text } : current);
    });
    observer.observe(document.body, { childList: true, subtree: true, characterData: true, attributes: true, attributeFilter: ['data-tooltip', 'aria-label'] });
    document.addEventListener('pointerover', over);
    document.addEventListener('pointerout', out);
    document.addEventListener('focusin', focus);
    document.addEventListener('focusout', blur);
    document.addEventListener('pointerdown', hide);
    document.addEventListener('keydown', key);
    window.addEventListener('scroll', hide, true);
    window.addEventListener('resize', hide);
    return () => {
      cancel(); observer.disconnect(); document.removeEventListener('pointerover', over); document.removeEventListener('pointerout', out);
      document.removeEventListener('focusin', focus); document.removeEventListener('focusout', blur);
      document.removeEventListener('pointerdown', hide); document.removeEventListener('keydown', key);
      window.removeEventListener('scroll', hide, true); window.removeEventListener('resize', hide);
    };
  }, []);
  useLayoutEffect(() => {
    if (!hint || !bubble.current) return;
    const rect = hint.anchor.getBoundingClientRect();
    const box = bubble.current.getBoundingClientRect();
    setPosition({ left: Math.max(8, Math.min(window.innerWidth - box.width - 8, rect.left + (rect.width - box.width) / 2)), top: rect.bottom + box.height + 18 < window.innerHeight ? rect.bottom + 8 : Math.max(8, rect.top - box.height - 8) });
    const old = hint.anchor.getAttribute('aria-describedby');
    hint.anchor.setAttribute('aria-describedby', [old, tooltipId].filter(Boolean).join(' '));
    return () => { if (old) hint.anchor.setAttribute('aria-describedby', old); else hint.anchor.removeAttribute('aria-describedby'); };
  }, [hint]);
  return <>{children}{hint && createPortal(<div ref={bubble} id={tooltipId} role="tooltip" className="control-tooltip" style={position}>{hint.text}</div>, document.body)}</>;
}
