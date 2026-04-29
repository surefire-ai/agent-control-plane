import { ArchiveX } from "lucide-react";

interface EmptyStateProps {
  title: string;
  description?: string;
  action?: React.ReactNode;
}

export function EmptyState({ title, description, action }: EmptyStateProps) {
  return (
    <div className="surface flex flex-col items-center justify-center rounded-lg px-6 py-16 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-lg border border-teal-200 bg-teal-50">
        <ArchiveX className="h-6 w-6 text-teal-700" aria-hidden="true" />
      </div>
      <h3 className="text-base font-semibold text-zinc-950">{title}</h3>
      {description && <p className="mt-1 max-w-sm text-sm leading-6 text-zinc-500">{description}</p>}
      {action && <div className="mt-4">{action}</div>}
    </div>
  );
}
