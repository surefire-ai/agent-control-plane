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
