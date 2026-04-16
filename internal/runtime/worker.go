package runtime

import (
	"context"
	"errors"
)

var ErrWorkerRuntimeNotImplemented = errors.New("worker runtime backend is not implemented yet")

type WorkerRuntime struct{}

func (r WorkerRuntime) Execute(ctx context.Context, request Request) (Result, error) {
	return Result{}, ErrWorkerRuntimeNotImplemented
}
