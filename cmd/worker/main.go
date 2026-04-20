package main

import (
	"context"
	"os"

	"github.com/surefire-ai/agent-control-plane/internal/worker"
)

func main() {
	if err := worker.Run(context.Background(), worker.ConfigFromEnv(), os.Stdout); err != nil {
		if writeErr := worker.WriteFailure(os.Stdout, err); writeErr != nil {
			os.Exit(1)
		}
		os.Exit(1)
	}
}
