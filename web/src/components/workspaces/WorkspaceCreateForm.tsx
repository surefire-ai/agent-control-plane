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

export function WorkspaceCreateForm({
  values,
  onChange,
  onSubmit,
  onCancel,
  isPending,
}: WorkspaceCreateFormProps) {
  const { t } = useTranslation();

  const statusOptions = [
    { value: "active", label: t("status.active") },
    { value: "inactive", label: t("status.inactive") },
  ];

  const set = (key: keyof CreateWorkspaceRequest, value: string) =>
    onChange({ ...values, [key]: value });

  return (
    <Card className="p-6">
      <form
        onSubmit={(e) => {
          e.preventDefault();
          onSubmit();
        }}
        className="space-y-6"
      >
        <Field label={t("workspace.fields.id")} htmlFor="id">
          <Input
            id="id"
            required
            placeholder="ws_my_workspace"
            value={values.id}
            onChange={(e) => set("id", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.slug")} htmlFor="slug">
          <Input
            id="slug"
            required
            placeholder="my-workspace"
            value={values.slug}
            onChange={(e) => set("slug", e.target.value)}
          />
        </Field>

        <Field label={t("workspace.fields.displayName")} htmlFor="displayName">
          <Input
            id="displayName"
            required
            placeholder="My Workspace"
            value={values.displayName}
            onChange={(e) => set("displayName", e.target.value)}
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
