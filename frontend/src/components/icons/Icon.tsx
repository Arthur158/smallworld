import React from 'react';
import { IconType } from 'react-icons';

interface IconProps {
  className?: string;
  icon?: IconType;
}

export default function Icon({ className, icon }: IconProps): JSX.Element | null {
  if (!icon) return null;
  return React.createElement(icon, {
    className,
    'aria-hidden': true,
  });
}
