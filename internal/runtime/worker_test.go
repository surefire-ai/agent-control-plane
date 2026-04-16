package runtime

import (
	"context"
	"errors"
	"strings"
	"testing"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestWorkerRuntimeCreatesJobAndReportsInProgress(t *testing.T) {
	scheme := testRuntimeScheme(t)
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	worker := NewWorkerRuntime(WorkerOptions{
		Client:  kubeClient,
		Image:   "busybox:test",
		Command: []string{"sh", "-c", "echo test"},
	})
	request := workerRequest()

	_, err := worker.Execute(context.Background(), request)
	if !errors.Is(err, ErrRuntimeInProgress) {
		t.Fatalf("expected ErrRuntimeInProgress, got %v", err)
	}

	var job batchv1.Job
	key := types.NamespacedName{Namespace: "ehs", Name: jobNameForRun(request.Run)}
	if err := kubeClient.Get(context.Background(), key, &job); err != nil {
		t.Fatalf("expected worker Job to be created: %v", err)
	}
	if job.Spec.Template.Spec.Containers[0].Image != "busybox:test" {
		t.Fatalf("unexpected worker image: %q", job.Spec.Template.Spec.Containers[0].Image)
	}
	if job.Labels["windosx.com/agentrun"] != request.Run.Name {
		t.Fatalf("expected AgentRun label, got %#v", job.Labels)
	}
}

func TestWorkerRuntimeReturnsResultWhenJobSucceeded(t *testing.T) {
	scheme := testRuntimeScheme(t)
	request := workerRequest()
	jobName := jobNameForRun(request.Run)
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: "ehs"},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&job).Build()
	worker := NewWorkerRuntime(WorkerOptions{Client: kubeClient})

	result, err := worker.Execute(context.Background(), request)
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}
	if result.Reason != "WorkerJobSucceeded" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
	if JSONString(result.TraceRef, "provider") != "kubernetes-job" {
		t.Fatalf("unexpected trace provider: %#v", result.TraceRef)
	}
}

func TestWorkerRuntimeReturnsErrorWhenJobFailed(t *testing.T) {
	scheme := testRuntimeScheme(t)
	request := workerRequest()
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobNameForRun(request.Run), Namespace: "ehs"},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Message: "pod failed",
				},
			},
		},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&job).Build()
	worker := NewWorkerRuntime(WorkerOptions{Client: kubeClient})

	_, err := worker.Execute(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "pod failed") {
		t.Fatalf("expected job failure error, got %v", err)
	}
}

func TestJobNameForRunIsDNSLabelSafe(t *testing.T) {
	run := apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "Run_With_A_Very_Long_Name_That_Should_Be_Shortened_Before_It_Becomes_A_Job_Name",
			Namespace: "ehs",
		},
	}
	name := jobNameForRun(run)
	if len(name) > 63 {
		t.Fatalf("expected job name to fit DNS label length, got %d: %s", len(name), name)
	}
	if strings.Contains(name, "_") {
		t.Fatalf("expected DNS-safe job name, got %q", name)
	}
}

func testRuntimeScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := batchv1.AddToScheme(scheme); err != nil {
		t.Fatalf("batch AddToScheme returned error: %v", err)
	}
	return scheme
}

func workerRequest() Request {
	return Request{
		Agent: apiv1alpha1.Agent{
			ObjectMeta: metav1.ObjectMeta{Name: "hazard-agent"},
			Status: apiv1alpha1.AgentStatus{
				CompiledRevision: "sha256:agent",
			},
		},
		Run: apiv1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "run-1",
				Namespace: "ehs",
				UID:       types.UID("run-uid"),
			},
		},
	}
}
