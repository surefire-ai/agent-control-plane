import { useTranslation } from "react-i18next";
import { Modal } from "./Modal";

interface ConfirmDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  title: string;
  message: string;
  confirmLabel?: string;
  isDestructive?: boolean;
  isPending?: boolean;
}

export function ConfirmDialog({
  open,
  onClose,
  onConfirm,
  title,
  message,
  confirmLabel,
  isDestructive = false,
  isPending = false,
}: ConfirmDialogProps) {
  const { t } = useTranslation();

  return (
    <Modal open={open} onClose={onClose} title={title}>
      <p className="text-sm leading-6 text-zinc-600">{message}</p>
      <div className="mt-6 flex justify-end gap-3">
        <button
          onClick={onClose}
          disabled={isPending}
          className="rounded-md border border-zinc-300 bg-white px-4 py-2 text-sm font-semibold text-zinc-700 transition-colors hover:bg-zinc-50 disabled:cursor-not-allowed disabled:opacity-50"
        >
          {t("common.cancel")}
        </button>
        <button
          onClick={onConfirm}
          disabled={isPending}
          className={`rounded-md px-4 py-2 text-sm font-semibold text-white transition-colors disabled:cursor-not-allowed disabled:opacity-50 ${
            isDestructive
              ? "bg-rose-600 hover:bg-rose-700"
              : "bg-zinc-950 hover:bg-zinc-800"
          }`}
        >
          {isPending ? t("common.deleting") : (confirmLabel ?? t("common.confirm"))}
        </button>
      </div>
    </Modal>
  );
}
