import { useTranslation } from "react-i18next";
import { TriangleAlert } from "lucide-react";
import { Button } from "./Button";

interface ErrorAlertProps {
  message: string;
  onRetry?: () => void;
}

export function ErrorAlert({ message, onRetry }: ErrorAlertProps) {
  const { t } = useTranslation();

  return (
    <div
      role="alert"
      className="alert-slide-in rounded-lg border border-rose-200/80 bg-gradient-to-b from-rose-50 to-rose-100/60 p-5"
    >
      <div className="flex items-start gap-3">
        <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-rose-100">
          <TriangleAlert
            className="h-4 w-4 text-rose-600"
            aria-hidden="true"
          />
        </div>
        <div className="flex-1 min-w-0">
          <p className="text-sm font-semibold text-rose-900">{message}</p>
          {onRetry && (
            <div className="mt-3">
              <Button variant="ghost" size="sm" onClick={onRetry} className="focus-ring-visible">
                {t("common.retry")}
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
