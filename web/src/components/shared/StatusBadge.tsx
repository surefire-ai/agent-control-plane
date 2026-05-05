import { useTranslation } from "react-i18next";

const styles: Record<string, { badge: string; dot: string }> = {
  active:    { badge: "bg-teal-50 text-teal-800 border-teal-200",    dot: "status-dot--success" },
  published: { badge: "bg-teal-50 text-teal-800 border-teal-200",    dot: "status-dot--success" },
  passed:    { badge: "bg-teal-50 text-teal-800 border-teal-200",    dot: "status-dot--success" },
  succeeded: { badge: "bg-teal-50 text-teal-800 border-teal-200",    dot: "status-dot--success" },
  failed:    { badge: "bg-rose-50 text-rose-800 border-rose-200",    dot: "status-dot--danger" },
  running:   { badge: "bg-cyan-50 text-cyan-800 border-cyan-200",    dot: "status-dot--info" },
  retrying:  { badge: "bg-cyan-50 text-cyan-800 border-cyan-200",    dot: "status-dot--info" },
  pending:   { badge: "bg-amber-50 text-amber-800 border-amber-200", dot: "status-dot--warning" },
  draft:     { badge: "bg-zinc-100 text-zinc-600 border-zinc-200",   dot: "status-dot--muted" },
  inactive:  { badge: "bg-zinc-100 text-zinc-600 border-zinc-200",   dot: "status-dot--muted" },
  archived:  { badge: "bg-amber-50 text-amber-800 border-amber-200", dot: "status-dot--warning" },
};

export function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation();
  const s = styles[status] ?? styles.inactive;

  return (
    <span
      className={`inline-flex items-center gap-1.5 rounded-md border px-2 py-0.5 text-xs font-semibold ${s.badge}`}
    >
      <span className={`status-dot ${s.dot}`} />
      {t(`status.${status}`, status)}
    </span>
  );
}
