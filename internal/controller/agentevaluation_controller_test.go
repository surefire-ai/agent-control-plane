package controller

import (
	"context"
	"testing"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	agentruntime "github.com/surefire-ai/agent-control-plane/internal/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func TestAgentEvaluationReconcilerMarksReadyWhenContractResolves(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-1",
			Namespace:  "ehs",
			Generation: 2,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Kind:     "Dataset",
				Name:     "ehs-hazard-benchmark-v1",
				Revision: "2026-04",
			},
			Baseline: &apiv1alpha1.EvaluationBaselineSpec{
				Revision: "agent-1-r0001",
			},
			Runtime: apiv1alpha1.FreeformObject{
				"sampleInput": agentruntime.JSONValue(map[string]interface{}{
					"task": "identify_hazard",
					"payload": map[string]interface{}{
						"text": "发现配电箱有裸露电线",
					},
				}),
			},
			Thresholds: []apiv1alpha1.EvaluationThresholdSpec{
				{Metric: "run_success", Target: 1.0, Blocking: true},
			},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent).
		Build()
	reconciler := &AgentEvaluationReconciler{
		Client: kubeClient,
		Scheme: scheme,
	}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-1"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.AgentEvaluation
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get AgentEvaluation returned error: %v", err)
	}
	if updated.Status.Phase != "Running" {
		t.Fatalf("expected Running phase, got %q", updated.Status.Phase)
	}
	if updated.Status.ObservedGeneration != 2 {
		t.Fatalf("expected observed generation 2, got %d", updated.Status.ObservedGeneration)
	}
	if updated.Status.Summary.DatasetRevision != "2026-04" {
		t.Fatalf("expected dataset revision, got %#v", updated.Status.Summary)
	}
	if updated.Status.Summary.BaselineRevision != "agent-1-r0001" {
		t.Fatalf("expected baseline revision, got %#v", updated.Status.Summary)
	}
	if updated.Status.LatestRunRef["agentRevision"] != "sha256:agent" {
		t.Fatalf("expected agent revision ref, got %#v", updated.Status.LatestRunRef)
	}
	if updated.Status.LatestRunRef["name"] != "eval-1-run-g2" {
		t.Fatalf("expected managed run name, got %#v", updated.Status.LatestRunRef)
	}
	if updated.Status.ReportRef["report"].Raw == nil {
		t.Fatalf("expected report ref, got %#v", updated.Status.ReportRef)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "EvaluationRunInProgress" {
		t.Fatalf("expected evaluation run in progress condition, got %#v", updated.Status.Conditions)
	}

	var managedRun apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: "ehs", Name: "eval-1-run-g2"}, &managedRun); err != nil {
		t.Fatalf("expected managed AgentRun to be created: %v", err)
	}
}

func TestAgentEvaluationReconcilerMarksPendingWhenAgentIsNotReady(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Name: "ehs-hazard-benchmark-v1",
			},
			Runtime: apiv1alpha1.FreeformObject{
				"sampleInput": agentruntime.JSONValue(map[string]interface{}{
					"task": "identify_hazard",
				}),
			},
		},
	}
	agent := &apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "agent-1", Namespace: "ehs"},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent).
		Build()
	reconciler := &AgentEvaluationReconciler{
		Client: kubeClient,
		Scheme: scheme,
	}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-1"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.AgentEvaluation
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get AgentEvaluation returned error: %v", err)
	}
	if updated.Status.Phase != "Pending" {
		t.Fatalf("expected Pending phase, got %q", updated.Status.Phase)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "WaitingForAgent" {
		t.Fatalf("expected WaitingForAgent condition, got %#v", updated.Status.Conditions)
	}
}

func TestAgentEvaluationReconcilerFailsWhenBaselineAgentMissing(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Name: "ehs-hazard-benchmark-v1",
			},
			Baseline: &apiv1alpha1.EvaluationBaselineSpec{
				AgentRef: &apiv1alpha1.LocalObjectReference{Name: "missing-baseline"},
			},
			Runtime: apiv1alpha1.FreeformObject{
				"sampleInput": agentruntime.JSONValue(map[string]interface{}{
					"task": "identify_hazard",
				}),
			},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent).
		Build()
	reconciler := &AgentEvaluationReconciler{
		Client: kubeClient,
		Scheme: scheme,
	}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-1"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.AgentEvaluation
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get AgentEvaluation returned error: %v", err)
	}
	if updated.Status.Phase != "NotReady" {
		t.Fatalf("expected NotReady phase, got %q", updated.Status.Phase)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "BaselineReferenceFailed" {
		t.Fatalf("expected BaselineReferenceFailed condition, got %#v", updated.Status.Conditions)
	}
}

func TestAgentEvaluationReconcilerAggregatesManagedRunResults(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-1",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Name:     "ehs-hazard-benchmark-v1",
				Revision: "2026-04",
			},
			Baseline: &apiv1alpha1.EvaluationBaselineSpec{
				Revision: "agent-1-r0001",
			},
			Runtime: apiv1alpha1.FreeformObject{
				"sampleInput": agentruntime.JSONValue(map[string]interface{}{
					"task": "identify_hazard",
					"payload": map[string]interface{}{
						"text": "发现配电箱有裸露电线",
					},
				}),
			},
			Thresholds: []apiv1alpha1.EvaluationThresholdSpec{
				{Metric: "run_success", Operator: "gte", Target: 1.0, Blocking: true},
				{Metric: "confidence", Operator: "gte", Target: 0.8, Blocking: true},
				{Metric: "schema_validity", Operator: "gte", Target: 1.0, Blocking: true},
			},
			Gate: apiv1alpha1.EvaluationGateSpec{
				Mode:        "all_blocking",
				Required:    []string{"run_success", "schema_validity"},
				BlockOnFail: true,
			},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")
	run := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eval-1-run-g1",
			Namespace: "ehs",
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:         string(apiv1alpha1.AgentRunPhaseSucceeded),
			AgentRevision: "sha256:agent",
			Output: apiv1alpha1.FreeformObject{
				"summary":          agentruntime.JSONValue("inspection complete"),
				"hazards":          agentruntime.JSONValue([]interface{}{}),
				"overallRiskLevel": agentruntime.JSONValue("medium"),
				"nextActions":      agentruntime.JSONValue([]string{"notify supervisor"}),
				"confidence":       agentruntime.JSONValue(0.93),
				"needsHumanReview": agentruntime.JSONValue(false),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent, run).
		Build()
	reconciler := &AgentEvaluationReconciler{
		Client: kubeClient,
		Scheme: scheme,
	}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-1"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.AgentEvaluation
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get AgentEvaluation returned error: %v", err)
	}
	if updated.Status.Phase != "Succeeded" {
		t.Fatalf("expected Succeeded phase, got %q", updated.Status.Phase)
	}
	if updated.Status.Summary.SamplesEvaluated != 1 {
		t.Fatalf("expected one evaluated sample, got %#v", updated.Status.Summary)
	}
	if !updated.Status.Summary.GatePassed {
		t.Fatalf("expected gate passed, got %#v", updated.Status.Summary)
	}
	if len(updated.Status.Results) != 3 {
		t.Fatalf("expected 3 metric results, got %#v", updated.Status.Results)
	}
	if updated.Status.LatestRunRef["name"] != "eval-1-run-g1" {
		t.Fatalf("expected latest run ref name, got %#v", updated.Status.LatestRunRef)
	}
}
