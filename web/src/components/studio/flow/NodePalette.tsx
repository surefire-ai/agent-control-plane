import { Cpu, Wrench, Users, BookOpen, Code2, Play, Flag } from "lucide-react";
import { useTranslation } from "react-i18next";

interface PaletteItem {
  kind: string;
  label: string;
  icon: React.ElementType;
  color: string;
  bg: string;
  border: string;
}

const items: PaletteItem[] = [
  { kind: "start", label: "Start", icon: Play, color: "text-emerald-600", bg: "bg-emerald-50", border: "border-emerald-200" },
  { kind: "model", label: "Model", icon: Cpu, color: "text-blue-600", bg: "bg-blue-50", border: "border-blue-200" },
  { kind: "tool", label: "Tool", icon: Wrench, color: "text-amber-600", bg: "bg-amber-50", border: "border-amber-200" },
  { kind: "agent", label: "Agent", icon: Users, color: "text-purple-600", bg: "bg-purple-50", border: "border-purple-200" },
  { kind: "knowledge", label: "Knowledge", icon: BookOpen, color: "text-orange-600", bg: "bg-orange-50", border: "border-orange-200" },
  { kind: "custom", label: "Custom", icon: Code2, color: "text-zinc-600", bg: "bg-zinc-50", border: "border-zinc-200" },
  { kind: "end", label: "End", icon: Flag, color: "text-rose-600", bg: "bg-rose-50", border: "border-rose-200" },
];

interface NodePaletteProps {
  onAddNode: (kind: string, position?: { x: number; y: number }) => void;
}

export function NodePalette({ onAddNode }: NodePaletteProps) {
  const { t } = useTranslation();

  const handleDragStart = (event: React.DragEvent, kind: string) => {
    event.dataTransfer.setData("application/korus-node-kind", kind);
    event.dataTransfer.effectAllowed = "move";
  };

  return (
    <div className="w-52 shrink-0 border-r border-zinc-200 bg-zinc-100/60 overflow-y-auto">
      <div className="sticky top-0 z-10 border-b border-zinc-200 bg-zinc-100/95 backdrop-blur-sm px-3 py-2.5">
        <h4 className="text-xs font-semibold uppercase tracking-wider text-zinc-500">
          {t("studio.workflow.palette")}
        </h4>
        <p className="mt-0.5 text-[10px] text-zinc-400">
          {t("studio.workflow.paletteHint")}
        </p>
      </div>

      <div className="px-3 pt-2.5 pb-1">
        <p className="text-[10px] font-medium uppercase tracking-wider text-zinc-400 mb-1.5">
          {t("studio.workflow.components")}
        </p>
      </div>

      <div className="space-y-1.5 px-3 pb-3">
        {items.map((item) => {
          const Icon = item.icon;
          return (
            <button
              key={item.kind}
              type="button"
              draggable
              onDragStart={(e) => handleDragStart(e, item.kind)}
              onClick={() => onAddNode(item.kind)}
              className={`flex w-full items-center gap-2.5 rounded-lg border ${item.border} ${item.bg} px-3 py-2.5 text-left text-sm transition-all hover:shadow-sm hover:scale-[1.01] active:scale-[0.98] cursor-grab active:cursor-grabbing hover:border-dashed`}
              title={t("studio.workflow.dragToAdd")}
            >
              <div className={`flex h-7 w-7 items-center justify-center rounded-md ${item.bg}`}>
                <Icon className={`h-4 w-4 ${item.color}`} strokeWidth={2} />
              </div>
              <div>
                <span className="font-medium text-zinc-700">{item.label}</span>
                <p className="text-[10px] text-zinc-400">
                  {t(`studio.workflow.kindDesc.${item.kind}`)}
                </p>
              </div>
            </button>
          );
        })}
      </div>

      {/* Quick tips */}
      <div className="border-t border-zinc-200 px-3 py-2.5">
        <p className="text-[10px] font-medium uppercase tracking-wider text-zinc-400 mb-1">{t("studio.workflow.tips")}</p>
        <ul className="space-y-0.5 text-[10px] text-zinc-400">
          <li>• {t("studio.workflow.tipClick")}</li>
          <li>• {t("studio.workflow.tipDrag")}</li>
          <li>• {t("studio.workflow.tipConnect")}</li>
        </ul>
      </div>
    </div>
  );
}
