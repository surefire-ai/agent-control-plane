interface CardProps {
  children: React.ReactNode;
  className?: string;
  variant?: "default" | "elevated" | "muted" | "interactive";
  padding?: "none" | "sm" | "md";
}

const variantCls: Record<string, string> = {
  default: "surface",
  elevated: "surface-elevated",
  muted: "surface-muted",
  interactive: "data-card data-card--interactive",
};

const paddingCls: Record<string, string> = {
  none: "",
  sm: "p-3",
  md: "p-5",
};

export function Card({
  children,
  className = "",
  variant = "default",
  padding = "none",
}: CardProps) {
  return (
    <div className={`${variantCls[variant]} rounded-lg ${paddingCls[padding]} ${className}`}>
      {children}
    </div>
  );
}
