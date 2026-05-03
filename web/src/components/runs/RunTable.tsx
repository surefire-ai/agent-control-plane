import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import type { Run } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface RunTableProps {
  runs: Run[];
}

export function RunTable({ runs }: RunTableProps) {
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
              {t("table.agent")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.runtime")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.started")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.summary")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.status")}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200/80 bg-white/50">
          {runs.map((run) => (
            <tr
              key={run.id}
              onClick={() => navigate(`/tenants/${run.tenantId}/runs/${run.id}`)}
              tabIndex={0}
              role="link"
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(`/tenants/${run.tenantId}/runs/${run.id}`); } }}
              className="cursor-pointer transition-colors hover:bg-teal-50/70"
            >
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-950">{run.id}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">
                  {run.traceRef ?? t("common.noData")}
                </p>
              </td>
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-800">{run.agentId}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">
                  {run.agentRevision ?? t("common.noData")}
                </p>
              </td>
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-800">{run.runtimeEngine}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{run.runnerClass}</p>
              </td>
              <td className="px-6 py-4 text-sm font-mono text-zinc-600">
                {run.startedAt ?? t("common.noData")}
              </td>
              <td className="px-6 py-4 text-sm text-zinc-700">
                {run.summary ?? t("common.noData")}
              </td>
              <td className="px-6 py-4">
                <StatusBadge status={run.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}