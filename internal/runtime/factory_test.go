package runtime

import (
	"testing"

	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestNewRunnerDefaultsToMockRuntime(t *testing.T) {
	runner, err := NewRunner(Options{})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}
	if _, ok := runner.(MockRuntime); !ok {
		t.Fatalf("expected MockRuntime, got %T", runner)
	}
}

func TestNewRunnerNormalizesBackendName(t *testing.T) {
	runner, err := NewRunner(Options{Backend: " MOCK "})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}
	if _, ok := runner.(MockRuntime); !ok {
		t.Fatalf("expected MockRuntime, got %T", runner)
	}
}

func TestNewRunnerRejectsUnknownBackend(t *testing.T) {
	_, err := NewRunner(Options{Backend: "banana"})
	if err == nil {
		t.Fatal("expected unsupported backend error")
	}
}

func TestNewRunnerCreatesWorkerRuntime(t *testing.T) {
	runner, err := NewRunner(Options{
		Backend: string(BackendWorker),
		Client:  fake.NewClientBuilder().Build(),
	})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}
	if _, ok := runner.(WorkerRuntime); !ok {
		t.Fatalf("expected WorkerRuntime, got %T", runner)
	}
}

func TestNewRunnerRequiresClientForWorkerBackend(t *testing.T) {
	_, err := NewRunner(Options{Backend: string(BackendWorker)})
	if err == nil {
		t.Fatal("expected worker client requirement error")
	}
}
