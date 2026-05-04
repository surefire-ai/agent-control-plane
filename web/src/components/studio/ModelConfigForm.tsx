import { useTranslation } from "react-i18next";
import { Plus, Trash2 } from "lucide-react";
import type { ModelConfig, SecretKeyReference } from "@/types/api";
import { Input } from "@/components/shared/Input";
import { Button } from "@/components/shared/Button";
import { Card } from "@/components/shared/Card";

interface ModelConfigFormProps {
  models: Record<string, ModelConfig>;
  onChange: (models: Record<string, ModelConfig>) => void;
}

export function ModelConfigForm({ models, onChange }: ModelConfigFormProps) {
  const { t } = useTranslation();

  const modelEntries = Object.entries(models);

  const addModel = () => {
    const key = `model-${Date.now()}`;
    onChange({ ...models, [key]: { provider: "", model: "", temperature: 0.7, maxTokens: 4096 } });
  };

  const removeModel = (key: string) => {
    const next = { ...models };
    delete next[key];
    onChange(next);
  };

  const updateModel = (key: string, field: keyof ModelConfig, value: string | number | SecretKeyReference | undefined) => {
    onChange({ ...models, [key]: { ...models[key], [field]: value } });
  };

  const credentialRefFor = (model: ModelConfig): SecretKeyReference => {
    if (typeof model.credentialRef === "string") {
      return { name: model.credentialRef, key: "" };
    }
    return model.credentialRef ?? { name: "", key: "" };
  };

  const updateCredentialRef = (key: string, field: keyof SecretKeyReference, value: string) => {
    const current = credentialRefFor(models[key]);
    const next = { ...current, [field]: value };
    updateModel(key, "credentialRef", next.name || next.key ? next : undefined);
  };

  return (
    <div>
      <div className="mb-4 flex items-center justify-between">
        <h3 className="text-lg font-semibold text-zinc-950">{t("studio.models.title")}</h3>
        <Button variant="secondary" size="sm" onClick={addModel} type="button">
          <Plus className="mr-1 h-3.5 w-3.5" />
          {t("studio.models.add")}
        </Button>
      </div>

      {modelEntries.length === 0 && (
        <Card className="p-6">
          <p className="text-center text-sm text-zinc-400">
            {t("studio.models.add")}
          </p>
        </Card>
      )}

      <div className="space-y-4">
        {modelEntries.map(([key, model]) => {
          const credentialRef = credentialRefFor(model);

          return (
            <Card key={key} className="p-4">
              <div className="mb-3 flex items-center justify-between">
                <span className="text-sm font-medium text-zinc-700">{key}</span>
                <button
                  type="button"
                  onClick={() => removeModel(key)}
                  className="rounded p-1 text-zinc-400 hover:bg-rose-50 hover:text-rose-600"
                >
                  <Trash2 className="h-4 w-4" />
                </button>
              </div>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.provider")}
                  </label>
                  <Input
                    value={model.provider ?? ""}
                    onChange={(e) => updateModel(key, "provider", e.target.value)}
                    placeholder={t("studio.models.providerPlaceholder")}
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.modelName")}
                  </label>
                  <Input
                    value={model.model ?? ""}
                    onChange={(e) => updateModel(key, "model", e.target.value)}
                    placeholder={t("studio.models.modelNamePlaceholder")}
                  />
                </div>
                <div className="sm:col-span-2">
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.baseURL")}
                  </label>
                  <Input
                    value={model.baseURL ?? ""}
                    onChange={(e) => updateModel(key, "baseURL", e.target.value || undefined)}
                    placeholder={t("studio.models.baseURLPlaceholder")}
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.credentialName")}
                  </label>
                  <Input
                    value={credentialRef.name}
                    onChange={(e) => updateCredentialRef(key, "name", e.target.value)}
                    placeholder={t("studio.models.credentialNamePlaceholder")}
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.credentialKey")}
                  </label>
                  <Input
                    value={credentialRef.key}
                    onChange={(e) => updateCredentialRef(key, "key", e.target.value)}
                    placeholder={t("studio.models.credentialKeyPlaceholder")}
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.temperature")}
                  </label>
                  <div className="flex items-center gap-3">
                    <input
                      type="range"
                      min={0}
                      max={2}
                      step={0.1}
                      value={model.temperature ?? 0.7}
                      onChange={(e) => updateModel(key, "temperature", Number(e.target.value))}
                      className="h-2 flex-1 cursor-pointer appearance-none rounded-lg bg-zinc-200 accent-teal-600"
                    />
                    <span className="min-w-[2.5rem] text-right text-sm font-mono text-zinc-600">
                      {(model.temperature ?? 0.7).toFixed(1)}
                    </span>
                  </div>
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.maxTokens")}
                  </label>
                  <Input
                    type="number"
                    min={1}
                    max={128000}
                    value={model.maxTokens ?? ""}
                    onChange={(e) => updateModel(key, "maxTokens", e.target.value ? Number(e.target.value) : undefined)}
                    placeholder="4096"
                  />
                </div>
                <div>
                  <label className="mb-1.5 block text-sm font-medium text-zinc-700">
                    {t("studio.models.timeoutSeconds")}
                  </label>
                  <Input
                    type="number"
                    min={1}
                    value={model.timeoutSeconds ?? ""}
                    onChange={(e) => updateModel(key, "timeoutSeconds", e.target.value ? Number(e.target.value) : undefined)}
                    placeholder="60"
                  />
                </div>
              </div>
            </Card>
          );
        })}
      </div>
    </div>
  );
}
