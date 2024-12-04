import { Language } from './misc';

export interface User {
  email: string;
  password: string;
}

export interface ApplicationState {
  language: Language;
  user: User | null; 
  error: string | null;
  selectedTribe: string | null,
  availableTribes: string[]
}

export type RootState = {
  application: ApplicationState;
};
