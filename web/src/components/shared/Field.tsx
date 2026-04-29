interface FieldProps {
  label: string;
  htmlFor?: string;
  error?: string;
  children: React.ReactNode;
}

export function Field({ label, htmlFor, error, children }: FieldProps) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={htmlFor} className="block text-sm font-semibold text-zinc-700">
        {label}
      </label>
      {children}
      {error && <p className="text-sm text-rose-600">{error}</p>}
    </div>
  );
}
