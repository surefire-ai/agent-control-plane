package runtime

import (
	"fmt"
	"strings"

	"github.com/surefire-ai/korus/internal/artifact"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Backend string

const (
	BackendMock   Backend = "mock"
	BackendWorker Backend = "worker"
)

type Options struct {
	Backend       string
	Client        client.Client
	Clientset     kubernetes.Interface
	JobImage      string
	JobCommand    []string
	ArtifactStore artifact.Store
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
			Client:        options.Client,
			Clientset:     options.Clientset,
			Image:         options.JobImage,
			Command:       options.JobCommand,
			ArtifactStore: options.ArtifactStore,
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
