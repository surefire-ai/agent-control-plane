import { useTranslation } from "react-i18next";
import { X, Cpu, Wrench, Users, BookOpen, Code2, Play, Flag, Trash2 } from "lucide-react";
import type { WorkflowNodeData } from "./WorkflowNode";
import { Input } from "@/components/shared/Input";
import { Button } from "@/components/shared/Button";

interface NodeConfigPanelProps {
  data: WorkflowNodeData;
  nodeId: string;
  onUpdate: (updates: Partial<WorkflowNodeData>) => void;
  onDelete: () => void;
  onClose: () => void;
}

const kindIcons: Record<string, React.ElementType> = {
  start: Play, end: Flag, model: Cpu, tool: Wrench, agent: Users, knowledge: BookOpen, custom: Code2,
};
const kindColors: Record<string, string> = {
  start: "text-emerald-600", end: "text-rose-600", model: "text-blue-600", tool: "text-amber-600",
  agent: "text-purple-600", knowledge: "text-orange-600", custom: "text-zinc-600",
};
const kindBgs: Record<string, string> = {
  start: "bg-emerald-50", end: "bg-rose-50", model: "bg-blue-50", tool: "bg-amber-50",
  agent: "bg-purple-50", knowledge: "bg-orange-50", custom: "bg-zinc-50",
};

function FieldLabel({ label }: { label: string }) {
  return <label className="mb-1.5 block text-xs font-medium text-zinc-600">{label}</label>;
}

export function NodeConfigPanel({ data, nodeId, onUpdate, onDelete, onClose }: NodeConfigPanelProps) {
  const { t } = useTranslation();
  const Icon = kindIcons[data.kind] ?? Code2;
  const color = kindColors[data.kind] ?? "text-zinc-600";
  const bg = kindBgs[data.kind] ?? "bg-zinc-50";
  const isTerminal = data.kind === "start" || data.kind === "end";

  return (
    <div className="w-80 shrink-0 border-l border-zinc-200 bg-white overflow-y-auto">
      {/* Header */}
      <div className="sticky top-0 z-10 flex items-center justify-between border-b border-zinc-200 bg-white px-4 py-3">
        <div className="flex items-center gap-2.5">
          <div className={`flex h-7 w-7 items-center justify-center rounded-lg ${bg} border border-zinc-200`}>
            <Icon className={`h-4 w-4 ${color}`} />
          </div>
          <div>
            <span className="text-sm font-semibold text-zinc-800 block">
              {t(`studio.workflow.kind.${data.kind}`)}
            </span>
            <span className="text-[10px] text-zinc-400">
              {t(`studio.workflow.kindDesc.${data.kind}`)}
            </span>
          </div>
        </div>
        <button type="button" onClick={onClose} className="rounded-lg p-1.5 text-zinc-400 hover:bg-zinc-100 hover:text-zinc-600 transition-colors">
          <X className="h-4 w-4" />
        </button>
      </div>

      {/* Fields */}
      <div className="space-y-4 p-4">
        {/* Node ID (read-only) */}
        <div>
          <FieldLabel label={t("studio.workflow.nodeId")} />
          <div className="rounded-lg border border-zinc-200 bg-zinc-50 px-3 py-2 text-xs font-mono text-zinc-500">
            {nodeId}
          </div>
          <p className="mt-1 text-[10px] text-zinc-400">
            {t("studio.workflow.nodeIdHint")}
          </p>
        </div>

        <div>
          <FieldLabel label={t("studio.workflow.nodeName")} />
          <Input
            value={data.label}
            onChange={(e) => onUpdate({ label: e.target.value })}
            placeholder={t("studio.workflow.nodeNamePlaceholder")}
          />
          <p className="mt-1 text-[10px] text-zinc-400">
            {t("studio.workflow.nodeNameHint")}
          </p>
        </div>

        {!isTerminal && data.kind === "model" && (
          <div>
            <FieldLabel label={t("studio.workflow.modelRef")} />
            <Input value={data.modelRef ?? ""} onChange={(e) => onUpdate({ modelRef: e.target.value })} placeholder={t("studio.workflow.placeholderModelRef")} />
            <p className="mt-1 text-[10px] text-zinc-400">{t("studio.workflow.modelRefHint")}</p>
          </div>
        )}

        {!isTerminal && data.kind === "tool" && (
          <div>
            <FieldLabel label={t("studio.workflow.toolRef")} />
            <Input value={data.toolRef ?? ""} onChange={(e) => onUpdate({ toolRef: e.target.value })} placeholder={t("studio.workflow.placeholderToolRef")} />
            <p className="mt-1 text-[10px] text-zinc-400">{t("studio.workflow.toolRefHint")}</p>
          </div>
        )}

        {!isTerminal && data.kind === "knowledge" && (
          <div>
            <FieldLabel label={t("studio.workflow.knowledgeRef")} />
            <Input value={data.knowledgeRef ?? ""} onChange={(e) => onUpdate({ knowledgeRef: e.target.value })} placeholder={t("studio.workflow.placeholderKnowledgeRef")} />
            <p className="mt-1 text-[10px] text-zinc-400">{t("studio.workflow.knowledgeRefHint")}</p>
          </div>
        )}

        {!isTerminal && data.kind === "agent" && (
          <div>
            <FieldLabel label={t("studio.workflow.agentRef")} />
            <Input value={data.agentRef ?? ""} onChange={(e) => onUpdate({ agentRef: e.target.value })} placeholder={t("studio.workflow.placeholderAgentRef")} />
            <p className="mt-1 text-[10px] text-zinc-400">{t("studio.workflow.agentRefHint")}</p>
          </div>
        )}

        {!isTerminal && data.kind === "custom" && (
          <div>
            <FieldLabel label={t("studio.workflow.implementation")} />
            <Input value={data.implementation ?? ""} onChange={(e) => onUpdate({ implementation: e.target.value })} placeholder={t("studio.workflow.placeholderImplementation")} />
            <p className="mt-1 text-[10px] text-zinc-400">{t("studio.workflow.implementationHint")}</p>
          </div>
        )}
      </div>

      {/* Delete button */}
      {!isTerminal && (
        <div className="border-t border-zinc-100 p-4">
          <Button variant="secondary" onClick={onDelete} className="w-full text-red-600 hover:bg-red-50 hover:text-red-700 border-red-200">
            <Trash2 className="h-3.5 w-3.5 mr-1.5" />
            {t("studio.workflow.deleteNode")}
          </Button>
        </div>
      )}
    </div>
  );
}
