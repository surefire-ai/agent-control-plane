package main

import (
	"flag"
	"os"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	"github.com/surefire-ai/korus/internal/controller"
	"github.com/surefire-ai/korus/internal/gateway"
	agentruntime "github.com/surefire-ai/korus/internal/runtime"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(apiv1alpha1.AddToScheme(scheme))
}

func main() {
	var metricsAddr string
	var probeAddr string
	var gatewayAddr string
	var enableLeaderElection bool
	var runtimeBackend string
	var workerJobImage string
	var workerJobCommand string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.StringVar(&gatewayAddr, "gateway-bind-address", ":8082", "The address the invoke gateway binds to. Set empty to disable.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.StringVar(&runtimeBackend, "runtime-backend", string(agentruntime.BackendMock), "AgentRun runtime backend to use: mock or worker.")
	flag.StringVar(&workerJobImage, "worker-job-image", "", "Container image for the worker runtime Kubernetes Job.")
	flag.StringVar(&workerJobCommand, "worker-job-command", "", "Shell command for the worker runtime Kubernetes Job.")
	opts := zap.Options{Development: true}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	config := ctrl.GetConfigOrDie()
	mgr, err := ctrl.NewManager(config, ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "korus.windosx.com",
	})
	if err != nil {
		ctrl.Log.Error(err, "unable to start manager")
		os.Exit(1)
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		ctrl.Log.Error(err, "unable to create Kubernetes clientset")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		ctrl.Log.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		ctrl.Log.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	runner, err := agentruntime.NewRunner(agentruntime.Options{
		Backend:    runtimeBackend,
		Client:     mgr.GetClient(),
		Clientset:  clientset,
		JobImage:   workerJobImage,
		JobCommand: shellCommand(workerJobCommand),
	})
	if err != nil {
		ctrl.Log.Error(err, "unable to create AgentRun runtime")
		os.Exit(1)
	}

	if err := (&controller.AgentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "unable to create Agent controller")
		os.Exit(1)
	}

	if err := (&controller.AgentRunReconciler{
		Client:  mgr.GetClient(),
		Scheme:  mgr.GetScheme(),
		Runtime: runner,
	}).SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "unable to create AgentRun controller")
		os.Exit(1)
	}

	if err := (&controller.AgentEvaluationReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "unable to create AgentEvaluation controller")
		os.Exit(1)
	}

	if err := (&controller.TenantReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "unable to create Tenant controller")
		os.Exit(1)
	}

	if err := (&controller.WorkspaceReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		ctrl.Log.Error(err, "unable to create Workspace controller")
		os.Exit(1)
	}

	if gatewayAddr != "" {
		if err := mgr.Add(gateway.Server{
			Addr:   gatewayAddr,
			Client: mgr.GetClient(),
		}); err != nil {
			ctrl.Log.Error(err, "unable to create invoke gateway")
			os.Exit(1)
		}
	}

	ctrl.Log.Info("starting agent control plane manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		ctrl.Log.Error(err, "problem running manager")
		os.Exit(1)
	}
}

func shellCommand(command string) []string {
	if command == "" {
		return nil
	}
	return []string{"sh", "-c", command}
}
