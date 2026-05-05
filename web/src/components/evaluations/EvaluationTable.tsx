import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import type { Evaluation } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface EvaluationTableProps {
  evaluations: Evaluation[];
}

export function EvaluationTable({ evaluations }: EvaluationTableProps) {
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
              {t("table.dataset")}
            </th>
            <th scope="col">
              {t("table.score")}
            </th>
            <th scope="col">
              {t("table.gate")}
            </th>
            <th scope="col">
              {t("table.samples")}
            </th>
            <th scope="col">
              {t("table.status")}
            </th>
          </tr>
        </thead>
        <tbody>
          {evaluations.map((evaluation) => (
            <tr
              key={evaluation.id}
              className="cursor-pointer transition-colors"
              onClick={() => navigate(`/tenants/${evaluation.tenantId}/evaluations/${evaluation.id}`)}
              onKeyDown={(e) => {
                if (e.key === "Enter" || e.key === " ") {
                  e.preventDefault();
                  navigate(`/tenants/${evaluation.tenantId}/evaluations/${evaluation.id}`);
                }
              }}
              tabIndex={0}
              role="link"
            >
              <td>
                <p className="text-sm font-semibold text-zinc-950">{evaluation.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{evaluation.id}</p>
              </td>
              <td>
                <p className="text-sm font-semibold text-zinc-800">{evaluation.datasetName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">
                  {evaluation.datasetRevision ?? t("common.noData")}
                </p>
              </td>
              <td className="text-sm font-semibold text-zinc-800">
                {Math.round(evaluation.score * 100)}%
              </td>
              <td>
                <StatusBadge status={evaluation.gatePassed ? "passed" : "failed"} />
              </td>
              <td className="text-sm font-mono text-zinc-600">
                {evaluation.samplesEvaluated}/{evaluation.samplesTotal}
              </td>
              <td>
                <StatusBadge status={evaluation.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
