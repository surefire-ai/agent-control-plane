package runtime

import (
	"fmt"
	"strings"
)

type Backend string

const (
	BackendMock   Backend = "mock"
	BackendWorker Backend = "worker"
)

type Options struct {
	Backend string
}

func NewRunner(options Options) (Runner, error) {
	backend := normalizeBackend(options.Backend)
	switch backend {
	case BackendMock:
		return NewMockRuntime(), nil
	case BackendWorker:
		return WorkerRuntime{}, nil
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
