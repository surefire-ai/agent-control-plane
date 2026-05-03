import { useTranslation } from "react-i18next";
import { TriangleAlert } from "lucide-react";

interface ErrorAlertProps {
  message: string;
  onRetry?: () => void;
}

export function ErrorAlert({ message, onRetry }: ErrorAlertProps) {
  const { t } = useTranslation();

  return (
    <div className="rounded-lg border border-rose-200 bg-rose-50 p-4">
      <div className="flex items-start gap-3">
        <TriangleAlert className="mt-0.5 h-5 w-5 text-rose-600" aria-hidden="true" />
        <div className="flex-1">
          <p className="text-sm font-semibold text-rose-900">{message}</p>
          {onRetry && (
            <button
              onClick={onRetry}
              type="button"
              className="mt-2 text-sm font-semibold text-rose-700 hover:text-rose-600 focus:outline-none focus:ring-2 focus:ring-rose-500 focus:ring-offset-2 rounded"
            >
              {t("common.retry")}
            </button>
          )}
        </div>
      </div>
    </div>
  );
}
