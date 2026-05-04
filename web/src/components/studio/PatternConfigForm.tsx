import { useTranslation } from "react-i18next";
import { Plus, X } from "lucide-react";
import type { PatternConfig, PatternRoute } from "@/types/api";
import { Input } from "@/components/shared/Input";
import { Button } from "@/components/shared/Button";

interface PatternConfigFormProps {
  pattern: string;
  config: PatternConfig;
  onChange: (config: PatternConfig) => void;
}

function FieldLabel({ label }: { label: string }) {
  return <label className="mb-1.5 block text-sm font-medium text-zinc-700">{label}</label>;
}

function ModelRefField({ value, onChange, label }: { value: string; onChange: (v: string) => void; label: string }) {
  const { t } = useTranslation();
  return (
    <div>
      <FieldLabel label={label} />
      <Input
        placeholder="e.g. default-model"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
      <p className="mt-1 text-xs text-zinc-400">{t("studio.config.modelRef")}</p>
    </div>
  );
}

function MaxIterationsField({ value, onChange }: { value: number | undefined; onChange: (v: number | undefined) => void }) {
  const { t } = useTranslation();
  return (
    <div>
      <FieldLabel label={t("studio.config.maxIterations")} />
      <Input
        type="number"
        min={1}
        max={100}
        placeholder="10"
        value={value ?? ""}
        onChange={(e) => onChange(e.target.value ? Number(e.target.value) : undefined)}
      />
    </div>
  );
}

function StopWhenField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const { t } = useTranslation();
  return (
    <div>
      <FieldLabel label={t("studio.config.stopWhen")} />
      <Input
        placeholder="e.g. task_complete"
        value={value}
        onChange={(e) => onChange(e.target.value)}
      />
    </div>
  );
}

function RouterRoutesField({
  routes,
  onChange,
}: {
  routes: PatternRoute[];
  onChange: (routes: PatternRoute[]) => void;
}) {
  const { t } = useTranslation();

  const addRoute = () => {
    onChange([...routes, { label: "", agentRef: "", modelRef: "", default: false }]);
  };

  const removeRoute = (index: number) => {
    onChange(routes.filter((_, i) => i !== index));
  };

  const updateRoute = (index: number, field: keyof PatternRoute, value: string | boolean) => {
    const updated = routes.map((r, i) => (i === index ? { ...r, [field]: value } : r));
    onChange(updated);
  };

  return (
    <div className="mt-4">
      <div className="mb-3 flex items-center justify-between">
        <h4 className="text-sm font-semibold text-zinc-800">{t("studio.routes.title")}</h4>
        <Button variant="secondary" size="sm" onClick={addRoute} type="button">
          <Plus className="mr-1 h-3.5 w-3.5" />
          {t("studio.routes.add")}
        </Button>
      </div>
      {routes.length === 0 && (
        <p className="rounded-md border border-dashed border-zinc-300 p-4 text-center text-sm text-zinc-400">
          {t("studio.routes.add")}
        </p>
      )}
      {routes.map((route, index) => (
        <div key={index} className="mb-3 rounded-md border border-zinc-200 bg-zinc-50/50 p-4">
          <div className="mb-3 flex items-center justify-between">
            <span className="text-xs font-medium text-zinc-500">#{index + 1}</span>
            <button
              type="button"
              onClick={() => removeRoute(index)}
              className="rounded p-1 text-zinc-400 hover:bg-rose-50 hover:text-rose-600"
            >
              <X className="h-4 w-4" />
            </button>
          </div>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-3">
            <div>
              <FieldLabel label={t("studio.routes.label")} />
              <Input
                value={route.label}
                onChange={(e) => updateRoute(index, "label", e.target.value)}
                placeholder="e.g. customer_support"
              />
            </div>
            <div>
              <FieldLabel label={t("studio.routes.agentRef")} />
              <Input
                value={route.agentRef ?? ""}
                onChange={(e) => updateRoute(index, "agentRef", e.target.value)}
                placeholder="e.g. support-agent"
              />
            </div>
            <div>
              <FieldLabel label={t("studio.routes.modelRef")} />
              <Input
                value={route.modelRef ?? ""}
                onChange={(e) => updateRoute(index, "modelRef", e.target.value)}
                placeholder="e.g. gpt-4o"
              />
            </div>
          </div>
          <label className="mt-3 flex items-center gap-2">
            <input
              type="checkbox"
              className="h-4 w-4 rounded border-zinc-300 text-teal-600 focus:ring-teal-500"
              checked={route.default ?? false}
              onChange={(e) => updateRoute(index, "default", e.target.checked)}
            />
            <span className="text-sm text-zinc-600">{t("studio.routes.default")}</span>
          </label>
        </div>
      ))}
    </div>
  );
}

export function PatternConfigForm({ pattern, config, onChange }: PatternConfigFormProps) {
  const { t } = useTranslation();

  const updateConfig = (updates: Partial<PatternConfig>) => {
    onChange({ ...config, ...updates });
  };

  return (
    <div className="mt-6">
      <h3 className="mb-4 text-base font-semibold text-zinc-800">
        {t("studio.pattern.select")} — {t(`studio.pattern.${pattern === "tool_calling" ? "toolCalling" : pattern === "plan_execute" ? "planExecute" : pattern}`)}
      </h3>

      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2">
        {pattern === "react" && (
          <>
            <ModelRefField value={config.modelRef ?? ""} onChange={(v) => updateConfig({ modelRef: v })} label={t("studio.config.modelRef")} />
            <MaxIterationsField value={config.maxIterations} onChange={(v) => updateConfig({ maxIterations: v })} />
            <div className="sm:col-span-2">
              <StopWhenField value={config.stopWhen ?? ""} onChange={(v) => updateConfig({ stopWhen: v })} />
            </div>
          </>
        )}

        {pattern === "router" && (
          <div className="sm:col-span-2">
            <ModelRefField value={config.modelRef ?? ""} onChange={(v) => updateConfig({ modelRef: v })} label={t("studio.config.modelRef")} />
            <RouterRoutesField routes={config.routes ?? []} onChange={(routes) => updateConfig({ routes })} />
          </div>
        )}

        {pattern === "reflection" && (
          <>
            <ModelRefField value={config.modelRef ?? ""} onChange={(v) => updateConfig({ modelRef: v })} label={t("studio.config.modelRef")} />
            <MaxIterationsField value={config.maxIterations} onChange={(v) => updateConfig({ maxIterations: v })} />
          </>
        )}

        {pattern === "tool_calling" && (
          <ModelRefField value={config.modelRef ?? ""} onChange={(v) => updateConfig({ modelRef: v })} label={t("studio.config.modelRef")} />
        )}

        {pattern === "plan_execute" && (
          <>
            <ModelRefField value={config.modelRef ?? ""} onChange={(v) => updateConfig({ modelRef: v })} label={t("studio.config.plannerModel")} />
            <ModelRefField value={config.executorModelRef ?? ""} onChange={(v) => updateConfig({ executorModelRef: v })} label={t("studio.config.executorModel")} />
          </>
        )}

      </div>
    </div>
  );
}
