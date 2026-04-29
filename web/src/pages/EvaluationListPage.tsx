import { useState } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useEvaluations } from "@/api/evaluations";
import { EvaluationTable } from "@/components/evaluations/EvaluationTable";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { PageHeader } from "@/components/shared/PageHeader";
import { Pagination } from "@/components/shared/Pagination";

const LIMIT = 10;

export function EvaluationListPage() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId: string }>();
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error, refetch } = useEvaluations(page, LIMIT, tenantId);

  return (
    <div>
      <PageHeader title={t("evaluation.title")} subtitle={t("evaluation.subtitle")} />

      {isLoading && <LoadingSkeleton />}

      {isError && (
        <ErrorAlert
          message={error instanceof Error ? error.message : t("evaluation.loadError")}
          onRetry={() => refetch()}
        />
      )}

      {data && data.evaluations.length === 0 && (
        <EmptyState title={t("evaluation.emptyTitle")} description={t("evaluation.emptyDescription")} />
      )}

      {data && data.evaluations.length > 0 && (
        <>
          <EvaluationTable evaluations={data.evaluations} />
          <Pagination page={page} limit={LIMIT} total={data.total} onPageChange={setPage} />
        </>
      )}
    </div>
  );
}
