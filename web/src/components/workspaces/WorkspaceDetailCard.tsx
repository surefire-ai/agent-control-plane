import { useTranslation } from "react-i18next";
import type { Workspace } from "@/types/api";
import { Card } from "@/components/shared/Card";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface WorkspaceDetailCardProps {
  workspace: Workspace;
}

export function WorkspaceDetailCard({ workspace }: WorkspaceDetailCardProps) {
  const { t } = useTranslation();
  const nd = t("common.noData");

  return (
    <Card className="p-6">
      <dl className="grid gap-4 sm:grid-cols-2">
        {/* Identity */}
        <dt className="detail-section-label sm:col-span-2">{t("detailSection.identity")}</dt>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.id")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{workspace.id}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.slug")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{workspace.slug}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.displayName")}</dt>
          <dd className="mt-2 text-sm font-semibold text-zinc-950">{workspace.displayName}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.status")}</dt>
          <dd className="mt-2">
            <StatusBadge status={workspace.status} />
          </dd>
        </div>
        <div className="surface-muted rounded-lg p-4 sm:col-span-2">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.description")}</dt>
          <dd className="mt-2 text-sm leading-6 text-zinc-800">{workspace.description ?? nd}</dd>
        </div>

        {/* Configuration */}
        <dt className="detail-section-label sm:col-span-2">{t("detailSection.configuration")}</dt>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.tenantId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{workspace.tenantId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.kubernetesNamespace")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{workspace.kubernetesNamespace ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4 sm:col-span-2">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("workspace.fields.kubernetesWorkspaceName")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{workspace.kubernetesWorkspaceName ?? nd}</dd>
        </div>
      </dl>
    </Card>
  );
}
