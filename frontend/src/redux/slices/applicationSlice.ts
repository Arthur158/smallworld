import { PayloadAction, createSlice } from '@reduxjs/toolkit';
import { ApplicationState, User } from '../../types/redux';
import { Language } from '../../types/misc';

const initialState: ApplicationState = {
  language: Language.NL,
  user: null, 
  error: null,
  selectedTribe: null,
  availableTribes: ["elves", "dwarves", "giants", "goblins", "trolls", "skelettons"],
};

const applicationSlice = createSlice({
  name: 'application',
  initialState,
  reducers: {
    setLanguage(state, action: PayloadAction<Language>): void {
      state.language = action.payload;
    },
    login(state, action: PayloadAction<User>): void {
      state.user = action.payload;
    },
    clearError(state) {
      state.error = null;
    },
    setError(state, action: PayloadAction<string>) {
      state.error = action.payload;
    },
    selectTribe(state, action) {
      state.selectedTribe = action.payload;
    },

  },
});

const applicationReducer = applicationSlice.reducer;

export const { setLanguage, login, clearError, setError, selectTribe } = applicationSlice.actions;
export default applicationReducer;
