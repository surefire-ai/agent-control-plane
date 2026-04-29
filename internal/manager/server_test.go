package manager

import (
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
			DatabaseDriver: "postgres",
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
	if response.DatabaseDriver != "postgres" {
		t.Fatalf("expected postgres database driver, got %#v", response)
	}
	if response.DatabaseStatus != "configured" {
		t.Fatalf("expected configured database status, got %#v", response)
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
