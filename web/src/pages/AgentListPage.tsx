import { useState } from "react";
import { useParams } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useAgents } from "@/api/agents";
import { AgentTable } from "@/components/agents/AgentTable";
import { EmptyState } from "@/components/shared/EmptyState";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { PageHeader } from "@/components/shared/PageHeader";
import { Pagination } from "@/components/shared/Pagination";

const LIMIT = 10;

export function AgentListPage() {
  const { t } = useTranslation();
  const { tenantId } = useParams<{ tenantId: string }>();
  const [page, setPage] = useState(1);
  const { data, isLoading, isError, error, refetch } = useAgents(page, LIMIT, tenantId);

  return (
    <div>
      <PageHeader title={t("agent.title")} subtitle={t("agent.subtitle")} />

      {isLoading && <LoadingSkeleton />}

      {isError && (
        <ErrorAlert
          message={error instanceof Error ? error.message : t("agent.loadError")}
          onRetry={() => refetch()}
        />
      )}

      {data && data.agents.length === 0 && (
        <EmptyState title={t("agent.emptyTitle")} description={t("agent.emptyDescription")} />
      )}

      {data && data.agents.length > 0 && (
        <>
          <AgentTable agents={data.agents} />
          <Pagination page={page} limit={LIMIT} total={data.total} onPageChange={setPage} />
        </>
      )}
    </div>
  );
}
