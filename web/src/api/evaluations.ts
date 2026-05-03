import { useQuery } from "@tanstack/react-query";
import type { Evaluation, PaginatedEvaluationsResponse } from "@/types/api";
import { api } from "./client";

export function useEvaluations(page: number, limit: number, tenantId?: string, workspaceId?: string, agentId?: string) {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (tenantId) params.set("tenantId", tenantId);
  if (workspaceId) params.set("workspaceId", workspaceId);
  if (agentId) params.set("agentId", agentId);

  return useQuery({
    queryKey: ["evaluations", page, limit, tenantId, workspaceId, agentId],
    queryFn: () => api.get<PaginatedEvaluationsResponse>(`/evaluations/?${params.toString()}`),
  });
}

export function useEvaluation(id: string | undefined) {
  return useQuery({
    queryKey: ["evaluations", id],
    queryFn: () => api.get<Evaluation>(`/evaluations/${id}`),
    enabled: !!id,
  });
}
