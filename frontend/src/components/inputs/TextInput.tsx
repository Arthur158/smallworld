import React, { InputHTMLAttributes } from 'react';
import { twMerge } from 'tailwind-merge';

type TextInputProps = {
  text?: string;
  setText?: (text: string) => void;
} & InputHTMLAttributes<HTMLInputElement>;

export default function TextInput({
  text,
  setText,
  className,
  value,
  onChange,
  ...props
}: TextInputProps): JSX.Element {
  return (
    <input
      {...props}
      type="text"
      className={twMerge('border w-full border-black border-2 h-10 px-2 rounded-[4px]', className)}
      value={value ?? text ?? ''}
      onChange={onChange ?? ((e): void => setText?.(e.target.value))}
    />
  );
}
