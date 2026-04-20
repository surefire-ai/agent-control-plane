package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	agentruntime "github.com/surefire-ai/agent-control-plane/internal/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestAgentRunReconcilerCompletesWithMockRuntime(t *testing.T) {
	scheme := testScheme(t)
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			Input: apiv1alpha1.FreeformObject{
				"task": agentruntime.JSONValue("identify_hazard"),
			},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentRun{}).
		WithObjects(run, agent).
		Build()
	reconciler := &AgentRunReconciler{
		Client:  kubeClient,
		Scheme:  scheme,
		Clock:   fixedClock(),
		Runtime: agentruntime.NewMockRuntime(),
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("first reconcile returned error: %v", err)
	}
	assertAgentRunPhase(t, kubeClient, "Pending")

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("second reconcile returned error: %v", err)
	}
	assertAgentRunPhase(t, kubeClient, "Running")

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("third reconcile returned error: %v", err)
	}

	var completed apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &completed); err != nil {
		t.Fatalf("get completed AgentRun returned error: %v", err)
	}
	if completed.Status.Phase != string(apiv1alpha1.AgentRunPhaseSucceeded) {
		t.Fatalf("expected Succeeded phase, got %q", completed.Status.Phase)
	}
	if completed.Status.AgentRevision != "sha256:agent" {
		t.Fatalf("expected agent revision, got %q", completed.Status.AgentRevision)
	}
	if completed.Status.Output["summary"].Raw == nil {
		t.Fatalf("expected mock output summary, got %#v", completed.Status.Output)
	}
	if completed.Status.TraceRef["provider"].Raw == nil {
		t.Fatalf("expected mock trace provider, got %#v", completed.Status.TraceRef)
	}
	if len(completed.Status.Conditions) != 1 || completed.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected true Completed condition, got %#v", completed.Status.Conditions)
	}
}

func TestAgentRunReconcilerFailsWhenAgentIsMissing(t *testing.T) {
	scheme := testScheme(t)
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "missing-agent"},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentRun{}).
		WithObjects(run).
		Build()
	reconciler := &AgentRunReconciler{
		Client:  kubeClient,
		Scheme:  scheme,
		Clock:   fixedClock(),
		Runtime: agentruntime.NewMockRuntime(),
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var failed apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &failed); err != nil {
		t.Fatalf("get failed AgentRun returned error: %v", err)
	}
	if failed.Status.Phase != string(apiv1alpha1.AgentRunPhaseFailed) {
		t.Fatalf("expected Failed phase, got %q", failed.Status.Phase)
	}
	if failed.Status.Conditions[0].Reason != "Failed" {
		t.Fatalf("expected Failed condition reason, got %#v", failed.Status.Conditions)
	}
}

func TestAgentRunReconcilerFailsWhenRuntimeReturnsError(t *testing.T) {
	scheme := testScheme(t)
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase: string(apiv1alpha1.AgentRunPhaseRunning),
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentRun{}).
		WithObjects(run, agent).
		Build()
	reconciler := &AgentRunReconciler{
		Client:  kubeClient,
		Scheme:  scheme,
		Clock:   fixedClock(),
		Runtime: failingRuntime{},
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var failed apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &failed); err != nil {
		t.Fatalf("get failed AgentRun returned error: %v", err)
	}
	if failed.Status.Phase != string(apiv1alpha1.AgentRunPhaseFailed) {
		t.Fatalf("expected Failed phase, got %q", failed.Status.Phase)
	}
	if failed.Status.Conditions[0].Message != "runtime exploded" {
		t.Fatalf("expected runtime error message, got %#v", failed.Status.Conditions)
	}
}

func TestAgentRunReconcilerPersistsRuntimeFailureDetails(t *testing.T) {
	scheme := testScheme(t)
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase: string(apiv1alpha1.AgentRunPhaseRunning),
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentRun{}).
		WithObjects(run, agent).
		Build()
	reconciler := &AgentRunReconciler{
		Client:  kubeClient,
		Scheme:  scheme,
		Clock:   fixedClock(),
		Runtime: structuredFailingRuntime{},
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var failed apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &failed); err != nil {
		t.Fatalf("get failed AgentRun returned error: %v", err)
	}
	if failed.Status.Phase != string(apiv1alpha1.AgentRunPhaseFailed) {
		t.Fatalf("expected Failed phase, got %q", failed.Status.Phase)
	}
	if failed.Status.AgentRevision != "sha256:agent" {
		t.Fatalf("expected agent revision, got %q", failed.Status.AgentRevision)
	}
	if failed.Status.Conditions[0].Reason != "WorkerFailed" {
		t.Fatalf("expected structured failure reason, got %#v", failed.Status.Conditions)
	}
	if agentruntime.JSONString(failed.Status.Output, "summary") != "worker contract failed" {
		t.Fatalf("expected failure output summary, got %#v", failed.Status.Output)
	}
	if agentruntime.JSONString(failed.Status.TraceRef, "podName") != "worker-pod" {
		t.Fatalf("expected failure trace pod, got %#v", failed.Status.TraceRef)
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := apiv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme returned error: %v", err)
	}
	return scheme
}

func readyAgent(name string, namespace string, revision string) *apiv1alpha1.Agent {
	return &apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Status: apiv1alpha1.AgentStatus{
			CompiledRevision: revision,
			CompiledArtifact: apiv1alpha1.FreeformObject{
				"kind": agentruntime.JSONValue("AgentCompiledArtifact"),
			},
			ConditionedStatus: apiv1alpha1.ConditionedStatus{
				Conditions: []metav1.Condition{
					{
						Type:               agentReadyCondition,
						Status:             metav1.ConditionTrue,
						Reason:             "CompilationSucceeded",
						Message:            "ready",
						ObservedGeneration: 1,
						LastTransitionTime: metav1.NewTime(time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC)),
					},
				},
			},
		},
	}
}

func fixedClock() func() metav1.Time {
	return func() metav1.Time {
		return metav1.NewTime(time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC))
	}
}

func assertAgentRunPhase(t *testing.T, kubeClient client.Client, expected string) {
	t.Helper()
	var run apiv1alpha1.AgentRun
	key := client.ObjectKey{Namespace: "ehs", Name: "run-1"}
	if err := kubeClient.Get(context.Background(), key, &run); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if run.Status.Phase != expected {
		t.Fatalf("expected phase %q, got %q", expected, run.Status.Phase)
	}
}

type failingRuntime struct{}

func (r failingRuntime) Execute(ctx context.Context, request agentruntime.Request) (agentruntime.Result, error) {
	return agentruntime.Result{}, errors.New("runtime exploded")
}

type structuredFailingRuntime struct{}

func (r structuredFailingRuntime) Execute(ctx context.Context, request agentruntime.Request) (agentruntime.Result, error) {
	return agentruntime.Result{}, agentruntime.Failure{
		Output: apiv1alpha1.FreeformObject{
			"summary": agentruntime.JSONValue("worker contract failed"),
		},
		TraceRef: apiv1alpha1.FreeformObject{
			"podName": agentruntime.JSONValue("worker-pod"),
		},
		Reason:  "WorkerFailed",
		Message: "worker contract failed",
	}
}
