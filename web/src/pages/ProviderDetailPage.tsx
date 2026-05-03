import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useProvider } from "@/api/providers";
import { ProviderDetailCard } from "@/components/providers/ProviderDetailCard";
import { PageHeader } from "@/components/shared/PageHeader";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ErrorAlert } from "@/components/shared/ErrorAlert";

export function ProviderDetailPage() {
  const { t } = useTranslation();
  const { tenantId: _tenantId, providerId } = useParams<{ tenantId: string; providerId: string }>();
  const { data: provider, isLoading, isError, error, refetch } = useProvider(providerId);
  useDocumentTitle(provider?.displayName);

  if (isLoading) return <LoadingSkeleton />;

  if (isError) {
    return (
      <ErrorAlert
        message={error instanceof Error ? error.message : t("provider.loadError")}
        onRetry={() => refetch()}
      />
    );
  }

  if (!provider) {
    return <ErrorAlert message={t("provider.notFound")} />;
  }

  return (
    <div>
      <PageHeader
        title={provider.displayName}
        subtitle={t("provider.detailSubtitle")}
      />
      <ProviderDetailCard provider={provider} />
    </div>
  );
}
