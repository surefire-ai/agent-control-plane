import { useTranslation } from "react-i18next";
import type { Agent } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface AgentTableProps {
  agents: Agent[];
}

export function AgentTable({ agents }: AgentTableProps) {
  const { t } = useTranslation();

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
              {t("table.pattern")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.model")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.runtime")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.status")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.revision")}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200/80 bg-white/50">
          {agents.map((agent) => (
            <tr key={agent.id} className="transition-colors hover:bg-teal-50/70">
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-950">{agent.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{agent.id}</p>
              </td>
              <td className="px-6 py-4 text-sm font-mono text-zinc-700">{agent.pattern}</td>
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-800">{agent.modelProvider ?? t("common.noData")}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{agent.modelName ?? t("common.noData")}</p>
              </td>
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-800">{agent.runtimeEngine}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{agent.runnerClass}</p>
              </td>
              <td className="px-6 py-4">
                <StatusBadge status={agent.status} />
              </td>
              <td className="px-6 py-4 text-sm font-mono text-zinc-600">
                {agent.latestRevision ?? t("common.noData")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}