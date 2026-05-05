import type { ButtonHTMLAttributes } from "react";

interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "ghost" | "danger";
  size?: "sm" | "md";
}

const variants: Record<string, string> = {
  primary:
    "bg-zinc-950 text-white hover:bg-zinc-800 focus:ring-teal-500 shadow-sm",
  secondary:
    "border border-zinc-300 bg-white text-zinc-800 hover:bg-zinc-50 hover:border-zinc-400 focus:ring-teal-500",
  ghost:
    "text-zinc-600 hover:bg-zinc-100 hover:text-zinc-950 focus:ring-teal-500",
  danger:
    "bg-rose-600 text-white hover:bg-rose-700 focus:ring-rose-500 shadow-sm",
};

const sizes: Record<string, string> = {
  sm: "px-3 py-1.5 text-xs",
  md: "px-4 py-2 text-sm",
};

export function Button({
  variant = "primary",
  size = "md",
  className = "",
  ...props
}: ButtonProps) {
  return (
    <button
      className={`control-button inline-flex items-center justify-center focus:outline-none focus:ring-2 focus:ring-offset-2 disabled:cursor-not-allowed disabled:opacity-50 ${variants[variant]} ${sizes[size]} ${className}`}
      {...props}
    />
  );
}
