import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import type { Agent } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface AgentTableProps {
  agents: Agent[];
}

export function AgentTable({ agents }: AgentTableProps) {
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
              {t("table.pattern")}
            </th>
            <th scope="col">
              {t("table.model")}
            </th>
            <th scope="col">
              {t("table.runtime")}
            </th>
            <th scope="col">
              {t("table.status")}
            </th>
            <th scope="col">
              {t("table.revision")}
            </th>
          </tr>
        </thead>
        <tbody>
          {agents.map((agent) => (
            <tr
              key={agent.id}
              onClick={() => navigate(`/tenants/${agent.tenantId}/agents/${agent.id}`)}
              tabIndex={0}
              role="link"
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(`/tenants/${agent.tenantId}/agents/${agent.id}`); } }}
              className="cursor-pointer transition-colors"
            >
              <td>
                <p className="text-sm font-semibold text-zinc-950">{agent.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{agent.id}</p>
              </td>
              <td className="text-sm font-mono text-zinc-700">{agent.pattern}</td>
              <td>
                <p className="text-sm font-semibold text-zinc-800">{agent.modelProvider ?? t("common.noData")}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{agent.modelName ?? t("common.noData")}</p>
              </td>
              <td>
                <p className="text-sm font-semibold text-zinc-800">{agent.runtimeEngine}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{agent.runnerClass}</p>
              </td>
              <td>
                <StatusBadge status={agent.status} />
              </td>
              <td className="text-sm font-mono text-zinc-600">
                {agent.latestRevision ?? t("common.noData")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
