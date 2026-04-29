import { useQuery } from "@tanstack/react-query";
import type { PaginatedTenantsResponse, Tenant } from "@/types/api";
import { api } from "./client";

export function useTenants(page: number, limit: number) {
  return useQuery({
    queryKey: ["tenants", page, limit],
    queryFn: () => api.get<PaginatedTenantsResponse>(`/tenants/?page=${page}&limit=${limit}`),
  });
}

export function useTenant(id: string | undefined) {
  return useQuery({
    queryKey: ["tenants", id],
    queryFn: () => api.get<Tenant>(`/tenants/${id}`),
    enabled: !!id,
  });
}
