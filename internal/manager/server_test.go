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
			Mode:            "managed",
			AutoMigrate:     true,
			DatabaseDriver:  "pgx",
			DatabaseURL:     "postgres://manager@example/agent-control-plane",
			Addr:            ":8090",
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
	request := httptest.NewRequest(http.MethodPost, "/api/v1/tenants/", nil)
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
