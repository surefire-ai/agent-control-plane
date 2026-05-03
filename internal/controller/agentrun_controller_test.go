package controller

import (
	"context"
	"errors"
	"testing"
	"time"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	agentruntime "github.com/surefire-ai/korus/internal/runtime"
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

func TestAgentRunReconcilerRecordsAgentWorkspace(t *testing.T) {
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
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")
	agent.Status.WorkspaceRef = "workspace-a"

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
		t.Fatalf("reconcile returned error: %v", err)
	}

	var pending apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &pending); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if pending.Status.WorkspaceRef != "workspace-a" {
		t.Fatalf("expected workspace ref, got %q", pending.Status.WorkspaceRef)
	}
}

func TestAgentRunReconcilerFailsWhenRunWorkspaceDoesNotMatchAgent(t *testing.T) {
	scheme := testScheme(t)
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:     apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			WorkspaceRef: &apiv1alpha1.LocalObjectReference{Name: "workspace-b"},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")
	agent.Status.WorkspaceRef = "workspace-a"

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
		t.Fatalf("reconcile returned error: %v", err)
	}

	var failed apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &failed); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if failed.Status.Phase != string(apiv1alpha1.AgentRunPhaseFailed) {
		t.Fatalf("expected Failed phase, got %q", failed.Status.Phase)
	}
	if failed.Status.WorkspaceRef != "workspace-b" {
		t.Fatalf("expected explicit workspace ref to be recorded, got %q", failed.Status.WorkspaceRef)
	}
	if failed.Status.Conditions[0].Reason != "WorkspaceMismatch" {
		t.Fatalf("expected WorkspaceMismatch condition reason, got %#v", failed.Status.Conditions)
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

// ── Cancel tests ──

func TestAgentRunReconcilerCancelsOnDeletionTimestamp(t *testing.T) {
	scheme := testScheme(t)
	now := time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)
	deletionTime := metav1.NewTime(now.Add(-time.Second))
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "run-1",
			Namespace:         "ehs",
			Generation:        1,
			DeletionTimestamp: &deletionTime,
			Finalizers:        []string{"kubernetes"}, // keep it alive for test
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:        string(apiv1alpha1.AgentRunPhaseRunning),
			WorkspaceRef: "ws",
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
		Clock:   func() metav1.Time { return metav1.NewTime(now) },
		Runtime: agentruntime.NewMockRuntime(),
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if result.Requeue || result.RequeueAfter > 0 {
		t.Fatalf("expected no requeue for canceled run, got %+v", result)
	}

	var canceled apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &canceled); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if canceled.Status.Phase != string(apiv1alpha1.AgentRunPhaseCanceled) {
		t.Fatalf("expected Canceled phase, got %q", canceled.Status.Phase)
	}
	if canceled.Status.Conditions[0].Reason != "Canceled" {
		t.Fatalf("expected Canceled condition reason, got %q", canceled.Status.Conditions[0].Reason)
	}
	if canceled.Status.FinishedAt == nil {
		t.Fatalf("expected FinishedAt to be set on canceled run")
	}
}

func TestAgentRunCanceledIsTerminal(t *testing.T) {
	if !isTerminalAgentRunPhase(string(apiv1alpha1.AgentRunPhaseCanceled)) {
		t.Fatal("Canceled should be a terminal phase")
	}
}

func TestAgentRunReconcilerCancelsOnSpecCancel(t *testing.T) {
	scheme := testScheme(t)
	cancel := true
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			Cancel:   &cancel,
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:        string(apiv1alpha1.AgentRunPhaseRunning),
			WorkspaceRef: "ws",
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

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if result.Requeue || result.RequeueAfter > 0 {
		t.Fatalf("expected no requeue for canceled run, got %+v", result)
	}

	var canceled apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &canceled); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if canceled.Status.Phase != string(apiv1alpha1.AgentRunPhaseCanceled) {
		t.Fatalf("expected Canceled phase, got %q", canceled.Status.Phase)
	}
	if canceled.Status.Conditions[0].Message != "AgentRun canceled by user request" {
		t.Fatalf("expected user-requested cancel message, got %q", canceled.Status.Conditions[0].Message)
	}
	if canceled.Status.FinishedAt == nil {
		t.Fatalf("expected FinishedAt to be set on canceled run")
	}
}

// ── Timeout tests ──

func TestAgentRunReconcilerTimesOut(t *testing.T) {
	scheme := testScheme(t)
	startTime := time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC)
	currentTime := time.Date(2026, 4, 16, 10, 0, 1, 0, time.UTC) // 1 hour + 1 second
	deadline := int64(3600)                                      // 1 hour

	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:              apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			ActiveDeadlineSeconds: &deadline,
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:     string(apiv1alpha1.AgentRunPhaseRunning),
			StartedAt: &metav1.Time{Time: startTime},
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
		Clock:   func() metav1.Time { return metav1.NewTime(currentTime) },
		Runtime: agentruntime.NewMockRuntime(),
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if result.Requeue || result.RequeueAfter > 0 {
		t.Fatalf("expected no requeue for timed-out run, got %+v", result)
	}

	var timedOut apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &timedOut); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if timedOut.Status.Phase != string(apiv1alpha1.AgentRunPhaseFailed) {
		t.Fatalf("expected Failed phase, got %q", timedOut.Status.Phase)
	}
	if timedOut.Status.Conditions[0].Reason != "DeadlineExceeded" {
		t.Fatalf("expected DeadlineExceeded reason, got %q", timedOut.Status.Conditions[0].Reason)
	}
}

func TestAgentRunReconcilerRequeuesBeforeDeadline(t *testing.T) {
	scheme := testScheme(t)
	startTime := time.Date(2026, 4, 16, 9, 0, 0, 0, time.UTC)
	currentTime := time.Date(2026, 4, 16, 9, 30, 0, 0, time.UTC) // only 30 min elapsed
	deadline := int64(3600)                                      // 1 hour

	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:              apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			ActiveDeadlineSeconds: &deadline,
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:     string(apiv1alpha1.AgentRunPhaseRunning),
			StartedAt: &metav1.Time{Time: startTime},
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
		Clock:   func() metav1.Time { return metav1.NewTime(currentTime) },
		Runtime: agentruntime.NewMockRuntime(),
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatalf("expected requeue before deadline, got no requeue")
	}
	// Should requeue after remaining time + 1s buffer
	expectedRemaining := 30*time.Minute + time.Second
	if result.RequeueAfter != expectedRemaining {
		t.Fatalf("expected requeue after %v, got %v", expectedRemaining, result.RequeueAfter)
	}
}

// ── Retry tests ──

func TestAgentRunReconcilerRetriesOnFailure(t *testing.T) {
	scheme := testScheme(t)
	maxRetries := int32(3)
	backoff := int64(5)

	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:            apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			MaxRetries:          &maxRetries,
			RetryBackoffSeconds: &backoff,
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

	var retrying apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &retrying); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if retrying.Status.Phase != string(apiv1alpha1.AgentRunPhaseRetrying) {
		t.Fatalf("expected Retrying phase, got %q", retrying.Status.Phase)
	}
	if retrying.Status.RetryCount != 1 {
		t.Fatalf("expected RetryCount 1, got %d", retrying.Status.RetryCount)
	}
	if retrying.Status.LastFailureReason != "runtime exploded" {
		t.Fatalf("expected last failure reason, got %q", retrying.Status.LastFailureReason)
	}
	if retrying.Status.Conditions[0].Reason != "Retrying" {
		t.Fatalf("expected Retrying condition reason, got %q", retrying.Status.Conditions[0].Reason)
	}
}

func TestAgentRunReconcilerFailsAfterMaxRetries(t *testing.T) {
	scheme := testScheme(t)
	maxRetries := int32(2)

	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:   apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			MaxRetries: &maxRetries,
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:      string(apiv1alpha1.AgentRunPhaseRunning),
			RetryCount: 2, // already exhausted
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
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if failed.Status.Phase != string(apiv1alpha1.AgentRunPhaseFailed) {
		t.Fatalf("expected Failed phase after max retries, got %q", failed.Status.Phase)
	}
}

func TestAgentRunReconcilerRetryingRespectsBackoff(t *testing.T) {
	scheme := testScheme(t)
	maxRetries := int32(3)
	backoff := int64(30)

	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:            apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			MaxRetries:          &maxRetries,
			RetryBackoffSeconds: &backoff,
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:      string(apiv1alpha1.AgentRunPhaseRetrying),
			RetryCount: 1,
			FinishedAt: &metav1.Time{Time: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentRun{}).
		WithObjects(run, agent).
		Build()

	// Only 10s elapsed — should wait remaining 20s
	currentTime := time.Date(2026, 4, 16, 10, 0, 10, 0, time.UTC)
	reconciler := &AgentRunReconciler{
		Client:  kubeClient,
		Scheme:  scheme,
		Clock:   func() metav1.Time { return metav1.NewTime(currentTime) },
		Runtime: agentruntime.NewMockRuntime(),
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if result.RequeueAfter != 20*time.Second {
		t.Fatalf("expected requeue after 20s remaining, got %v", result.RequeueAfter)
	}
}

func TestAgentRunReconcilerRetryingTransitionsToPendingAfterBackoff(t *testing.T) {
	scheme := testScheme(t)
	maxRetries := int32(3)
	backoff := int64(5)

	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "run-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:            apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			MaxRetries:          &maxRetries,
			RetryBackoffSeconds: &backoff,
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:      string(apiv1alpha1.AgentRunPhaseRetrying),
			RetryCount: 1,
			FinishedAt: &metav1.Time{Time: time.Date(2026, 4, 16, 10, 0, 0, 0, time.UTC)},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentRun{}).
		WithObjects(run, agent).
		Build()

	// 10s elapsed — backoff (5s) already passed
	currentTime := time.Date(2026, 4, 16, 10, 0, 10, 0, time.UTC)
	reconciler := &AgentRunReconciler{
		Client:  kubeClient,
		Scheme:  scheme,
		Clock:   func() metav1.Time { return metav1.NewTime(currentTime) },
		Runtime: agentruntime.NewMockRuntime(),
	}
	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "run-1"}}

	result, err := reconciler.Reconcile(context.Background(), request)
	if err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}
	if !result.Requeue {
		t.Fatalf("expected requeue after transitioning to Pending, got %+v", result)
	}

	var pending apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &pending); err != nil {
		t.Fatalf("get AgentRun returned error: %v", err)
	}
	if pending.Status.Phase != string(apiv1alpha1.AgentRunPhasePending) {
		t.Fatalf("expected Pending phase after backoff, got %q", pending.Status.Phase)
	}
}

// ── Helpers ──

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
		Spec: apiv1alpha1.AgentSpec{
			Interfaces: apiv1alpha1.AgentInterfaceSpec{
				Output: apiv1alpha1.SchemaEnvelope{
					Schema: apiv1alpha1.JSONSchema{
						Raw: []byte(`{"type":"object","required":["summary","hazards","overallRiskLevel","nextActions","confidence","needsHumanReview"]}`),
					},
				},
			},
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

func readyWorkspace(name string, namespace string, tenantRef string) *apiv1alpha1.Workspace {
	return &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: apiv1alpha1.WorkspaceSpec{
			TenantRef: apiv1alpha1.LocalObjectReference{Name: tenantRef},
		},
		Status: apiv1alpha1.WorkspaceStatus{
			Phase:     "Ready",
			TenantRef: tenantRef,
			Namespace: namespace,
			ConditionedStatus: apiv1alpha1.ConditionedStatus{
				Conditions: []metav1.Condition{
					{
						Type:               workspaceReadyCondition,
						Status:             metav1.ConditionTrue,
						Reason:             "TenantResolved",
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
