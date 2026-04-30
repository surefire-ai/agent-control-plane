import { Check, Minus } from "lucide-react";
import { useTranslation } from "react-i18next";
import type { ProviderAccount } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface ProviderTableProps {
  providers: ProviderAccount[];
}

export function ProviderTable({ providers }: ProviderTableProps) {
  const { t } = useTranslation();

  return (
    <div className="surface overflow-hidden rounded-lg">
      <table className="min-w-full divide-y divide-zinc-200/80">
        <thead className="bg-zinc-50/80">
          <tr>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.name")}
            </th>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.provider")}
            </th>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.capabilities")}
            </th>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.credential")}
            </th>
            <th className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.status")}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200/80 bg-white/50">
          {providers.map((provider) => (
            <tr key={provider.id} className="transition-colors hover:bg-teal-50/70">
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-950">{provider.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{provider.id}</p>
              </td>
              <td className="px-6 py-4">
                <div className="flex flex-wrap items-center gap-2">
                  <span className="text-sm font-semibold text-zinc-800">{provider.provider}</span>
                  {provider.domestic && (
                    <span className="rounded-md border border-amber-200 bg-amber-50 px-2 py-0.5 text-xs font-medium text-amber-700">
                      {t("provider.domestic")}
                    </span>
                  )}
                </div>
                <p className="mt-1 text-xs font-mono text-zinc-500">{provider.family}</p>
              </td>
              <td className="px-6 py-4">
                <Capability enabled={provider.supportsJsonSchema} label={t("provider.jsonSchema")} />
                <Capability enabled={provider.supportsToolCalling} label={t("provider.toolCalling")} />
              </td>
              <td className="px-6 py-4">
                <p className="text-sm font-mono text-zinc-700">
                  {provider.credentialRef ?? t("common.noData")}
                </p>
                <p className="mt-1 max-w-xs truncate text-xs font-mono text-zinc-500">
                  {provider.baseUrl ?? t("common.noData")}
                </p>
              </td>
              <td className="px-6 py-4">
                <StatusBadge status={provider.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function Capability({ enabled, label }: { enabled: boolean; label: string }) {
  const Icon = enabled ? Check : Minus;
  return (
    <div className="flex items-center gap-2 text-sm text-zinc-700">
      <Icon className={enabled ? "h-4 w-4 text-teal-600" : "h-4 w-4 text-zinc-300"} aria-hidden="true" />
      <span>{label}</span>
    </div>
  );
}
