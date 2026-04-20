package contract

import (
	"strings"
	"testing"
)

func TestRuntimeIdentityFromSpecDefaultsToEinoADK(t *testing.T) {
	identity := RuntimeIdentityFromSpec(RuntimeSpec{})

	if identity.Engine != RuntimeEngineEino {
		t.Fatalf("unexpected engine: %#v", identity)
	}
	if identity.RunnerClass != RunnerClassADK {
		t.Fatalf("unexpected runner class: %#v", identity)
	}
}

func TestRuntimeIdentityFromMapDefaultsToEinoADK(t *testing.T) {
	identity := RuntimeIdentityFromMap(map[string]interface{}{})

	if identity.Engine != RuntimeEngineEino {
		t.Fatalf("unexpected engine: %#v", identity)
	}
	if identity.RunnerClass != RunnerClassADK {
		t.Fatalf("unexpected runner class: %#v", identity)
	}
}

func TestRuntimeIdentityFromMapAcceptsExplicitValues(t *testing.T) {
	identity := RuntimeIdentityFromMap(map[string]interface{}{
		"engine":      "eino",
		"runnerClass": "adk",
	})

	if identity.Engine != RuntimeEngineEino {
		t.Fatalf("unexpected engine: %#v", identity)
	}
	if identity.RunnerClass != RunnerClassADK {
		t.Fatalf("unexpected runner class: %#v", identity)
	}
}

func TestRuntimeIdentityRejectsUnsupportedEngine(t *testing.T) {
	err := RuntimeIdentity{
		Engine:      "langgraph",
		RunnerClass: RunnerClassADK,
	}.ValidateSupported()

	if err == nil || !strings.Contains(err.Error(), "unsupported runtime engine") {
		t.Fatalf("expected unsupported runtime engine error, got %v", err)
	}
}

func TestRuntimeIdentityRejectsUnsupportedRunnerClass(t *testing.T) {
	err := RuntimeIdentity{
		Engine:      RuntimeEngineEino,
		RunnerClass: "custom",
	}.ValidateSupported()

	if err == nil || !strings.Contains(err.Error(), "unsupported runner class") {
		t.Fatalf("expected unsupported runner class error, got %v", err)
	}
}
