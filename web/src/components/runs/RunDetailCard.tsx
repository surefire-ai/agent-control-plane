import { useTranslation } from "react-i18next";
import type { Run } from "@/types/api";
import { Card } from "@/components/shared/Card";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface RunDetailCardProps {
  run: Run;
}

function formatDuration(start: string, end: string): string {
  const ms = new Date(end).getTime() - new Date(start).getTime();
  if (ms < 0) return "—";
  if (ms < 1000) return `${ms}ms`;
  if (ms < 60000) return `${(ms / 1000).toFixed(1)}s`;
  const min = Math.floor(ms / 60000);
  const sec = Math.floor((ms % 60000) / 1000);
  return `${min}m ${sec}s`;
}

export function RunDetailCard({ run }: RunDetailCardProps) {
  const { t } = useTranslation();
  const nd = t("common.noData");

  const duration = run.startedAt && run.completedAt
    ? formatDuration(run.startedAt, run.completedAt)
    : null;

  return (
    <Card className="p-6">
      <dl className="grid gap-4 sm:grid-cols-2">
        {/* Identity */}
        <dt className="detail-section-label sm:col-span-2">{t("detailSection.identity")}</dt>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.id")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{run.id}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.status")}</dt>
          <dd className="mt-2">
            <StatusBadge status={run.status} />
          </dd>
        </div>

        {/* Runtime */}
        <dt className="detail-section-label sm:col-span-2">{t("detailSection.runtime")}</dt>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.runtimeEngine")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-950">{run.runtimeEngine}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.runnerClass")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-950">{run.runnerClass}</dd>
        </div>

        {/* Timing */}
        <dt className="detail-section-label sm:col-span-2">{t("detailSection.timing")}</dt>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.startedAt")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-800">{run.startedAt ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.completedAt")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-800">{run.completedAt ?? nd}</dd>
        </div>
        {duration && (
          <div className="surface-muted rounded-lg p-4">
            <dt className="text-xs font-semibold uppercase text-zinc-500">Duration</dt>
            <dd className="mt-2 text-lg font-bold text-teal-600">{duration}</dd>
          </div>
        )}

        {/* References */}
        <dt className="detail-section-label sm:col-span-2">{t("detailSection.references")}</dt>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.agentId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{run.agentId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.agentRevision")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{run.agentRevision ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.evaluationId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{run.evaluationId ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.tenantId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{run.tenantId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.workspaceId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{run.workspaceId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4 sm:col-span-2">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.traceRef")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{run.traceRef ?? nd}</dd>
        </div>

        {/* Summary */}
        <dt className="detail-section-label sm:col-span-2">{t("detailSection.summary")}</dt>
        <div className="surface-muted rounded-lg p-4 sm:col-span-2">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("run.fields.summary")}</dt>
          <dd className="mt-2 text-sm leading-6 text-zinc-800">{run.summary ?? nd}</dd>
        </div>
      </dl>
    </Card>
  );
}
