import { useState, useEffect, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { CheckCircle2, XCircle, AlertTriangle, Info, X } from "lucide-react";

type ToastVariant = "success" | "error" | "warning" | "info";

interface ToastData {
  id: string;
  variant: ToastVariant;
  message: string;
}

interface ToastProps extends ToastData {
  onDismiss: (id: string) => void;
  duration?: number;
}

const variantConfig: Record<
  ToastVariant,
  { icon: typeof CheckCircle2; bg: string; border: string; text: string; iconColor: string }
> = {
  success: {
    icon: CheckCircle2,
    bg: "bg-emerald-50",
    border: "border-emerald-200/80",
    text: "text-emerald-800",
    iconColor: "text-emerald-500",
  },
  error: {
    icon: XCircle,
    bg: "bg-rose-50",
    border: "border-rose-200/80",
    text: "text-rose-800",
    iconColor: "text-rose-500",
  },
  warning: {
    icon: AlertTriangle,
    bg: "bg-amber-50",
    border: "border-amber-200/80",
    text: "text-amber-800",
    iconColor: "text-amber-500",
  },
  info: {
    icon: Info,
    bg: "bg-sky-50",
    border: "border-sky-200/80",
    text: "text-sky-800",
    iconColor: "text-sky-500",
  },
};

function Toast({ id, variant, message, onDismiss, duration = 4000 }: ToastProps) {
  const { t } = useTranslation();
  const [exiting, setExiting] = useState(false);
  const config = variantConfig[variant];
  const Icon = config.icon;

  const handleDismiss = useCallback(() => {
    setExiting(true);
    setTimeout(() => onDismiss(id), 200);
  }, [id, onDismiss]);

  useEffect(() => {
    const timer = setTimeout(handleDismiss, duration);
    return () => clearTimeout(timer);
  }, [duration, handleDismiss]);

  return (
    <div
      role="alert"
      className={`${exiting ? "toast-exit" : "toast-enter"} ${config.bg} ${config.border} flex items-center gap-3 rounded-lg border px-4 py-3 shadow-lg`}
    >
      <Icon className={`h-4 w-4 shrink-0 ${config.iconColor}`} aria-hidden="true" />
      <p className={`flex-1 text-sm font-medium ${config.text}`}>{message}</p>
      <button
        onClick={handleDismiss}
        className={`shrink-0 rounded p-0.5 ${config.text} opacity-60 transition-opacity hover:opacity-100 focus:outline-none focus:ring-2 focus:ring-teal-500/20`}
        aria-label={t("common.dismiss")}
      >
        <X className="h-3.5 w-3.5" />
      </button>
    </div>
  );
}

/* ── Toast container (use via useToast hook) ── */
export function ToastContainer({
  toasts,
  onDismiss,
}: {
  toasts: ToastData[];
  onDismiss: (id: string) => void;
}) {
  if (toasts.length === 0) return null;

  return (
    <div className="fixed bottom-6 right-6 z-[60] flex flex-col gap-2 w-80">
      {toasts.map((toast) => (
        <Toast key={toast.id} {...toast} onDismiss={onDismiss} />
      ))}
    </div>
  );
}

/* ── useToast hook ── */
let toastCounter = 0;

export function useToast() {
  const [toasts, setToasts] = useState<ToastData[]>([]);

  const addToast = useCallback(
    (variant: ToastVariant, message: string) => {
      const id = `toast-${++toastCounter}`;
      setToasts((prev) => [...prev, { id, variant, message }]);
    },
    [],
  );

  const dismissToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((t) => t.id !== id));
  }, []);

  return {
    toasts,
    addToast,
    dismissToast,
    success: (msg: string) => addToast("success", msg),
    error: (msg: string) => addToast("error", msg),
    warning: (msg: string) => addToast("warning", msg),
    info: (msg: string) => addToast("info", msg),
  };
}
