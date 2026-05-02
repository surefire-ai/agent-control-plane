package worker

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestRunnerForReturnsEinoADKRunner(t *testing.T) {
	runner, err := runnerFor(contract.DefaultRuntimeIdentity())
	if err != nil {
		t.Fatalf("runnerFor returned error: %v", err)
	}
	if _, ok := runner.(EinoADKRunner); !ok {
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
			Skills: map[string]contract.SkillSpec{
				"risk-scoring": {
					Name:          "risk-scoring",
					Ref:           "ehs-risk-scoring-skill",
					Description:   "EHS风险评分能力",
					PromptRefs:    map[string]string{"system": "ehs-hazard-identification-system"},
					ToolRefs:      []string{"rectify-ticket-api"},
					KnowledgeRefs: []contract.CompiledSkillKnowledgeRef{{Name: "regulations", Ref: "ehs-regulations"}},
					Functions:     []string{"app.skills.ehs:score_risk_by_matrix"},
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
	if info.Skills["risk-scoring"].Ref != "ehs-risk-scoring-skill" || info.Skills["risk-scoring"].FunctionCount != 1 {
		t.Fatalf("expected skill runtime info, got %#v", info.Skills)
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
				Skills: map[string]contract.SkillSpec{
					"risk-scoring": {
						Name:       "risk-scoring",
						Ref:        "ehs-risk-scoring-skill",
						Functions:  []string{"app.skills.ehs:score_risk_by_matrix"},
						ToolRefs:   []string{"rectify-ticket-api"},
						PromptRefs: map[string]string{"system": "ehs-hazard-identification-system"},
					},
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
	if got := result.Output["resolvedSkills"]; got != 1 {
		t.Fatalf("expected resolvedSkills=1, got %#v", got)
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
				Skills: map[string]contract.SkillSpec{
					"risk-scoring": {
						Name:      "risk-scoring",
						Ref:       "ehs-risk-scoring-skill",
						Functions: []string{"app.skills.ehs:score_risk_by_matrix"},
					},
				},
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
				Skills: map[string]contract.SkillSpec{
					"risk-scoring": {
						Name:      "risk-scoring",
						Ref:       "ehs-risk-scoring-skill",
						Functions: []string{"app.skills.ehs:score_risk_by_matrix"},
					},
				},
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

func TestPlaceholderRunnerExecutesStepToolNode(t *testing.T) {
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ticketId":"T-103","status":"created"}`))
	}))
	defer server.Close()

	runner := EinoADKPlaceholderRunner{
		ToolInvoker: HTTPToolInvoker{Client: server.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
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
				Skills: map[string]contract.SkillSpec{
					"risk-scoring": {
						Name:      "risk-scoring",
						Ref:       "ehs-risk-scoring-skill",
						Functions: []string{"app.skills.ehs:score_risk_by_matrix"},
					},
				},
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
	step, ok := result.Output["step"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected step output, got %#v", result.Output)
	}
	if step["node"] != "create_ticket" || step["name"] != "rectify-ticket-api" {
		t.Fatalf("unexpected step tool metadata: %#v", step)
	}
}

func TestPlaceholderRunnerExecutesStepRetrievalNode(t *testing.T) {
	runner := EinoADKPlaceholderRunner{}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"node": "retrieve_regulations",
					"input": map[string]interface{}{
						"query": "危化品 储存 风险",
						"topK":  1,
					},
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Skills: map[string]contract.SkillSpec{
					"risk-scoring": {
						Name:      "risk-scoring",
						Ref:       "ehs-risk-scoring-skill",
						Functions: []string{"app.skills.ehs:score_risk_by_matrix"},
					},
				},
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
	step, ok := result.Output["step"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected step output, got %#v", result.Output)
	}
	if step["node"] != "retrieve_regulations" || step["name"] != "regulations" {
		t.Fatalf("unexpected step retrieval metadata: %#v", step)
	}
}

func TestPlaceholderRunnerExecutesStepSequence(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"step complete\",\"title\":\"Repair cabinet\",\"hazards\":[],\"overallRiskLevel\":\"low\",\"nextActions\":[],\"confidence\":0.88,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	toolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["title"] != "Repair cabinet" {
			t.Fatalf("expected title from prior llm step, got %#v", body)
		}
		_, _ = w.Write([]byte(`{"ticketId":"T-104","status":"created"}`))
	}))
	defer toolServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker:     EinoOpenAIInvoker{Client: modelServer.Client()},
		ToolInvoker: EinoToolInvoker{Client: toolServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"sequence": []interface{}{"classify_task", "create_ticket"},
					"input": map[string]interface{}{
						"text": "配电箱门未关闭",
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
							"name":     "classify_task",
							"kind":     "llm",
							"modelRef": "planner",
						},
						map[string]interface{}{
							"name":    "create_ticket",
							"kind":    "tool",
							"toolRef": "rectify-ticket-api",
						},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Name: "rectify-ticket-api",
						Type: "http",
						HTTP: map[string]interface{}{
							"url": toolServer.URL,
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
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	step, ok := result.Output["step"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected sequence step output, got %#v", result.Output)
	}
	if step["currentNode"] != "create_ticket" {
		t.Fatalf("unexpected currentNode: %#v", step)
	}
	switch steps := step["steps"].(type) {
	case []map[string]interface{}:
		if len(steps) != 2 {
			t.Fatalf("expected 2 steps, got %#v", step)
		}
	case []interface{}:
		if len(steps) != 2 {
			t.Fatalf("expected 2 steps, got %#v", step)
		}
	default:
		t.Fatalf("unexpected steps type: %T", step["steps"])
	}
	finalState, _ := step["finalState"].(map[string]interface{})
	toolOutput, _ := finalState["toolOutput"].(map[string]interface{})
	if toolOutput["ticketId"] != "T-104" {
		t.Fatalf("unexpected finalState tool output: %#v", finalState)
	}
	if finalState["title"] != "Repair cabinet" {
		t.Fatalf("expected llm result merged into finalState, got %#v", finalState)
	}
}

func TestPlaceholderRunnerExecutesAutoStepSequenceFromGraphEdges(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"step complete\",\"title\":\"Repair cabinet\",\"hazards\":[],\"overallRiskLevel\":\"low\",\"nextActions\":[],\"confidence\":0.88,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	toolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["title"] != "Repair cabinet" {
			t.Fatalf("expected title from prior llm step, got %#v", body)
		}
		_, _ = w.Write([]byte(`{"ticketId":"T-105","status":"created"}`))
	}))
	defer toolServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker:     EinoOpenAIInvoker{Client: modelServer.Client()},
		ToolInvoker: EinoToolInvoker{Client: toolServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"auto": true,
					"input": map[string]interface{}{
						"text": "配电箱门未关闭",
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
						map[string]interface{}{"name": "classify_task", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "create_ticket", "kind": "tool", "toolRef": "rectify-ticket-api"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "classify_task"},
						map[string]interface{}{"from": "classify_task", "to": "create_ticket"},
						map[string]interface{}{"from": "create_ticket", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Name: "rectify-ticket-api",
						Type: "http",
						HTTP: map[string]interface{}{
							"url": toolServer.URL,
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
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	step, ok := result.Output["step"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected auto step output, got %#v", result.Output)
	}
	sequence, _ := step["sequence"].([]string)
	if len(sequence) != 2 || sequence[0] != "classify_task" || sequence[1] != "create_ticket" {
		t.Fatalf("unexpected auto sequence: %#v", step)
	}
}

func TestPlaceholderRunnerAutoStepSequenceInjectsRetrievalContextIntoLLM(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	requestCount := 0
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode model request: %v", err)
		}
		messages, _ := body["messages"].([]interface{})
		if len(messages) < 2 {
			t.Fatalf("expected chat messages, got %#v", body)
		}
		userMessage, _ := messages[1].(map[string]interface{})
		content, _ := userMessage["content"].(string)
		if requestCount == 2 {
			if !strings.Contains(content, "有限空间 作业 风险") {
				t.Fatalf("expected retrieval query to come from payload text, got %q", content)
			}
			if !strings.Contains(content, "retrieved context 1") {
				t.Fatalf("expected retrieval context to be injected into llm input, got %q", content)
			}
		}
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"reasoned with retrieval\",\"hazards\":[],\"overallRiskLevel\":\"low\",\"nextActions\":[],\"confidence\":0.91,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker: EinoOpenAIInvoker{Client: modelServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"auto": true,
					"input": map[string]interface{}{
						"payload": map[string]interface{}{
							"text": "有限空间 作业 风险",
						},
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
						map[string]interface{}{"name": "retrieve_regulations", "kind": "retrieval", "knowledgeRef": "regulations"},
						map[string]interface{}{"name": "reason", "kind": "llm", "modelRef": "planner"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "retrieve_regulations"},
						map[string]interface{}{"from": "retrieve_regulations", "to": "reason"},
						map[string]interface{}{"from": "reason", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
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
						Retrieval: map[string]interface{}{
							"defaultTopK": float64(2),
						},
					},
				},
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	if requestCount != 1 {
		t.Fatalf("expected only the step llm call, got %d", requestCount)
	}
	step, _ := result.Output["step"].(map[string]interface{})
	finalState, _ := step["finalState"].(map[string]interface{})
	retrieval, _ := finalState["retrieval"].(map[string]interface{})
	if _, ok := retrieval["regulations"]; !ok {
		t.Fatalf("expected retrieval state keyed by binding name, got %#v", finalState)
	}
}

func TestPlaceholderRunnerAutoStepSequenceInjectsToolContextIntoFinalizeLLM(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	requestCount := 0
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode model request: %v", err)
		}
		messages, _ := body["messages"].([]interface{})
		if len(messages) < 2 {
			t.Fatalf("expected chat messages, got %#v", body)
		}
		userMessage, _ := messages[1].(map[string]interface{})
		content, _ := userMessage["content"].(string)
		switch requestCount {
		case 1:
			if strings.Contains(content, "T-106") {
				t.Fatalf("did not expect tool result before tool node ran, got %q", content)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"classified\",\"title\":\"Repair cabinet\",\"hazards\":[],\"overallRiskLevel\":\"high\",\"nextActions\":[],\"confidence\":0.9,\"needsHumanReview\":false}"}}]}`))
		case 2:
			if !strings.Contains(content, "T-106") {
				t.Fatalf("expected finalize llm input to contain tool result, got %q", content)
			}
			if !strings.Contains(content, "Repair cabinet") {
				t.Fatalf("expected finalize llm input to retain prior llm result, got %q", content)
			}
			_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"finalized\",\"hazards\":[],\"overallRiskLevel\":\"high\",\"nextActions\":[\"notify supervisor\"],\"confidence\":0.94,\"needsHumanReview\":true}"}}]}`))
		default:
			t.Fatalf("unexpected extra llm call #%d", requestCount)
		}
	}))
	defer modelServer.Close()

	toolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode tool request: %v", err)
		}
		if body["title"] != "Repair cabinet" {
			t.Fatalf("expected tool input to contain prior llm result, got %#v", body)
		}
		_, _ = w.Write([]byte(`{"ticketId":"T-106","status":"created"}`))
	}))
	defer toolServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker:     EinoOpenAIInvoker{Client: modelServer.Client()},
		ToolInvoker: EinoToolInvoker{Client: toolServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"auto": true,
					"input": map[string]interface{}{
						"text": "配电箱门未关闭",
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
						map[string]interface{}{"name": "classify_task", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "create_ticket", "kind": "tool", "toolRef": "rectify-ticket-api"},
						map[string]interface{}{"name": "finalize", "kind": "llm", "modelRef": "planner"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "classify_task"},
						map[string]interface{}{"from": "classify_task", "to": "create_ticket"},
						map[string]interface{}{"from": "create_ticket", "to": "finalize"},
						map[string]interface{}{"from": "finalize", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Name: "rectify-ticket-api",
						Type: "http",
						HTTP: map[string]interface{}{
							"url": toolServer.URL,
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
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	if requestCount != 2 {
		t.Fatalf("expected classify and finalize llm calls, got %d", requestCount)
	}
	step, _ := result.Output["step"].(map[string]interface{})
	finalState, _ := step["finalState"].(map[string]interface{})
	tools, _ := finalState["tools"].(map[string]interface{})
	if _, ok := tools["rectify-ticket-api"]; !ok {
		t.Fatalf("expected tools state keyed by tool name, got %#v", finalState)
	}
}

func TestPlaceholderRunnerAutoStepSequenceStopsOnFinalAnswerPattern(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	requestCount := 0
	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"done\",\"hazards\":[],\"overallRiskLevel\":\"low\",\"nextActions\":[],\"confidence\":0.92,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	toolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Fatal("tool node should not execute after final_answer stop")
		return
	}))
	defer toolServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker:     EinoOpenAIInvoker{Client: modelServer.Client()},
		ToolInvoker: EinoToolInvoker{Client: toolServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{"auto": true},
			},
		},
		Artifact: contract.CompiledArtifact{
			Pattern: contract.ArtifactPattern{
				Type:     "react",
				StopWhen: "final_answer",
			},
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{"name": "reason", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "finalize", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "create_ticket", "kind": "tool", "toolRef": "rectify-ticket-api"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "reason"},
						map[string]interface{}{"from": "reason", "to": "finalize"},
						map[string]interface{}{"from": "finalize", "to": "create_ticket"},
						map[string]interface{}{"from": "create_ticket", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Name: "rectify-ticket-api",
						Type: "http",
						HTTP: map[string]interface{}{"url": toolServer.URL},
					},
				},
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	if requestCount != 2 {
		t.Fatalf("expected reason and finalize llm calls only, got %d", requestCount)
	}
	step, _ := result.Output["step"].(map[string]interface{})
	if step["currentNode"] != "finalize" || step["stopReason"] != "final_answer" {
		t.Fatalf("expected final_answer stop metadata, got %#v", step)
	}
}

func TestPlaceholderRunnerAutoStepSequenceRejectsPatternMaxIterationsExceeded(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"loop\",\"hazards\":[],\"overallRiskLevel\":\"low\",\"nextActions\":[],\"confidence\":0.81,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker: EinoOpenAIInvoker{Client: modelServer.Client()},
	}
	_, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{"auto": true},
			},
		},
		Artifact: contract.CompiledArtifact{
			Pattern: contract.ArtifactPattern{
				Type:          "react",
				MaxIterations: 2,
			},
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{"name": "reason_a", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "reason_b", "kind": "llm", "modelRef": "planner"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "reason_a"},
						map[string]interface{}{"from": "reason_a", "to": "reason_b"},
						map[string]interface{}{"from": "reason_b", "to": "reason_a"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
						},
					},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err == nil {
		t.Fatal("expected maxIterations error")
	}
	if err.Error() != "automatic graph execution exceeded pattern.maxIterations=2" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceholderRunnerAutoStepSequenceMatchesInputCondition(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"ok\",\"hazards\":[],\"overallRiskLevel\":\"low\",\"nextActions\":[],\"confidence\":0.88,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker: EinoOpenAIInvoker{Client: modelServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"auto": true,
				},
				"payload": map[string]interface{}{
					"images": []interface{}{"s3://ehs/image-1.jpg"},
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
						map[string]interface{}{"name": "extract_facts", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "inspect_images", "kind": "llm", "modelRef": "planner"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "extract_facts"},
						map[string]interface{}{"from": "extract_facts", "to": "inspect_images", "when": "input.payload.images != null && len(input.payload.images) > 0"},
						map[string]interface{}{"from": "inspect_images", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	step, _ := result.Output["step"].(map[string]interface{})
	sequence, _ := step["sequence"].([]string)
	if len(sequence) != 2 || sequence[1] != "inspect_images" {
		t.Fatalf("expected conditional image path, got %#v", step)
	}
}

func TestPlaceholderRunnerAutoStepSequenceMatchesFinalResponseCondition(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")
	t.Setenv("TOOL_RECTIFY_TICKET_API_AUTH_TOKEN", "test-token")

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"reviewed\",\"hazards\":[],\"overallRiskLevel\":\"high\",\"nextActions\":[],\"confidence\":0.88,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	toolServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"ticketId":"T-106","status":"created"}`))
	}))
	defer toolServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker:     EinoOpenAIInvoker{Client: modelServer.Client()},
		ToolInvoker: EinoToolInvoker{Client: toolServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{"auto": true},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{"name": "review_output", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "create_ticket", "kind": "tool", "toolRef": "rectify-ticket-api"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "review_output"},
						map[string]interface{}{"from": "review_output", "to": "create_ticket", "when": "finalResponse.overallRiskLevel in ['high', 'critical']"},
						map[string]interface{}{"from": "review_output", "to": "END"},
						map[string]interface{}{"from": "create_ticket", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Tools: map[string]contract.ToolSpec{
					"rectify-ticket-api": {
						Name: "rectify-ticket-api",
						Type: "http",
						HTTP: map[string]interface{}{
							"url": toolServer.URL,
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
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	step, _ := result.Output["step"].(map[string]interface{})
	sequence, _ := step["sequence"].([]string)
	if len(sequence) != 2 || sequence[1] != "create_ticket" {
		t.Fatalf("expected high-risk branch to create ticket, got %#v", step)
	}
}

func TestRequestedStepRejectsInvalidSequence(t *testing.T) {
	_, _, err := requestedStep(map[string]interface{}{
		"step": map[string]interface{}{
			"sequence": "classify_task",
		},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "step.sequence must be an array of node names" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRequestedStepRequiresNodeSequenceOrAuto(t *testing.T) {
	_, _, err := requestedStep(map[string]interface{}{
		"step": map[string]interface{}{},
	})
	if err == nil {
		t.Fatal("expected validation error")
	}
	if err.Error() != "step.node, step.sequence, or step.auto is required" {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceholderRunnerRejectsUnsupportedStepNodeKind(t *testing.T) {
	runner := EinoADKPlaceholderRunner{}
	_, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"node": "classify_task",
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
							"name": "classify_task",
							"kind": "function",
						},
					},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `graph node "classify_task" is missing implementation` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceholderRunnerExecutesStepFunctionNode(t *testing.T) {
	runner := EinoADKPlaceholderRunner{}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"node": "score_risk",
					"input": map[string]interface{}{
						"hazards": []interface{}{
							map[string]interface{}{"riskLevel": "medium"},
							map[string]interface{}{"riskLevel": "high"},
						},
					},
				},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Skills: map[string]contract.SkillSpec{
					"risk-scoring": {
						Name:      "risk-scoring",
						Ref:       "ehs-risk-scoring-skill",
						Functions: []string{"app.skills.ehs:score_risk_by_matrix"},
					},
				},
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{
							"name":           "score_risk",
							"kind":           "function",
							"implementation": "app.skills.ehs:score_risk_by_matrix",
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
	step, ok := result.Output["step"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected function step output, got %#v", result.Output)
	}
	if step["node"] != "score_risk" || step["implementation"] != "app.skills.ehs:score_risk_by_matrix" {
		t.Fatalf("unexpected function step metadata: %#v", step)
	}
	if step["skillBinding"] != "risk-scoring" || step["skillRef"] != "ehs-risk-scoring-skill" {
		t.Fatalf("expected declared skill metadata, got %#v", step)
	}
	if step["skill"] != "ehs" || step["function"] != "score_risk_by_matrix" {
		t.Fatalf("expected builtin skill metadata, got %#v", step)
	}
	parsed, _ := step["result"].(map[string]interface{})
	if parsed["overallRiskLevel"] != "high" {
		t.Fatalf("unexpected function result: %#v", step)
	}
}

func TestResolveBuiltinSkillFunctionRejectsUnknownSkill(t *testing.T) {
	_, _, _, err := resolveBuiltinSkillFunction("app.skills.unknown:do_work")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `skill "unknown" is not supported yet` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveDeclaredSkillFunctionRejectsUndeclaredImplementation(t *testing.T) {
	_, _, err := resolveDeclaredSkillFunction(contract.CompiledArtifact{
		Runner: contract.ArtifactRunner{
			Skills: map[string]contract.SkillSpec{
				"ticketing": {Name: "ticketing", Ref: "ehs-ticketing-skill", Functions: []string{"app.skills.ehs:create_ticket"}},
			},
		},
	}, "app.skills.ehs:score_risk_by_matrix")
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `function implementation "app.skills.ehs:score_risk_by_matrix" is not declared by any resolved skill` {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestPlaceholderRunnerAutoStepSequenceExecutesFunctionNode(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	modelServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"identified\",\"hazards\":[{\"riskLevel\":\"high\"}],\"overallRiskLevel\":\"high\",\"nextActions\":[],\"confidence\":0.88,\"needsHumanReview\":false}"}}]}`))
	}))
	defer modelServer.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker: EinoOpenAIInvoker{Client: modelServer.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{"auto": true},
			},
		},
		Artifact: contract.CompiledArtifact{
			Runtime: contract.ArtifactRuntime{Entrypoint: "ehs.hazard_identification"},
			Runner: contract.ArtifactRunner{
				Kind:       "EinoADKRunner",
				Entrypoint: "ehs.hazard_identification",
				Skills: map[string]contract.SkillSpec{
					"risk-scoring": {
						Name:      "risk-scoring",
						Ref:       "ehs-risk-scoring-skill",
						Functions: []string{"app.skills.ehs:score_risk_by_matrix"},
					},
				},
				Graph: map[string]interface{}{
					"nodes": []interface{}{
						map[string]interface{}{"name": "identify_hazards", "kind": "llm", "modelRef": "planner"},
						map[string]interface{}{"name": "score_risk", "kind": "function", "implementation": "app.skills.ehs:score_risk_by_matrix"},
					},
					"edges": []interface{}{
						map[string]interface{}{"from": "START", "to": "identify_hazards"},
						map[string]interface{}{"from": "identify_hazards", "to": "score_risk"},
						map[string]interface{}{"from": "score_risk", "to": "END"},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       modelServer.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	step, _ := result.Output["step"].(map[string]interface{})
	finalState, _ := step["finalState"].(map[string]interface{})
	if finalState["overallRiskLevel"] != "high" {
		t.Fatalf("expected function result merged into state, got %#v", finalState)
	}
}

func TestPlaceholderRunnerExecutesStepLLMNode(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"{\"summary\":\"step complete\",\"hazards\":[],\"overallRiskLevel\":\"low\",\"nextActions\":[],\"confidence\":0.88,\"needsHumanReview\":false}"}}]}`))
	}))
	defer server.Close()

	runner := EinoADKPlaceholderRunner{
		Invoker: EinoOpenAIInvoker{Client: server.Client()},
	}
	result, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"node": "classify_task",
					"input": map[string]interface{}{
						"text": "配电箱门未关闭",
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
							"name":     "classify_task",
							"kind":     "llm",
							"modelRef": "planner",
						},
					},
				},
				Prompts: map[string]contract.PromptSpec{
					"system": {Name: "system", Template: "You are an EHS assistant."},
				},
				Models: map[string]contract.ModelConfig{
					"planner": {
						Provider:      "openai",
						Model:         "gpt-4.1",
						BaseURL:       server.URL,
						CredentialRef: &contract.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					},
				},
				Output: map[string]interface{}{
					"schema": map[string]interface{}{
						"type": "object",
						"required": []interface{}{
							"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview",
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
	step, ok := result.Output["step"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected step output, got %#v", result.Output)
	}
	if step["node"] != "classify_task" || step["model"] != "planner" {
		t.Fatalf("unexpected llm step metadata: %#v", step)
	}
	parsed, _ := step["result"].(map[string]interface{})
	if parsed["summary"] != "step complete" {
		t.Fatalf("unexpected llm step result: %#v", step)
	}
}

func TestPlaceholderRunnerRejectsStepLLMNodeWithoutModelRef(t *testing.T) {
	runner := EinoADKPlaceholderRunner{}
	_, err := runner.Run(context.Background(), RunRequest{
		Config: Config{
			ParsedRunInput: map[string]interface{}{
				"step": map[string]interface{}{
					"node": "classify_task",
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
							"name": "classify_task",
							"kind": "llm",
						},
					},
				},
			},
		},
		RuntimeIdentity: contract.DefaultRuntimeIdentity(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if err.Error() != `graph node "classify_task" is missing modelRef` {
		t.Fatalf("unexpected error: %v", err)
	}
}
