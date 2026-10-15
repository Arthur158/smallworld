import React, { createContext, useCallback, useContext, useEffect, useState } from 'react';
import { useSelector } from 'react-redux';
import { RootState } from '../../redux/store';
import { PieceStack, Tile, Tribe } from '../../types/Board';
import WikiDialog from './WikiDialog';

export type WikiTarget =
  | { kind: 'tribe'; tribe: Tribe }
  | { kind: 'piece'; stack: PieceStack; tribe?: Tribe; tileId?: string }
  | { kind: 'zone'; tile: Tile }
  | { kind: 'article'; id: string };
type InspectEvent = Pick<React.MouseEvent | React.KeyboardEvent, 'ctrlKey' | 'metaKey' | 'preventDefault' | 'stopPropagation'>;
interface WikiContextValue {
  isOpen: boolean;
  inspectMode: boolean;
  setInspectMode: (value: boolean) => void;
  openWiki: (target?: WikiTarget) => void;
  inspect: (event: InspectEvent, target: WikiTarget) => boolean;
}
const WikiContext = createContext<WikiContextValue>({ isOpen: false, inspectMode: false, setInspectMode: () => {}, openWiki: () => {}, inspect: () => false });
export const useWiki = () => useContext(WikiContext);

export function WikiProvider({ children }: { children: React.ReactNode }) {
  const [isOpen, setIsOpen] = useState(false);
  const [inspectMode, setInspectMode] = useState(false);
  const [target, setTarget] = useState<WikiTarget>();
  const roomId = useSelector((state: RootState) => state.application.roomid);
  const authenticated = useSelector((state: RootState) => state.application.isAuthenticated);
  useEffect(() => { setIsOpen(false); setInspectMode(false); setTarget(undefined); }, [roomId, authenticated]);
  const openWiki = useCallback((next?: WikiTarget) => { setTarget(next); setIsOpen(true); }, []);
  const inspect = useCallback((event: InspectEvent, next: WikiTarget) => {
    if (!event.ctrlKey && !event.metaKey && !inspectMode) return false;
    event.preventDefault(); event.stopPropagation(); openWiki(next); return true;
  }, [inspectMode, openWiki]);
  useEffect(() => {
    const onKey = (event: KeyboardEvent) => {
      if (event.key === 'Escape' && !isOpen) setInspectMode(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [isOpen]);
  return <WikiContext.Provider value={{ isOpen, inspectMode, setInspectMode, openWiki, inspect }}>
    {children}
    {isOpen && <WikiDialog initialTarget={target} onClose={() => setIsOpen(false)} />}
  </WikiContext.Provider>;
}
