import type { InputHTMLAttributes } from "react";

interface InputProps extends InputHTMLAttributes<HTMLInputElement> {
  hasError?: boolean;
}

export function Input({ hasError, className = "", ...props }: InputProps) {
  return (
    <input
      className={`control-input block w-full px-3 py-2 text-sm text-zinc-950 placeholder-zinc-400 ${
        hasError
          ? "border-rose-300 focus:border-rose-500 focus:!shadow-[0_0_0_3px_rgba(239,68,68,0.15)]"
          : ""
      } ${className}`}
      {...props}
    />
  );
}
