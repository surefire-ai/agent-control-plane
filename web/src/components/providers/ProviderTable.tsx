import { Check, Minus } from "lucide-react";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import type { ProviderAccount } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface ProviderTableProps {
  providers: ProviderAccount[];
}

export function ProviderTable({ providers }: ProviderTableProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <div className="surface overflow-hidden rounded-lg">
      <table className="data-table min-w-full">
        <caption className="sr-only">{t("table.name")}</caption>
        <thead>
          <tr>
            <th scope="col">
              {t("table.name")}
            </th>
            <th scope="col">
              {t("table.provider")}
            </th>
            <th scope="col">
              {t("table.capabilities")}
            </th>
            <th scope="col">
              {t("table.credential")}
            </th>
            <th scope="col">
              {t("table.status")}
            </th>
          </tr>
        </thead>
        <tbody>
          {providers.map((provider) => (
            <tr
              key={provider.id}
              onClick={() => navigate(`/tenants/${provider.tenantId}/providers/${provider.id}`)}
              tabIndex={0}
              role="link"
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(`/tenants/${provider.tenantId}/providers/${provider.id}`); } }}
              className="cursor-pointer transition-colors"
            >
              <td>
                <p className="text-sm font-semibold text-zinc-950">{provider.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{provider.id}</p>
              </td>
              <td>
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
              <td>
                <Capability enabled={provider.supportsJsonSchema} label={t("provider.jsonSchema")} />
                <Capability enabled={provider.supportsToolCalling} label={t("provider.toolCalling")} />
              </td>
              <td>
                <p className="text-sm font-mono text-zinc-700">
                  {provider.credentialRef ?? t("common.noData")}
                </p>
                <p className="mt-1 max-w-xs truncate text-xs font-mono text-zinc-500">
                  {provider.baseUrl ?? t("common.noData")}
                </p>
              </td>
              <td>
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
