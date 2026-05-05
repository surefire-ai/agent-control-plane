import { useTranslation } from "react-i18next";
import { Modal } from "./Modal";
import { Button } from "./Button";

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
      <div className="mt-6 flex justify-end gap-2">
        <Button variant="secondary" size="sm" onClick={onClose} disabled={isPending}>
          {t("common.cancel")}
        </Button>
        <Button
          variant={isDestructive ? "danger" : "primary"}
          size="sm"
          onClick={onConfirm}
          disabled={isPending}
        >
          {isPending ? (
            <span className="inline-flex items-center gap-1.5">
              <span className="h-3 w-3 animate-spin rounded-full border-2 border-current border-t-transparent" />
              {t("common.deleting")}
            </span>
          ) : (confirmLabel ?? t("common.confirm"))}
        </Button>
      </div>
    </Modal>
  );
}
