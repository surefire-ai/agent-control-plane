import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useRun } from "@/api/runs";
import { RunDetailCard } from "@/components/runs/RunDetailCard";
import { PageHeader } from "@/components/shared/PageHeader";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ErrorAlert } from "@/components/shared/ErrorAlert";

export function RunDetailPage() {
  const { t } = useTranslation();
  const { tenantId: _tenantId, runId } = useParams<{ tenantId: string; runId: string }>();
  const { data: run, isLoading, isError, error, refetch } = useRun(runId);
  useDocumentTitle(run?.id);

  if (isLoading) return <LoadingSkeleton />;

  if (isError) {
    return (
      <ErrorAlert
        message={error instanceof Error ? error.message : t("run.loadError")}
        onRetry={() => refetch()}
      />
    );
  }

  if (!run) {
    return <ErrorAlert message={t("run.notFound")} />;
  }

  return (
    <div>
      <PageHeader
        title={run.id}
        subtitle={t("run.detailSubtitle")}
      />
      <RunDetailCard run={run} />
    </div>
  );
}
