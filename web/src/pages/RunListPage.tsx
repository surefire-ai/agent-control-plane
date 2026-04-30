import { useState } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useRuns } from "@/api/runs";
import { RunTable } from "@/components/runs/RunTable";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { PageHeader } from "@/components/shared/PageHeader";
import { Pagination } from "@/components/shared/Pagination";

const LIMIT = 10;

export function RunListPage() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId: string }>();
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error, refetch } = useRuns(page, LIMIT, tenantId);

  return (
    <div>
      <PageHeader title={t("run.title")} subtitle={t("run.subtitle")} />

      {isLoading && <LoadingSkeleton />}

      {isError && (
        <ErrorAlert
          message={error instanceof Error ? error.message : t("run.loadError")}
          onRetry={() => refetch()}
        />
      )}

      {data && data.runs.length === 0 && (
        <EmptyState title={t("run.emptyTitle")} description={t("run.emptyDescription")} />
      )}

      {data && data.runs.length > 0 && (
        <>
          <RunTable runs={data.runs} />
          <Pagination page={page} limit={LIMIT} total={data.total} onPageChange={setPage} />
        </>
      )}
    </div>
  );
}
