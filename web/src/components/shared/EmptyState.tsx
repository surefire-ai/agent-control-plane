import type { LucideIcon } from "lucide-react";
import { ArchiveX } from "lucide-react";

interface EmptyStateProps {
  title: string;
  description?: string;
  action?: React.ReactNode;
  icon?: LucideIcon;
}

export function EmptyState({
  title,
  description,
  action,
  icon: Icon,
}: EmptyStateProps) {
  const DisplayIcon = Icon ?? ArchiveX;

  return (
    <div className="surface-panel flex flex-col items-center justify-center rounded-lg px-6 py-16 text-center">
      <div className="mb-4 flex h-12 w-12 items-center justify-center rounded-xl border border-teal-200/60 bg-gradient-to-b from-teal-50 to-teal-100/50">
        <DisplayIcon className="h-6 w-6 text-teal-600" aria-hidden="true" />
      </div>
      <h3 className="text-base font-semibold text-zinc-900">{title}</h3>
      {description && (
        <p className="mt-1.5 max-w-sm text-sm leading-6 text-zinc-500">
          {description}
        </p>
      )}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}
