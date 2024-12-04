import { useSelector } from 'react-redux';
import { RootState } from '../../types/redux';
import { Language } from '../../types/misc';

export default function useLanguage(): Language {
  return useSelector((state: RootState): Language => state.application.language);
}
