import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useTenants } from "@/api/tenants";
import { TenantTable } from "@/components/tenants/TenantTable";
import { PageHeader } from "@/components/shared/PageHeader";
import { Pagination } from "@/components/shared/Pagination";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { EmptyState } from "@/components/shared/EmptyState";

const LIMIT = 10;

export function TenantListPage() {
  const { t } = useTranslation();
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error, refetch } = useTenants(page, LIMIT);

  return (
    <div>
      <PageHeader title={t("tenant.title")} subtitle={t("tenant.subtitle")} />

      {isLoading && <LoadingSkeleton />}

      {isError && (
        <ErrorAlert
          message={error instanceof Error ? error.message : t("tenant.loadError")}
          onRetry={() => refetch()}
        />
      )}

      {data && data.tenants.length === 0 && (
        <EmptyState title={t("tenant.emptyTitle")} description={t("tenant.emptyDescription")} />
      )}

      {data && data.tenants.length > 0 && (
        <>
          <TenantTable tenants={data.tenants} />
          <Pagination page={page} limit={LIMIT} total={data.total} onPageChange={setPage} />
        </>
      )}
    </div>
  );
}
