package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/windosx/agent-control-plane/internal/contract"
)

type Config struct {
	AgentName              string           `json:"agentName"`
	AgentRunName           string           `json:"agentRunName"`
	AgentRunNamespace      string           `json:"agentRunNamespace"`
	AgentRevision          string           `json:"agentRevision"`
	AgentCompiledArtifact  string           `json:"-"`
	ParsedCompiledArtifact CompiledArtifact `json:"-"`
}

type CompiledArtifact struct {
	APIVersion string                 `json:"apiVersion,omitempty"`
	Kind       string                 `json:"kind,omitempty"`
	Runtime    map[string]interface{} `json:"runtime,omitempty"`
	PolicyRef  string                 `json:"policyRef,omitempty"`
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
	artifact, err := parseCompiledArtifact(config.AgentCompiledArtifact)
	if err != nil {
		return err
	}
	if err := validateRuntimeIdentity(artifact); err != nil {
		return err
	}
	config.ParsedCompiledArtifact = artifact

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	result := contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          "agent control plane worker placeholder completed",
		Config:           config,
		CompiledArtifact: summarizeArtifact(artifact),
		StartedAt:        time.Now().UTC(),
	}

	return contract.WriteWorkerResult(writer, result)
}

func WriteFailure(writer io.Writer, err error) error {
	result := contract.WorkerResult{
		Status:    contract.WorkerStatusFailed,
		Reason:    "WorkerFailed",
		Message:   err.Error(),
		StartedAt: time.Now().UTC(),
	}
	return contract.WriteWorkerResult(writer, result)
}

func parseCompiledArtifact(raw string) (CompiledArtifact, error) {
	var artifact CompiledArtifact
	if err := json.Unmarshal([]byte(raw), &artifact); err != nil {
		return CompiledArtifact{}, fmt.Errorf("AGENT_COMPILED_ARTIFACT must be valid JSON: %w", err)
	}
	if artifact.Kind == "" {
		return CompiledArtifact{}, fmt.Errorf("AGENT_COMPILED_ARTIFACT kind is required")
	}
	return artifact, nil
}

func summarizeArtifact(artifact CompiledArtifact) contract.ArtifactSummary {
	identity := runtimeIdentity(artifact.Runtime)
	return contract.ArtifactSummary{
		APIVersion:    artifact.APIVersion,
		Kind:          artifact.Kind,
		RuntimeEngine: identity.engine,
		RunnerClass:   identity.runnerClass,
		PolicyRef:     artifact.PolicyRef,
	}
}

func validateRuntimeIdentity(artifact CompiledArtifact) error {
	identity := runtimeIdentity(artifact.Runtime)
	if identity.engine != contract.RuntimeEngineEino {
		return fmt.Errorf("unsupported runtime engine %q", identity.engine)
	}
	if identity.runnerClass != contract.RunnerClassADK {
		return fmt.Errorf("unsupported runner class %q for runtime engine %q", identity.runnerClass, identity.engine)
	}
	return nil
}

type runtimeIdentityValue struct {
	engine      string
	runnerClass string
}

func runtimeIdentity(runtime map[string]interface{}) runtimeIdentityValue {
	identity := runtimeIdentityValue{
		engine:      contract.RuntimeEngineEino,
		runnerClass: contract.RunnerClassADK,
	}
	if engine := runtimeString(runtime, "engine"); engine != "" {
		identity.engine = engine
	}
	if runnerClass := runtimeString(runtime, "runnerClass"); runnerClass != "" {
		identity.runnerClass = runnerClass
	}
	return identity
}

func runtimeString(runtime map[string]interface{}, key string) string {
	value, ok := runtime[key]
	if !ok {
		return ""
	}
	output, _ := value.(string)
	return output
}
