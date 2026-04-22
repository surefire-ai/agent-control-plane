package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

func TestHTTPToolInvokerInvoke(t *testing.T) {
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/tickets" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["title"] != "Repair cabinet" {
			t.Fatalf("unexpected request body: %#v", body)
		}
		_, _ = w.Write([]byte(`{"ticketId":"T-100","status":"created","url":"https://example.internal/T-100"}`))
	}))
	defer server.Close()

	result, err := HTTPToolInvoker{Client: server.Client()}.Invoke(
		context.Background(),
		contract.WorkerToolRuntime{
			Type:         "http",
			Capabilities: []string{"http"},
			AuthTokenEnv: "TOOL_RECTIFY_TICKET_API_AUTH_TOKEN",
		},
		contract.ToolSpec{
			Name: "rectify-ticket-api",
			HTTP: map[string]interface{}{
				"method": "POST",
				"url":    server.URL + "/tickets",
				"auth": map[string]interface{}{
					"type": "bearerToken",
				},
			},
			Schema: map[string]interface{}{
				"output": map[string]interface{}{
					"type":     "object",
					"required": []interface{}{"ticketId", "status"},
				},
			},
		},
		map[string]interface{}{"title": "Repair cabinet"},
	)
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if result.Output["ticketId"] != "T-100" {
		t.Fatalf("unexpected tool output: %#v", result.Output)
	}
}

func TestHTTPToolInvokerRejectsSchemaMismatch(t *testing.T) {
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"status":"created"}`))
	}))
	defer server.Close()

	_, err := HTTPToolInvoker{Client: server.Client()}.Invoke(
		context.Background(),
		contract.WorkerToolRuntime{
			Type:         "http",
			Capabilities: []string{"http"},
			AuthTokenEnv: "TOOL_RECTIFY_TICKET_API_AUTH_TOKEN",
		},
		contract.ToolSpec{
			Name: "rectify-ticket-api",
			HTTP: map[string]interface{}{
				"url": server.URL,
				"auth": map[string]interface{}{
					"type": "bearerToken",
				},
			},
			Schema: map[string]interface{}{
				"output": map[string]interface{}{
					"type":     "object",
					"required": []interface{}{"ticketId"},
				},
			},
		},
		map[string]interface{}{"title": "Repair cabinet"},
	)
	if err == nil || !strings.Contains(err.Error(), "missing required field") {
		t.Fatalf("expected schema validation error, got %v", err)
	}
}
