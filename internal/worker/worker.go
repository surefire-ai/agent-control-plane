package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

type Config struct {
	AgentName         string `json:"agentName"`
	AgentRunName      string `json:"agentRunName"`
	AgentRunNamespace string `json:"agentRunNamespace"`
	AgentRevision     string `json:"agentRevision"`
}

type Result struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Config    Config    `json:"config"`
	StartedAt time.Time `json:"startedAt"`
}

func ConfigFromEnv() Config {
	return Config{
		AgentName:         os.Getenv("AGENT_NAME"),
		AgentRunName:      os.Getenv("AGENT_RUN_NAME"),
		AgentRunNamespace: os.Getenv("AGENT_RUN_NAMESPACE"),
		AgentRevision:     os.Getenv("AGENT_REVISION"),
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
	return nil
}

func Run(ctx context.Context, config Config, writer io.Writer) error {
	if err := config.Validate(); err != nil {
		return err
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	result := Result{
		Status:    "succeeded",
		Message:   "agent control plane worker placeholder completed",
		Config:    config,
		StartedAt: time.Now().UTC(),
	}

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}
