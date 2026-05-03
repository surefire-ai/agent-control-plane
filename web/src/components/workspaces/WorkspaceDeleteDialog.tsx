import { useTranslation } from "react-i18next";
import { ConfirmDialog } from "@/components/shared/ConfirmDialog";
import type { Workspace } from "@/types/api";

interface WorkspaceDeleteDialogProps {
  workspace: Workspace | null;
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  isPending: boolean;
}

export function WorkspaceDeleteDialog({
  workspace,
  open,
  onClose,
  onConfirm,
  isPending,
}: WorkspaceDeleteDialogProps) {
  const { t } = useTranslation();
  if (!workspace) return null;

  return (
    <ConfirmDialog
      open={open}
      onClose={onClose}
      onConfirm={onConfirm}
      title={t("workspace.deleteTitle")}
      message={t("workspace.deleteConfirm", { name: workspace.displayName, id: workspace.id })}
      confirmLabel={t("workspace.deleteButton")}
      isDestructive
      isPending={isPending}
    />
  );
}
