package worker

import (
	"context"
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
