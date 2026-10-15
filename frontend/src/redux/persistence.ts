import type { ApplicationState } from '../types/redux';
import { Language } from '../types/misc';
import applicationReducer from './slices/applicationSlice';

// Authentication and game state are server-owned. Browser snapshots cannot
// establish a session and their blob URLs are invalid after a reload.
export function persistApplicationPreferences(state: ApplicationState): Pick<ApplicationState, 'language'> {
  return { language: state.language };
}

export function restoreApplicationPreferences(persisted: unknown): ApplicationState {
  const initial = applicationReducer(undefined, { type: 'application/restorePreferences' });
  const language = persisted && typeof persisted === 'object' && 'language' in persisted
    ? (persisted as { language?: unknown }).language : undefined;
  return {
    ...initial,
    language: language === Language.EN || language === Language.NL ? language : initial.language,
  };
}
