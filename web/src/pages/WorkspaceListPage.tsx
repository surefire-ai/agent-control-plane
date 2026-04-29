import { useState } from "react";
import { useParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useWorkspaces } from "@/api/workspaces";
import { WorkspaceTable } from "@/components/workspaces/WorkspaceTable";
import { PageHeader } from "@/components/shared/PageHeader";
import { Pagination } from "@/components/shared/Pagination";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { EmptyState } from "@/components/shared/EmptyState";
import { Button } from "@/components/shared/Button";

const LIMIT = 10;

export function WorkspaceListPage() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId: string }>();
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error, refetch } = useWorkspaces(page, LIMIT, tenantId);

  return (
    <div>
      <PageHeader
        title={t("workspace.listTitle")}
        subtitle={t("workspace.listSubtitle")}
        actions={
          <Link to={`/tenants/${tenantId}/workspaces/new`}>
            <Button>{t("workspace.newButton")}</Button>
          </Link>
        }
      />

      {isLoading && <LoadingSkeleton />}

      {isError && (
        <ErrorAlert
          message={error instanceof Error ? error.message : t("workspace.listLoadError")}
          onRetry={() => refetch()}
        />
      )}

      {data && data.workspaces.length === 0 && (
        <EmptyState
          title={t("workspace.emptyTitle")}
          description={t("workspace.emptyDescription")}
          action={
            <Link to={`/tenants/${tenantId}/workspaces/new`}>
              <Button>{t("workspace.createButton")}</Button>
            </Link>
          }
        />
      )}

      {data && data.workspaces.length > 0 && (
        <>
          <WorkspaceTable workspaces={data.workspaces} />
          <Pagination page={page} limit={LIMIT} total={data.total} onPageChange={setPage} />
        </>
      )}
    </div>
  );
}
