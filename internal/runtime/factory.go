package runtime

import (
	"fmt"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backend string

const (
	BackendMock   Backend = "mock"
	BackendWorker Backend = "worker"
)

type Options struct {
	Backend    string
	Client     client.Client
	JobImage   string
	JobCommand []string
}

func NewRunner(options Options) (Runner, error) {
	backend := normalizeBackend(options.Backend)
	switch backend {
	case BackendMock:
		return NewMockRuntime(), nil
	case BackendWorker:
		if options.Client == nil {
			return nil, fmt.Errorf("worker runtime requires a Kubernetes client")
		}
		return NewWorkerRuntime(WorkerOptions{
			Client:  options.Client,
			Image:   options.JobImage,
			Command: options.JobCommand,
		}), nil
	default:
		return nil, fmt.Errorf("unsupported runtime backend %q", options.Backend)
	}
}

func normalizeBackend(value string) Backend {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return BackendMock
	}
	return Backend(trimmed)
}
