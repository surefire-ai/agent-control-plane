import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { Workspace, UpdateWorkspaceRequest } from "@/types/api";
import { Button } from "@/components/shared/Button";
import { Input } from "@/components/shared/Input";
import { Textarea } from "@/components/shared/Textarea";
import { Select } from "@/components/shared/Select";
import { Field } from "@/components/shared/Field";
import { Card } from "@/components/shared/Card";

interface WorkspaceEditFormProps {
  workspace: Workspace;
  values: UpdateWorkspaceRequest;
  onChange: (values: UpdateWorkspaceRequest) => void;
  onSubmit: () => void;
  onCancel: () => void;
  isPending: boolean;
}

interface FormErrors {
  displayName?: string;
}

export function WorkspaceEditForm({
  workspace,
  values,
  onChange,
  onSubmit,
  onCancel,
  isPending,
}: WorkspaceEditFormProps) {
  const { t } = useTranslation();
  const [errors, setErrors] = useState<FormErrors>({});
  const [submitted, setSubmitted] = useState(false);

  const statusOptions = [
    { value: "active", label: t("status.active") },
    { value: "inactive", label: t("status.inactive") },
    { value: "archived", label: t("status.archived") },
  ];

  const set = (key: keyof UpdateWorkspaceRequest, value: string) => {
    onChange({ ...values, [key]: value });
    if (submitted && key === "displayName") {
      validateDisplayName(value);
    }
  };

  const validateDisplayName = (value: string) => {
    const newErrors = { ...errors };
    if (!value.trim()) {
      newErrors.displayName = t("validation.required");
    } else {
      delete newErrors.displayName;
    }
    setErrors(newErrors);
  };

  const validate = (): boolean => {
    const displayName = values.displayName ?? workspace.displayName;
    const newErrors: FormErrors = {};
    if (!displayName.trim()) {
      newErrors.displayName = t("validation.required");
    }
    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  };

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    setSubmitted(true);
    if (validate()) {
      onSubmit();
    }
  };

  return (
    <Card className="p-6">
      <form onSubmit={handleSubmit} className="space-y-6" noValidate>
        <Field label={t("workspace.fields.displayName")} htmlFor="displayName" required error={errors.displayName}>
          <Input
            id="displayName"
            value={values.displayName ?? workspace.displayName}
            onChange={(e) => set("displayName", e.target.value)}
            hasError={!!errors.displayName}
          />
        </Field>

        <Field label={t("workspace.fields.description")} htmlFor="description">
          <Textarea
            id="description"
            rows={3}
            value={values.description ?? workspace.description ?? ""}
            onChange={(e) => set("description", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.status")} htmlFor="status">
          <Select
            id="status"
            options={statusOptions}
            value={values.status ?? workspace.status}
            onChange={(e) => set("status", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.kubernetesNamespace")} htmlFor="kubernetesNamespace">
          <Input
            id="kubernetesNamespace"
            value={values.kubernetesNamespace ?? workspace.kubernetesNamespace ?? ""}
            onChange={(e) => set("kubernetesNamespace", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.kubernetesWorkspaceName")} htmlFor="kubernetesWorkspaceName">
          <Input
            id="kubernetesWorkspaceName"
            value={values.kubernetesWorkspaceName ?? workspace.kubernetesWorkspaceName ?? ""}
            onChange={(e) => set("kubernetesWorkspaceName", e.target.value)}
          />
        </Field>

        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" disabled={isPending}>
            {isPending ? t("common.saving") : t("common.saveChanges")}
          </Button>
        </div>
      </form>
    </Card>
  );
}
