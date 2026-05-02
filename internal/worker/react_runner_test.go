package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestIsReactPattern(t *testing.T) {
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
			name: "pattern.type react",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "react"},
			},
			want: true,
		},
		{
			name: "runner.pattern.type react",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "react"},
				},
			},
			want: true,
		},
		{
			name: "pattern.type workflow",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "workflow"},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isReactPattern(tt.artifact); got != tt.want {
				t.Errorf("isReactPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReactMaxIterations(t *testing.T) {
	tests := []struct {
		name     string
		artifact contract.CompiledArtifact
		want     int32
	}{
		{
			name:     "default",
			artifact: contract.CompiledArtifact{},
			want:     defaultReactMaxIterations,
		},
		{
			name: "from pattern",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{MaxIterations: 10},
			},
			want: 10,
		},
		{
			name: "from runner pattern float64",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"maxIterations": float64(8)},
				},
			},
			want: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := reactMaxIterations(tt.artifact); got != tt.want {
				t.Errorf("reactMaxIterations() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseReactDecision(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		parsed    map[string]interface{}
		wantFinal bool
		wantTool  string
	}{
		{
			name:      "final_answer from parsed",
			content:   `{"final_answer": {"summary": "done"}}`,
			parsed:    map[string]interface{}{"final_answer": map[string]interface{}{"summary": "done"}},
			wantFinal: true,
		},
		{
			name:      "action from parsed",
			content:   `{"action": "rectify-ticket-api", "action_input": {"id": "123"}}`,
			parsed:    map[string]interface{}{"action": "rectify-ticket-api", "action_input": map[string]interface{}{"id": "123"}},
			wantFinal: false,
			wantTool:  "rectify-ticket-api",
		},
		{
			name:      "extract JSON from code block",
			content:   "Here is my answer:\n```json\n{\"final_answer\": {\"summary\": \"done\"}}\n```",
			parsed:    nil,
			wantFinal: true,
		},
		{
			name:      "extract JSON from plain text",
			content:   `Some text before {"action": "my-tool", "action_input": {}} some text after`,
			parsed:    nil,
			wantFinal: false,
			wantTool:  "my-tool",
		},
		{
			name:      "fallback to raw content",
			content:   "I don't know",
			parsed:    nil,
			wantFinal: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			decision := parseReactDecision(tt.content, tt.parsed)
			if decision.IsFinal != tt.wantFinal {
				t.Errorf("IsFinal = %v, want %v", decision.IsFinal, tt.wantFinal)
			}
			if !tt.wantFinal && decision.Action != tt.wantTool {
				t.Errorf("Action = %q, want %q", decision.Action, tt.wantTool)
			}
		})
	}
}

func TestExtractJSONFromContent(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantNil bool
	}{
		{
			name:    "json code block",
			content: "```json\n{\"key\": \"value\"}\n```",
			wantNil: false,
		},
		{
			name:    "plain json in text",
			content: `result: {"key": "value"} done`,
			wantNil: false,
		},
		{
			name:    "no json",
			content: "no json here",
			wantNil: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractJSONFromContent(tt.content)
			if tt.wantNil && result != nil {
				t.Errorf("expected nil, got %v", result)
			}
			if !tt.wantNil && result == nil {
				t.Error("expected non-nil result")
			}
		})
	}
}

func TestBuildReactSystemPrompt(t *testing.T) {
	base := contract.PromptSpec{Template: "You are an EHS assistant."}
	artifact := contract.CompiledArtifact{
		Runner: contract.ArtifactRunner{
			Tools: map[string]contract.ToolSpec{
				"rectify-ticket-api": {Description: "Create rectification tickets"},
			},
			Knowledge: map[string]contract.KnowledgeSpec{
				"regulations": {Description: "Safety regulations database"},
			},
		},
	}

	prompt := buildReactSystemPrompt(base, artifact)

	if prompt == base.Template {
		t.Error("expected augmented prompt to be longer than base")
	}
	if !strings.Contains(prompt, "rectify-ticket-api") {
		t.Error("expected tool name in prompt")
	}
	if !strings.Contains(prompt, "regulations") {
		t.Error("expected knowledge name in prompt")
	}
	if !strings.Contains(prompt, "final_answer") {
		t.Error("expected ReAct format instructions in prompt")
	}
}

func TestExecuteReactLoopFinalAnswer(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		resp := map[string]interface{}{
			"id": "chatcmpl-1",
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": `{"final_answer": {"summary": "hazard identified", "risk_level": "high"}}`,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
				"payload": map[string]interface{}{
					"text": "配电箱门打开，地面有积水",
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Pattern: contract.ArtifactPattern{
				Type:          "react",
				ModelRef:      "planner",
				MaxIterations: 4,
			},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Output["pattern"] != "react" {
		t.Fatalf("expected pattern=react in output, got %#v", result.Output)
	}
	if result.Output["iterations"] != 1 {
		t.Fatalf("expected 1 iteration, got %v", result.Output["iterations"])
	}
	if atomic.LoadInt32(&callCount) != 1 {
		t.Fatalf("expected 1 model call, got %d", callCount)
	}
}

func TestExecuteReactLoopWithToolCall(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		var content string
		if n == 1 {
			// First call: request tool use.
			content = `{"action": "rectify-ticket-api", "action_input": {"hazard_id": "H-001", "action": "fix wiring"}}`
		} else {
			// Second call: final answer.
			content = `{"final_answer": {"summary": "ticket created", "ticket_id": "T-001"}}`
		}
		resp := map[string]interface{}{
			"id": "chatcmpl-1",
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": content,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	toolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"ticket_id": "T-001", "status": "created"})
	}))
	defer toolServer.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_and_fix",
				"payload": map[string]interface{}{
					"text": "配电箱门打开",
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Pattern: contract.ArtifactPattern{
				Type:          "react",
				ModelRef:      "planner",
				MaxIterations: 4,
			},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Type:        "http",
						Description: "Create rectification tickets",
						HTTP: map[string]interface{}{
							"url":    toolServer.URL + "/tickets",
							"method": "POST",
						},
					},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Output["iterations"] != 2 {
		t.Fatalf("expected 2 iterations, got %v", result.Output["iterations"])
	}
	if atomic.LoadInt32(&callCount) != 2 {
		t.Fatalf("expected 2 model calls, got %d", callCount)
	}

	// Verify reasoning trace.
	trace, ok := result.Output["reasoning"].([]reactStep)
	if !ok {
		t.Fatalf("expected reasoning trace, got %T", result.Output["reasoning"])
	}
	if len(trace) != 2 {
		t.Fatalf("expected 2 trace steps, got %d", len(trace))
	}
	if trace[0].Action != "rectify-ticket-api" {
		t.Fatalf("expected first step action=rectify-ticket-api, got %q", trace[0].Action)
	}
	if trace[1].FinalAnswer == nil {
		t.Fatal("expected second step to have final answer")
	}
}

func TestExecuteReactLoopMaxIterations(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	callCount := int32(0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		// Always request tool use — never give final answer.
		resp := map[string]interface{}{
			"id": "chatcmpl-1",
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": `{"action": "some-tool", "action_input": {"query": "test"}}`,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	toolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{"result": "ok"})
	}))
	defer toolServer.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "test",
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			Pattern: contract.ArtifactPattern{
				Type:          "react",
				ModelRef:      "planner",
				MaxIterations: 3,
			},
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
				},
				Tools: map[string]contract.ToolSpec{
					"some-tool": {
						Type: "http",
						HTTP: map[string]interface{}{
							"url":    toolServer.URL + "/query",
							"method": "POST",
						},
					},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	// Should have maxIterations tool calls + 1 final forced step.
	if result.Output["iterations"] != 4 {
		t.Fatalf("expected 4 steps (3 iterations + 1 forced), got %v", result.Output["iterations"])
	}
	if atomic.LoadInt32(&callCount) != 3 {
		t.Fatalf("expected 3 model calls (max iterations), got %d", callCount)
	}
}

func TestReactLoopSkipsWhenNoPattern(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"id": "chatcmpl-1",
			"choices": []interface{}{
				map[string]interface{}{
					"message": map[string]interface{}{
						"role":    "assistant",
						"content": `{"summary": "done"}`,
					},
				},
			},
		}
		_ = json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	runner := EinoADKRunner{Invoker: EinoOpenAIInvoker{}}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			AgentName:         "hazard-agent",
			AgentRunName:      "run-1",
			AgentRunNamespace: "ehs",
			AgentRevision:     "sha256:test",
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Engine: "eino", RunnerClass: "adk"},
			// No Pattern set — should fall through to single-model mode.
			Runner: contract.ArtifactRunner{
				Kind: "EinoADKRunner",
				Prompts: map[string]contract.PromptSpec{
					"system": {Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {Provider: "openai", Model: "gpt-4.1", BaseURL: server.URL},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	// Should NOT have react-specific output fields.
	if _, ok := result.Output["reasoning"]; ok {
		t.Fatal("expected single-model output, not ReAct output")
	}
}
