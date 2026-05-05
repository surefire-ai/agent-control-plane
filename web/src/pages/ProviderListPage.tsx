import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useProviders } from "@/api/providers";
import { ProviderTable } from "@/components/providers/ProviderTable";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { PageHeader } from "@/components/shared/PageHeader";
import { Pagination } from "@/components/shared/Pagination";

const LIMIT = 10;

export function ProviderListPage() {
  const { t } = useTranslation();
  useDocumentTitle(t("provider.title"));
  const { tenantId } = useParams<{ tenantId: string }>();
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error, refetch } = useProviders(page, LIMIT, tenantId);

  useEffect(() => { setPage(1); }, [tenantId]);

  return (
    <div>
      <PageHeader title={t("provider.title")} subtitle={t("provider.subtitle")} />

      {isLoading && <LoadingSkeleton />}

      {isError && (
        <ErrorAlert
          message={error instanceof Error ? error.message : t("provider.loadError")}
          onRetry={() => refetch()}
        />
      )}

      {data && data.total > 0 && (
        <div className="summary-strip mb-4">
          <span><strong>{data.total}</strong> {t("table.totalResults", "total")}</span>
        </div>
      )}

      {data && data.providers.length === 0 && (
        <EmptyState title={t("provider.emptyTitle")} description={t("provider.emptyDescription")} />
      )}

      {data && data.providers.length > 0 && (
        <>
          <ProviderTable providers={data.providers} />
          <Pagination page={page} limit={LIMIT} total={data.total} onPageChange={setPage} />
        </>
      )}
    </div>
  );
}
