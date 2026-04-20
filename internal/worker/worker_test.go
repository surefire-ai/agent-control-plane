package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/windosx/agent-control-plane/internal/contract"
)

func TestRunWritesStructuredResult(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","runtime":{"engine":"eino"},"policyRef":"ehs-policy"}`,
	}

	if err := Run(context.Background(), config, &buffer); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var result contract.WorkerResult
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatalf("worker output is not JSON: %v", err)
	}
	if result.Status != contract.WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Config == nil {
		t.Fatalf("unexpected config: %#v", result.Config)
	}
	if result.CompiledArtifact.Kind != "AgentCompiledArtifact" {
		t.Fatalf("unexpected compiled artifact summary: %#v", result.CompiledArtifact)
	}
	if result.CompiledArtifact.RuntimeEngine != "eino" {
		t.Fatalf("unexpected runtime engine: %#v", result.CompiledArtifact)
	}
	if result.CompiledArtifact.PolicyRef != "ehs-policy" {
		t.Fatalf("unexpected policy ref: %#v", result.CompiledArtifact)
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

func TestRunRejectsInvalidCompiledArtifact(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentCompiledArtifact: `{`,
	}

	err := Run(context.Background(), config, &buffer)
	if err == nil {
		t.Fatal("expected invalid compiled artifact error")
	}
	if !strings.Contains(err.Error(), "AGENT_COMPILED_ARTIFACT") {
		t.Fatalf("expected compiled artifact error, got %v", err)
	}
}

func TestWriteFailureWritesStructuredResult(t *testing.T) {
	var buffer bytes.Buffer

	if err := WriteFailure(&buffer, context.Canceled); err != nil {
		t.Fatalf("WriteFailure returned error: %v", err)
	}

	var result contract.WorkerResult
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatalf("failure output is not JSON: %v", err)
	}
	if result.Status != contract.WorkerStatusFailed {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Reason != "WorkerFailed" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
	if result.Message == "" {
		t.Fatalf("expected failure message, got %#v", result)
	}
}
