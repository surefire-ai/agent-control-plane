import { useTranslation } from "react-i18next";
import type { ProviderAccount } from "@/types/api";
import { Card } from "@/components/shared/Card";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface ProviderDetailCardProps {
  provider: ProviderAccount;
}

function CapabilityBadge({ supported, label }: { supported: boolean; label: string }) {
  return (
    <span className={supported
      ? "inline-flex items-center gap-1 rounded-full bg-emerald-50 px-2.5 py-1 text-xs font-semibold text-emerald-700"
      : "inline-flex items-center gap-1 rounded-full bg-zinc-100 px-2.5 py-1 text-xs font-medium text-zinc-500"
    }>
      {supported ? "✓" : "✗"} {label}
    </span>
  );
}

export function ProviderDetailCard({ provider }: ProviderDetailCardProps) {
  const { t } = useTranslation();
  const nd = t("common.noData");

  return (
    <Card className="p-6">
      <dl className="grid gap-4 sm:grid-cols-2">
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.id")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{provider.id}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.status")}</dt>
          <dd className="mt-2">
            <StatusBadge status={provider.status} />
          </dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.provider")}</dt>
          <dd className="mt-2 text-sm font-semibold text-zinc-950">{provider.provider}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.displayName")}</dt>
          <dd className="mt-2 text-sm font-semibold text-zinc-950">{provider.displayName}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.family")}</dt>
          <dd className="mt-2 text-sm font-mono text-zinc-950">{provider.family}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.baseUrl")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{provider.baseUrl ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.credentialRef")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{provider.credentialRef ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.tenantId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{provider.tenantId}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4">
          <dt className="text-xs font-semibold uppercase text-zinc-500">{t("provider.fields.workspaceId")}</dt>
          <dd className="mt-2 break-all text-sm font-mono text-zinc-950">{provider.workspaceId ?? nd}</dd>
        </div>
        <div className="surface-muted rounded-lg p-4 sm:col-span-2">
          <dt className="text-xs font-semibold uppercase text-zinc-500">Capabilities</dt>
          <dd className="mt-2 flex flex-wrap gap-2">
            <CapabilityBadge supported={provider.domestic} label={t("provider.domestic")} />
            <CapabilityBadge supported={provider.supportsJsonSchema} label={t("provider.jsonSchema")} />
            <CapabilityBadge supported={provider.supportsToolCalling} label={t("provider.toolCalling")} />
          </dd>
        </div>
      </dl>
    </Card>
  );
}
