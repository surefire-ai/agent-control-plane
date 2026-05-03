package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type fakeWorkspaceStore struct {
	records    map[string]WorkspaceRecord
	orderedIDs []string
}

func (s fakeWorkspaceStore) GetWorkspace(ctx context.Context, id string) (*WorkspaceRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s fakeWorkspaceStore) ListWorkspaces(ctx context.Context, page, limit int) ([]WorkspaceRecord, int, error) {
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

func (s fakeWorkspaceStore) ListWorkspacesByTenant(ctx context.Context, tenantID string, page, limit int) ([]WorkspaceRecord, int, error) {
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

func (s *fakeWorkspaceStore) CreateWorkspace(ctx context.Context, workspace WorkspaceRecord) error {
	if _, exists := s.records[workspace.ID]; exists {
		return ErrConflict
	}
	s.records[workspace.ID] = workspace
	s.orderedIDs = append(s.orderedIDs, workspace.ID)
	return nil
}

func (s *fakeWorkspaceStore) UpdateWorkspace(ctx context.Context, id string, fields map[string]string) (*WorkspaceRecord, error) {
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

func (s *fakeWorkspaceStore) DeleteWorkspace(ctx context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type fakeTenantStore struct {
	records    map[string]TenantRecord
	orderedIDs []string
}

func (s fakeTenantStore) GetTenant(ctx context.Context, id string) (*TenantRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s fakeTenantStore) ListTenants(ctx context.Context, page, limit int) ([]TenantRecord, int, error) {
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

func (s *fakeTenantStore) CreateTenant(_ context.Context, tenant TenantRecord) error {
	if _, exists := s.records[tenant.ID]; exists {
		return ErrConflict
	}
	s.records[tenant.ID] = tenant
	s.orderedIDs = append(s.orderedIDs, tenant.ID)
	return nil
}

func (s *fakeTenantStore) UpdateTenant(_ context.Context, id string, fields map[string]string) (*TenantRecord, error) {
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

func (s *fakeTenantStore) DeleteTenant(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type fakeAgentStore struct {
	records    map[string]AgentRecord
	orderedIDs []string
}

func (s fakeAgentStore) GetAgent(ctx context.Context, id string) (*AgentRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s fakeAgentStore) ListAgents(ctx context.Context, page, limit int) ([]AgentRecord, int, error) {
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

func (s fakeAgentStore) ListAgentsByTenant(ctx context.Context, tenantID string, page, limit int) ([]AgentRecord, int, error) {
	filtered := make([]AgentRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestAgents(filtered, page, limit), len(filtered), nil
}

func (s fakeAgentStore) ListAgentsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]AgentRecord, int, error) {
	filtered := make([]AgentRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestAgents(filtered, page, limit), len(filtered), nil
}

func paginateTestAgents(records []AgentRecord, page, limit int) []AgentRecord {
	start := (page - 1) * limit
	if start >= len(records) {
		return []AgentRecord{}
	}
	end := min(start+limit, len(records))
	return records[start:end]
}

func (s *fakeAgentStore) CreateAgent(_ context.Context, agent AgentRecord) error {
	if _, exists := s.records[agent.ID]; exists {
		return ErrConflict
	}
	s.records[agent.ID] = agent
	s.orderedIDs = append(s.orderedIDs, agent.ID)
	return nil
}

func (s *fakeAgentStore) UpdateAgent(_ context.Context, id string, fields map[string]string) (*AgentRecord, error) {
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

func (s *fakeAgentStore) DeleteAgent(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type fakeEvaluationStore struct {
	records    map[string]EvaluationRecord
	orderedIDs []string
}

func (s fakeEvaluationStore) GetEvaluation(ctx context.Context, id string) (*EvaluationRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s fakeEvaluationStore) ListEvaluations(ctx context.Context, page, limit int) ([]EvaluationRecord, int, error) {
	return paginateTestEvaluationsFromIDs(s.records, s.orderedIDs, page, limit), len(s.records), nil
}

func (s fakeEvaluationStore) ListEvaluationsByTenant(ctx context.Context, tenantID string, page, limit int) ([]EvaluationRecord, int, error) {
	filtered := make([]EvaluationRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestEvaluations(filtered, page, limit), len(filtered), nil
}

func (s fakeEvaluationStore) ListEvaluationsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]EvaluationRecord, int, error) {
	filtered := make([]EvaluationRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestEvaluations(filtered, page, limit), len(filtered), nil
}

func (s fakeEvaluationStore) ListEvaluationsByAgent(ctx context.Context, agentID string, page, limit int) ([]EvaluationRecord, int, error) {
	filtered := make([]EvaluationRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].AgentID == agentID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestEvaluations(filtered, page, limit), len(filtered), nil
}

func paginateTestEvaluationsFromIDs(records map[string]EvaluationRecord, orderedIDs []string, page, limit int) []EvaluationRecord {
	all := make([]EvaluationRecord, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		all = append(all, records[id])
	}
	return paginateTestEvaluations(all, page, limit)
}

func paginateTestEvaluations(records []EvaluationRecord, page, limit int) []EvaluationRecord {
	start := (page - 1) * limit
	if start >= len(records) {
		return []EvaluationRecord{}
	}
	end := min(start+limit, len(records))
	return records[start:end]
}

func (s *fakeEvaluationStore) CreateEvaluation(_ context.Context, evaluation EvaluationRecord) error {
	if _, exists := s.records[evaluation.ID]; exists {
		return ErrConflict
	}
	s.records[evaluation.ID] = evaluation
	s.orderedIDs = append(s.orderedIDs, evaluation.ID)
	return nil
}

func (s *fakeEvaluationStore) UpdateEvaluation(_ context.Context, id string, fields map[string]string) (*EvaluationRecord, error) {
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
	if v, ok := fields["latest_run_id"]; ok {
		rec.LatestRunID = v
	}
	if v, ok := fields["report_ref"]; ok {
		rec.ReportRef = v
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *fakeEvaluationStore) DeleteEvaluation(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type fakeProviderStore struct {
	records    map[string]ProviderRecord
	orderedIDs []string
}

func (s fakeProviderStore) GetProvider(ctx context.Context, id string) (*ProviderRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s fakeProviderStore) ListProviders(ctx context.Context, page, limit int) ([]ProviderRecord, int, error) {
	return paginateTestProvidersFromIDs(s.records, s.orderedIDs, page, limit), len(s.records), nil
}

func (s fakeProviderStore) ListProvidersByTenant(ctx context.Context, tenantID string, page, limit int) ([]ProviderRecord, int, error) {
	filtered := make([]ProviderRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestProviders(filtered, page, limit), len(filtered), nil
}

func (s fakeProviderStore) ListProvidersByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]ProviderRecord, int, error) {
	filtered := make([]ProviderRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestProviders(filtered, page, limit), len(filtered), nil
}

func paginateTestProvidersFromIDs(records map[string]ProviderRecord, orderedIDs []string, page, limit int) []ProviderRecord {
	all := make([]ProviderRecord, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		all = append(all, records[id])
	}
	return paginateTestProviders(all, page, limit)
}

func paginateTestProviders(records []ProviderRecord, page, limit int) []ProviderRecord {
	start := (page - 1) * limit
	if start >= len(records) {
		return []ProviderRecord{}
	}
	end := min(start+limit, len(records))
	return records[start:end]
}

func (s *fakeProviderStore) CreateProvider(_ context.Context, provider ProviderRecord) error {
	if _, exists := s.records[provider.ID]; exists {
		return ErrConflict
	}
	s.records[provider.ID] = provider
	s.orderedIDs = append(s.orderedIDs, provider.ID)
	return nil
}

func (s *fakeProviderStore) UpdateProvider(_ context.Context, id string, fields map[string]string) (*ProviderRecord, error) {
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
		if v == "true" {
			rec.Domestic = true
		}
	}
	if v, ok := fields["supports_json_schema"]; ok {
		if v == "true" {
			rec.SupportsJSONSchema = true
		}
	}
	if v, ok := fields["supports_tool_calling"]; ok {
		if v == "true" {
			rec.SupportsToolCalling = true
		}
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *fakeProviderStore) DeleteProvider(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type fakeRunStore struct {
	records    map[string]RunRecord
	orderedIDs []string
}

func (s fakeRunStore) GetRun(ctx context.Context, id string) (*RunRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s fakeRunStore) ListRuns(ctx context.Context, page, limit int) ([]RunRecord, int, error) {
	return paginateTestRunsFromIDs(s.records, s.orderedIDs, page, limit), len(s.records), nil
}

func (s fakeRunStore) ListRunsByTenant(ctx context.Context, tenantID string, page, limit int) ([]RunRecord, int, error) {
	filtered := make([]RunRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestRuns(filtered, page, limit), len(filtered), nil
}

func (s fakeRunStore) ListRunsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]RunRecord, int, error) {
	filtered := make([]RunRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].WorkspaceID == workspaceID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestRuns(filtered, page, limit), len(filtered), nil
}

func (s fakeRunStore) ListRunsByAgent(ctx context.Context, agentID string, page, limit int) ([]RunRecord, int, error) {
	filtered := make([]RunRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].AgentID == agentID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestRuns(filtered, page, limit), len(filtered), nil
}

func (s fakeRunStore) ListRunsByEvaluation(ctx context.Context, evaluationID string, page, limit int) ([]RunRecord, int, error) {
	filtered := make([]RunRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].EvaluationID == evaluationID {
			filtered = append(filtered, s.records[id])
		}
	}
	return paginateTestRuns(filtered, page, limit), len(filtered), nil
}

func paginateTestRunsFromIDs(records map[string]RunRecord, orderedIDs []string, page, limit int) []RunRecord {
	all := make([]RunRecord, 0, len(orderedIDs))
	for _, id := range orderedIDs {
		all = append(all, records[id])
	}
	return paginateTestRuns(all, page, limit)
}

func paginateTestRuns(records []RunRecord, page, limit int) []RunRecord {
	start := (page - 1) * limit
	if start >= len(records) {
		return []RunRecord{}
	}
	end := min(start+limit, len(records))
	return records[start:end]
}

func (s *fakeRunStore) CreateRun(_ context.Context, run RunRecord) error {
	if _, exists := s.records[run.ID]; exists {
		return ErrConflict
	}
	s.records[run.ID] = run
	s.orderedIDs = append(s.orderedIDs, run.ID)
	return nil
}

func (s *fakeRunStore) UpdateRun(_ context.Context, id string, fields map[string]string) (*RunRecord, error) {
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

func (s *fakeRunStore) DeleteRun(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

func TestManagerHealthAndReadiness(t *testing.T) {
	handler := Server{}.Handler()
	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, path, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status 200, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerInfo(t *testing.T) {
	server := Server{
		Config: Config{
			Mode:           "managed",
			AutoMigrate:    true,
			DatabaseDriver: "pgx",
			DatabaseURL:    "postgres://manager@example/korus",
			Addr:           ":8090",
		},
	}
	handler := server.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var info InfoResponse
	if err := json.NewDecoder(recorder.Body).Decode(&info); err != nil {
		t.Fatalf("failed to decode info response: %v", err)
	}
	if info.Component != "manager" {
		t.Fatalf("expected component 'manager', got %q", info.Component)
	}
	if info.Mode != "managed" {
		t.Fatalf("expected mode 'managed', got %q", info.Mode)
	}
	if !info.DatabaseConfigured {
		t.Fatalf("expected databaseConfigured true")
	}
	if info.DatabaseDriver != "pgx" {
		t.Fatalf("expected databaseDriver 'pgx', got %q", info.DatabaseDriver)
	}
	if info.DatabaseStatus != "configured" {
		t.Fatalf("expected databaseStatus 'configured', got %q", info.DatabaseStatus)
	}
	if !info.MigrateOnStart {
		t.Fatalf("expected migrateOnStart true")
	}
}

func TestManagerRejectsUnsupportedMethod(t *testing.T) {
	handler := Server{}.Handler()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/info", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

func TestManagerGetWorkspace(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Workspaces: &fakeWorkspaceStore{
				records: map[string]WorkspaceRecord{
					"ws_123": {
						ID: "ws_123", TenantID: "tenant_123", Slug: "ehs",
						DisplayName: "EHS", Description: "Safety workspace", Status: "active",
						KubernetesNamespace: "ehs", KubernetesWorkspaceName: "workspace-ehs",
					},
				},
				orderedIDs: []string{"ws_123"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_123", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp WorkspaceResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "ws_123" || resp.TenantID != "tenant_123" {
		t.Fatalf("unexpected workspace response: %#v", resp)
	}
	if resp.KubernetesNamespace != "ehs" || resp.KubernetesWorkspaceName != "workspace-ehs" {
		t.Fatalf("unexpected Kubernetes mapping: %#v", resp)
	}
}

func TestManagerGetWorkspaceRequiresStore(t *testing.T) {
	handler := Server{}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_123", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}
}

func TestManagerGetWorkspaceNotFound(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Workspaces: &fakeWorkspaceStore{
				records:    map[string]WorkspaceRecord{},
				orderedIDs: []string{},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/missing", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestManagerListWorkspaces(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Workspaces: &fakeWorkspaceStore{
				records: map[string]WorkspaceRecord{
					"ws_1": {ID: "ws_1", TenantID: "t_1", Slug: "ws-1", DisplayName: "Workspace 1", Status: "active"},
					"ws_2": {ID: "ws_2", TenantID: "t_1", Slug: "ws-2", DisplayName: "Workspace 2", Status: "active"},
				},
				orderedIDs: []string{"ws_1", "ws_2"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedWorkspacesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Workspaces) != 2 {
		t.Fatalf("expected 2 workspaces, got %d", len(resp.Workspaces))
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
	if resp.Page != 1 {
		t.Fatalf("expected page 1, got %d", resp.Page)
	}
}

func TestManagerListWorkspacesPagination(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Workspaces: &fakeWorkspaceStore{
				records: map[string]WorkspaceRecord{
					"ws_1": {ID: "ws_1", TenantID: "t_1", Slug: "ws-1", DisplayName: "Workspace 1", Status: "active"},
					"ws_2": {ID: "ws_2", TenantID: "t_1", Slug: "ws-2", DisplayName: "Workspace 2", Status: "active"},
					"ws_3": {ID: "ws_3", TenantID: "t_1", Slug: "ws-3", DisplayName: "Workspace 3", Status: "active"},
				},
				orderedIDs: []string{"ws_1", "ws_2", "ws_3"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/?limit=2&page=2", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedWorkspacesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Workspaces) != 1 {
		t.Fatalf("expected 1 workspace on page 2, got %d", len(resp.Workspaces))
	}
	if resp.Total != 3 {
		t.Fatalf("expected total 3, got %d", resp.Total)
	}
	if resp.Page != 2 {
		t.Fatalf("expected page 2, got %d", resp.Page)
	}
}

func TestManagerListWorkspacesEmpty(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Workspaces: &fakeWorkspaceStore{
				records:    map[string]WorkspaceRecord{},
				orderedIDs: []string{},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedWorkspacesResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Workspaces) != 0 {
		t.Fatalf("expected 0 workspaces, got %d", len(resp.Workspaces))
	}
}

func TestManagerCreateWorkspace(t *testing.T) {
	store := &fakeWorkspaceStore{
		records:    map[string]WorkspaceRecord{},
		orderedIDs: []string{},
	}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	body := `{"id":"ws_1","tenantId":"t_1","slug":"my-ws","displayName":"My WS"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp WorkspaceResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "active" {
		t.Fatalf("expected default status 'active', got %q", resp.Status)
	}
	if _, ok := store.records["ws_1"]; !ok {
		t.Fatalf("expected workspace ws_1 to be stored")
	}
}

func TestManagerCreateWorkspaceMissingFields(t *testing.T) {
	store := &fakeWorkspaceStore{records: map[string]WorkspaceRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	tests := []struct {
		name string
		body string
	}{
		{"missing id", `{"tenantId":"t_1","slug":"ws","displayName":"WS"}`},
		{"missing tenantId", `{"id":"ws_1","slug":"ws","displayName":"WS"}`},
		{"missing slug", `{"id":"ws_1","tenantId":"t_1","displayName":"WS"}`},
		{"missing displayName", `{"id":"ws_1","tenantId":"t_1","slug":"ws"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerCreateWorkspaceConflict(t *testing.T) {
	store := &fakeWorkspaceStore{
		records:    map[string]WorkspaceRecord{"ws_1": {ID: "ws_1"}},
		orderedIDs: []string{"ws_1"},
	}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	body := `{"id":"ws_1","tenantId":"t_1","slug":"dup","displayName":"Dup WS"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", recorder.Code)
	}
}

func TestManagerUpdateWorkspace(t *testing.T) {
	store := &fakeWorkspaceStore{
		records: map[string]WorkspaceRecord{
			"ws_1": {ID: "ws_1", TenantID: "t_1", Slug: "ws-1", DisplayName: "Old", Status: "active"},
		},
		orderedIDs: []string{"ws_1"},
	}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	body := `{"displayName":"New Name","status":"inactive"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp WorkspaceResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.DisplayName != "New Name" {
		t.Fatalf("expected displayName 'New Name', got %q", resp.DisplayName)
	}
	if resp.Status != "inactive" {
		t.Fatalf("expected status 'inactive', got %q", resp.Status)
	}
	if resp.Slug != "ws-1" {
		t.Fatalf("expected slug unchanged 'ws-1', got %q", resp.Slug)
	}
}

func TestManagerUpdateWorkspaceNotFound(t *testing.T) {
	store := &fakeWorkspaceStore{records: map[string]WorkspaceRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	body := `{"displayName":"X"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/missing", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestManagerUpdateWorkspaceNoFields(t *testing.T) {
	store := &fakeWorkspaceStore{
		records:    map[string]WorkspaceRecord{"ws_1": {ID: "ws_1"}},
		orderedIDs: []string{"ws_1"},
	}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	body := `{}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/workspaces/ws_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestManagerDeleteWorkspace(t *testing.T) {
	store := &fakeWorkspaceStore{
		records:    map[string]WorkspaceRecord{"ws_1": {ID: "ws_1"}},
		orderedIDs: []string{"ws_1"},
	}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/ws_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
	if _, ok := store.records["ws_1"]; ok {
		t.Fatalf("expected workspace ws_1 to be deleted")
	}
}

func TestManagerDeleteWorkspaceIdempotent(t *testing.T) {
	store := &fakeWorkspaceStore{records: map[string]WorkspaceRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/workspaces/missing", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
}

func TestManagerWorkspaceRejectsUnsupportedMethods(t *testing.T) {
	store := &fakeWorkspaceStore{
		records:    map[string]WorkspaceRecord{"ws_1": {ID: "ws_1"}},
		orderedIDs: []string{"ws_1"},
	}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"put on collection", http.MethodPut, "/api/v1/workspaces/"},
		{"put on resource", http.MethodPut, "/api/v1/workspaces/ws_1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(tt.method, tt.path, nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected status 405, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerCollectionRejectsNonGetPost(t *testing.T) {
	store := &fakeWorkspaceStore{records: map[string]WorkspaceRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	for _, method := range []string{http.MethodPut, http.MethodPatch, http.MethodDelete} {
		t.Run(fmt.Sprintf("%s /api/v1/workspaces", method), func(t *testing.T) {
			request := httptest.NewRequest(method, "/api/v1/workspaces/", nil)
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusMethodNotAllowed {
				t.Fatalf("expected status 405 for %s, got %d", method, recorder.Code)
			}
		})
	}
}

func TestManagerGetTenant(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Tenants: &fakeTenantStore{
				records: map[string]TenantRecord{
					"t_1": {ID: "t_1", OrganizationID: "org_1", Slug: "my-tenant", DisplayName: "My Tenant", Status: "active", DefaultRegion: "us-east-1"},
				},
				orderedIDs: []string{"t_1"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp TenantResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Slug != "my-tenant" {
		t.Fatalf("expected slug 'my-tenant', got %q", resp.Slug)
	}
	if resp.DefaultRegion != "us-east-1" {
		t.Fatalf("expected defaultRegion 'us-east-1', got %q", resp.DefaultRegion)
	}
}

func TestManagerGetTenantNotFound(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Tenants: &fakeTenantStore{records: map[string]TenantRecord{}, orderedIDs: []string{}},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d", recorder.Code)
	}
}

func TestManagerListTenants(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Tenants: &fakeTenantStore{
				records: map[string]TenantRecord{
					"t_1": {ID: "t_1", OrganizationID: "org_1", Slug: "t-1", DisplayName: "Tenant 1", Status: "active"},
					"t_2": {ID: "t_2", OrganizationID: "org_1", Slug: "t-2", DisplayName: "Tenant 2", Status: "active"},
				},
				orderedIDs: []string{"t_1", "t_2"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedTenantsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Tenants) != 2 {
		t.Fatalf("expected 2 tenants, got %d", len(resp.Tenants))
	}
	if resp.Total != 2 {
		t.Fatalf("expected total 2, got %d", resp.Total)
	}
}

func TestManagerListTenantsEmpty(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Tenants: &fakeTenantStore{records: map[string]TenantRecord{}, orderedIDs: []string{}},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedTenantsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Tenants) != 0 {
		t.Fatalf("expected 0 tenants, got %d", len(resp.Tenants))
	}
}

func TestManagerTenantRequiresStore(t *testing.T) {
	handler := Server{}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}
}

func TestManagerTenantRejectsUnsupportedMethod(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Tenants: &fakeTenantStore{records: map[string]TenantRecord{}, orderedIDs: []string{}},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/tenants/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

func TestManagerWorkspaceWithSubPath(t *testing.T) {
	store := &fakeWorkspaceStore{
		records:    map[string]WorkspaceRecord{"ws_1": {ID: "ws_1"}},
		orderedIDs: []string{"ws_1"},
	}
	handler := Server{Stores: Stores{Workspaces: store}}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_1/extra", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for sub-path, got %d", recorder.Code)
	}
}

func TestManagerTenantWithSubPath(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Tenants: &fakeTenantStore{
				records:    map[string]TenantRecord{"t_1": {ID: "t_1"}},
				orderedIDs: []string{"t_1"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/tenants/t_1/extra", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status 404 for sub-path, got %d", recorder.Code)
	}
}

func TestManagerGetAgent(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Agents: &fakeAgentStore{
				records: map[string]AgentRecord{
					"agent_1": {
						ID: "agent_1", TenantID: "t_1", WorkspaceID: "ws_1", Slug: "triage",
						DisplayName: "Triage Agent", Status: "published", Pattern: "react",
						RuntimeEngine: "eino", RunnerClass: "adk", ModelProvider: "qwen", ModelName: "qwen-plus",
						LatestRevision: "rev-1",
					},
				},
				orderedIDs: []string{"agent_1"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/agents/agent_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp AgentResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "agent_1" || resp.ModelProvider != "qwen" || resp.RuntimeEngine != "eino" {
		t.Fatalf("unexpected agent response: %#v", resp)
	}
}

func TestManagerListAgentsByTenant(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Agents: &fakeAgentStore{
				records: map[string]AgentRecord{
					"agent_1": {ID: "agent_1", TenantID: "t_1", WorkspaceID: "ws_1", Slug: "a1", DisplayName: "Agent 1", Status: "published", Pattern: "react", RuntimeEngine: "eino", RunnerClass: "adk"},
					"agent_2": {ID: "agent_2", TenantID: "t_2", WorkspaceID: "ws_2", Slug: "a2", DisplayName: "Agent 2", Status: "draft", Pattern: "react", RuntimeEngine: "eino", RunnerClass: "adk"},
					"agent_3": {ID: "agent_3", TenantID: "t_1", WorkspaceID: "ws_3", Slug: "a3", DisplayName: "Agent 3", Status: "draft", Pattern: "react", RuntimeEngine: "eino", RunnerClass: "adk"},
				},
				orderedIDs: []string{"agent_1", "agent_2", "agent_3"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/agents/?tenantId=t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedAgentsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Agents) != 2 || resp.Total != 2 {
		t.Fatalf("expected 2 tenant agents, got len=%d total=%d", len(resp.Agents), resp.Total)
	}
}

func TestManagerAgentRequiresStore(t *testing.T) {
	handler := Server{}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/agents/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}
}

func TestManagerAgentRejectsUnsupportedMethod(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Agents: &fakeAgentStore{records: map[string]AgentRecord{}, orderedIDs: []string{}},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/agents/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

func TestManagerGetEvaluation(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Evaluations: &fakeEvaluationStore{
				records: map[string]EvaluationRecord{
					"eval_1": {
						ID: "eval_1", TenantID: "t_1", WorkspaceID: "ws_1", AgentID: "agent_1",
						Slug: "release-gate", DisplayName: "Release Gate", Status: "passed",
						DatasetName: "golden-set", DatasetRevision: "dataset-rev-1",
						BaselineRevision: "rev-0", Score: 0.93, GatePassed: true,
						SamplesTotal: 10, SamplesEvaluated: 10, LatestRunID: "run_1",
					},
				},
				orderedIDs: []string{"eval_1"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/eval_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp EvaluationResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "eval_1" || !resp.GatePassed || resp.Score != 0.93 {
		t.Fatalf("unexpected evaluation response: %#v", resp)
	}
}

func TestManagerListEvaluationsByTenant(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Evaluations: &fakeEvaluationStore{
				records: map[string]EvaluationRecord{
					"eval_1": {ID: "eval_1", TenantID: "t_1", WorkspaceID: "ws_1", AgentID: "agent_1", Slug: "e1", DisplayName: "Eval 1", Status: "passed", DatasetName: "ds", Score: 0.9, GatePassed: true},
					"eval_2": {ID: "eval_2", TenantID: "t_2", WorkspaceID: "ws_2", AgentID: "agent_2", Slug: "e2", DisplayName: "Eval 2", Status: "failed", DatasetName: "ds", Score: 0.6},
					"eval_3": {ID: "eval_3", TenantID: "t_1", WorkspaceID: "ws_3", AgentID: "agent_3", Slug: "e3", DisplayName: "Eval 3", Status: "running", DatasetName: "ds", Score: 0.7},
				},
				orderedIDs: []string{"eval_1", "eval_2", "eval_3"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/?tenantId=t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedEvaluationsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Evaluations) != 2 || resp.Total != 2 {
		t.Fatalf("expected 2 tenant evaluations, got len=%d total=%d", len(resp.Evaluations), resp.Total)
	}
}

func TestManagerEvaluationRequiresStore(t *testing.T) {
	handler := Server{}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/evaluations/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}
}

func TestManagerEvaluationRejectsUnsupportedMethod(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Evaluations: &fakeEvaluationStore{records: map[string]EvaluationRecord{}, orderedIDs: []string{}},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/evaluations/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

func TestManagerGetProvider(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Providers: &fakeProviderStore{
				records: map[string]ProviderRecord{
					"provider_1": {
						ID: "provider_1", TenantID: "t_1", WorkspaceID: "ws_1",
						Provider: "qwen", DisplayName: "Qwen Production", Family: "openai-compatible",
						BaseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1", CredentialRef: "secret://demo/qwen",
						Status: "active", Domestic: true, SupportsJSONSchema: true, SupportsToolCalling: true,
					},
				},
				orderedIDs: []string{"provider_1"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/providers/provider_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp ProviderResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "provider_1" || resp.Provider != "qwen" || !resp.Domestic || !resp.SupportsToolCalling {
		t.Fatalf("unexpected provider response: %#v", resp)
	}
}

func TestManagerListProvidersByTenant(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Providers: &fakeProviderStore{
				records: map[string]ProviderRecord{
					"provider_1": {ID: "provider_1", TenantID: "t_1", WorkspaceID: "ws_1", Provider: "qwen", DisplayName: "Qwen", Family: "openai-compatible", Status: "active"},
					"provider_2": {ID: "provider_2", TenantID: "t_2", WorkspaceID: "ws_2", Provider: "openai", DisplayName: "OpenAI", Family: "openai-compatible", Status: "active"},
					"provider_3": {ID: "provider_3", TenantID: "t_1", WorkspaceID: "ws_3", Provider: "deepseek", DisplayName: "DeepSeek", Family: "openai-compatible", Status: "inactive"},
				},
				orderedIDs: []string{"provider_1", "provider_2", "provider_3"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/providers/?tenantId=t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedProvidersResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Providers) != 2 || resp.Total != 2 {
		t.Fatalf("expected 2 tenant providers, got len=%d total=%d", len(resp.Providers), resp.Total)
	}
}

func TestManagerProviderRequiresStore(t *testing.T) {
	handler := Server{}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/providers/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}
}

func TestManagerProviderRejectsUnsupportedMethod(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Providers: &fakeProviderStore{records: map[string]ProviderRecord{}, orderedIDs: []string{}},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/providers/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

func TestManagerGetRun(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Runs: &fakeRunStore{
				records: map[string]RunRecord{
					"run_1": {
						ID: "run_1", TenantID: "t_1", WorkspaceID: "ws_1", AgentID: "agent_1",
						AgentRevision: "rev-1", Status: "succeeded", RuntimeEngine: "eino", RunnerClass: "adk",
						Summary: "inspection complete", TraceRef: "pod/run-1",
					},
				},
				orderedIDs: []string{"run_1"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runs/run_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp RunResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.ID != "run_1" || resp.Status != "succeeded" || resp.TraceRef != "pod/run-1" {
		t.Fatalf("unexpected run response: %#v", resp)
	}
}

func TestManagerListRunsByTenant(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Runs: &fakeRunStore{
				records: map[string]RunRecord{
					"run_1": {ID: "run_1", TenantID: "t_1", WorkspaceID: "ws_1", AgentID: "agent_1", Status: "succeeded"},
					"run_2": {ID: "run_2", TenantID: "t_2", WorkspaceID: "ws_2", AgentID: "agent_2", Status: "running"},
					"run_3": {ID: "run_3", TenantID: "t_1", WorkspaceID: "ws_3", AgentID: "agent_3", Status: "failed"},
				},
				orderedIDs: []string{"run_1", "run_2", "run_3"},
			},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runs/?tenantId=t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	var resp PaginatedRunsResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if len(resp.Runs) != 2 || resp.Total != 2 {
		t.Fatalf("expected 2 tenant runs, got len=%d total=%d", len(resp.Runs), resp.Total)
	}
}

func TestManagerRunRequiresStore(t *testing.T) {
	handler := Server{}.Handler()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/runs/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d", recorder.Code)
	}
}

func TestManagerRunRejectsUnsupportedMethod(t *testing.T) {
	handler := Server{
		Stores: Stores{
			Runs: &fakeRunStore{records: map[string]RunRecord{}, orderedIDs: []string{}},
		},
	}.Handler()
	request := httptest.NewRequest(http.MethodPut, "/api/v1/runs/", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status 405, got %d", recorder.Code)
	}
}

// --- Tenant CRUD tests ---

func TestManagerCreateTenant(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Tenants: store}}.Handler()
	body := `{"id":"t_new","organizationId":"org_1","slug":"new-tenant","displayName":"New Tenant"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp TenantResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "active" {
		t.Fatalf("expected default status 'active', got %q", resp.Status)
	}
	if _, ok := store.records["t_new"]; !ok {
		t.Fatalf("expected tenant t_new to be stored")
	}
}

func TestManagerCreateTenantMissingFields(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Tenants: store}}.Handler()
	tests := []struct {
		name string
		body string
	}{
		{"missing id", `{"organizationId":"org_1","slug":"t","displayName":"T"}`},
		{"missing organizationId", `{"id":"t_1","slug":"t","displayName":"T"}`},
		{"missing slug", `{"id":"t_1","organizationId":"org_1","displayName":"T"}`},
		{"missing displayName", `{"id":"t_1","organizationId":"org_1","slug":"t"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerCreateTenantConflict(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{"t_1": {ID: "t_1"}}, orderedIDs: []string{"t_1"}}
	handler := Server{Stores: Stores{Tenants: store}}.Handler()
	body := `{"id":"t_1","organizationId":"org_1","slug":"dup","displayName":"Dup"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status 409, got %d", recorder.Code)
	}
}

func TestManagerUpdateTenant(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{"t_1": {ID: "t_1", DisplayName: "Old", Status: "active"}}, orderedIDs: []string{"t_1"}}
	handler := Server{Stores: Stores{Tenants: store}}.Handler()
	body := `{"displayName":"New Name","status":"inactive"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/tenants/t_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp TenantResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.DisplayName != "New Name" {
		t.Fatalf("expected displayName 'New Name', got %q", resp.DisplayName)
	}
	if resp.Status != "inactive" {
		t.Fatalf("expected status 'inactive', got %q", resp.Status)
	}
}

func TestManagerDeleteTenant(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{"t_1": {ID: "t_1"}}, orderedIDs: []string{"t_1"}}
	handler := Server{Stores: Stores{Tenants: store}}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
	if _, ok := store.records["t_1"]; ok {
		t.Fatalf("expected tenant t_1 to be deleted")
	}
}

// --- Agent CRUD tests ---

func TestManagerCreateAgent(t *testing.T) {
	store := &fakeAgentStore{records: map[string]AgentRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Agents: store}}.Handler()
	body := `{"id":"agent_new","tenantId":"t_1","workspaceId":"ws_1","slug":"new-agent","displayName":"New Agent"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/agents/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp AgentResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "draft" {
		t.Fatalf("expected default status 'draft', got %q", resp.Status)
	}
	if resp.Pattern != "react" {
		t.Fatalf("expected default pattern 'react', got %q", resp.Pattern)
	}
}

func TestManagerCreateAgentMissingFields(t *testing.T) {
	store := &fakeAgentStore{records: map[string]AgentRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Agents: store}}.Handler()
	tests := []struct {
		name string
		body string
	}{
		{"missing id", `{"tenantId":"t_1","workspaceId":"ws_1","slug":"a","displayName":"A"}`},
		{"missing tenantId", `{"id":"a_1","workspaceId":"ws_1","slug":"a","displayName":"A"}`},
		{"missing workspaceId", `{"id":"a_1","tenantId":"t_1","slug":"a","displayName":"A"}`},
		{"missing slug", `{"id":"a_1","tenantId":"t_1","workspaceId":"ws_1","displayName":"A"}`},
		{"missing displayName", `{"id":"a_1","tenantId":"t_1","workspaceId":"ws_1","slug":"a"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/agents/", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerUpdateAgent(t *testing.T) {
	store := &fakeAgentStore{records: map[string]AgentRecord{"a_1": {ID: "a_1", DisplayName: "Old"}}, orderedIDs: []string{"a_1"}}
	handler := Server{Stores: Stores{Agents: store}}.Handler()
	body := `{"displayName":"Updated Agent","modelName":"gpt-4"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/a_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp AgentResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.DisplayName != "Updated Agent" {
		t.Fatalf("expected displayName 'Updated Agent', got %q", resp.DisplayName)
	}
}

func TestManagerDeleteAgent(t *testing.T) {
	store := &fakeAgentStore{records: map[string]AgentRecord{"a_1": {ID: "a_1"}}, orderedIDs: []string{"a_1"}}
	handler := Server{Stores: Stores{Agents: store}}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/agents/a_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
	if _, ok := store.records["a_1"]; ok {
		t.Fatalf("expected agent a_1 to be deleted")
	}
}

// --- Evaluation CRUD tests ---

func TestManagerCreateEvaluation(t *testing.T) {
	store := &fakeEvaluationStore{records: map[string]EvaluationRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Evaluations: store}}.Handler()
	body := `{"id":"eval_new","tenantId":"t_1","workspaceId":"ws_1","agentId":"a_1","slug":"new-eval","displayName":"New Eval"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/evaluations/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp EvaluationResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected default status 'pending', got %q", resp.Status)
	}
}

func TestManagerCreateEvaluationMissingFields(t *testing.T) {
	store := &fakeEvaluationStore{records: map[string]EvaluationRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Evaluations: store}}.Handler()
	tests := []struct {
		name string
		body string
	}{
		{"missing id", `{"tenantId":"t_1","workspaceId":"ws_1","agentId":"a_1","slug":"e","displayName":"E"}`},
		{"missing tenantId", `{"id":"e_1","workspaceId":"ws_1","agentId":"a_1","slug":"e","displayName":"E"}`},
		{"missing workspaceId", `{"id":"e_1","tenantId":"t_1","agentId":"a_1","slug":"e","displayName":"E"}`},
		{"missing agentId", `{"id":"e_1","tenantId":"t_1","workspaceId":"ws_1","slug":"e","displayName":"E"}`},
		{"missing slug", `{"id":"e_1","tenantId":"t_1","workspaceId":"ws_1","agentId":"a_1","displayName":"E"}`},
		{"missing displayName", `{"id":"e_1","tenantId":"t_1","workspaceId":"ws_1","agentId":"a_1","slug":"e"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/evaluations/", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerUpdateEvaluation(t *testing.T) {
	store := &fakeEvaluationStore{records: map[string]EvaluationRecord{"e_1": {ID: "e_1", DisplayName: "Old"}}, orderedIDs: []string{"e_1"}}
	handler := Server{Stores: Stores{Evaluations: store}}.Handler()
	body := `{"displayName":"Updated Eval","status":"passed"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/evaluations/e_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp EvaluationResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.DisplayName != "Updated Eval" {
		t.Fatalf("expected displayName 'Updated Eval', got %q", resp.DisplayName)
	}
	if resp.Status != "passed" {
		t.Fatalf("expected status 'passed', got %q", resp.Status)
	}
}

func TestManagerDeleteEvaluation(t *testing.T) {
	store := &fakeEvaluationStore{records: map[string]EvaluationRecord{"e_1": {ID: "e_1"}}, orderedIDs: []string{"e_1"}}
	handler := Server{Stores: Stores{Evaluations: store}}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/evaluations/e_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
	if _, ok := store.records["e_1"]; ok {
		t.Fatalf("expected evaluation e_1 to be deleted")
	}
}

// --- Provider CRUD tests ---

func TestManagerCreateProvider(t *testing.T) {
	store := &fakeProviderStore{records: map[string]ProviderRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Providers: store}}.Handler()
	body := `{"id":"p_new","tenantId":"t_1","provider":"openai","displayName":"New Provider"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/providers/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp ProviderResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "active" {
		t.Fatalf("expected default status 'active', got %q", resp.Status)
	}
}

func TestManagerCreateProviderMissingFields(t *testing.T) {
	store := &fakeProviderStore{records: map[string]ProviderRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Providers: store}}.Handler()
	tests := []struct {
		name string
		body string
	}{
		{"missing id", `{"tenantId":"t_1","provider":"openai","displayName":"P"}`},
		{"missing tenantId", `{"id":"p_1","provider":"openai","displayName":"P"}`},
		{"missing provider", `{"id":"p_1","tenantId":"t_1","displayName":"P"}`},
		{"missing displayName", `{"id":"p_1","tenantId":"t_1","provider":"openai"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/providers/", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerUpdateProvider(t *testing.T) {
	store := &fakeProviderStore{records: map[string]ProviderRecord{"p_1": {ID: "p_1", DisplayName: "Old"}}, orderedIDs: []string{"p_1"}}
	handler := Server{Stores: Stores{Providers: store}}.Handler()
	body := `{"displayName":"Updated Provider","status":"inactive"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/providers/p_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp ProviderResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.DisplayName != "Updated Provider" {
		t.Fatalf("expected displayName 'Updated Provider', got %q", resp.DisplayName)
	}
}

func TestManagerDeleteProvider(t *testing.T) {
	store := &fakeProviderStore{records: map[string]ProviderRecord{"p_1": {ID: "p_1"}}, orderedIDs: []string{"p_1"}}
	handler := Server{Stores: Stores{Providers: store}}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/providers/p_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
	if _, ok := store.records["p_1"]; ok {
		t.Fatalf("expected provider p_1 to be deleted")
	}
}

// --- Run CRUD tests ---

func TestManagerCreateRun(t *testing.T) {
	store := &fakeRunStore{records: map[string]RunRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Runs: store}}.Handler()
	body := `{"id":"run_new","tenantId":"t_1","workspaceId":"ws_1","agentId":"a_1"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/runs/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp RunResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "pending" {
		t.Fatalf("expected default status 'pending', got %q", resp.Status)
	}
}

func TestManagerCreateRunMissingFields(t *testing.T) {
	store := &fakeRunStore{records: map[string]RunRecord{}, orderedIDs: []string{}}
	handler := Server{Stores: Stores{Runs: store}}.Handler()
	tests := []struct {
		name string
		body string
	}{
		{"missing id", `{"tenantId":"t_1","workspaceId":"ws_1","agentId":"a_1"}`},
		{"missing tenantId", `{"id":"r_1","workspaceId":"ws_1","agentId":"a_1"}`},
		{"missing workspaceId", `{"id":"r_1","tenantId":"t_1","agentId":"a_1"}`},
		{"missing agentId", `{"id":"r_1","tenantId":"t_1","workspaceId":"ws_1"}`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/api/v1/runs/", strings.NewReader(tt.body))
			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, request)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("expected status 400, got %d", recorder.Code)
			}
		})
	}
}

func TestManagerUpdateRun(t *testing.T) {
	store := &fakeRunStore{records: map[string]RunRecord{"r_1": {ID: "r_1", Status: "pending"}}, orderedIDs: []string{"r_1"}}
	handler := Server{Stores: Stores{Runs: store}}.Handler()
	body := `{"status":"succeeded","summary":"done"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/runs/r_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp RunResponse
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if resp.Status != "succeeded" {
		t.Fatalf("expected status 'succeeded', got %q", resp.Status)
	}
	if resp.Summary != "done" {
		t.Fatalf("expected summary 'done', got %q", resp.Summary)
	}
}

func TestManagerDeleteRun(t *testing.T) {
	store := &fakeRunStore{records: map[string]RunRecord{"r_1": {ID: "r_1"}}, orderedIDs: []string{"r_1"}}
	handler := Server{Stores: Stores{Runs: store}}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/runs/r_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected status 204, got %d", recorder.Code)
	}
	if _, ok := store.records["r_1"]; ok {
		t.Fatalf("expected run r_1 to be deleted")
	}
}

// --- Syncer integration tests ---

type trackingSyncer struct {
	synced    []string
	deleted   []string
	syncErr   error
	deleteErr error
}

func (s *trackingSyncer) SyncTenant(_ context.Context, r TenantRecord) error {
	s.synced = append(s.synced, "tenant:"+r.ID)
	return s.syncErr
}
func (s *trackingSyncer) DeleteTenant(_ context.Context, id string) error {
	s.deleted = append(s.deleted, "tenant:"+id)
	return s.deleteErr
}
func (s *trackingSyncer) SyncWorkspace(_ context.Context, r WorkspaceRecord) error {
	s.synced = append(s.synced, "workspace:"+r.ID)
	return s.syncErr
}
func (s *trackingSyncer) DeleteWorkspace(_ context.Context, id string) error {
	s.deleted = append(s.deleted, "workspace:"+id)
	return s.deleteErr
}
func (s *trackingSyncer) SyncAgent(_ context.Context, r AgentRecord) error {
	s.synced = append(s.synced, "agent:"+r.ID)
	return s.syncErr
}
func (s *trackingSyncer) DeleteAgent(_ context.Context, id string) error {
	s.deleted = append(s.deleted, "agent:"+id)
	return s.deleteErr
}
func (s *trackingSyncer) SyncEvaluation(_ context.Context, r EvaluationRecord) error {
	s.synced = append(s.synced, "evaluation:"+r.ID)
	return s.syncErr
}
func (s *trackingSyncer) DeleteEvaluation(_ context.Context, id string) error {
	s.deleted = append(s.deleted, "evaluation:"+id)
	return s.deleteErr
}
func (s *trackingSyncer) SyncProvider(_ context.Context, r ProviderRecord) error {
	s.synced = append(s.synced, "provider:"+r.ID)
	return s.syncErr
}
func (s *trackingSyncer) DeleteProvider(_ context.Context, id string) error {
	s.deleted = append(s.deleted, "provider:"+id)
	return s.deleteErr
}

func TestSyncerCalledOnTenantCreate(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{}, orderedIDs: []string{}}
	syncer := &trackingSyncer{}
	handler := Server{Stores: Stores{Tenants: store}, Syncer: syncer}.Handler()
	body := `{"id":"t_sync","organizationId":"org_1","slug":"sync","displayName":"Sync"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}
	if len(syncer.synced) != 1 || syncer.synced[0] != "tenant:t_sync" {
		t.Fatalf("expected syncer to be called with tenant:t_sync, got %v", syncer.synced)
	}
}

func TestSyncerCalledOnTenantDelete(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{"t_1": {ID: "t_1"}}, orderedIDs: []string{"t_1"}}
	syncer := &trackingSyncer{}
	handler := Server{Stores: Stores{Tenants: store}, Syncer: syncer}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/tenants/t_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	if len(syncer.deleted) != 1 || syncer.deleted[0] != "tenant:t_1" {
		t.Fatalf("expected syncer to be called with tenant:t_1, got %v", syncer.deleted)
	}
}

func TestSyncerCalledOnWorkspaceCreate(t *testing.T) {
	store := &fakeWorkspaceStore{records: map[string]WorkspaceRecord{}, orderedIDs: []string{}}
	syncer := &trackingSyncer{}
	handler := Server{Stores: Stores{Workspaces: store}, Syncer: syncer}.Handler()
	body := `{"id":"ws_sync","tenantId":"t_1","slug":"sync","displayName":"Sync"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/workspaces/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", recorder.Code)
	}
	if len(syncer.synced) != 1 || syncer.synced[0] != "workspace:ws_sync" {
		t.Fatalf("expected syncer to be called with workspace:ws_sync, got %v", syncer.synced)
	}
}

func TestSyncerCalledOnAgentUpdate(t *testing.T) {
	store := &fakeAgentStore{records: map[string]AgentRecord{"a_1": {ID: "a_1", DisplayName: "Old"}}, orderedIDs: []string{"a_1"}}
	syncer := &trackingSyncer{}
	handler := Server{Stores: Stores{Agents: store}, Syncer: syncer}.Handler()
	body := `{"displayName":"Updated"}`
	request := httptest.NewRequest(http.MethodPatch, "/api/v1/agents/a_1", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}
	if len(syncer.synced) != 1 || syncer.synced[0] != "agent:a_1" {
		t.Fatalf("expected syncer to be called with agent:a_1, got %v", syncer.synced)
	}
}

func TestSyncerCalledOnRunDelete(t *testing.T) {
	store := &fakeRunStore{records: map[string]RunRecord{"r_1": {ID: "r_1"}}, orderedIDs: []string{"r_1"}}
	syncer := &trackingSyncer{}
	handler := Server{Stores: Stores{Runs: store}, Syncer: syncer}.Handler()
	request := httptest.NewRequest(http.MethodDelete, "/api/v1/runs/r_1", nil)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", recorder.Code)
	}
	// Run has no CRD sync, so syncer should not be called
	if len(syncer.deleted) != 0 {
		t.Fatalf("expected syncer not to be called for run delete, got %v", syncer.deleted)
	}
}

func TestSyncerErrorDoesNotFailRequest(t *testing.T) {
	store := &fakeTenantStore{records: map[string]TenantRecord{}, orderedIDs: []string{}}
	syncer := &trackingSyncer{syncErr: fmt.Errorf("k8s unavailable")}
	handler := Server{Stores: Stores{Tenants: store}, Syncer: syncer}.Handler()
	body := `{"id":"t_err","organizationId":"org_1","slug":"err","displayName":"Err"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", strings.NewReader(body))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	// Should still return 201 even though syncer failed
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected 201 despite syncer error, got %d", recorder.Code)
	}
	if _, ok := store.records["t_err"]; !ok {
		t.Fatalf("expected tenant to be created in store despite syncer error")
	}
}

func TestNoopSyncerDoesNothing(t *testing.T) {
	noop := NoopCRDSyncer{}
	ctx := context.Background()
	if err := noop.SyncTenant(ctx, TenantRecord{ID: "t_1"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := noop.DeleteTenant(ctx, "t_1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := noop.SyncWorkspace(ctx, WorkspaceRecord{ID: "ws_1"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := noop.DeleteWorkspace(ctx, "ws_1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := noop.SyncAgent(ctx, AgentRecord{ID: "a_1"}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := noop.DeleteAgent(ctx, "a_1"); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}
