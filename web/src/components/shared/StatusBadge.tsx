import { useTranslation } from "react-i18next";

const styles: Record<string, string> = {
  active: "bg-teal-50 text-teal-800 border-teal-200",
  published: "bg-teal-50 text-teal-800 border-teal-200",
  passed: "bg-teal-50 text-teal-800 border-teal-200",
  failed: "bg-rose-50 text-rose-800 border-rose-200",
  running: "bg-cyan-50 text-cyan-800 border-cyan-200",
  draft: "bg-zinc-100 text-zinc-600 border-zinc-200",
  inactive: "bg-zinc-100 text-zinc-600 border-zinc-200",
  archived: "bg-amber-50 text-amber-800 border-amber-200",
};

export function StatusBadge({ status }: { status: string }) {
  const { t } = useTranslation();
  const cls = styles[status] ?? styles.inactive;

  return (
    <span className={`inline-flex items-center rounded-md border px-2 py-0.5 text-xs font-semibold ${cls}`}>
      {t(`status.${status}`, status)}
    </span>
  );
}
