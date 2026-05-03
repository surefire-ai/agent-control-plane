import { useState } from "react";
import { useTranslation } from "react-i18next";
import type { CreateWorkspaceRequest } from "@/types/api";
import { Button } from "@/components/shared/Button";
import { Input } from "@/components/shared/Input";
import { Textarea } from "@/components/shared/Textarea";
import { Select } from "@/components/shared/Select";
import { Field } from "@/components/shared/Field";
import { Card } from "@/components/shared/Card";

interface WorkspaceCreateFormProps {
  values: CreateWorkspaceRequest;
  onChange: (values: CreateWorkspaceRequest) => void;
  onSubmit: () => void;
  onCancel: () => void;
  isPending: boolean;
}

interface FormErrors {
  id?: string;
  slug?: string;
  displayName?: string;
}

export function WorkspaceCreateForm({
  values,
  onChange,
  onSubmit,
  onCancel,
  isPending,
}: WorkspaceCreateFormProps) {
  const { t } = useTranslation();
  const [errors, setErrors] = useState<FormErrors>({});
  const [submitted, setSubmitted] = useState(false);

  const statusOptions = [
    { value: "active", label: t("status.active") },
    { value: "inactive", label: t("status.inactive") },
  ];

  const set = (key: keyof CreateWorkspaceRequest, value: string) => {
    onChange({ ...values, [key]: value });
    if (submitted) {
      validateField(key, value);
    }
  };

  const validateField = (key: string, value: string) => {
    const newErrors = { ...errors };
    if (key === "id" || key === "slug" || key === "displayName") {
      if (!value.trim()) {
        newErrors[key as keyof FormErrors] = t("validation.required");
      } else {
        delete newErrors[key as keyof FormErrors];
      }
    }
    setErrors(newErrors);
  };

  const validate = (): boolean => {
    const newErrors: FormErrors = {};
    if (!values.id.trim()) newErrors.id = t("validation.required");
    if (!values.slug.trim()) newErrors.slug = t("validation.required");
    if (!values.displayName.trim()) newErrors.displayName = t("validation.required");
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
        <Field label={t("workspace.fields.id")} htmlFor="id" required error={errors.id}>
          <Input
            id="id"
            placeholder="ws_my_workspace"
            value={values.id}
            onChange={(e) => set("id", e.target.value)}
            hasError={!!errors.id}
          />
        </Field>

        <Field label={t("workspace.fields.slug")} htmlFor="slug" required error={errors.slug}>
          <Input
            id="slug"
            placeholder="my-workspace"
            value={values.slug}
            onChange={(e) => set("slug", e.target.value)}
            hasError={!!errors.slug}
          />
        </Field>

        <Field label={t("workspace.fields.displayName")} htmlFor="displayName" required error={errors.displayName}>
          <Input
            id="displayName"
            placeholder="My Workspace"
            value={values.displayName}
            onChange={(e) => set("displayName", e.target.value)}
            hasError={!!errors.displayName}
          />
        </Field>

        <Field label={t("workspace.fields.description")} htmlFor="description">
          <Textarea
            id="description"
            rows={3}
            value={values.description ?? ""}
            onChange={(e) => set("description", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.status")} htmlFor="status">
          <Select
            id="status"
            options={statusOptions}
            value={values.status ?? "active"}
            onChange={(e) => set("status", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.kubernetesNamespace")} htmlFor="kubernetesNamespace">
          <Input
            id="kubernetesNamespace"
            value={values.kubernetesNamespace ?? ""}
            onChange={(e) => set("kubernetesNamespace", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.kubernetesWorkspaceName")} htmlFor="kubernetesWorkspaceName">
          <Input
            id="kubernetesWorkspaceName"
            value={values.kubernetesWorkspaceName ?? ""}
            onChange={(e) => set("kubernetesWorkspaceName", e.target.value)}
          />
        </Field>

        <div className="flex justify-end gap-3 pt-2">
          <Button type="button" variant="secondary" onClick={onCancel}>
            {t("common.cancel")}
          </Button>
          <Button type="submit" disabled={isPending}>
            {isPending ? t("common.creating") : t("workspace.createButton")}
          </Button>
        </div>
      </form>
    </Card>
  );
}
