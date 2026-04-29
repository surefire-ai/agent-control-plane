import { useTranslation } from "react-i18next";

interface PaginationProps {
  page: number;
  limit: number;
  total: number;
  onPageChange: (page: number) => void;
}

export function Pagination({ page, limit, total, onPageChange }: PaginationProps) {
  const { t } = useTranslation();
  const lastPage = Math.max(1, Math.ceil(total / limit));

  if (total <= limit) return null;

  return (
    <div className="flex flex-col gap-3 pt-4 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-sm text-zinc-500">
        {t("pagination.showing", {
          start: (page - 1) * limit + 1,
          end: Math.min(page * limit, total),
          total,
        })}
      </p>
      <div className="flex items-center gap-2">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm font-semibold text-zinc-700 transition-colors hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t("common.previous")}
        </button>
        <span className="text-sm text-zinc-600">
          {t("pagination.page", { page, lastPage })}
        </span>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= lastPage}
          className="rounded-md border border-zinc-300 bg-white px-3 py-1.5 text-sm font-semibold text-zinc-700 transition-colors hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t("common.next")}
        </button>
      </div>
    </div>
  );
}
