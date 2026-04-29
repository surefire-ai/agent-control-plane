interface CardProps {
  children: React.ReactNode;
  className?: string;
}

export function Card({ children, className = "" }: CardProps) {
  return (
    <div className={`surface rounded-lg ${className}`}>
      {children}
    </div>
  );
}
