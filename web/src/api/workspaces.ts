import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import type {
  CreateWorkspaceRequest,
  PaginatedWorkspacesResponse,
  UpdateWorkspaceRequest,
  Workspace,
} from "@/types/api";
import { api } from "./client";

export function useWorkspaces(page: number, limit: number, tenantId?: string) {
  const params = new URLSearchParams({ page: String(page), limit: String(limit) });
  if (tenantId) params.set("tenantId", tenantId);

  return useQuery({
    queryKey: ["workspaces", page, limit, tenantId],
    queryFn: () => api.get<PaginatedWorkspacesResponse>(`/workspaces/?${params.toString()}`),
  });
}

export function useWorkspace(id: string | undefined) {
  return useQuery({
    queryKey: ["workspaces", id],
    queryFn: () => api.get<Workspace>(`/workspaces/${id}`),
    enabled: !!id,
  });
}

export function useCreateWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (data: CreateWorkspaceRequest) =>
      api.post<Workspace>("/workspaces/", data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workspaces"] });
    },
  });
}

export function useUpdateWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: ({ id, ...data }: { id: string } & UpdateWorkspaceRequest) =>
      api.patch<Workspace>(`/workspaces/${id}`, data),
    onSuccess: (_data, variables) => {
      queryClient.invalidateQueries({ queryKey: ["workspaces"] });
      queryClient.invalidateQueries({ queryKey: ["workspaces", variables.id] });
    },
  });
}

export function useDeleteWorkspace() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (id: string) => api.delete<void>(`/workspaces/${id}`),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["workspaces"] });
    },
  });
}
