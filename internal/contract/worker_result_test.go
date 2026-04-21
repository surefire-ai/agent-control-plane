package contract

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseWorkerResultAcceptsSucceededResult(t *testing.T) {
	result, err := ParseWorkerResult(`{
  "status": "succeeded",
  "message": "done",
  "output": {
    "summary": "validated runtime"
  },
  "artifacts": [
    {
      "name": "model-runtime",
      "kind": "json",
      "inline": {
        "models": 1
      }
    }
  ],
  "runtime": {
    "engine": "eino",
    "runnerClass": "adk",
    "runner": "EinoADKPlaceholderRunner"
  },
  "compiledArtifact": {
    "kind": "AgentCompiledArtifact",
    "runtimeEngine": "eino",
    "runnerClass": "adk"
  }
}`)
	if err != nil {
		t.Fatalf("ParseWorkerResult returned error: %v", err)
	}
	if result.Status != WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.CompiledArtifact.RuntimeEngine != "eino" {
		t.Fatalf("unexpected artifact summary: %#v", result.CompiledArtifact)
	}
	if result.CompiledArtifact.RunnerClass != "adk" {
		t.Fatalf("unexpected artifact summary: %#v", result.CompiledArtifact)
	}
	if result.Output["summary"] != "validated runtime" {
		t.Fatalf("unexpected output: %#v", result.Output)
	}
	if len(result.Artifacts) != 1 || result.Artifacts[0].Name != "model-runtime" {
		t.Fatalf("unexpected artifacts: %#v", result.Artifacts)
	}
	if result.Runtime == nil || result.Runtime.Runner != "EinoADKPlaceholderRunner" {
		t.Fatalf("unexpected runtime info: %#v", result.Runtime)
	}
}

func TestParseWorkerResultAcceptsFailedResult(t *testing.T) {
	result, err := ParseWorkerResult(`{
  "status": "failed",
  "reason": "WorkerFailed",
  "message": "boom"
}`)
	if err != nil {
		t.Fatalf("ParseWorkerResult returned error: %v", err)
	}
	if result.Status != WorkerStatusFailed {
		t.Fatalf("unexpected status: %q", result.Status)
	}
	if result.Reason != "WorkerFailed" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
}

func TestParseWorkerResultIgnoresUnknownFields(t *testing.T) {
	result, err := ParseWorkerResult(`{
  "status": "succeeded",
  "message": "done",
  "future": {"tokenUsage": 12}
}`)
	if err != nil {
		t.Fatalf("ParseWorkerResult returned error: %v", err)
	}
	if result.Message != "done" {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestParseWorkerResultRejectsMissingStatus(t *testing.T) {
	_, err := ParseWorkerResult(`{"message":"missing status"}`)
	if err == nil || !strings.Contains(err.Error(), "status") {
		t.Fatalf("expected status error, got %v", err)
	}
}

func TestParseWorkerResultRejectsMultipleDocuments(t *testing.T) {
	_, err := ParseWorkerResult(`{"status":"succeeded"} {"status":"failed"}`)
	if err == nil || !strings.Contains(err.Error(), "single JSON document") {
		t.Fatalf("expected single document error, got %v", err)
	}
}

func TestWriteWorkerResultWritesJSON(t *testing.T) {
	var buffer bytes.Buffer

	err := WriteWorkerResult(&buffer, WorkerResult{
		Status:  WorkerStatusSucceeded,
		Message: "done",
	})
	if err != nil {
		t.Fatalf("WriteWorkerResult returned error: %v", err)
	}
	result, err := ParseWorkerResult(buffer.String())
	if err != nil {
		t.Fatalf("written result did not parse: %v", err)
	}
	if result.Status != WorkerStatusSucceeded {
		t.Fatalf("unexpected status: %q", result.Status)
	}
}
