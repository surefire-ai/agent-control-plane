import { useTranslation } from "react-i18next";
import type { Evaluation } from "@/types/api";
import { StatusBadge } from "@/components/shared/StatusBadge";

interface EvaluationTableProps {
  evaluations: Evaluation[];
}

export function EvaluationTable({ evaluations }: EvaluationTableProps) {
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
              {t("table.dataset")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.score")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.gate")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.samples")}
            </th>
            <th scope="col" className="px-6 py-3 text-left text-xs font-semibold uppercase text-zinc-500">
              {t("table.status")}
            </th>
          </tr>
        </thead>
        <tbody className="divide-y divide-zinc-200/80 bg-white/50">
          {evaluations.map((evaluation) => (
            <tr key={evaluation.id} className="transition-colors hover:bg-teal-50/70">
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-950">{evaluation.displayName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">{evaluation.id}</p>
              </td>
              <td className="px-6 py-4">
                <p className="text-sm font-semibold text-zinc-800">{evaluation.datasetName}</p>
                <p className="mt-1 text-xs font-mono text-zinc-500">
                  {evaluation.datasetRevision ?? t("common.noData")}
                </p>
              </td>
              <td className="px-6 py-4 text-sm font-semibold text-zinc-800">
                {Math.round(evaluation.score * 100)}%
              </td>
              <td className="px-6 py-4">
                <StatusBadge status={evaluation.gatePassed ? "passed" : "failed"} />
              </td>
              <td className="px-6 py-4 text-sm font-mono text-zinc-600">
                {evaluation.samplesEvaluated}/{evaluation.samplesTotal}
              </td>
              <td className="px-6 py-4">
                <StatusBadge status={evaluation.status} />
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}