package contract

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type WorkerStatus string

const (
	WorkerStatusSucceeded WorkerStatus = "succeeded"
	WorkerStatusFailed    WorkerStatus = "failed"
)

type WorkerResult struct {
	Status           WorkerStatus    `json:"status"`
	Reason           string          `json:"reason,omitempty"`
	Message          string          `json:"message"`
	Config           interface{}     `json:"config,omitempty"`
	CompiledArtifact ArtifactSummary `json:"compiledArtifact,omitempty"`
	StartedAt        time.Time       `json:"startedAt,omitempty"`
}

type ArtifactSummary struct {
	APIVersion    string `json:"apiVersion,omitempty"`
	Kind          string `json:"kind,omitempty"`
	RuntimeEngine string `json:"runtimeEngine,omitempty"`
	RunnerClass   string `json:"runnerClass,omitempty"`
	PolicyRef     string `json:"policyRef,omitempty"`
}

func ParseWorkerResult(raw string) (WorkerResult, error) {
	var result WorkerResult
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&result); err != nil {
		return WorkerResult{}, fmt.Errorf("worker result must be valid JSON: %w", err)
	}
	if err := ensureSingleJSONDocument(decoder); err != nil {
		return WorkerResult{}, err
	}
	if result.Status == "" {
		return WorkerResult{}, fmt.Errorf("worker result status is required")
	}
	return result, nil
}

func WriteWorkerResult(writer io.Writer, result WorkerResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func ensureSingleJSONDocument(decoder *json.Decoder) error {
	var trailing interface{}
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("worker result must contain a single JSON document")
	} else if err != io.EOF {
		return fmt.Errorf("worker result has invalid trailing data: %w", err)
	}
	return nil
}
