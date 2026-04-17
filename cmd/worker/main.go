package main

import (
	"context"
	"fmt"
	"os"

	"github.com/windosx/agent-control-plane/internal/worker"
)

func main() {
	if err := worker.Run(context.Background(), worker.ConfigFromEnv(), os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "worker failed: %v\n", err)
		os.Exit(1)
	}
}
