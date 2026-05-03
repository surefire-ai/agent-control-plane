import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useWorkspace, useUpdateWorkspace, useDeleteWorkspace } from "@/api/workspaces";
import { WorkspaceDetailCard } from "@/components/workspaces/WorkspaceDetailCard";
import { WorkspaceEditForm } from "@/components/workspaces/WorkspaceEditForm";
import { WorkspaceDeleteDialog } from "@/components/workspaces/WorkspaceDeleteDialog";
import { WorkspaceProviderBindings } from "@/components/providers/WorkspaceProviderBindings";
import { PageHeader } from "@/components/shared/PageHeader";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { Button } from "@/components/shared/Button";
import type { UpdateWorkspaceRequest } from "@/types/api";

export function WorkspaceDetailPage() {
  const { t } = useTranslation();
  const { tenantId, workspaceId } = useParams<{ tenantId: string; workspaceId: string }>();
  const navigate = useNavigate();
  const { data: workspace, isLoading, isError, error, refetch } = useWorkspace(workspaceId);
  useDocumentTitle(workspace?.displayName);
  const updateMutation = useUpdateWorkspace();
  const deleteMutation = useDeleteWorkspace();

  const [isEditing, setIsEditing] = useState(false);
  const [editValues, setEditValues] = useState<UpdateWorkspaceRequest>({});
  const [showDeleteDialog, setShowDeleteDialog] = useState(false);

  const handleEdit = () => {
    setEditValues({});
    setIsEditing(true);
  };

  const handleSave = () => {
    if (!workspace) return;
    updateMutation.mutate(
      { id: workspace.id, ...editValues },
      {
        onSuccess: () => setIsEditing(false),
      },
    );
  };

  const handleDelete = () => {
    if (!workspace) return;
    deleteMutation.mutate(workspace.id, {
      onSuccess: () => navigate(`/tenants/${tenantId}/workspaces`),
    });
  };

  if (isLoading) return <LoadingSkeleton />;

  if (isError) {
    return (
      <ErrorAlert
        message={error instanceof Error ? error.message : t("workspace.loadError")}
        onRetry={() => refetch()}
      />
    );
  }

  if (!workspace) {
    return <ErrorAlert message={t("workspace.notFound")} />;
  }

  return (
    <div>
      <PageHeader
        title={workspace.displayName}
        subtitle={t("workspace.detailSubtitle")}
        actions={
          !isEditing ? (
            <>
              <Button variant="secondary" onClick={handleEdit}>
                {t("common.edit")}
              </Button>
              <Button variant="danger" onClick={() => setShowDeleteDialog(true)}>
                {t("common.delete")}
              </Button>
            </>
          ) : null
        }
      />

      {updateMutation.isError && (
        <div className="mb-4">
          <ErrorAlert
            message={
              updateMutation.error instanceof Error
                ? updateMutation.error.message
                : t("workspace.updateError")
            }
          />
        </div>
      )}

      {isEditing ? (
        <WorkspaceEditForm
          workspace={workspace}
          values={editValues}
          onChange={setEditValues}
          onSubmit={handleSave}
          onCancel={() => setIsEditing(false)}
          isPending={updateMutation.isPending}
        />
      ) : (
        <>
          <WorkspaceDetailCard workspace={workspace} />
          <WorkspaceProviderBindings tenantId={tenantId} workspaceId={workspace.id} />
        </>
      )}

      <WorkspaceDeleteDialog
        workspace={workspace}
        open={showDeleteDialog}
        onClose={() => setShowDeleteDialog(false)}
        onConfirm={handleDelete}
        isPending={deleteMutation.isPending}
      />
    </div>
  );
}
