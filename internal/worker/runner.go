package worker

import (
	"context"
	"time"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type Runner interface {
	Run(ctx context.Context, request RunRequest) (contract.WorkerResult, error)
}

type RunRequest struct {
	Config          Config
	Artifact        CompiledArtifact
	RuntimeIdentity contract.RuntimeIdentity
}

type EinoADKPlaceholderRunner struct{}

func (r EinoADKPlaceholderRunner) Run(ctx context.Context, request RunRequest) (contract.WorkerResult, error) {
	select {
	case <-ctx.Done():
		return contract.WorkerResult{}, ctx.Err()
	default:
	}

	return contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          "agent control plane worker placeholder completed",
		Config:           request.Config,
		CompiledArtifact: summarizeArtifact(request.Artifact),
		StartedAt:        time.Now().UTC(),
	}, nil
}

func runnerFor(identity contract.RuntimeIdentity) (Runner, error) {
	if err := identity.ValidateSupported(); err != nil {
		return nil, err
	}
	return EinoADKPlaceholderRunner{}, nil
}
