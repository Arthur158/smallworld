import React, { ButtonHTMLAttributes } from 'react';
import { twMerge } from 'tailwind-merge';

type ButtonProps = { size?: 'sm' | 'md' | 'lg' } & ButtonHTMLAttributes<HTMLButtonElement>;

const sizes = {
  sm: 'px-2 py-1 rounded-[8px]',
  md: 'px-3 py-1.5 text-md rounded-[10px]',
  lg: 'px-4 py-2 text-lg rounded-[12px]',
};

export default function Button({
  className,
  size = 'md',
  ...props
}: ButtonProps): JSX.Element {
  return (
    <button
      type="button"
      {...props}
      className={twMerge(
        'disabled:opacity-20 transition-all flex whitespace-nowrap flex-row hover:bg-opacity-80 disabled:hover:bg-opacity-100 h-fit items-center justify-center flex font-[600] gap-1 bg-black text-white',
        sizes[size],
        className,
      )}

    />
  );
}
