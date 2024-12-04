import React from 'react';
import { useDispatch } from 'react-redux';
import useLanguage from '../../hooks/selectors/useLanguage';
import { Language } from '../../types/misc';
import { setLanguage } from '../../redux/slices/applicationSlice';

export default function LanguageButton(): JSX.Element {
  const lng = useLanguage();
  const dispatch = useDispatch();
  const oppLng = lng === Language.EN ? Language.NL : Language.EN;
  return (
    <img
      alt="Language Switch"
      src={`${process.env.PUBLIC_URL}/assets/flags/${oppLng}.svg`}
      onClick={() => dispatch(setLanguage(oppLng))}
      className="w-6 h-6 cursor-pointer"
    />
  );
}
