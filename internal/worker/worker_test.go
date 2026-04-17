package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunWritesStructuredResult(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:         "hazard-agent",
		AgentRunName:      "run-1",
		AgentRunNamespace: "ehs",
		AgentRevision:     "sha256:test",
	}

	if err := Run(context.Background(), config, &buffer); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var result Result
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatalf("worker output is not JSON: %v", err)
	}
	if result.Status != "succeeded" {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Config.AgentRunName != "run-1" {
		t.Fatalf("unexpected config: %#v", result.Config)
	}
}

func TestConfigValidateRequiresRunIdentity(t *testing.T) {
	err := Config{}.Validate()
	if err == nil {
		t.Fatal("expected validation error")
	}
	if !strings.Contains(err.Error(), "AGENT_NAME") {
		t.Fatalf("expected AGENT_NAME error, got %v", err)
	}
}
