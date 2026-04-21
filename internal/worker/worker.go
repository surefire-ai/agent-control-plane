package worker

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type Config struct {
	AgentName              string                    `json:"agentName"`
	AgentRunName           string                    `json:"agentRunName"`
	AgentRunNamespace      string                    `json:"agentRunNamespace"`
	AgentRevision          string                    `json:"agentRevision"`
	AgentCompiledArtifact  string                    `json:"-"`
	ParsedCompiledArtifact contract.CompiledArtifact `json:"-"`
}

func ConfigFromEnv() Config {
	return Config{
		AgentName:             os.Getenv("AGENT_NAME"),
		AgentRunName:          os.Getenv("AGENT_RUN_NAME"),
		AgentRunNamespace:     os.Getenv("AGENT_RUN_NAMESPACE"),
		AgentRevision:         os.Getenv("AGENT_REVISION"),
		AgentCompiledArtifact: os.Getenv("AGENT_COMPILED_ARTIFACT"),
	}
}

func (c Config) Validate() error {
	if c.AgentName == "" {
		return fmt.Errorf("AGENT_NAME is required")
	}
	if c.AgentRunName == "" {
		return fmt.Errorf("AGENT_RUN_NAME is required")
	}
	if c.AgentRunNamespace == "" {
		return fmt.Errorf("AGENT_RUN_NAMESPACE is required")
	}
	if c.AgentRevision == "" {
		return fmt.Errorf("AGENT_REVISION is required")
	}
	if c.AgentCompiledArtifact == "" {
		return fmt.Errorf("AGENT_COMPILED_ARTIFACT is required")
	}
	return nil
}

func Run(ctx context.Context, config Config, writer io.Writer) error {
	if err := config.Validate(); err != nil {
		return err
	}
	artifact, err := contract.ParseCompiledArtifact(config.AgentCompiledArtifact)
	if err != nil {
		return err
	}
	identity := artifact.RuntimeIdentity()
	runner, err := runnerFor(identity)
	if err != nil {
		return err
	}
	config.ParsedCompiledArtifact = artifact

	result, err := runner.Run(ctx, RunRequest{
		Config:          config,
		Artifact:        artifact,
		RuntimeIdentity: identity,
	})
	if err != nil {
		return err
	}

	return contract.WriteWorkerResult(writer, result)
}

func WriteFailure(writer io.Writer, err error) error {
	result := contract.WorkerResult{
		Status:    contract.WorkerStatusFailed,
		Reason:    failureReason(err),
		Message:   err.Error(),
		StartedAt: time.Now().UTC(),
	}
	return contract.WriteWorkerResult(writer, result)
}

func summarizeArtifact(artifact contract.CompiledArtifact) contract.ArtifactSummary {
	return artifact.Summary()
}

func failureReason(err error) string {
	var reasoned FailureReasonError
	if errors.As(err, &reasoned) && reasoned.Reason != "" {
		return reasoned.Reason
	}
	return "WorkerFailed"
}
