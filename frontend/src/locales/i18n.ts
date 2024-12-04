import i18n from 'i18next';
import { initReactI18next } from 'react-i18next';
import { store } from '../redux/store';
import en from './resources/en.json';
import nl from './resources/nl.json';

i18n.use(initReactI18next).init({
  lng: 'nl',
  debug: process.env.NODE_ENV !== 'production',
  resources: {
    nl,
    en,
  },
  interpolation: { escapeValue: false },
});

store.subscribe((): void => {
  const { language } = store.getState().application;
  if (language !== i18n.language) {
    i18n.changeLanguage(language);
  }
});

export default i18n;
