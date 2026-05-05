import { memo } from "react";
import { Handle, Position, type NodeProps } from "@xyflow/react";
import { Cpu, Wrench, Users, BookOpen, Code2, Play, Flag, X } from "lucide-react";
import { useTranslation } from "react-i18next";

export interface WorkflowNodeData {
  label: string;
  kind: "start" | "end" | "model" | "tool" | "agent" | "knowledge" | "custom";
  modelRef?: string;
  toolRef?: string;
  knowledgeRef?: string;
  agentRef?: string;
  implementation?: string;
  [key: string]: unknown;
}

const kindConfig: Record<
  string,
  { icon: React.ElementType; color: string; bg: string; border: string; selectedRing: string }
> = {
  start: { icon: Play, color: "text-emerald-600", bg: "bg-emerald-50", border: "border-emerald-400", selectedRing: "ring-emerald-500" },
  end: { icon: Flag, color: "text-rose-600", bg: "bg-rose-50", border: "border-rose-400", selectedRing: "ring-rose-500" },
  model: { icon: Cpu, color: "text-blue-600", bg: "bg-blue-50", border: "border-blue-400", selectedRing: "ring-blue-500" },
  tool: { icon: Wrench, color: "text-amber-600", bg: "bg-amber-50", border: "border-amber-400", selectedRing: "ring-amber-500" },
  agent: { icon: Users, color: "text-purple-600", bg: "bg-purple-50", border: "border-purple-400", selectedRing: "ring-purple-500" },
  knowledge: { icon: BookOpen, color: "text-orange-600", bg: "bg-orange-50", border: "border-orange-400", selectedRing: "ring-orange-500" },
  custom: { icon: Code2, color: "text-zinc-600", bg: "bg-zinc-50", border: "border-zinc-400", selectedRing: "ring-zinc-500" },
};

function WorkflowNodeInner({ data, selected }: NodeProps & { data: WorkflowNodeData }) {
  const { t } = useTranslation();
  const cfg = kindConfig[data.kind] ?? kindConfig.custom;
  const Icon = cfg.icon;
  const isTerminal = data.kind === "start" || data.kind === "end";

  const subtitle = (() => {
    switch (data.kind) {
      case "model":
        return data.modelRef || t("studio.workflow.noModel");
      case "tool":
        return data.toolRef || t("studio.workflow.noTool");
      case "agent":
        return data.agentRef || t("studio.workflow.noAgent");
      case "knowledge":
        return data.knowledgeRef || t("studio.workflow.noKnowledge");
      case "custom":
        return data.implementation || t("studio.workflow.noImpl");
      default:
        return "";
    }
  })();

  return (
    <div
      className={`
        group relative rounded-xl border-2 shadow-sm transition-all
        ${cfg.bg} ${cfg.border}
        ${selected ? `ring-2 ${cfg.selectedRing} ring-offset-1 shadow-md scale-[1.02]` : "hover:shadow-md hover:scale-[1.01]"}
        ${isTerminal ? "rounded-full px-5 py-2.5" : "min-w-[160px] px-4 py-3 shadow-[inset_0_1px_0_rgba(255,255,255,0.5)]"}
      `}
    >
      {/* Delete button on hover */}
      {selected && !isTerminal && (
        <button
          type="button"
          onClick={(e) => {
            e.stopPropagation();
            // Dispatch keyboard event to trigger deletion handler
            document.dispatchEvent(new KeyboardEvent("keydown", { key: "Delete" }));
          }}
          className="absolute -top-2 -right-2 z-10 flex h-5 w-5 items-center justify-center rounded-full bg-red-500 text-white shadow-sm opacity-0 group-hover:opacity-100 transition-opacity hover:bg-red-600"
          title={t("studio.workflow.deleteNodeTitle")}
        >
          <X className="h-3 w-3" />
        </button>
      )}

      {/* Input handle (not on start node) */}
      {data.kind !== "start" && (
        <Handle
          type="target"
          position={Position.Left}
          className="!w-3 !h-3 group-hover:!w-3.5 group-hover:!h-3.5 !bg-zinc-400 !border-2 !border-white group-hover:!bg-teal-500 transition-all"
        />
      )}

      <div className="flex items-center gap-2.5">
        <div className={`flex h-8 w-8 items-center justify-center rounded-lg ${cfg.bg} border ${cfg.border}`}>
          <Icon className={`h-4 w-4 ${cfg.color}`} strokeWidth={2} />
        </div>
        <div className="min-w-0">
          {!isTerminal && (
            <p className="text-[9px] uppercase tracking-wider font-semibold text-zinc-400 mb-0.5">
              {t(`studio.workflow.kind.${data.kind}`)}
            </p>
          )}
          <p className={`text-sm font-semibold ${cfg.color} truncate`}>
            {data.label || t("studio.workflow.unnamed")}
          </p>
          {subtitle && (
            <p className="text-[11px] text-zinc-500 truncate">{subtitle}</p>
          )}
        </div>
      </div>

      {/* Output handle (not on end node) */}
      {data.kind !== "end" && (
        <Handle
          type="source"
          position={Position.Right}
          className="!w-3 !h-3 group-hover:!w-3.5 group-hover:!h-3.5 !bg-zinc-400 !border-2 !border-white group-hover:!bg-teal-500 transition-all"
        />
      )}
    </div>
  );
}

export const WorkflowNode = memo(WorkflowNodeInner);
