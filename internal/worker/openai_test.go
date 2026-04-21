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

func TestOpenAICompatibleInvokerInvoke(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-secret" {
			t.Fatalf("unexpected auth header: %q", got)
		}
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["model"] != "gpt-4.1" {
			t.Fatalf("unexpected model request: %#v", body)
		}
		if _, ok := body["response_format"]; !ok {
			t.Fatalf("expected response_format in request: %#v", body)
		}
		_, _ = w.Write([]byte(`{"id":"chatcmpl-1","model":"gpt-4.1","choices":[{"index":0,"message":{"role":"assistant","content":"{\"summary\":\"inspection complete\"}"}}]}`))
	}))
	defer server.Close()

	result, err := OpenAICompatibleInvoker{Client: server.Client()}.Invoke(
		context.Background(),
		contract.WorkerModelRuntime{
			Provider:  "openai",
			Model:     "gpt-4.1",
			BaseURL:   server.URL,
			APIKeyEnv: "MODEL_PLANNER_API_KEY",
		},
		contract.ModelConfig{
			Model:       "gpt-4.1",
			Temperature: 0.1,
			MaxTokens:   4000,
		},
		contract.PromptSpec{
			Name:     "system",
			Language: "zh-CN",
			Template: "You are an EHS assistant.",
		},
		map[string]interface{}{"task": "identify_hazard"},
		map[string]interface{}{"schema": map[string]interface{}{"type": "object"}},
	)
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if result.Content != "{\"summary\":\"inspection complete\"}" {
		t.Fatalf("unexpected content: %q", result.Content)
	}
	if result.RequestBody["model"] != "gpt-4.1" {
		t.Fatalf("unexpected request body: %#v", result.RequestBody)
	}
}

func TestOpenAICompatibleInvokerRejectsNonSuccessStatus(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", http.StatusBadGateway)
	}))
	defer server.Close()

	_, err := OpenAICompatibleInvoker{Client: server.Client()}.Invoke(
		context.Background(),
		contract.WorkerModelRuntime{
			BaseURL:   server.URL,
			APIKeyEnv: "MODEL_PLANNER_API_KEY",
		},
		contract.ModelConfig{Model: "gpt-4.1"},
		contract.PromptSpec{Template: "hello"},
		map[string]interface{}{"task": "identify_hazard"},
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "model call returned status") {
		t.Fatalf("expected status error, got %v", err)
	}
}
