package runtime

import (
	"context"
	"errors"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
)

var ErrRuntimeInProgress = errors.New("runtime execution is still in progress")

type Request struct {
	Agent apiv1alpha1.Agent
	Run   apiv1alpha1.AgentRun
}

type Result struct {
	Output   apiv1alpha1.FreeformObject
	TraceRef apiv1alpha1.FreeformObject
	Reason   string
	Message  string
}

type Failure struct {
	Output   apiv1alpha1.FreeformObject
	TraceRef apiv1alpha1.FreeformObject
	Reason   string
	Message  string
	Err      error
}

func (e Failure) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Err != nil {
		return e.Err.Error()
	}
	return "runtime failed"
}

func (e Failure) Unwrap() error {
	return e.Err
}

type Runner interface {
	Execute(ctx context.Context, request Request) (Result, error)
}
