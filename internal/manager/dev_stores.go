package manager

import (
	"context"
	"fmt"
	"strconv"
)

type devWorkspaceStore struct {
	records    map[string]WorkspaceRecord
	orderedIDs []string
}

func (s devWorkspaceStore) GetWorkspace(_ context.Context, id string) (*WorkspaceRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devWorkspaceStore) ListWorkspaces(_ context.Context, page, limit int) ([]WorkspaceRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []WorkspaceRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]WorkspaceRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func (s devWorkspaceStore) ListWorkspacesByTenant(_ context.Context, tenantID string, page, limit int) ([]WorkspaceRecord, int, error) {
	filtered := make([]WorkspaceRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	total := len(filtered)
	start := (page - 1) * limit
	if start >= total {
		return []WorkspaceRecord{}, total, nil
	}
	end := min(start+limit, total)
	return filtered[start:end], total, nil
}

func (s *devWorkspaceStore) CreateWorkspace(_ context.Context, workspace WorkspaceRecord) error {
	if _, exists := s.records[workspace.ID]; exists {
		return ErrConflict
	}
	s.records[workspace.ID] = workspace
	s.orderedIDs = append(s.orderedIDs, workspace.ID)
	return nil
}

func (s *devWorkspaceStore) UpdateWorkspace(_ context.Context, id string, fields map[string]string) (*WorkspaceRecord, error) {
	rec, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if v, ok := fields["display_name"]; ok {
		rec.DisplayName = v
	}
	if v, ok := fields["description"]; ok {
		rec.Description = v
	}
	if v, ok := fields["status"]; ok {
		rec.Status = v
	}
	if v, ok := fields["kubernetes_namespace"]; ok {
		rec.KubernetesNamespace = v
	}
	if v, ok := fields["kubernetes_workspace_name"]; ok {
		rec.KubernetesWorkspaceName = v
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *devWorkspaceStore) DeleteWorkspace(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type devTenantStore struct {
	records    map[string]TenantRecord
	orderedIDs []string
}

func (s devTenantStore) GetTenant(_ context.Context, id string) (*TenantRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devTenantStore) ListTenants(_ context.Context, page, limit int) ([]TenantRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []TenantRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]TenantRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func (s *devTenantStore) CreateTenant(_ context.Context, tenant TenantRecord) error {
	if _, exists := s.records[tenant.ID]; exists {
		return ErrConflict
	}
	s.records[tenant.ID] = tenant
	s.orderedIDs = append(s.orderedIDs, tenant.ID)
	return nil
}

func (s *devTenantStore) UpdateTenant(_ context.Context, id string, fields map[string]string) (*TenantRecord, error) {
	rec, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if v, ok := fields["display_name"]; ok {
		rec.DisplayName = v
	}
	if v, ok := fields["status"]; ok {
		rec.Status = v
	}
	if v, ok := fields["default_region"]; ok {
		rec.DefaultRegion = v
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *devTenantStore) DeleteTenant(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type devAgentStore struct {
	records    map[string]AgentRecord
	orderedIDs []string
}

func (s devAgentStore) GetAgent(_ context.Context, id string) (*AgentRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devAgentStore) ListAgents(_ context.Context, page, limit int) ([]AgentRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []AgentRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]AgentRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func (s devAgentStore) ListAgentsByTenant(_ context.Context, tenantID string, page, limit int) ([]AgentRecord, int, error) {
	filtered := make([]AgentRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateAgents(filtered, page, limit)
}

func (s devAgentStore) ListAgentsByWorkspace(_ context.Context, workspaceID string, page, limit int) ([]AgentRecord, int, error) {
	filtered := make([]AgentRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateAgents(filtered, page, limit)
}

func paginateAgents(records []AgentRecord, page, limit int) ([]AgentRecord, int, error) {
	total := len(records)
	start := (page - 1) * limit
	if start >= total {
		return []AgentRecord{}, total, nil
	}
	end := min(start+limit, total)
	return records[start:end], total, nil
}

func (s *devAgentStore) CreateAgent(_ context.Context, agent AgentRecord) error {
	if _, exists := s.records[agent.ID]; exists {
		return ErrConflict
	}
	s.records[agent.ID] = agent
	s.orderedIDs = append(s.orderedIDs, agent.ID)
	return nil
}

func (s *devAgentStore) UpdateAgent(_ context.Context, id string, fields map[string]string) (*AgentRecord, error) {
	rec, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if v, ok := fields["display_name"]; ok {
		rec.DisplayName = v
	}
	if v, ok := fields["description"]; ok {
		rec.Description = v
	}
	if v, ok := fields["status"]; ok {
		rec.Status = v
	}
	if v, ok := fields["pattern"]; ok {
		rec.Pattern = v
	}
	if v, ok := fields["runtime_engine"]; ok {
		rec.RuntimeEngine = v
	}
	if v, ok := fields["runner_class"]; ok {
		rec.RunnerClass = v
	}
	if v, ok := fields["model_provider"]; ok {
		rec.ModelProvider = v
	}
	if v, ok := fields["model_name"]; ok {
		rec.ModelName = v
	}
	if v, ok := fields["latest_revision"]; ok {
		rec.LatestRevision = v
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *devAgentStore) DeleteAgent(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type devEvaluationStore struct {
	records    map[string]EvaluationRecord
	orderedIDs []string
}

func (s devEvaluationStore) GetEvaluation(_ context.Context, id string) (*EvaluationRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devEvaluationStore) ListEvaluations(_ context.Context, page, limit int) ([]EvaluationRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []EvaluationRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]EvaluationRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func (s devEvaluationStore) ListEvaluationsByTenant(_ context.Context, tenantID string, page, limit int) ([]EvaluationRecord, int, error) {
	filtered := make([]EvaluationRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateEvaluations(filtered, page, limit)
}

func (s devEvaluationStore) ListEvaluationsByWorkspace(_ context.Context, workspaceID string, page, limit int) ([]EvaluationRecord, int, error) {
	filtered := make([]EvaluationRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateEvaluations(filtered, page, limit)
}

func (s devEvaluationStore) ListEvaluationsByAgent(_ context.Context, agentID string, page, limit int) ([]EvaluationRecord, int, error) {
	filtered := make([]EvaluationRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].AgentID == agentID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateEvaluations(filtered, page, limit)
}

func paginateEvaluations(records []EvaluationRecord, page, limit int) ([]EvaluationRecord, int, error) {
	total := len(records)
	start := (page - 1) * limit
	if start >= total {
		return []EvaluationRecord{}, total, nil
	}
	end := min(start+limit, total)
	return records[start:end], total, nil
}

func (s *devEvaluationStore) CreateEvaluation(_ context.Context, eval EvaluationRecord) error {
	if _, exists := s.records[eval.ID]; exists {
		return ErrConflict
	}
	s.records[eval.ID] = eval
	s.orderedIDs = append(s.orderedIDs, eval.ID)
	return nil
}

func (s *devEvaluationStore) UpdateEvaluation(_ context.Context, id string, fields map[string]string) (*EvaluationRecord, error) {
	rec, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if v, ok := fields["display_name"]; ok {
		rec.DisplayName = v
	}
	if v, ok := fields["description"]; ok {
		rec.Description = v
	}
	if v, ok := fields["status"]; ok {
		rec.Status = v
	}
	if v, ok := fields["dataset_name"]; ok {
		rec.DatasetName = v
	}
	if v, ok := fields["dataset_revision"]; ok {
		rec.DatasetRevision = v
	}
	if v, ok := fields["baseline_revision"]; ok {
		rec.BaselineRevision = v
	}
	if v, ok := fields["score"]; ok {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			rec.Score = f
		} else {
			return nil, fmt.Errorf("invalid score %q: %w", v, err)
		}
	}
	if v, ok := fields["gate_passed"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			rec.GatePassed = b
		} else {
			return nil, fmt.Errorf("invalid gate_passed %q: %w", v, err)
		}
	}
	if v, ok := fields["samples_total"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			rec.SamplesTotal = n
		} else {
			return nil, fmt.Errorf("invalid samples_total %q: %w", v, err)
		}
	}
	if v, ok := fields["samples_evaluated"]; ok {
		if n, err := strconv.Atoi(v); err == nil {
			rec.SamplesEvaluated = n
		} else {
			return nil, fmt.Errorf("invalid samples_evaluated %q: %w", v, err)
		}
	}
	if v, ok := fields["latest_run_id"]; ok {
		rec.LatestRunID = v
	}
	if v, ok := fields["report_ref"]; ok {
		rec.ReportRef = v
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *devEvaluationStore) DeleteEvaluation(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type devProviderStore struct {
	records    map[string]ProviderRecord
	orderedIDs []string
}

func (s devProviderStore) GetProvider(_ context.Context, id string) (*ProviderRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devProviderStore) ListProviders(_ context.Context, page, limit int) ([]ProviderRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []ProviderRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]ProviderRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func (s devProviderStore) ListProvidersByTenant(_ context.Context, tenantID string, page, limit int) ([]ProviderRecord, int, error) {
	filtered := make([]ProviderRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateProviders(filtered, page, limit)
}

func (s devProviderStore) ListProvidersByWorkspace(_ context.Context, workspaceID string, page, limit int) ([]ProviderRecord, int, error) {
	filtered := make([]ProviderRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateProviders(filtered, page, limit)
}

func paginateProviders(records []ProviderRecord, page, limit int) ([]ProviderRecord, int, error) {
	total := len(records)
	start := (page - 1) * limit
	if start >= total {
		return []ProviderRecord{}, total, nil
	}
	end := min(start+limit, total)
	return records[start:end], total, nil
}

func (s *devProviderStore) CreateProvider(_ context.Context, provider ProviderRecord) error {
	if _, exists := s.records[provider.ID]; exists {
		return ErrConflict
	}
	s.records[provider.ID] = provider
	s.orderedIDs = append(s.orderedIDs, provider.ID)
	return nil
}

func (s *devProviderStore) UpdateProvider(_ context.Context, id string, fields map[string]string) (*ProviderRecord, error) {
	rec, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if v, ok := fields["display_name"]; ok {
		rec.DisplayName = v
	}
	if v, ok := fields["family"]; ok {
		rec.Family = v
	}
	if v, ok := fields["base_url"]; ok {
		rec.BaseURL = v
	}
	if v, ok := fields["credential_ref"]; ok {
		rec.CredentialRef = v
	}
	if v, ok := fields["status"]; ok {
		rec.Status = v
	}
	if v, ok := fields["domestic"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			rec.Domestic = b
		} else {
			return nil, fmt.Errorf("invalid domestic %q: %w", v, err)
		}
	}
	if v, ok := fields["supports_json_schema"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			rec.SupportsJSONSchema = b
		} else {
			return nil, fmt.Errorf("invalid supports_json_schema %q: %w", v, err)
		}
	}
	if v, ok := fields["supports_tool_calling"]; ok {
		if b, err := strconv.ParseBool(v); err == nil {
			rec.SupportsToolCalling = b
		} else {
			return nil, fmt.Errorf("invalid supports_tool_calling %q: %w", v, err)
		}
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *devProviderStore) DeleteProvider(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type devRunStore struct {
	records    map[string]RunRecord
	orderedIDs []string
}

func (s devRunStore) GetRun(_ context.Context, id string) (*RunRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devRunStore) ListRuns(_ context.Context, page, limit int) ([]RunRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []RunRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]RunRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func (s devRunStore) ListRunsByTenant(_ context.Context, tenantID string, page, limit int) ([]RunRecord, int, error) {
	filtered := make([]RunRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateRuns(filtered, page, limit)
}

func (s devRunStore) ListRunsByWorkspace(_ context.Context, workspaceID string, page, limit int) ([]RunRecord, int, error) {
	filtered := make([]RunRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateRuns(filtered, page, limit)
}

func (s devRunStore) ListRunsByAgent(_ context.Context, agentID string, page, limit int) ([]RunRecord, int, error) {
	filtered := make([]RunRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].AgentID == agentID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateRuns(filtered, page, limit)
}

func paginateRuns(records []RunRecord, page, limit int) ([]RunRecord, int, error) {
	total := len(records)
	start := (page - 1) * limit
	if start >= total {
		return []RunRecord{}, total, nil
	}
	end := min(start+limit, total)
	return records[start:end], total, nil
}

func (s *devRunStore) CreateRun(_ context.Context, run RunRecord) error {
	if _, exists := s.records[run.ID]; exists {
		return ErrConflict
	}
	s.records[run.ID] = run
	s.orderedIDs = append(s.orderedIDs, run.ID)
	return nil
}

func (s *devRunStore) UpdateRun(_ context.Context, id string, fields map[string]string) (*RunRecord, error) {
	rec, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if v, ok := fields["status"]; ok {
		rec.Status = v
	}
	if v, ok := fields["started_at"]; ok {
		rec.StartedAt = v
	}
	if v, ok := fields["completed_at"]; ok {
		rec.CompletedAt = v
	}
	if v, ok := fields["summary"]; ok {
		rec.Summary = v
	}
	if v, ok := fields["trace_ref"]; ok {
		rec.TraceRef = v
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *devRunStore) DeleteRun(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

func NewFakeStores() Stores {
	workspaces := &devWorkspaceStore{
		records: map[string]WorkspaceRecord{
			"ws_demo":       {ID: "ws_demo", TenantID: "t_demo", Slug: "demo-ws", DisplayName: "Demo Workspace", Description: "A demo workspace for development", Status: "active", KubernetesNamespace: "demo", KubernetesWorkspaceName: "workspace-demo"},
			"ws_staging":    {ID: "ws_staging", TenantID: "t_demo", Slug: "staging-ws", DisplayName: "Staging Workspace", Status: "active", KubernetesNamespace: "staging"},
			"ws_enterprise": {ID: "ws_enterprise", TenantID: "t_enterprise", Slug: "enterprise-ws", DisplayName: "Enterprise Workspace", Description: "Enterprise customer workspace", Status: "active", KubernetesNamespace: "enterprise", KubernetesWorkspaceName: "workspace-enterprise"},
		},
		orderedIDs: []string{"ws_demo", "ws_staging", "ws_enterprise"},
	}
	tenants := &devTenantStore{
		records: map[string]TenantRecord{
			"t_demo":       {ID: "t_demo", OrganizationID: "org_1", Slug: "demo-tenant", DisplayName: "Demo Tenant", Status: "active", DefaultRegion: "us-east-1"},
			"t_enterprise": {ID: "t_enterprise", OrganizationID: "org_1", Slug: "enterprise-tenant", DisplayName: "Enterprise Tenant", Status: "active", DefaultRegion: "eu-west-1"},
			"t_inactive":   {ID: "t_inactive", OrganizationID: "org_2", Slug: "inactive-tenant", DisplayName: "Inactive Tenant", Status: "inactive"},
		},
		orderedIDs: []string{"t_demo", "t_enterprise", "t_inactive"},
	}
	agents := &devAgentStore{
		records: map[string]AgentRecord{
			"agent_ehs_react": {
				ID: "agent_ehs_react", TenantID: "t_demo", WorkspaceID: "ws_demo", Slug: "ehs-react",
				DisplayName: "EHS ReAct Agent", Description: "Safety incident triage with tools and knowledge", Status: "published",
				Pattern: "react", RuntimeEngine: "eino", RunnerClass: "adk", ModelProvider: "openai", ModelName: "gpt-4.1-mini", LatestRevision: "rev-20260429-001",
			},
			"agent_eval_guard": {
				ID: "agent_eval_guard", TenantID: "t_demo", WorkspaceID: "ws_staging", Slug: "eval-guard",
				DisplayName: "Evaluation Guard", Description: "Release gate evaluator for regression checks", Status: "draft",
				Pattern: "react", RuntimeEngine: "eino", RunnerClass: "adk", ModelProvider: "qwen", ModelName: "qwen-plus", LatestRevision: "rev-20260429-002",
			},
			"agent_enterprise_ops": {
				ID: "agent_enterprise_ops", TenantID: "t_enterprise", WorkspaceID: "ws_enterprise", Slug: "enterprise-ops",
				DisplayName: "Enterprise Ops Agent", Description: "Operations assistant for enterprise workflows", Status: "published",
				Pattern: "react", RuntimeEngine: "eino", RunnerClass: "adk", ModelProvider: "deepseek", ModelName: "deepseek-chat", LatestRevision: "rev-20260429-003",
			},
		},
		orderedIDs: []string{"agent_ehs_react", "agent_eval_guard", "agent_enterprise_ops"},
	}
	evaluations := &devEvaluationStore{
		records: map[string]EvaluationRecord{
			"eval_ehs_regression": {
				ID: "eval_ehs_regression", TenantID: "t_demo", WorkspaceID: "ws_demo", AgentID: "agent_ehs_react",
				Slug: "ehs-regression", DisplayName: "EHS Regression Gate", Description: "Pre-release regression suite for safety incident triage",
				Status: "passed", DatasetName: "ehs-golden-set", DatasetRevision: "dataset-rev-12", BaselineRevision: "rev-20260420-007",
				Score: 0.94, GatePassed: true, SamplesTotal: 128, SamplesEvaluated: 128, LatestRunID: "evalrun-20260429-001", ReportRef: "s3://reports/ehs-regression/latest.json",
			},
			"eval_guardrail_release": {
				ID: "eval_guardrail_release", TenantID: "t_demo", WorkspaceID: "ws_staging", AgentID: "agent_eval_guard",
				Slug: "guardrail-release", DisplayName: "Guardrail Release Check", Description: "Blocking gate for release candidate risk checks",
				Status: "failed", DatasetName: "risk-gate-set", DatasetRevision: "dataset-rev-4", BaselineRevision: "rev-20260421-002",
				Score: 0.72, GatePassed: false, SamplesTotal: 64, SamplesEvaluated: 64, LatestRunID: "evalrun-20260429-002", ReportRef: "s3://reports/guardrail-release/latest.json",
			},
			"eval_enterprise_ops": {
				ID: "eval_enterprise_ops", TenantID: "t_enterprise", WorkspaceID: "ws_enterprise", AgentID: "agent_enterprise_ops",
				Slug: "enterprise-ops-weekly", DisplayName: "Enterprise Ops Weekly", Description: "Weekly regression monitor for enterprise operations",
				Status: "running", DatasetName: "enterprise-ops-set", DatasetRevision: "dataset-rev-9", BaselineRevision: "rev-20260422-001",
				Score: 0.88, GatePassed: true, SamplesTotal: 240, SamplesEvaluated: 180, LatestRunID: "evalrun-20260429-003", ReportRef: "",
			},
		},
		orderedIDs: []string{"eval_ehs_regression", "eval_guardrail_release", "eval_enterprise_ops"},
	}
	providers := &devProviderStore{
		records: map[string]ProviderRecord{
			"provider_qwen_prod": {
				ID: "provider_qwen_prod", TenantID: "t_demo", WorkspaceID: "ws_demo",
				Provider: "qwen", DisplayName: "Qwen Production", Family: "openai-compatible",
				BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", CredentialRef: "secret://demo/qwen-api-key",
				Status: "active", Domestic: true, SupportsJSONSchema: true, SupportsToolCalling: true,
			},
			"provider_deepseek_gate": {
				ID: "provider_deepseek_gate", TenantID: "t_demo", WorkspaceID: "ws_staging",
				Provider: "deepseek", DisplayName: "DeepSeek Release Gate", Family: "openai-compatible",
				BaseURL: "https://api.deepseek.com/v1", CredentialRef: "secret://staging/deepseek-api-key",
				Status: "active", Domestic: true, SupportsJSONSchema: true, SupportsToolCalling: true,
			},
			"provider_openai_fallback": {
				ID: "provider_openai_fallback", TenantID: "t_enterprise", WorkspaceID: "ws_enterprise",
				Provider: "openai", DisplayName: "OpenAI Fallback", Family: "openai-compatible",
				BaseURL: "https://api.openai.com/v1", CredentialRef: "secret://enterprise/openai-api-key",
				Status: "active", SupportsJSONSchema: true, SupportsToolCalling: true,
			},
		},
		orderedIDs: []string{"provider_qwen_prod", "provider_deepseek_gate", "provider_openai_fallback"},
	}
	runs := &devRunStore{
		records: map[string]RunRecord{
			"run_ehs_20260429_001": {
				ID: "run_ehs_20260429_001", TenantID: "t_demo", WorkspaceID: "ws_demo", AgentID: "agent_ehs_react",
				AgentRevision: "rev-20260429-001", Status: "succeeded", RuntimeEngine: "eino", RunnerClass: "adk",
				StartedAt: "2026-04-29T09:10:00Z", CompletedAt: "2026-04-29T09:10:14Z",
				Summary: "inspection complete", TraceRef: "pod/run-ehs-20260429-001",
			},
			"run_guardrail_20260429_002": {
				ID: "run_guardrail_20260429_002", TenantID: "t_demo", WorkspaceID: "ws_staging", AgentID: "agent_eval_guard",
				EvaluationID: "eval_guardrail_release", AgentRevision: "rev-20260429-002", Status: "failed",
				RuntimeEngine: "eino", RunnerClass: "adk", StartedAt: "2026-04-29T10:25:00Z",
				CompletedAt: "2026-04-29T10:25:31Z", Summary: "release gate failed", TraceRef: "pod/run-guardrail-20260429-002",
			},
			"run_enterprise_20260429_003": {
				ID: "run_enterprise_20260429_003", TenantID: "t_enterprise", WorkspaceID: "ws_enterprise", AgentID: "agent_enterprise_ops",
				EvaluationID: "eval_enterprise_ops", AgentRevision: "rev-20260429-003", Status: "running",
				RuntimeEngine: "eino", RunnerClass: "adk", StartedAt: "2026-04-29T11:40:00Z",
				Summary: "weekly regression in progress", TraceRef: "pod/run-enterprise-20260429-003",
			},
		},
		orderedIDs: []string{"run_ehs_20260429_001", "run_guardrail_20260429_002", "run_enterprise_20260429_003"},
	}
	return Stores{
		Workspaces:  workspaces,
		Tenants:     tenants,
		Agents:      agents,
		Evaluations: evaluations,
		Providers:   providers,
		Runs:        runs,
	}
}
