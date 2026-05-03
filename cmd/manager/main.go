package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	"github.com/surefire-ai/korus/internal/manager"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiv1alpha1.AddToScheme(scheme))
}

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

	// Wire CRD syncer.
	if config.Mode == "fake" {
		server.Syncer = manager.NoopCRDSyncer{}
	} else if syncer, err := newK8sSyncer(); err == nil {
		server.Syncer = syncer
		log.Printf("CRD syncer enabled (K8s)")
	} else {
		log.Printf("CRD syncer disabled (no kubeconfig available): %v", err)
		server.Syncer = manager.NoopCRDSyncer{}
	}

	if err := server.Start(ctx); err != nil {
		log.Printf("manager exited: %v", err)
		os.Exit(1)
	}
}

// newK8sSyncer attempts to create a K8sCRDSyncer by discovering the kubeconfig
// and building a controller-runtime client. Returns an error if kubeconfig is
// not available (e.g., running outside a cluster without a kubeconfig file).
func newK8sSyncer() (*manager.K8sCRDSyncer, error) {
	restConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, err
	}
	c, err := client.New(restConfig, client.Options{Scheme: scheme})
	if err != nil {
		return nil, err
	}
	return manager.NewK8sCRDSyncer(c, scheme), nil
}
