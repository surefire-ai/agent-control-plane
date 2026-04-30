import { useQuery } from "@tanstack/react-query";
import type { PaginatedProvidersResponse, ProviderAccount } from "@/types/api";
import { api } from "./client";

export function useProviders(page: number, limit: number, tenantId?: string, workspaceId?: string) {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (tenantId) params.set("tenantId", tenantId);
  if (workspaceId) params.set("workspaceId", workspaceId);

  return useQuery({
    queryKey: ["providers", page, limit, tenantId, workspaceId],
    queryFn: () => api.get<PaginatedProvidersResponse>(`/providers/?${params.toString()}`),
  });
}

export function useProvider(id: string | undefined) {
  return useQuery({
    queryKey: ["providers", id],
    queryFn: () => api.get<ProviderAccount>(`/providers/${id}`),
    enabled: !!id,
  });
}
