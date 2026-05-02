package worker

import (
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestIsToolCallingPattern(t *testing.T) {
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
			name: "pattern.type tool_calling",
			artifact: contract.CompiledArtifact{
				Pattern: contract.ArtifactPattern{Type: "tool_calling"},
			},
			want: true,
		},
		{
			name: "runner.pattern.type tool_calling",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "tool_calling"},
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
			name: "runner.pattern.type workflow",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": "workflow"},
				},
			},
			want: false,
		},
		{
			name: "runner.pattern non-string type",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Pattern: map[string]interface{}{"type": 123},
				},
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isToolCallingPattern(tt.artifact); got != tt.want {
				t.Errorf("isToolCallingPattern() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildToolDefinitions(t *testing.T) {
	tests := []struct {
		name        string
		artifact    contract.CompiledArtifact
		runtimeInfo contract.WorkerRuntimeInfo
		wantCount   int
		wantFirst   string
	}{
		{
			name:        "no tools",
			artifact:    contract.CompiledArtifact{},
			runtimeInfo: contract.WorkerRuntimeInfo{},
			wantCount:   0,
		},
		{
			name: "tool with matching runtime",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Tools: map[string]contract.ToolSpec{
						"create-ticket": {
							Description: "Create a support ticket",
						},
					},
				},
			},
			runtimeInfo: contract.WorkerRuntimeInfo{
				Tools: map[string]contract.WorkerToolRuntime{
					"create-ticket": {Type: "http"},
				},
			},
			wantCount: 1,
			wantFirst: "create-ticket",
		},
		{
			name: "tool without matching runtime is skipped",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Tools: map[string]contract.ToolSpec{
						"orphan-tool": {Description: "No runtime entry"},
					},
				},
			},
			runtimeInfo: contract.WorkerRuntimeInfo{
				Tools: map[string]contract.WorkerToolRuntime{},
			},
			wantCount: 0,
		},
		{
			name: "tool with empty runtime type is skipped",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Tools: map[string]contract.ToolSpec{
						"bad-tool": {Description: "Empty type"},
					},
				},
			},
			runtimeInfo: contract.WorkerRuntimeInfo{
				Tools: map[string]contract.WorkerToolRuntime{
					"bad-tool": {Type: ""},
				},
			},
			wantCount: 0,
		},
		{
			name: "tool with schema",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Tools: map[string]contract.ToolSpec{
						"search": {
							Description: "Search database",
							Schema: map[string]interface{}{
								"type": "object",
								"properties": map[string]interface{}{
									"query": map[string]interface{}{"type": "string"},
								},
							},
						},
					},
				},
			},
			runtimeInfo: contract.WorkerRuntimeInfo{
				Tools: map[string]contract.WorkerToolRuntime{
					"search": {Type: "http"},
				},
			},
			wantCount: 1,
			wantFirst: "search",
		},
		{
			name: "multiple tools",
			artifact: contract.CompiledArtifact{
				Runner: contract.ArtifactRunner{
					Tools: map[string]contract.ToolSpec{
						"tool-a": {Description: "Tool A"},
						"tool-b": {Description: "Tool B"},
					},
				},
			},
			runtimeInfo: contract.WorkerRuntimeInfo{
				Tools: map[string]contract.WorkerToolRuntime{
					"tool-a": {Type: "http"},
					"tool-b": {Type: "http"},
				},
			},
			wantCount: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defs := buildToolDefinitions(tt.artifact, tt.runtimeInfo)
			if len(defs) != tt.wantCount {
				t.Fatalf("buildToolDefinitions() returned %d defs, want %d", len(defs), tt.wantCount)
			}
			if tt.wantFirst != "" && len(defs) > 0 {
				if defs[0].Function.Name != tt.wantFirst {
					// Order may vary for maps; check existence instead.
					found := false
					for _, d := range defs {
						if d.Function.Name == tt.wantFirst {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("expected tool %q in definitions", tt.wantFirst)
					}
				}
				if defs[0].Type != "function" {
					t.Errorf("expected type 'function', got %q", defs[0].Type)
				}
			}
		})
	}
}

func TestBuildToolDefinitionsWithSchema(t *testing.T) {
	schema := map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"hazard_id": map[string]interface{}{"type": "string"},
		},
		"required": []interface{}{"hazard_id"},
	}
	artifact := contract.CompiledArtifact{
		Runner: contract.ArtifactRunner{
			Tools: map[string]contract.ToolSpec{
				"rectify": {
					Description: "Rectify a hazard",
					Schema:      schema,
				},
			},
		},
	}
	runtimeInfo := contract.WorkerRuntimeInfo{
		Tools: map[string]contract.WorkerToolRuntime{
			"rectify": {Type: "http"},
		},
	}

	defs := buildToolDefinitions(artifact, runtimeInfo)
	if len(defs) != 1 {
		t.Fatalf("expected 1 definition, got %d", len(defs))
	}
	if defs[0].Function.Parameters == nil {
		t.Fatal("expected parameters to be set from schema")
	}
	params, ok := defs[0].Function.Parameters.(map[string]interface{})
	if !ok {
		t.Fatalf("expected parameters to be map, got %T", defs[0].Function.Parameters)
	}
	if params["type"] != "object" {
		t.Errorf("expected schema type=object, got %v", params["type"])
	}
}

func TestParseToolCallingResponse(t *testing.T) {
	tests := []struct {
		name          string
		result        ModelInvocationResult
		wantToolCalls int
		wantFirstID   string
		wantFirstFunc string
		wantContent   string
	}{
		{
			name: "no tool calls",
			result: ModelInvocationResult{
				Content: "Here is the final answer.",
				ResponseBody: map[string]interface{}{
					"choices": []interface{}{
						map[string]interface{}{
							"message": map[string]interface{}{
								"role":    "assistant",
								"content": "Here is the final answer.",
							},
						},
					},
				},
			},
			wantToolCalls: 0,
			wantContent:   "Here is the final answer.",
		},
		{
			name: "with tool calls",
			result: ModelInvocationResult{
				Content: "",
				ResponseBody: map[string]interface{}{
					"choices": []interface{}{
						map[string]interface{}{
							"message": map[string]interface{}{
								"role": "assistant",
								"tool_calls": []interface{}{
									map[string]interface{}{
										"id":   "call_abc123",
										"type": "function",
										"function": map[string]interface{}{
											"name":      "create-ticket",
											"arguments": `{"hazard_id": "H-001"}`,
										},
									},
								},
							},
						},
					},
				},
			},
			wantToolCalls: 1,
			wantFirstID:   "call_abc123",
			wantFirstFunc: "create-ticket",
			wantContent:   "",
		},
		{
			name: "multiple tool calls",
			result: ModelInvocationResult{
				Content: "",
				ResponseBody: map[string]interface{}{
					"choices": []interface{}{
						map[string]interface{}{
							"message": map[string]interface{}{
								"role": "assistant",
								"tool_calls": []interface{}{
									map[string]interface{}{
										"id":   "call_1",
										"type": "function",
										"function": map[string]interface{}{
											"name":      "tool-a",
											"arguments": `{}`,
										},
									},
									map[string]interface{}{
										"id":   "call_2",
										"type": "function",
										"function": map[string]interface{}{
											"name":      "tool-b",
											"arguments": `{"key": "value"}`,
										},
									},
								},
							},
						},
					},
				},
			},
			wantToolCalls: 2,
			wantFirstID:   "call_1",
			wantFirstFunc: "tool-a",
		},
		{
			name: "nil response body",
			result: ModelInvocationResult{
				Content:      "fallback content",
				ResponseBody: nil,
			},
			wantToolCalls: 0,
			wantContent:   "fallback content",
		},
		{
			name: "empty choices",
			result: ModelInvocationResult{
				Content: "some content",
				ResponseBody: map[string]interface{}{
					"choices": []interface{}{},
				},
			},
			wantToolCalls: 0,
			wantContent:   "some content",
		},
		{
			name: "content updated from response message",
			result: ModelInvocationResult{
				Content: "original",
				ResponseBody: map[string]interface{}{
					"choices": []interface{}{
						map[string]interface{}{
							"message": map[string]interface{}{
								"role":    "assistant",
								"content": "updated content from response",
							},
						},
					},
				},
			},
			wantToolCalls: 0,
			wantContent:   "updated content from response",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := parseToolCallingResponse(tt.result)
			if len(msg.ToolCalls) != tt.wantToolCalls {
				t.Fatalf("ToolCalls count = %d, want %d", len(msg.ToolCalls), tt.wantToolCalls)
			}
			if tt.wantFirstID != "" && msg.ToolCalls[0].ID != tt.wantFirstID {
				t.Errorf("ToolCalls[0].ID = %q, want %q", msg.ToolCalls[0].ID, tt.wantFirstID)
			}
			if tt.wantFirstFunc != "" && msg.ToolCalls[0].Function.Name != tt.wantFirstFunc {
				t.Errorf("ToolCalls[0].Function.Name = %q, want %q", msg.ToolCalls[0].Function.Name, tt.wantFirstFunc)
			}
			if tt.wantContent != "" && msg.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", msg.Content, tt.wantContent)
			}
			if msg.Role != "assistant" {
				t.Errorf("Role = %q, want %q", msg.Role, "assistant")
			}
		})
	}
}

func TestGetStringField(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]interface{}
		key  string
		want string
	}{
		{
			name: "existing string field",
			m:    map[string]interface{}{"name": "create-ticket"},
			key:  "name",
			want: "create-ticket",
		},
		{
			name: "missing field",
			m:    map[string]interface{}{},
			key:  "name",
			want: "",
		},
		{
			name: "non-string field",
			m:    map[string]interface{}{"count": 42},
			key:  "count",
			want: "",
		},
		{
			name: "nil value",
			m:    map[string]interface{}{"key": nil},
			key:  "key",
			want: "",
		},
		{
			name: "empty string",
			m:    map[string]interface{}{"key": ""},
			key:  "key",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := getStringField(tt.m, tt.key); got != tt.want {
				t.Errorf("getStringField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestResolveAPIKey(t *testing.T) {
	tests := []struct {
		name      string
		model     contract.WorkerModelRuntime
		modelName string
		envVal    string
		wantKey   string
		wantErr   bool
		wantRsn   string
	}{
		{
			name: "valid api key",
			model: contract.WorkerModelRuntime{
				APIKeyEnv: "TEST_API_KEY",
			},
			modelName: "gpt-4",
			envVal:    "sk-test-secret",
			wantKey:   "sk-test-secret",
			wantErr:   false,
		},
		{
			name: "empty env var name",
			model: contract.WorkerModelRuntime{
				APIKeyEnv: "",
			},
			modelName: "gpt-4",
			wantErr:   true,
			wantRsn:   "MissingModelCredentials",
		},
		{
			name: "env var not set",
			model: contract.WorkerModelRuntime{
				APIKeyEnv: "NONEXISTENT_KEY_VAR",
			},
			modelName: "gpt-4",
			envVal:    "",
			wantErr:   true,
			wantRsn:   "MissingModelCredentials",
		},
		{
			name: "api key with whitespace is trimmed",
			model: contract.WorkerModelRuntime{
				APIKeyEnv: "WHITESPACE_KEY",
			},
			modelName: "gpt-4",
			envVal:    "  sk-padded  ",
			wantKey:   "sk-padded",
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Override lookupEnv to control env var resolution.
			origLookup := lookupEnv
			lookupEnv = func(key string) string {
				if key == tt.model.APIKeyEnv {
					return tt.envVal
				}
				return ""
			}
			defer func() { lookupEnv = origLookup }()

			got, err := resolveAPIKey(tt.model, tt.modelName)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				fre, ok := err.(FailureReasonError)
				if !ok {
					t.Fatalf("expected FailureReasonError, got %T", err)
				}
				if fre.Reason != tt.wantRsn {
					t.Errorf("Reason = %q, want %q", fre.Reason, tt.wantRsn)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.wantKey {
				t.Errorf("resolveAPIKey() = %q, want %q", got, tt.wantKey)
			}
		})
	}
}

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name  string
		model contract.WorkerModelRuntime
		want  string
	}{
		{
			name:  "custom base URL",
			model: contract.WorkerModelRuntime{BaseURL: "https://custom.api.com/v1"},
			want:  "https://custom.api.com/v1",
		},
		{
			name:  "custom base URL with trailing slash",
			model: contract.WorkerModelRuntime{BaseURL: "https://custom.api.com/v1/"},
			want:  "https://custom.api.com/v1",
		},
		{
			name:  "empty base URL returns default",
			model: contract.WorkerModelRuntime{BaseURL: ""},
			want:  "https://api.openai.com/v1",
		},
		{
			name:  "multiple trailing slashes stripped",
			model: contract.WorkerModelRuntime{BaseURL: "https://example.com///"},
			want:  "https://example.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveBaseURL(tt.model); got != tt.want {
				t.Errorf("resolveBaseURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
