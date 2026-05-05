import { useTranslation } from "react-i18next";
import { Zap, GitBranch, RefreshCw, Wrench, ListChecks, Network } from "lucide-react";

interface PatternSelectorProps {
  selected: string;
  onSelect: (pattern: string) => void;
}

interface PatternOption {
  key: string;
  nameKey: string;
  descKey: string;
  icon: React.ReactNode;
  tagKeys: string[];
}

const patterns: PatternOption[] = [
  { key: "react", nameKey: "studio.pattern.react", descKey: "studio.pattern.reactDesc", icon: <Zap className="h-6 w-6" />, tagKeys: ["studio.patternTags.reactSingleModel", "studio.patternTags.reactToolUse"] },
  { key: "router", nameKey: "studio.pattern.router", descKey: "studio.pattern.routerDesc", icon: <GitBranch className="h-6 w-6" />, tagKeys: ["studio.patternTags.routerMultiRoute", "studio.patternTags.routerConditional"] },
  { key: "reflection", nameKey: "studio.pattern.reflection", descKey: "studio.pattern.reflectionDesc", icon: <RefreshCw className="h-6 w-6" />, tagKeys: ["studio.patternTags.reflectionSelfCorrect", "studio.patternTags.reflectionIterative"] },
  { key: "tool_calling", nameKey: "studio.pattern.toolCalling", descKey: "studio.pattern.toolCallingDesc", icon: <Wrench className="h-6 w-6" />, tagKeys: ["studio.patternTags.toolCallingFunctionCall", "studio.patternTags.toolCallingExternalApis"] },
  { key: "plan_execute", nameKey: "studio.pattern.planExecute", descKey: "studio.pattern.planExecuteDesc", icon: <ListChecks className="h-6 w-6" />, tagKeys: ["studio.patternTags.planExecutePlanning", "studio.patternTags.planExecuteMultiStep"] },
  { key: "workflow", nameKey: "studio.pattern.workflow", descKey: "studio.pattern.workflowDesc", icon: <Network className="h-6 w-6" />, tagKeys: ["studio.patternTags.workflowGraphEditor", "studio.patternTags.workflowVisualCanvas"] },
];

export function PatternSelector({ selected, onSelect }: PatternSelectorProps) {
  const { t } = useTranslation();

  return (
    <div>
      <h3 className="mb-4 text-lg font-semibold text-zinc-950">{t("studio.pattern.select")}</h3>
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {patterns.map((p) => {
          const isSelected = selected === p.key;
          return (
            <button
              key={p.key}
              onClick={() => onSelect(p.key)}
              className={`flex flex-col items-start gap-3 rounded-lg border-2 p-5 text-left transition-all border-t-4 ${
                isSelected
                  ? "border-teal-500 border-t-teal-500 bg-teal-50/50 shadow-sm shadow-[0_0_20px_rgba(20,184,166,0.12)]"
                  : "border-zinc-200 border-t-transparent bg-white hover:border-zinc-300 hover:shadow-sm"
              }`}
            >
              <div className={`rounded-md p-2 ${isSelected ? "bg-teal-100 text-teal-700" : "bg-zinc-100 text-zinc-600"}`}>
                {p.icon}
              </div>
              <div>
                <p className={`text-sm font-semibold ${isSelected ? "text-teal-700" : "text-zinc-900"}`}>
                  {t(p.nameKey)}
                </p>
                <p className="mt-1 text-xs text-zinc-500">{t(p.descKey)}</p>
              </div>
              <div className="flex flex-wrap gap-1.5">
                {p.tagKeys.map((tagKey) => (
                  <span
                    key={tagKey}
                    className="text-[10px] rounded-full px-2 py-0.5 bg-zinc-100 text-zinc-500"
                  >
                    {t(tagKey)}
                  </span>
                ))}
              </div>
            </button>
          );
        })}
      </div>
    </div>
  );
}
