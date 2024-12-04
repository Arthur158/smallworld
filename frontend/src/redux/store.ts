// src/store.ts
import { AnyAction, configureStore } from '@reduxjs/toolkit';
import storage from 'redux-persist/lib/storage';
import { persistReducer, persistStore } from 'redux-persist';
import thunk from 'redux-thunk';
import rootReducer from './reducer';

// Configuration for redux-persist
const persistConfig = {
  key: 'root',
  storage,
  whitelist: ['application'], // Specify which slices of the state should be persisted
};

// Wrap the root reducer with persistReducer
const persistedReducer = persistReducer<ReturnType<typeof rootReducer>, AnyAction>(
  persistConfig,
  rootReducer,
);

// Create the Redux store with persisted reducer and middleware (thunk)
export const store = configureStore({
  reducer: persistedReducer,
  devTools: process.env.NODE_ENV !== 'production', // Enable Redux DevTools in development
  middleware: (getDefaultMiddleware) =>
    getDefaultMiddleware().concat(thunk),
});

// Export persistor to persist the store
export const persistor = persistStore(store);

// Export RootState and AppDispatch types
export type RootState = ReturnType<typeof store.getState>; // Fix for the missing RootState
export type AppDispatch = typeof store.dispatch;
