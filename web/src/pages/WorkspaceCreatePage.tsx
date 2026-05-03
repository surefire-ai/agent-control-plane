import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useCreateWorkspace } from "@/api/workspaces";
import { WorkspaceCreateForm } from "@/components/workspaces/WorkspaceCreateForm";
import { PageHeader } from "@/components/shared/PageHeader";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import type { CreateWorkspaceRequest } from "@/types/api";

export function WorkspaceCreatePage() {
  const { t } = useTranslation();
  useDocumentTitle(t("workspace.createTitle"));
  const { tenantId } = useParams<{ tenantId: string }>();
  const navigate = useNavigate();
  const createMutation = useCreateWorkspace();

  const [values, setValues] = useState<CreateWorkspaceRequest>({
    id: "",
    tenantId: tenantId ?? "",
    slug: "",
    displayName: "",
    status: "active",
  });

  const handleChange = (newValues: CreateWorkspaceRequest) => {
    setValues((prev) => ({ ...prev, ...newValues, tenantId: tenantId ?? "" }));
  };

  const handleSubmit = () => {
    createMutation.mutate(values, {
      onSuccess: () => navigate(`/tenants/${tenantId}/workspaces`),
    });
  };

  return (
    <div>
      <PageHeader
        title={t("workspace.createTitle")}
        subtitle={t("workspace.createSubtitle")}
      />

      {createMutation.isError && (
        <div className="mb-4">
          <ErrorAlert
            message={
              createMutation.error instanceof Error
                ? createMutation.error.message
                : t("workspace.createError")
            }
          />
        </div>
      )}

      <WorkspaceCreateForm
        values={values}
        onChange={handleChange}
        onSubmit={handleSubmit}
        onCancel={() => navigate(`/tenants/${tenantId}/workspaces`)}
        isPending={createMutation.isPending}
      />
    </div>
  );
}