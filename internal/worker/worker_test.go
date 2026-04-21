package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

func TestRunWritesStructuredResult(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentRunInput:         `{"task":"identify_hazard","payload":{"text":"inspect line 3"}}`,
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","runtime":{"engine":"eino","runnerClass":"adk"},"runner":{"kind":"EinoADKRunner","models":{"planner":{"provider":"openai","model":"gpt-4.1","credentialRef":{"name":"openai-credentials","key":"apiKey"}}}},"policyRef":"ehs-policy"}`,
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
	if result.CompiledArtifact.RunnerClass != "adk" {
		t.Fatalf("unexpected runner class: %#v", result.CompiledArtifact)
	}
	if result.CompiledArtifact.PolicyRef != "ehs-policy" {
		t.Fatalf("unexpected policy ref: %#v", result.CompiledArtifact)
	}
	if result.Runtime == nil || result.Runtime.Runner == "" {
		t.Fatalf("expected runtime info, got %#v", result.Runtime)
	}
	if result.Runtime.Models["planner"].CredentialInjected != true {
		t.Fatalf("expected planner credentials to be marked injected, got %#v", result.Runtime.Models)
	}
	if result.Output["validatedModels"] != float64(1) {
		t.Fatalf("unexpected output payload: %#v", result.Output)
	}
	if result.Output["task"] != "identify_hazard" {
		t.Fatalf("unexpected task payload: %#v", result.Output)
	}
	inputKeys, ok := result.Output["inputKeys"].([]interface{})
	if !ok || len(inputKeys) != 2 {
		t.Fatalf("unexpected input keys: %#v", result.Output)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Name != "runtime-model-bindings" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
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
		AgentRunInput:         `{}`,
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

func TestRunDefaultsRunnerClass(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentRunInput:         `{}`,
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","runtime":{"engine":"eino"},"policyRef":"ehs-policy"}`,
	}

	if err := Run(context.Background(), config, &buffer); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var result contract.WorkerResult
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatalf("worker output is not JSON: %v", err)
	}
	if result.CompiledArtifact.RunnerClass != contract.RunnerClassADK {
		t.Fatalf("unexpected runner class: %#v", result.CompiledArtifact)
	}
}

func TestRunAcceptsV1RunnerArtifact(t *testing.T) {
	t.Setenv("MODEL_PLANNER_API_KEY", "test-secret")
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentRunInput:         `{"task":"identify_hazard"}`,
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","schemaVersion":"v1","runtime":{"engine":"eino","runnerClass":"adk","entrypoint":"ehs.hazard_identification"},"runner":{"kind":"EinoADKRunner","entrypoint":"ehs.hazard_identification","prompts":{"system":{"name":"system","language":"zh-CN","template":"hello"}},"models":{"planner":{"provider":"openai","model":"gpt-4.1","credentialRef":{"name":"openai-credentials","key":"apiKey"}}}},"policyRef":"ehs-policy"}`,
	}

	if err := Run(context.Background(), config, &buffer); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	var result contract.WorkerResult
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatalf("worker output is not JSON: %v", err)
	}
	if result.CompiledArtifact.RuntimeEngine != contract.RuntimeEngineEino {
		t.Fatalf("unexpected runtime engine: %#v", result.CompiledArtifact)
	}
	if result.CompiledArtifact.RunnerClass != contract.RunnerClassADK {
		t.Fatalf("unexpected runner class: %#v", result.CompiledArtifact)
	}
	if result.Runtime == nil || result.Runtime.Entrypoint != "ehs.hazard_identification" {
		t.Fatalf("unexpected runtime info: %#v", result.Runtime)
	}
}

func TestRunFailsWhenModelCredentialIsMissing(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentRunInput:         `{"task":"identify_hazard"}`,
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","runtime":{"engine":"eino","runnerClass":"adk"},"runner":{"kind":"EinoADKRunner","models":{"planner":{"provider":"openai","model":"gpt-4.1","credentialRef":{"name":"openai-credentials","key":"apiKey"}}}}}`,
	}

	err := Run(context.Background(), config, &buffer)
	if err == nil {
		t.Fatal("expected missing credentials error")
	}
	if !strings.Contains(err.Error(), "missing model credentials") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunRejectsUnsupportedRuntimeEngine(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentRunInput:         `{}`,
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","runtime":{"engine":"langgraph","runnerClass":"adk"},"policyRef":"ehs-policy"}`,
	}

	err := Run(context.Background(), config, &buffer)
	if err == nil {
		t.Fatal("expected unsupported runtime engine error")
	}
	if !strings.Contains(err.Error(), "unsupported runtime engine") {
		t.Fatalf("expected unsupported runtime error, got %v", err)
	}
}

func TestRunRejectsUnsupportedRunnerClass(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentRunInput:         `{}`,
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","runtime":{"engine":"eino","runnerClass":"custom"},"policyRef":"ehs-policy"}`,
	}

	err := Run(context.Background(), config, &buffer)
	if err == nil {
		t.Fatal("expected unsupported runner class error")
	}
	if !strings.Contains(err.Error(), "unsupported runner class") {
		t.Fatalf("expected unsupported runner class error, got %v", err)
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

func TestWriteFailurePreservesFailureReason(t *testing.T) {
	var buffer bytes.Buffer

	if err := WriteFailure(&buffer, FailureReasonError{
		Reason:  "MissingModelCredentials",
		Message: "missing model credentials for \"planner\" via MODEL_PLANNER_API_KEY",
	}); err != nil {
		t.Fatalf("WriteFailure returned error: %v", err)
	}

	var result contract.WorkerResult
	if err := json.Unmarshal(buffer.Bytes(), &result); err != nil {
		t.Fatalf("failure output is not JSON: %v", err)
	}
	if result.Reason != "MissingModelCredentials" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
	if result.Message == "" {
		t.Fatalf("expected failure message, got %#v", result)
	}
}

func TestRunRejectsInvalidRunInput(t *testing.T) {
	var buffer bytes.Buffer
	config := Config{
		AgentName:             "hazard-agent",
		AgentRunName:          "run-1",
		AgentRunNamespace:     "ehs",
		AgentRevision:         "sha256:test",
		AgentRunInput:         `{`,
		AgentCompiledArtifact: `{"apiVersion":"windosx.com/v1alpha1","kind":"AgentCompiledArtifact","runtime":{"engine":"eino","runnerClass":"adk"}}`,
	}

	err := Run(context.Background(), config, &buffer)
	if err == nil {
		t.Fatal("expected invalid run input error")
	}
	if !strings.Contains(err.Error(), "AGENT_RUN_INPUT must be valid JSON") {
		t.Fatalf("unexpected error: %v", err)
	}
	if failureReason(err) != "InvalidRunInput" {
		t.Fatalf("unexpected failure reason: %q", failureReason(err))
	}
}
