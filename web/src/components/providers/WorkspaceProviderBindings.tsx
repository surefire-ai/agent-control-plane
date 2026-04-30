import { KeyRound } from "lucide-react";
import { useTranslation } from "react-i18next";
import { useProviders } from "@/api/providers";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ProviderTable } from "./ProviderTable";

interface WorkspaceProviderBindingsProps {
  tenantId?: string;
  workspaceId: string;
}

export function WorkspaceProviderBindings({ tenantId, workspaceId }: WorkspaceProviderBindingsProps) {
  const { t } = useTranslation();
  const { data, isLoading, isError, error, refetch } = useProviders(1, 10, tenantId, workspaceId);

  return (
    <section className="mt-6">
      <div className="mb-4 flex items-center gap-3">
        <div className="flex h-10 w-10 items-center justify-center rounded-md border border-teal-200 bg-teal-50 text-teal-700">
          <KeyRound className="h-5 w-5" aria-hidden="true" />
        </div>
        <div>
          <h2 className="text-base font-semibold text-zinc-950">
            {t("provider.workspaceBindingsTitle")}
          </h2>
          <p className="mt-1 text-sm text-zinc-500">{t("provider.workspaceBindingsSubtitle")}</p>
        </div>
      </div>

      {isLoading && <LoadingSkeleton />}

      {isError && (
        <ErrorAlert
          message={error instanceof Error ? error.message : t("provider.loadError")}
          onRetry={() => refetch()}
        />
      )}

      {data && data.providers.length === 0 && (
        <EmptyState
          title={t("provider.emptyWorkspaceTitle")}
          description={t("provider.emptyWorkspaceDescription")}
        />
      )}

      {data && data.providers.length > 0 && <ProviderTable providers={data.providers} />}
    </section>
  );
}
