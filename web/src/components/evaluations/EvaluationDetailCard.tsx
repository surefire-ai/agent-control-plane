import { useTranslation } from "react-i18next";
import type { Evaluation } from "@/types/api";
import { Card } from "@/components/shared/Card";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface EvaluationDetailCardProps {
  evaluation: Evaluation;
}

export function EvaluationDetailCard({ evaluation }: EvaluationDetailCardProps) {
  const { t } = useTranslation();
  const nd = t("common.noData");

  const scorePercent = Math.round(evaluation.score * 100);
  const scoreColor = evaluation.score >= 0.8 ? "text-emerald-600" : evaluation.score >= 0.6 ? "text-amber-600" : "text-red-600";

  return (
    <Card className="p-6">
      <dl className="grid gap-4 sm:grid-cols-2">
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.id")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{evaluation.id}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.slug")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{evaluation.slug}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.displayName")}</dt>
          <dd className="mt-2 text-sm font-semibold text-zinc-950">{evaluation.displayName}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.status")}</dt>
          <dd className="mt-2">
            <StatusBadge status={evaluation.status} />
          </dd>
        </div>
        <div className="surface-muted rounded-lg p-4 sm:col-span-2">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.description")}</dt>
          <dd className="mt-2 text-sm leading-6 text-zinc-800">{evaluation.description ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.score")}</dt>
          <dd className={`mt-2 text-2xl font-bold ${scoreColor}`}>{scorePercent}%</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.gatePassed")}</dt>
          <dd className="mt-2">
            <span className={evaluation.gatePassed ? "inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-semibold text-emerald-700" : "inline-flex items-center gap-1 rounded-full bg-red-50 px-2.5 py-1 text-xs font-semibold text-red-700"}>
              {evaluation.gatePassed ? t("evaluation.gatePassed") : t("evaluation.gateFailed")}
            </span>
          </dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.datasetName")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-950">{evaluation.datasetName}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.datasetRevision")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-950">{evaluation.datasetRevision ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.baselineRevision")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-950">{evaluation.baselineRevision ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.samplesTotal")}</dt>
          <dd className="mt-2 text-sm font-semibold text-zinc-800">{evaluation.samplesEvaluated} / {evaluation.samplesTotal}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.agentId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{evaluation.agentId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.tenantId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{evaluation.tenantId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.workspaceId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{evaluation.workspaceId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.latestRunId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{evaluation.latestRunId ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("evaluation.fields.reportRef")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{evaluation.reportRef ?? nd}</dd>
        </div>
      </dl>
    </Card>
  );
}
