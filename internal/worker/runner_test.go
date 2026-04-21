package worker

import (
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
