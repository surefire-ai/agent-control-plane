package runtime

import (
	"context"
	"errors"
	"testing"
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

func TestWorkerRuntimeIsExplicitlyUnimplemented(t *testing.T) {
	runner, err := NewRunner(Options{Backend: string(BackendWorker)})
	if err != nil {
		t.Fatalf("NewRunner returned error: %v", err)
	}
	_, err = runner.Execute(context.Background(), Request{})
	if !errors.Is(err, ErrWorkerRuntimeNotImplemented) {
		t.Fatalf("expected ErrWorkerRuntimeNotImplemented, got %v", err)
	}
}
