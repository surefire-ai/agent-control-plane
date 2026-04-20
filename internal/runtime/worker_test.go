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
	if job.Spec.Template.Spec.Containers[0].ImagePullPolicy != corev1.PullIfNotPresent {
		t.Fatalf("unexpected worker image pull policy: %q", job.Spec.Template.Spec.Containers[0].ImagePullPolicy)
	}
	if job.Labels["windosx.com/agentrun"] != request.Run.Name {
		t.Fatalf("expected AgentRun label, got %#v", job.Labels)
	}
	if job.Spec.Template.Spec.SecurityContext == nil || job.Spec.Template.Spec.SecurityContext.RunAsUser == nil || *job.Spec.Template.Spec.SecurityContext.RunAsUser != 65532 {
		t.Fatalf("expected worker Job to run as nonroot UID 65532, got %#v", job.Spec.Template.Spec.SecurityContext)
	}
	if job.Spec.Template.Spec.Containers[0].SecurityContext == nil || job.Spec.Template.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation == nil || *job.Spec.Template.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation {
		t.Fatalf("expected worker container privilege escalation to be disabled")
	}
	if envValue(job.Spec.Template.Spec.Containers[0].Env, "AGENT_COMPILED_ARTIFACT") == "" {
		t.Fatal("expected worker Job to receive AGENT_COMPILED_ARTIFACT")
	}
	if !strings.Contains(envValue(job.Spec.Template.Spec.Containers[0].Env, "AGENT_COMPILED_ARTIFACT"), "AgentCompiledArtifact") {
		t.Fatalf("unexpected compiled artifact env: %q", envValue(job.Spec.Template.Spec.Containers[0].Env, "AGENT_COMPILED_ARTIFACT"))
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
	worker := NewWorkerRuntime(WorkerOptions{
		Client: kubeClient,
		LogReader: staticPodLogReader{
			logs: PodLogs{
				PodName:       jobName + "-pod",
				ContainerName: workerContainerName,
				Text:          workerResultLog(),
			},
		},
	})

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
	if JSONString(result.TraceRef, "podName") != jobName+"-pod" {
		t.Fatalf("unexpected trace pod: %#v", result.TraceRef)
	}
	if JSONString(result.Output, "summary") != "agent control plane worker placeholder completed" {
		t.Fatalf("unexpected output summary: %#v", result.Output)
	}
	if result.Output["compiledArtifact"].Raw == nil {
		t.Fatalf("expected compiled artifact summary in output: %#v", result.Output)
	}
}

func TestWorkerRuntimeRejectsInvalidWorkerResult(t *testing.T) {
	scheme := testRuntimeScheme(t)
	request := workerRequest()
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobNameForRun(request.Run), Namespace: "ehs"},
		Status: batchv1.JobStatus{
			Succeeded: 1,
		},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&job).Build()
	worker := NewWorkerRuntime(WorkerOptions{
		Client: kubeClient,
		LogReader: staticPodLogReader{
			logs: PodLogs{
				PodName:       job.Name + "-pod",
				ContainerName: workerContainerName,
				Text:          `{`,
			},
		},
	})

	_, err := worker.Execute(context.Background(), request)
	if err == nil || !strings.Contains(err.Error(), "worker result") {
		t.Fatalf("expected worker result error, got %v", err)
	}
}

func TestWorkerRuntimeReturnsStructuredFailureWhenWorkerFailed(t *testing.T) {
	scheme := testRuntimeScheme(t)
	request := workerRequest()
	jobName := jobNameForRun(request.Run)
	job := batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{Name: jobName, Namespace: "ehs"},
		Status: batchv1.JobStatus{
			Conditions: []batchv1.JobCondition{
				{
					Type:    batchv1.JobFailed,
					Status:  corev1.ConditionTrue,
					Message: "worker exited with status 1",
				},
			},
		},
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&job).Build()
	worker := NewWorkerRuntime(WorkerOptions{
		Client: kubeClient,
		LogReader: staticPodLogReader{
			logs: PodLogs{
				PodName:       jobName + "-pod",
				ContainerName: workerContainerName,
				Text:          workerFailureLog(),
			},
		},
	})

	_, err := worker.Execute(context.Background(), request)
	var failure Failure
	if !errors.As(err, &failure) {
		t.Fatalf("expected structured runtime failure, got %T %v", err, err)
	}
	if failure.Reason != "WorkerFailed" {
		t.Fatalf("unexpected failure reason: %#v", failure)
	}
	if JSONString(failure.TraceRef, "podName") != jobName+"-pod" {
		t.Fatalf("unexpected trace ref: %#v", failure.TraceRef)
	}
	if JSONString(failure.Output, "summary") != "AGENT_COMPILED_ARTIFACT kind is required" {
		t.Fatalf("unexpected failure output: %#v", failure.Output)
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
				CompiledArtifact: apiv1alpha1.FreeformObject{
					"kind": JSONValue("AgentCompiledArtifact"),
					"runtime": JSONValue(map[string]interface{}{
						"engine":      "eino",
						"runnerClass": "adk",
					}),
				},
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

func envValue(env []corev1.EnvVar, name string) string {
	for _, item := range env {
		if item.Name == name {
			return item.Value
		}
	}
	return ""
}

type staticPodLogReader struct {
	logs PodLogs
	err  error
}

func (r staticPodLogReader) ReadJobPodLogs(ctx context.Context, namespace string, jobName string) (PodLogs, error) {
	return r.logs, r.err
}

func workerResultLog() string {
	return `{
  "status": "succeeded",
  "message": "agent control plane worker placeholder completed",
  "config": {
    "agentName": "hazard-agent",
    "agentRunName": "run-1",
    "agentRunNamespace": "ehs",
    "agentRevision": "sha256:agent"
  },
  "compiledArtifact": {
    "apiVersion": "windosx.com/v1alpha1",
    "kind": "AgentCompiledArtifact",
    "runtimeEngine": "eino",
    "runnerClass": "adk",
    "policyRef": "ehs-default-safety-policy"
  },
  "startedAt": "2026-04-17T06:16:59.241012625Z"
}`
}

func workerFailureLog() string {
	return `{
  "status": "failed",
  "reason": "WorkerFailed",
  "message": "AGENT_COMPILED_ARTIFACT kind is required",
  "startedAt": "2026-04-17T06:16:59.241012625Z"
}`
}
