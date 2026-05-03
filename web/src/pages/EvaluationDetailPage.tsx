import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useParams, Link } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useEvaluation } from "@/api/evaluations";
import { useRuns } from "@/api/runs";
import { EvaluationDetailCard } from "@/components/evaluations/EvaluationDetailCard";
import { RunTable } from "@/components/runs/RunTable";
import { PageHeader } from "@/components/shared/PageHeader";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { Card } from "@/components/shared/Card";
import { EmptyState } from "@/components/shared/EmptyState";

const LINKED_LIMIT = 5;

export function EvaluationDetailPage() {
  const { t } = useTranslation();
  const { tenantId, evaluationId } = useParams<{ tenantId: string; evaluationId: string }>();
  const { data: evaluation, isLoading, isError, error, refetch } = useEvaluation(evaluationId);
  useDocumentTitle(evaluation?.displayName);

  const { data: runsData } = useRuns(1, LINKED_LIMIT, tenantId, undefined, undefined, evaluationId);

  if (isLoading) return <LoadingSkeleton />;

  if (isError) {
    return (
      <ErrorAlert
        message={error instanceof Error ? error.message : t("evaluation.loadError")}
        onRetry={() => refetch()}
      />
    );
  }

  if (!evaluation) {
    return <ErrorAlert message={t("evaluation.notFound")} />;
  }

  return (
    <div>
      <PageHeader
        title={evaluation.displayName}
        subtitle={t("evaluation.detailSubtitle")}
      />

      <EvaluationDetailCard evaluation={evaluation} />

      {/* Linked Runs */}
      <div className="mt-8">
        <div className="mb-4 flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-zinc-950">{t("evaluation.linkedRuns")}</h2>
            <p className="mt-1 text-sm text-zinc-500">{t("evaluation.linkedRunsSubtitle")}</p>
          </div>
          {runsData && runsData.runs.length > 0 && (
            <Link
              to={`/tenants/${tenantId}/runs`}
              className="text-sm font-medium text-teal-600 hover:text-teal-700"
            >
              {t("common.viewAll", "View all")}
            </Link>
          )}
        </div>
        {runsData && runsData.runs.length === 0 && (
          <Card className="p-6">
            <EmptyState title={t("run.emptyTitle")} description={t("run.emptyDescription")} />
          </Card>
        )}
        {runsData && runsData.runs.length > 0 && <RunTable runs={runsData.runs} />}
      </div>
    </div>
  );
}
