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

export interface Agent {
  id: string;
  tenantId: string;
  workspaceId: string;
  slug: string;
  displayName: string;
  description?: string;
  status: string;
  pattern: string;
  runtimeEngine: string;
  runnerClass: string;
  modelProvider?: string;
  modelName?: string;
  latestRevision?: string;
}

export interface PaginatedAgentsResponse {
  agents: Agent[];
  page: number;
  limit: number;
  total: number;
}

export interface Evaluation {
  id: string;
  tenantId: string;
  workspaceId: string;
  agentId: string;
  slug: string;
  displayName: string;
  description?: string;
  status: string;
  datasetName: string;
  datasetRevision?: string;
  baselineRevision?: string;
  score: number;
  gatePassed: boolean;
  samplesTotal: number;
  samplesEvaluated: number;
  latestRunId?: string;
  reportRef?: string;
}

export interface PaginatedEvaluationsResponse {
  evaluations: Evaluation[];
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
