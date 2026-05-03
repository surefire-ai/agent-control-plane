import { useQuery } from "@tanstack/react-query";
import type { PaginatedRunsResponse, Run } from "@/types/api";
import { api } from "./client";

export function useRuns(page: number, limit: number, tenantId?: string, workspaceId?: string, agentId?: string) {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (tenantId) params.set("tenantId", tenantId);
  if (workspaceId) params.set("workspaceId", workspaceId);
  if (agentId) params.set("agentId", agentId);

  return useQuery({
    queryKey: ["runs", page, limit, tenantId, workspaceId, agentId],
    queryFn: () => api.get<PaginatedRunsResponse>(`/runs/?${params.toString()}`),
  });
}

export function useRun(id: string | undefined) {
  return useQuery({
    queryKey: ["runs", id],
    queryFn: () => api.get<Run>(`/runs/${id}`),
    enabled: !!id,
  });
}
