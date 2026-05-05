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
      <table className="data-table min-w-full">
        <caption className="sr-only">{t("table.name")}</caption>
        <thead>
          <tr>
            <th scope="col">
              {t("table.name")}
            </th>
            <th scope="col">
              {t("table.agent")}
            </th>
            <th scope="col">
              {t("table.runtime")}
            </th>
            <th scope="col">
              {t("table.started")}
            </th>
            <th scope="col">
              {t("table.summary")}
            </th>
            <th scope="col">
              {t("table.status")}
            </th>
          </tr>
        </thead>
        <tbody>
          {runs.map((run) => (
            <tr
              key={run.id}
              onClick={() => navigate(`/tenants/${run.tenantId}/runs/${run.id}`)}
              tabIndex={0}
              role="link"
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(`/tenants/${run.tenantId}/runs/${run.id}`); } }}
              className="cursor-pointer transition-colors"
            >
              <td>
                <p className="text-sm font-semibold text-zinc-950">{run.id}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">
                  {run.traceRef ?? t("common.noData")}
                </p>
              </td>
              <td>
                <p className="text-sm font-semibold text-zinc-800">{run.agentId}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">
                  {run.agentRevision ?? t("common.noData")}
                </p>
              </td>
              <td>
                <p className="text-sm font-semibold text-zinc-800">{run.runtimeEngine}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{run.runnerClass}</p>
              </td>
              <td className="text-sm font-mono text-zinc-600">
                {run.startedAt ?? t("common.noData")}
              </td>
              <td className="text-sm text-zinc-700">
                {run.summary ?? t("common.noData")}
              </td>
              <td>
                <StatusBadge status={run.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
