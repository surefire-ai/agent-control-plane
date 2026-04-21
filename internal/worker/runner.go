package worker

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type Runner interface {
	Run(ctx context.Context, request RunRequest) (contract.WorkerResult, error)
}

type RunRequest struct {
	Config          Config
	Artifact        contract.CompiledArtifact
	RuntimeIdentity contract.RuntimeIdentity
}

type EinoADKPlaceholderRunner struct{}

type FailureReasonError struct {
	Reason  string
	Message string
}

func (e FailureReasonError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Reason != "" {
		return e.Reason
	}
	return "worker failed"
}

func (r EinoADKPlaceholderRunner) Run(ctx context.Context, request RunRequest) (contract.WorkerResult, error) {
	select {
	case <-ctx.Done():
		return contract.WorkerResult{}, ctx.Err()
	default:
	}

	runtimeInfo, artifacts, err := runtimeInfoForArtifact(request.Artifact, request.RuntimeIdentity)
	if err != nil {
		return contract.WorkerResult{}, err
	}

	modelCount := len(runtimeInfo.Models)
	message := "agent control plane worker placeholder completed"
	if modelCount > 0 {
		message = fmt.Sprintf("agent control plane worker placeholder validated %d model binding(s)", modelCount)
	}

	return contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          message,
		Config:           request.Config,
		CompiledArtifact: summarizeArtifact(request.Artifact),
		Output: map[string]interface{}{
			"summary":           message,
			"validatedModels":   modelCount,
			"runtimeEntrypoint": runtimeInfo.Entrypoint,
		},
		Artifacts: artifacts,
		Runtime:   &runtimeInfo,
		StartedAt: time.Now().UTC(),
	}, nil
}

func runnerFor(identity contract.RuntimeIdentity) (Runner, error) {
	if err := identity.ValidateSupported(); err != nil {
		return nil, err
	}
	return EinoADKPlaceholderRunner{}, nil
}

func runtimeInfoForArtifact(artifact contract.CompiledArtifact, identity contract.RuntimeIdentity) (contract.WorkerRuntimeInfo, []contract.WorkerArtifact, error) {
	info := contract.WorkerRuntimeInfo{
		Engine:      identity.Engine,
		RunnerClass: identity.RunnerClass,
		Runner:      artifact.Runner.Kind,
		Entrypoint:  artifact.Runner.Entrypoint,
		Models:      make(map[string]contract.WorkerModelRuntime, len(artifact.Runner.Models)),
	}
	if info.Runner == "" {
		info.Runner = "EinoADKPlaceholderRunner"
	}
	if info.Entrypoint == "" {
		info.Entrypoint = artifact.Runtime.Entrypoint
	}

	modelNames := sortedModelNames(artifact.Runner.Models, artifact.Models)
	for _, name := range modelNames {
		model := artifact.Runner.Models[name]
		if model.Provider == "" && model.Model == "" && model.CredentialRef == nil && model.BaseURL == "" {
			model = artifact.Models[name]
		}

		apiKeyEnv := modelAPIKeyEnvName(name)
		modelRuntime := contract.WorkerModelRuntime{
			Provider:  model.Provider,
			Model:     model.Model,
			BaseURL:   model.BaseURL,
			APIKeyEnv: apiKeyEnv,
		}
		if model.CredentialRef != nil {
			if os.Getenv(apiKeyEnv) == "" {
				return contract.WorkerRuntimeInfo{}, nil, FailureReasonError{
					Reason:  "MissingModelCredentials",
					Message: fmt.Sprintf("missing model credentials for %q via %s", name, apiKeyEnv),
				}
			}
			modelRuntime.CredentialInjected = true
		}
		info.Models[name] = modelRuntime
	}

	artifacts := []contract.WorkerArtifact{
		{
			Name: "runtime-model-bindings",
			Kind: "json",
			Inline: map[string]interface{}{
				"models": info.Models,
			},
		},
	}
	return info, artifacts, nil
}

func sortedModelNames(modelSets ...map[string]contract.ModelConfig) []string {
	seen := map[string]struct{}{}
	for _, models := range modelSets {
		for name := range models {
			seen[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func modelAPIKeyEnvName(name string) string {
	return modelEnvPrefix(name) + "_API_KEY"
}

func modelEnvPrefix(name string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case 'a' <= r && r <= 'z':
			builder.WriteRune(r - ('a' - 'A'))
			lastUnderscore = false
		case 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
			builder.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore:
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	prefix := strings.Trim(builder.String(), "_")
	if prefix == "" {
		prefix = "MODEL"
	}
	return "MODEL_" + prefix
}
