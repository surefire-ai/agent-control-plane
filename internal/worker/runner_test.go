package worker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

func TestRunnerForReturnsEinoADKPlaceholderRunner(t *testing.T) {
	runner, err := runnerFor(contract.DefaultRuntimeIdentity())
	if err != nil {
		t.Fatalf("runnerFor returned error: %v", err)
	}
	if _, ok := runner.(EinoADKPlaceholderRunner); !ok {
		t.Fatalf("unexpected runner type: %T", runner)
	}
}

func TestRunnerForRejectsUnsupportedIdentity(t *testing.T) {
	_, err := runnerFor(contract.RuntimeIdentity{
		Engine:      contract.RuntimeEngineEino,
		RunnerClass: "custom",
	})
	if err == nil {
		t.Fatal("expected unsupported identity error")
	}
}

func TestPrimaryModelConfigPrefersPlanner(t *testing.T) {
	name, model, ok := primaryModelConfig(contract.CompiledArtifact{
		Runner: contract.ArtifactRunner{
			Models: map[string]contract.ModelConfig{
				"extractor": {Provider: "openai", Model: "gpt-4.1-mini"},
				"planner":   {Provider: "openai", Model: "gpt-4.1"},
			},
		},
	})
	if !ok {
		t.Fatal("expected primary model config")
	}
	if name != "planner" || model.Model != "gpt-4.1" {
		t.Fatalf("expected planner model to be selected, got %q %#v", name, model)
	}
}

func TestRuntimeInfoForArtifactIncludesToolsAndKnowledge(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "dummy")

	info, artifacts, err := runtimeInfoForArtifact(contract.CompiledArtifact{
		Runner: contract.ArtifactRunner{
			Models: map[string]contract.ModelConfig{
				"planner": {
					Provider:      "openai",
					Model:         "gpt-4.1",
					BaseURL:       "http://mock-openai:8080",
					CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
				},
			},
			Tools: map[string]contract.ToolSpec{
				"vision-inspection-tool": {
					Name:        "vision-inspection-tool",
					Type:        "multimodal",
					Description: "图片巡检工具",
					Runtime:     map[string]interface{}{"provider": "internal-runtime"},
				},
			},
			Knowledge: map[string]contract.KnowledgeSpec{
				"regulations": {
					Name:        "regulations",
					Ref:         "ehs-regulations",
					Description: "法规库",
					Sources:     []map[string]interface{}{{"name": "source-a", "uri": "s3://bucket/a"}},
					Binding:     map[string]interface{}{"retrieval": map[string]interface{}{"topK": float64(5)}},
					Retrieval:   map[string]interface{}{"defaultTopK": float64(5), "defaultScoreThreshold": 0.72},
				},
			},
		},
	}, contract.DefaultRuntimeIdentity())
	if err != nil {
		t.Fatalf("runtimeInfoForArtifact returned error: %v", err)
	}
	if info.Tools["vision-inspection-tool"].Type != "multimodal" {
		t.Fatalf("expected tool runtime info, got %#v", info.Tools)
	}
	if info.Knowledge["regulations"].Ref != "ehs-regulations" || info.Knowledge["regulations"].SourceCount != 1 {
		t.Fatalf("expected knowledge runtime info, got %#v", info.Knowledge)
	}
	if len(artifacts) != 2 {
		t.Fatalf("expected model and dependency artifacts, got %#v", artifacts)
	}
}

func TestPlaceholderRunnerReportsResolvedDependencies(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "dummy")

	runner := EinoADKPlaceholderRunner{}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"task": "identify_hazard",
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {Name: "rectify-ticket-api", Type: "http", HTTP: map[string]interface{}{"url": "https://example.internal"}},
				},
				Knowledge: map[string]contract.KnowledgeSpec{
					"cases": {Name: "cases", Ref: "ehs-hazard-cases", Sources: []map[string]interface{}{{"name": "a"}}},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if got := result.Output["resolvedTools"]; got != 1 {
		t.Fatalf("expected resolvedTools=1, got %#v", got)
	}
	if got := result.Output["resolvedKnowledge"]; got != 1 {
		t.Fatalf("expected resolvedKnowledge=1, got %#v", got)
	}
}

func TestPlaceholderRunnerExecutesRequestedTool(t *testing.T) {
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ticketId":"T-101","status":"created"}`))
	}))
	defer server.Close()

	runner := EinoADKPlaceholderRunner{
		ToolInvoker: HTTPToolInvoker{Client: server.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"toolCall": map[string]interface{}{
					"name": "rectify-ticket-api",
					"input": map[string]interface{}{
						"title": "Repair cabinet",
					},
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Name: "rectify-ticket-api",
						Type: "http",
						HTTP: map[string]interface{}{
							"url": server.URL,
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
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	toolCall, ok := result.Output["toolCall"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected toolCall output, got %#v", result.Output)
	}
	output, _ := toolCall["output"].(map[string]interface{})
	if output["ticketId"] != "T-101" {
		t.Fatalf("unexpected tool output: %#v", toolCall)
	}
}

func TestPlaceholderRunnerExecutesGraphToolNode(t *testing.T) {
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ticketId":"T-102","status":"created"}`))
	}))
	defer server.Close()

	runner := EinoADKPlaceholderRunner{
		ToolInvoker: HTTPToolInvoker{Client: server.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"toolCall": map[string]interface{}{
					"node": "create_ticket",
					"input": map[string]interface{}{
						"title": "Repair cabinet",
					},
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{
							"name":    "create_ticket",
							"kind":    "tool",
							"toolRef": "rectify-ticket-api",
						},
					},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Name: "rectify-ticket-api",
						Type: "http",
						HTTP: map[string]interface{}{
							"url": server.URL,
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
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	toolCall, ok := result.Output["toolCall"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected toolCall output, got %#v", result.Output)
	}
	if toolCall["name"] != "rectify-ticket-api" || toolCall["node"] != "create_ticket" {
		t.Fatalf("unexpected graph tool metadata: %#v", toolCall)
	}
	output, _ := toolCall["output"].(map[string]interface{})
	if output["ticketId"] != "T-102" {
		t.Fatalf("unexpected graph tool output: %#v", toolCall)
	}
}

func TestRequestedToolCallRejectsMissingNameAndNode(t *testing.T) {
	_, _, err := requestedToolCall(map[string]interface{}{
		"toolCall": map[string]interface{}{
			"input": map[string]interface{}{},
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "toolCall.name or toolCall.node is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceholderRunnerExecutesRequestedRetrieval(t *testing.T) {
	runner := EinoADKPlaceholderRunner{}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"retrievalCall": map[string]interface{}{
					"name":  "regulations",
					"query": "配电箱 积水 触电 风险",
					"topK":  2,
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Knowledge: map[string]contract.KnowledgeSpec{
					"regulations": {
						Name:        "regulations",
						Ref:         "ehs-regulations",
						Description: "法规库",
						Sources: []map[string]interface{}{
							{"name": "国家安全生产法规", "uri": "s3://ehs-kb/regulations/national/"},
							{"name": "企业内部EHS制度", "uri": "s3://ehs-kb/regulations/internal/"},
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
	retrievalCall, ok := result.Output["retrievalCall"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retrievalCall output, got %#v", result.Output)
	}
	results, _ := retrievalCall["results"].([]map[string]interface{})
	if len(results) != 2 {
		// json-ish map values from runtime path still stay typed here
		rawResults, _ := retrievalCall["results"].([]interface{})
		if len(rawResults) != 2 {
			t.Fatalf("expected 2 retrieval results, got %#v", retrievalCall)
		}
	}
}

func TestPlaceholderRunnerExecutesGraphRetrievalNode(t *testing.T) {
	runner := EinoADKPlaceholderRunner{}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"retrievalCall": map[string]interface{}{
					"node":  "retrieve_regulations",
					"query": "有限空间 作业 风险",
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{
							"name":         "retrieve_regulations",
							"kind":         "retrieval",
							"knowledgeRef": "regulations",
						},
					},
				},
				Knowledge: map[string]contract.KnowledgeSpec{
					"regulations": {
						Name:        "regulations",
						Ref:         "ehs-regulations",
						Description: "法规库",
						Sources: []map[string]interface{}{
							{"name": "国家安全生产法规", "uri": "s3://ehs-kb/regulations/national/"},
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
	retrievalCall, ok := result.Output["retrievalCall"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected retrievalCall output, got %#v", result.Output)
	}
	if retrievalCall["name"] != "regulations" || retrievalCall["node"] != "retrieve_regulations" {
		t.Fatalf("unexpected graph retrieval metadata: %#v", retrievalCall)
	}
}

func TestRequestedRetrievalCallRejectsMissingQuery(t *testing.T) {
	_, _, err := requestedRetrievalCall(map[string]interface{}{
		"retrievalCall": map[string]interface{}{
			"name": "regulations",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "retrievalCall.query is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}
