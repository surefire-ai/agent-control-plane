package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestIsRouterPattern(t *testing.T) {
	tests := []struct {
		name     string
		artifact contract.CompiledArtifact
		want     bool
	}{
		{
			name:     "empty artifact",
			artifact: contract.CompiledArtifact{},
			want:     false,
		},
		{
			name: "pattern.type router",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "router"},
			},
			want: true,
		},
		{
			name: "runner.pattern.type router",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "router"},
				},
			},
			want: true,
		},
		{
			name: "pattern.type react",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "react"},
			},
			want: false,
		},
		{
			name: "both pattern.type and runner.pattern set",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "router"},
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "react"},
				},
			},
			want: true,
		},
		{
			name: "runner.pattern.type not a string",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": 42},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRouterPattern(tt.artifact); got != tt.want {
				t.Errorf("isRouterPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouterClassifierPrompt(t *testing.T) {
	tests := []struct {
		name      string
		routes    []contract.PatternRouteConfig
		wantLabel []string
		wantJSON  bool
	}{
		{
			name:      "single route",
			routes:    []contract.PatternRouteConfig{{Label: "billing"}},
			wantLabel: []string{"billing"},
			wantJSON:  true,
		},
		{
			name: "multiple routes",
			routes: []contract.PatternRouteConfig{
				{Label: "billing"},
				{Label: "support"},
				{Label: "general"},
			},
			wantLabel: []string{"billing", "support", "general"},
			wantJSON:  true,
		},
		{
			name:      "empty routes",
			routes:    []contract.PatternRouteConfig{},
			wantLabel: nil,
			wantJSON:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := routerClassifierPrompt(tt.routes)
			if prompt == "" {
				t.Fatal("expected non-empty prompt")
			}
			for _, label := range tt.wantLabel {
				if !strings.Contains(prompt, "- "+label) {
					t.Errorf("expected label %q in prompt", label)
				}
			}
			if tt.wantJSON && !strings.Contains(prompt, `"classification"`) {
				t.Error("expected JSON format instruction in prompt")
			}
		})
	}
}

func TestPollAgentRunSucceeded(t *testing.T) {
	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		var phase string
		if n < 2 {
			phase = "Running"
		} else {
			phase = "Succeeded"
		}
		resp := map[string]interface{}{
			"status": map[string]interface{}{
				"phase": phase,
				"output": map[string]interface{}{
					"result": "ok",
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	result, err := pollAgentRun(context.Background(), httpClient, server.URL, "test-ns", "run-1")
	if err != nil {
		t.Fatalf("pollAgentRun returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("expected status %q, got %q", contract.WorkerStatusSucceeded, result.Status)
	}
	if result.Output["result"] != "ok" {
		t.Fatalf("expected output result=ok, got %v", result.Output)
	}
	if atomic.LoadInt32(&callCount) < 2 {
		t.Fatalf("expected at least 2 poll calls, got %d", callCount)
	}
}

func TestPollAgentRunFailed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status": map[string]interface{}{
				"phase": "Failed",
				"conditions": []interface{}{
					map[string]interface{}{
						"message": "model rate limited",
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	_, err := pollAgentRun(context.Background(), httpClient, server.URL, "test-ns", "run-1")
	if err == nil {
		t.Fatal("expected error for Failed phase")
	}
	if !strings.Contains(err.Error(), "failed") {
		t.Errorf("expected error to mention 'failed', got: %v", err)
	}
}

func TestPollAgentRunCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status": map[string]interface{}{
				"phase": "Canceled",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	_, err := pollAgentRun(context.Background(), httpClient, server.URL, "test-ns", "run-1")
	if err == nil {
		t.Fatal("expected error for Canceled phase")
	}
	if !strings.Contains(err.Error(), "canceled") {
		t.Errorf("expected error to mention 'canceled', got: %v", err)
	}
}

func TestPollAgentRunContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"status": map[string]interface{}{
				"phase": "Running",
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	_, err := pollAgentRun(ctx, httpClient, server.URL, "test-ns", "run-1")
	if err == nil {
		t.Fatal("expected error when context is canceled")
	}
}

func TestPollAgentRunNoStatus(t *testing.T) {
	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		var resp map[string]interface{}
		if n == 1 {
			resp = map[string]interface{}{"metadata": map[string]interface{}{}}
		} else {
			resp = map[string]interface{}{
				"status": map[string]interface{}{
					"phase":  "Succeeded",
					"output": map[string]interface{}{"done": true},
				},
			}
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	httpClient := &http.Client{Timeout: 10 * time.Second}
	result, err := pollAgentRun(context.Background(), httpClient, server.URL, "test-ns", "run-1")
	if err != nil {
		t.Fatalf("pollAgentRun returned error: %v", err)
	}
	if result.Output["done"] != true {
		t.Fatalf("expected output done=true, got %v", result.Output)
	}
}

// NOTE: parseAgentRunPath is an unexported function in the gateway package
// (github.com/surefire-ai/korus/internal/gateway) and cannot be tested
// directly from the worker package. Its tests belong in
// internal/gateway/gateway_test.go.
