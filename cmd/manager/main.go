package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/surefire-ai/agent-control-plane/internal/manager"
)

func main() {
	config := manager.ConfigFromEnv()
	flag.StringVar(&config.Addr, "bind-address", config.Addr, "The address the manager HTTP server binds to.")
	flag.BoolVar(&config.AutoMigrate, "migrate-on-start", config.AutoMigrate, "Run built-in manager database migrations during startup when a database URL is configured.")
	flag.StringVar(&config.DatabaseDriver, "database-driver", config.DatabaseDriver, "Manager database driver name.")
	flag.StringVar(&config.DatabaseURL, "database-url", config.DatabaseURL, "Manager database URL. Optional for the current scaffold.")
	flag.StringVar(&config.Mode, "mode", config.Mode, "Manager operating mode.")
	flag.Parse()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := manager.Server{Config: config}
	if config.Mode == "fake" {
		server.Stores = manager.NewFakeStores()
	}
	if err := server.Start(ctx); err != nil {
		log.Printf("manager exited: %v", err)
		os.Exit(1)
	}
}
