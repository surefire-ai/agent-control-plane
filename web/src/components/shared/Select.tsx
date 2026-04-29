import type { SelectHTMLAttributes } from "react";

interface SelectProps extends SelectHTMLAttributes<HTMLSelectElement> {
  hasError?: boolean;
  options: { value: string; label: string }[];
}

export function Select({ hasError, options, className = "", ...props }: SelectProps) {
  return (
    <select
      className={`block w-full rounded-md border bg-white/90 px-3 py-2 text-sm text-zinc-950 outline-none transition focus:ring-2 focus:ring-offset-0 ${
        hasError
          ? "border-rose-300 focus:border-rose-500 focus:ring-rose-500/20"
          : "border-zinc-300 focus:border-teal-600 focus:ring-teal-500/20"
      } ${className}`}
      {...props}
    >
      {options.map((opt) => (
        <option key={opt.value} value={opt.value}>
          {opt.label}
        </option>
      ))}
    </select>
  );
}
