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

export interface RuntimeConfig {
  engine?: string;
  runnerClass?: string;
  mode?: string;
  entrypoint?: string;
}

export interface ModelConfig {
  provider?: string;
  model?: string;
  baseURL?: string;
  credentialRef?: string;
  temperature?: number;
  maxTokens?: number;
  timeoutSeconds?: number;
}

export interface IdentityConfig {
  displayName?: string;
  role?: string;
  description?: string;
}

export interface PatternRoute {
  label: string;
  agentRef?: string;
  modelRef?: string;
  default?: boolean;
}

export interface PatternConfig {
  type?: string;
  version?: string;
  modelRef?: string;
  executorModelRef?: string;
  toolRefs?: string[];
  knowledgeRefs?: string[];
  maxIterations?: number;
  stopWhen?: string;
  routes?: PatternRoute[];
}

export interface PromptRefsConfig {
  system?: string;
}

export interface KnowledgeBinding {
  name: string;
  ref: string;
  topK?: number;
  scoreThreshold?: number;
}

export interface SkillBinding {
  name: string;
  ref: string;
}

export interface SubAgentBinding {
  name: string;
  ref: string;
  namespace?: string;
}

export interface SchemaConfig {
  schema?: Record<string, unknown>;
}

export interface InterfaceConfig {
  input?: SchemaConfig;
  output?: SchemaConfig;
}

export interface GraphNode {
  name: string;
  kind: string;
  modelRef?: string;
  toolRef?: string;
  knowledgeRef?: string;
  agentRef?: string;
  implementation?: string;
  position?: { x: number; y: number };
}

export interface GraphEdge {
  from: string;
  to: string;
  when?: string;
}

export interface GraphConfig {
  nodes?: GraphNode[];
  edges?: GraphEdge[];
}

export interface AgentSpecData {
  runtime?: RuntimeConfig;
  models?: Record<string, ModelConfig>;
  identity?: IdentityConfig;
  pattern?: PatternConfig;
  promptRefs?: PromptRefsConfig;
  knowledgeRefs?: KnowledgeBinding[];
  toolRefs?: string[];
  skillRefs?: SkillBinding[];
  subAgentRefs?: SubAgentBinding[];
  mcpRefs?: string[];
  policyRef?: string;
  interfaces?: InterfaceConfig;
  graph?: GraphConfig;
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
  spec?: AgentSpecData;
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

export interface ProviderAccount {
  id: string;
  tenantId: string;
  workspaceId?: string;
  provider: string;
  displayName: string;
  family: string;
  baseUrl?: string;
  credentialRef?: string;
  status: string;
  domestic: boolean;
  supportsJsonSchema: boolean;
  supportsToolCalling: boolean;
}

export interface PaginatedProvidersResponse {
  providers: ProviderAccount[];
  page: number;
  limit: number;
  total: number;
}

export interface Run {
  id: string;
  tenantId: string;
  workspaceId: string;
  agentId: string;
  evaluationId?: string;
  agentRevision?: string;
  status: string;
  runtimeEngine: string;
  runnerClass: string;
  startedAt?: string;
  completedAt?: string;
  summary?: string;
  traceRef?: string;
}

export interface PaginatedRunsResponse {
  runs: Run[];
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
