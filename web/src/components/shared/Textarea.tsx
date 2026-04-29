import type { TextareaHTMLAttributes } from "react";

interface TextareaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  hasError?: boolean;
}

export function Textarea({ hasError, className = "", ...props }: TextareaProps) {
  return (
    <textarea
      className={`block w-full rounded-md border bg-white/90 px-3 py-2 text-sm text-zinc-950 placeholder-zinc-400 outline-none transition focus:ring-2 focus:ring-offset-0 ${
        hasError
          ? "border-rose-300 focus:border-rose-500 focus:ring-rose-500/20"
          : "border-zinc-300 focus:border-teal-600 focus:ring-teal-500/20"
      } ${className}`}
      {...props}
    />
  );
}
