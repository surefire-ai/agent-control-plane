import { useTranslation } from "react-i18next";
import { ChevronLeft, ChevronRight } from "lucide-react";

interface PaginationProps {
  page: number;
  limit: number;
  total: number;
  onPageChange: (page: number) => void;
}

export function Pagination({
  page,
  limit,
  total,
  onPageChange,
}: PaginationProps) {
  const { t } = useTranslation();
  const lastPage = Math.max(1, Math.ceil(total / limit));

  if (total <= limit) return null;

  return (
    <div className="flex flex-col gap-3 pt-5 sm:flex-row sm:items-center sm:justify-between">
      <p className="text-xs text-zinc-500">
        {t("pagination.showing", {
          start: (page - 1) * limit + 1,
          end: Math.min(page * limit, total),
          total,
        })}
      </p>
      <div className="flex items-center gap-1.5">
        <button
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          aria-label={t("common.previous")}
          className="control-button inline-flex h-8 items-center gap-1 rounded-md border border-zinc-200 bg-white px-2.5 text-xs font-medium text-zinc-600 transition-colors hover:border-zinc-300 hover:text-zinc-800 disabled:cursor-not-allowed disabled:opacity-40"
        >
          <ChevronLeft className="h-3.5 w-3.5" aria-hidden="true" />
          {t("common.previous")}
        </button>
        <span className="min-w-[5rem] text-center text-xs font-medium text-zinc-500">
          {t("pagination.page", { page, lastPage })}
        </span>
        <button
          onClick={() => onPageChange(page + 1)}
          disabled={page >= lastPage}
          aria-label={t("common.next")}
          className="control-button inline-flex h-8 items-center gap-1 rounded-md border border-zinc-200 bg-white px-2.5 text-xs font-medium text-zinc-600 transition-colors hover:border-zinc-300 hover:text-zinc-800 disabled:cursor-not-allowed disabled:opacity-40"
        >
          {t("common.next")}
          <ChevronRight className="h-3.5 w-3.5" aria-hidden="true" />
        </button>
      </div>
    </div>
  );
}
