import { useNavigate, useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import type { Workspace } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface WorkspaceTableProps {
  workspaces: Workspace[];
}

export function WorkspaceTable({ workspaces }: WorkspaceTableProps) {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { tenantId } = useParams<{ tenantId: string }>();

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
              {t("table.slug")}
            </th>
            <th scope="col">
              {t("table.status")}
            </th>
            <th scope="col">
              {t("table.namespace")}
            </th>
          </tr>
        </thead>
        <tbody>
          {workspaces.map((ws) => (
            <tr
              key={ws.id}
              onClick={() => navigate(`/tenants/${tenantId}/workspaces/${ws.id}`)}
              tabIndex={0}
              role="link"
              onKeyDown={(e) => { if (e.key === "Enter" || e.key === " ") { e.preventDefault(); navigate(`/tenants/${tenantId}/workspaces/${ws.id}`); } }}
              className="cursor-pointer transition-colors"
            >
              <td>
                <p className="text-sm font-semibold text-zinc-950">{ws.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{ws.id}</p>
              </td>
              <td className="text-sm font-mono text-zinc-700">{ws.slug}</td>
              <td>
                <StatusBadge status={ws.status} />
              </td>
              <td className="text-sm font-mono text-zinc-600">
                {ws.kubernetesNamespace ?? t("common.noData")}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
