import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import type { Agent, PaginatedAgentsResponse } from "@/types/api";
import { api } from "./client";

export function useAgents(page: number, limit: number, tenantId?: string, workspaceId?: string) {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (tenantId) params.set("tenantId", tenantId);
  if (workspaceId) params.set("workspaceId", workspaceId);

  return useQuery({
    queryKey: ["agents", page, limit, tenantId, workspaceId],
    queryFn: () => api.get<PaginatedAgentsResponse>(`/agents/?${params.toString()}`),
  });
}

export function useAgent(id: string | undefined) {
  return useQuery({
    queryKey: ["agents", id],
    queryFn: () => api.get<Agent>(`/agents/${id}`),
    enabled: !!id,
  });
}

export function useUpdateAgent() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: ({ id, ...body }: { id: string } & Partial<Agent>) =>
      api.patch<Agent>(`/agents/${id}`, body),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["agents"] });
    },
  });
}
