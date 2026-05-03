import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import type { Tenant } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface TenantTableProps {
  tenants: Tenant[];
}

export function TenantTable({ tenants }: TenantTableProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();

  return (
    <div className="surface overflow-hidden rounded-lg">
      <table className="min-w-full divide-y divide-zinc-200/80">
        <caption className="sr-only">{t("table.name")}</caption>
        <thead className="bg-zinc-50/80">
          <tr>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.name")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.slug")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.status")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.region")}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200/80 bg-white/50">
          {tenants.map((tenant) => (
            <tr
              key={tenant.id}
              onClick={() => navigate(`/tenants/${tenant.id}/workspaces`)}
              tabIndex={0}
              role="link"
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(`/tenants/${tenant.id}/workspaces`); } }}
              className="cursor-pointer transition-colors hover:bg-teal-50/70"
            >
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-950">{tenant.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{tenant.id}</p>
              </td>
              <td className="px-6 py-4 text-sm font-mono text-zinc-700">{tenant.slug}</td>
              <td className="px-6 py-4">
                <StatusBadge status={tenant.status} />
              </td>
              <td className="px-6 py-4 text-sm text-zinc-600">
                {tenant.defaultRegion ?? t("common.noData")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}