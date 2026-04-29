package manager

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestManagerHealthAndReadiness(t *testing.T) {
	server := Server{}.Handler()

	for _, path := range []string{"/healthz", "/readyz"} {
		t.Run(path, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, path, nil)

			server.ServeHTTP(recorder, request)

			if recorder.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
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
			DatabaseURL:    "postgres://manager@example/agent-control-plane",
		},
	}.Handler()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/info", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response InfoResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Component != "manager" {
		t.Fatalf("expected manager component, got %#v", response)
	}
	if response.Mode != "managed" {
		t.Fatalf("expected managed mode, got %#v", response)
	}
	if !response.DatabaseConfigured {
		t.Fatalf("expected configured database, got %#v", response)
	}
	if response.DatabaseDriver != "pgx" {
		t.Fatalf("expected pgx database driver, got %#v", response)
	}
	if response.DatabaseStatus != "configured" {
		t.Fatalf("expected configured database status, got %#v", response)
	}
	if !response.MigrateOnStart {
		t.Fatalf("expected migrate-on-start to be enabled, got %#v", response)
	}
}

func TestManagerRejectsUnsupportedMethod(t *testing.T) {
	server := Server{}.Handler()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/info", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected status %d, got %d: %s", http.StatusMethodNotAllowed, recorder.Code, recorder.Body.String())
	}
}

func TestManagerGetWorkspace(t *testing.T) {
	server := Server{
		Stores: Stores{
			Workspaces: fakeWorkspaceStore{
				records: map[string]WorkspaceRecord{
					"ws_123": {
						ID:                      "ws_123",
						TenantID:                "tenant_123",
						Slug:                    "ehs",
						DisplayName:             "EHS",
						Description:             "Safety workspace",
						Status:                  "active",
						KubernetesNamespace:     "ehs",
						KubernetesWorkspaceName: "workspace-ehs",
					},
				},
			},
		},
	}.Handler()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_123", nil)
	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, recorder.Code, recorder.Body.String())
	}
	var response WorkspaceResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.ID != "ws_123" || response.TenantID != "tenant_123" {
		t.Fatalf("unexpected workspace response: %#v", response)
	}
	if response.KubernetesNamespace != "ehs" || response.KubernetesWorkspaceName != "workspace-ehs" {
		t.Fatalf("unexpected Kubernetes mapping: %#v", response)
	}
}

func TestManagerGetWorkspaceRequiresStore(t *testing.T) {
	server := Server{}.Handler()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/ws_123", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status %d, got %d: %s", http.StatusServiceUnavailable, recorder.Code, recorder.Body.String())
	}
}

func TestManagerGetWorkspaceNotFound(t *testing.T) {
	server := Server{
		Stores: Stores{
			Workspaces: fakeWorkspaceStore{records: map[string]WorkspaceRecord{}},
		},
	}.Handler()
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/v1/workspaces/missing", nil)

	server.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, recorder.Code, recorder.Body.String())
	}
}

type fakeWorkspaceStore struct {
	records map[string]WorkspaceRecord
}

func (s fakeWorkspaceStore) GetWorkspace(ctx context.Context, id string) (*WorkspaceRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}
