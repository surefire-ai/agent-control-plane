export interface Tenant {
  id: string;
  organizationId: string;
  slug: string;
  displayName: string;
  status: string;
  defaultRegion?: string;
}

export interface PaginatedTenantsResponse {
  tenants: Tenant[];
  page: number;
  limit: number;
  total: number;
}

export interface Workspace {
  id: string;
  tenantId: string;
  slug: string;
  displayName: string;
  description?: string;
  status: string;
  kubernetesNamespace?: string;
  kubernetesWorkspaceName?: string;
}

export interface PaginatedWorkspacesResponse {
  workspaces: Workspace[];
  page: number;
  limit: number;
  total: number;
}

export interface CreateWorkspaceRequest {
  id: string;
  tenantId: string;
  slug: string;
  displayName: string;
  description?: string;
  status?: string;
  kubernetesNamespace?: string;
  kubernetesWorkspaceName?: string;
}

export interface UpdateWorkspaceRequest {
  displayName?: string;
  description?: string;
  status?: string;
  kubernetesNamespace?: string;
  kubernetesWorkspaceName?: string;
}

export interface ApiErrorResponse {
  error: string;
}
