package controller

import (
	"context"
	"encoding/json"
	"testing"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	agentruntime "github.com/surefire-ai/korus/internal/runtime"
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
			AgentRef:     apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			WorkspaceRef: &apiv1alpha1.LocalObjectReference{Name: "workspace-a"},
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
	workspace := readyWorkspace("workspace-a", "ehs", "tenant-a")

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent, workspace).
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
	if updated.Status.WorkspaceRef != "workspace-a" {
		t.Fatalf("expected workspace ref, got %#v", updated.Status)
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
	if updated.Status.LatestRunRef["name"] != "eval-1-run-g2-sample-0" {
		t.Fatalf("expected managed run name, got %#v", updated.Status.LatestRunRef)
	}
	if updated.Status.ReportRef["report"].Raw == nil {
		t.Fatalf("expected report ref, got %#v", updated.Status.ReportRef)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "EvaluationRunInProgress" {
		t.Fatalf("expected evaluation run in progress condition, got %#v", updated.Status.Conditions)
	}

	var managedRun apiv1alpha1.AgentRun
	if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: "ehs", Name: "eval-1-run-g2-sample-0"}, &managedRun); err != nil {
		t.Fatalf("expected managed AgentRun to be created: %v", err)
	}
	if managedRun.Spec.WorkspaceRef == nil || managedRun.Spec.WorkspaceRef.Name != "workspace-a" {
		t.Fatalf("expected managed AgentRun workspace ref, got %#v", managedRun.Spec.WorkspaceRef)
	}
}

func TestAgentEvaluationReconcilerFailsWhenWorkspaceMissing(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-workspace",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef:     apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			WorkspaceRef: &apiv1alpha1.LocalObjectReference{Name: "missing-workspace"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Name: "ehs-hazard-benchmark-v1",
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

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-workspace"}}
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
	if updated.Status.WorkspaceRef != "missing-workspace" {
		t.Fatalf("expected workspace ref, got %#v", updated.Status)
	}
	if len(updated.Status.Conditions) != 1 || updated.Status.Conditions[0].Reason != "WorkspaceReferenceFailed" {
		t.Fatalf("expected WorkspaceReferenceFailed condition, got %#v", updated.Status.Conditions)
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
				"samples": agentruntime.JSONValue([]map[string]interface{}{
					{
						"name": "case-a",
						"input": map[string]interface{}{
							"task": "identify_hazard",
							"payload": map[string]interface{}{
								"text": "发现配电箱有裸露电线",
							},
						},
						"expected": map[string]interface{}{
							"overallRiskLevel": "medium",
							"hazards_count":    1,
							"hazards":          []map[string]interface{}{{"category": "electrical"}},
						},
					},
					{
						"name": "case-b",
						"input": map[string]interface{}{
							"task": "identify_hazard",
							"payload": map[string]interface{}{
								"text": "灭火器被遮挡",
							},
						},
						"expected": map[string]interface{}{
							"overallRiskLevel": "low",
							"hazards_count":    1,
							"hazards":          []map[string]interface{}{{"category": "fire"}},
						},
					},
				}),
			},
			Thresholds: []apiv1alpha1.EvaluationThresholdSpec{
				{Metric: "run_success", Operator: "gte", Target: 1.0, Blocking: true},
				{Metric: "confidence", Operator: "gte", Target: 0.8, Blocking: true},
				{Metric: "schema_validity", Operator: "gte", Target: 1.0, Blocking: true},
				{Metric: "overallRiskLevel", Operator: "gte", Target: 1.0, Blocking: true},
				{Metric: "hazards_count", Operator: "gte", Target: 1.0, Blocking: true},
				{Metric: "risk_level_match", Operator: "gte", Target: 1.0, Blocking: true},
				{Metric: "hazard_coverage", Operator: "gte", Target: 1.0, Blocking: true},
			},
			Gate: apiv1alpha1.EvaluationGateSpec{
				Mode:        "all_blocking",
				Required:    []string{"run_success", "schema_validity"},
				BlockOnFail: true,
			},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")
	runA := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eval-1-run-g1-case-a",
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
				"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{"category": "electrical", "title": "裸露电线"}}),
				"overallRiskLevel": agentruntime.JSONValue("medium"),
				"nextActions":      agentruntime.JSONValue([]string{"notify supervisor"}),
				"confidence":       agentruntime.JSONValue(0.93),
				"needsHumanReview": agentruntime.JSONValue(false),
			},
		},
	}
	runB := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eval-1-run-g1-case-b",
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
				"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{"category": "fire", "title": "消防通道受阻"}}),
				"overallRiskLevel": agentruntime.JSONValue("low"),
				"nextActions":      agentruntime.JSONValue([]string{"clear pathway"}),
				"confidence":       agentruntime.JSONValue(0.87),
				"needsHumanReview": agentruntime.JSONValue(false),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent, runA, runB).
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
	if updated.Status.Summary.SamplesEvaluated != 2 || updated.Status.Summary.SamplesTotal != 2 {
		t.Fatalf("expected two evaluated samples, got %#v", updated.Status.Summary)
	}
	if !updated.Status.Summary.GatePassed {
		t.Fatalf("expected gate passed, got %#v", updated.Status.Summary)
	}
	if len(updated.Status.Results) != 7 {
		t.Fatalf("expected 7 metric results, got %#v", updated.Status.Results)
	}
	if updated.Status.LatestRunRef["name"] != "eval-1-run-g1-case-b" {
		t.Fatalf("expected latest run ref name, got %#v", updated.Status.LatestRunRef)
	}
}

func TestAgentEvaluationReconcilerBuildsBaselineComparison(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-compare",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-current"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Name:     "ehs-hazard-benchmark-v1",
				Revision: "2026-04",
			},
			Baseline: &apiv1alpha1.EvaluationBaselineSpec{
				AgentRef: &apiv1alpha1.LocalObjectReference{Name: "agent-baseline"},
			},
			Runtime: apiv1alpha1.FreeformObject{
				"samples": agentruntime.JSONValue([]map[string]interface{}{
					{
						"name": "case-a",
						"input": map[string]interface{}{
							"task": "identify_hazard",
							"payload": map[string]interface{}{
								"text": "发现配电箱有裸露电线",
							},
						},
						"expected": map[string]interface{}{
							"overallRiskLevel": "medium",
						},
					},
				}),
			},
			Thresholds: []apiv1alpha1.EvaluationThresholdSpec{
				{Metric: "run_success", Operator: "gte", Target: 1.0, Blocking: true},
				{Metric: "confidence", Operator: "gte", Target: 0.8, Blocking: true},
				{Metric: "overallRiskLevel", Operator: "gte", Target: 1.0, Blocking: true},
			},
			Gate: apiv1alpha1.EvaluationGateSpec{
				Mode:        "all_blocking",
				BlockOnFail: true,
			},
		},
	}
	currentAgent := readyAgent("agent-current", "ehs", "sha256:current")
	baselineAgent := readyAgent("agent-baseline", "ehs", "sha256:baseline")
	currentRun := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eval-compare-run-g1-case-a",
			Namespace: "ehs",
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-current"},
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:         string(apiv1alpha1.AgentRunPhaseSucceeded),
			AgentRevision: "sha256:current",
			Output: apiv1alpha1.FreeformObject{
				"overallRiskLevel": agentruntime.JSONValue("medium"),
				"confidence":       agentruntime.JSONValue(0.91),
			},
		},
	}
	baselineRun := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eval-compare-baseline-run-g1-case-a",
			Namespace: "ehs",
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-baseline"},
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase:         string(apiv1alpha1.AgentRunPhaseSucceeded),
			AgentRevision: "sha256:baseline",
			Output: apiv1alpha1.FreeformObject{
				"overallRiskLevel": agentruntime.JSONValue("low"),
				"confidence":       agentruntime.JSONValue(0.62),
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, currentAgent, baselineAgent, currentRun, baselineRun).
		Build()
	reconciler := &AgentEvaluationReconciler{
		Client: kubeClient,
		Scheme: scheme,
	}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-compare"}}
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
	if updated.Status.Comparison == nil {
		t.Fatalf("expected comparison status, got %#v", updated.Status)
	}
	if updated.Status.Comparison.BaselineAgentRef != "agent-baseline" {
		t.Fatalf("expected baseline agent ref, got %#v", updated.Status.Comparison)
	}
	if updated.Status.Comparison.CurrentScore <= updated.Status.Comparison.BaselineScore {
		t.Fatalf("expected current score to beat baseline, got %#v", updated.Status.Comparison)
	}
	if updated.Status.Comparison.ScoreDelta <= 0 {
		t.Fatalf("expected positive score delta, got %#v", updated.Status.Comparison)
	}
	if !updated.Status.Comparison.CurrentGatePassed || updated.Status.Comparison.BaselineGatePassed {
		t.Fatalf("expected current gate pass and baseline gate fail, got %#v", updated.Status.Comparison)
	}
}

func TestExpectedMetricScoreSupportsExactMatchAndCount(t *testing.T) {
	sample := evaluationSample{
		Name: "case-a",
		Expected: apiv1alpha1.FreeformObject{
			"overallRiskLevel": agentruntime.JSONValue("high"),
			"hazards_count":    agentruntime.JSONValue(2),
		},
	}
	run := apiv1alpha1.AgentRun{
		Status: apiv1alpha1.AgentRunStatus{
			Output: apiv1alpha1.FreeformObject{
				"overallRiskLevel": agentruntime.JSONValue("high"),
				"hazards": agentruntime.JSONValue([]interface{}{
					map[string]interface{}{"title": "wire"},
					map[string]interface{}{"title": "smoke"},
				}),
			},
		},
	}

	if score, ok := expectedMetricScore(run, sample, "overallRiskLevel"); !ok || score != 1 {
		t.Fatalf("expected exact match metric score 1, got %v %v", score, ok)
	}
	if score, ok := expectedMetricScore(run, sample, "hazards_count"); !ok || score != 1 {
		t.Fatalf("expected count metric score 1, got %v %v", score, ok)
	}
}

func TestStructuredMetricScoresRiskLevelAndHazardCoverage(t *testing.T) {
	sample := evaluationSample{
		Name: "case-a",
		Expected: apiv1alpha1.FreeformObject{
			"overallRiskLevel": agentruntime.JSONValue("high"),
			"hazards": agentruntime.JSONValue([]map[string]interface{}{
				{"category": "electrical"},
				{"category": "fire"},
			}),
		},
	}
	run := apiv1alpha1.AgentRun{
		Status: apiv1alpha1.AgentRunStatus{
			Output: apiv1alpha1.FreeformObject{
				"overallRiskLevel": agentruntime.JSONValue("medium"),
				"hazards": agentruntime.JSONValue([]map[string]interface{}{
					{"category": "electrical", "title": "裸露电线"},
				}),
			},
		},
	}

	if score, ok := riskLevelMatchScore(run, sample); !ok || score != 0.5 {
		t.Fatalf("expected tolerant risk level score 0.5, got %v %v", score, ok)
	}
	if score, ok := hazardCoverageScore(run, sample); !ok || score != 0.5 {
		t.Fatalf("expected hazard coverage score 0.5, got %v %v", score, ok)
	}
}

func TestAgentEvaluationReconcilerCreatesMultipleManagedRuns(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-2",
			Namespace:  "ehs",
			Generation: 3,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Name:     "ehs-hazard-benchmark-v2",
				Revision: "2026-05",
			},
			Runtime: apiv1alpha1.FreeformObject{
				"samples": agentruntime.JSONValue([]map[string]interface{}{
					{
						"name": "power-box",
						"input": map[string]interface{}{
							"task":    "identify_hazard",
							"payload": map[string]interface{}{"text": "配电箱外壳破损"},
						},
					},
					{
						"name": "fire-lane",
						"input": map[string]interface{}{
							"task":    "identify_hazard",
							"payload": map[string]interface{}{"text": "消防通道堆放杂物"},
						},
					},
				}),
			},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")
	dataset := &apiv1alpha1.Dataset{
		ObjectMeta: metav1.ObjectMeta{Name: "ehs-hazard-benchmark-v2", Namespace: "ehs"},
		Spec: apiv1alpha1.DatasetSpec{
			Revision: "2026-05",
			Samples: []apiv1alpha1.DatasetSampleSpec{
				{
					Name: "power-box",
					Input: apiv1alpha1.FreeformObject{
						"task":    agentruntime.JSONValue("identify_hazard"),
						"payload": agentruntime.JSONValue(map[string]interface{}{"text": "配电箱外壳破损"}),
					},
				},
				{
					Name: "fire-lane",
					Input: apiv1alpha1.FreeformObject{
						"task":    agentruntime.JSONValue("identify_hazard"),
						"payload": agentruntime.JSONValue(map[string]interface{}{"text": "消防通道堆放杂物"}),
					},
				},
			},
		},
	}
	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent, dataset).
		Build()
	reconciler := &AgentEvaluationReconciler{Client: kubeClient, Scheme: scheme}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-2"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.AgentEvaluation
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get AgentEvaluation returned error: %v", err)
	}
	if updated.Status.Summary.SamplesTotal != 2 {
		t.Fatalf("expected two total samples, got %#v", updated.Status.Summary)
	}

	for _, runName := range []string{"eval-2-run-g3-power-box", "eval-2-run-g3-fire-lane"} {
		var managedRun apiv1alpha1.AgentRun
		if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: "ehs", Name: runName}, &managedRun); err != nil {
			t.Fatalf("expected managed AgentRun %q to be created: %v", runName, err)
		}
	}
}

func TestAgentEvaluationReconcilerCreatesManagedRunsFromDataset(t *testing.T) {
	scheme := testScheme(t)
	evaluation := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "eval-3",
			Namespace:  "ehs",
			Generation: 1,
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "agent-1"},
			DatasetRef: apiv1alpha1.EvaluationDatasetReference{
				Kind:     "Dataset",
				Name:     "ehs-hazard-benchmark-v3",
				Revision: "2026-06",
			},
		},
	}
	agent := readyAgent("agent-1", "ehs", "sha256:agent")
	dataset := &apiv1alpha1.Dataset{
		ObjectMeta: metav1.ObjectMeta{Name: "ehs-hazard-benchmark-v3", Namespace: "ehs"},
		Spec: apiv1alpha1.DatasetSpec{
			Revision: "2026-06",
			Samples: []apiv1alpha1.DatasetSampleSpec{
				{
					Name: "electrical-room",
					Input: apiv1alpha1.FreeformObject{
						"task":    agentruntime.JSONValue("identify_hazard"),
						"payload": agentruntime.JSONValue(map[string]interface{}{"text": "电气间有积水"}),
					},
				},
				{
					Name: "blocked-exit",
					Input: apiv1alpha1.FreeformObject{
						"task":    agentruntime.JSONValue("identify_hazard"),
						"payload": agentruntime.JSONValue(map[string]interface{}{"text": "安全出口被货箱堵塞"}),
					},
				},
			},
		},
	}
	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithStatusSubresource(&apiv1alpha1.AgentEvaluation{}, &apiv1alpha1.AgentRun{}).
		WithObjects(evaluation, agent, dataset).
		Build()
	reconciler := &AgentEvaluationReconciler{Client: kubeClient, Scheme: scheme}

	request := reconcile.Request{NamespacedName: client.ObjectKey{Namespace: "ehs", Name: "eval-3"}}
	if _, err := reconciler.Reconcile(context.Background(), request); err != nil {
		t.Fatalf("reconcile returned error: %v", err)
	}

	var updated apiv1alpha1.AgentEvaluation
	if err := kubeClient.Get(context.Background(), request.NamespacedName, &updated); err != nil {
		t.Fatalf("get AgentEvaluation returned error: %v", err)
	}
	if updated.Status.Summary.DatasetRevision != "2026-06" || updated.Status.Summary.SamplesTotal != 2 {
		t.Fatalf("expected dataset revision and sample total from Dataset, got %#v", updated.Status.Summary)
	}
	for _, runName := range []string{"eval-3-run-g1-electrical-room", "eval-3-run-g1-blocked-exit"} {
		var managedRun apiv1alpha1.AgentRun
		if err := kubeClient.Get(context.Background(), client.ObjectKey{Namespace: "ehs", Name: runName}, &managedRun); err != nil {
			t.Fatalf("expected managed AgentRun %q to be created from Dataset: %v", runName, err)
		}
	}
}

func TestResponseCompletenessScore(t *testing.T) {
	agent := *readyAgent("test-agent", "default", "sha256:test")

	// All fields populated with meaningful values
	t.Run("all fields populated", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"summary":          agentruntime.JSONValue("found hazards"),
					"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{"title": "wire"}}),
					"overallRiskLevel": agentruntime.JSONValue("high"),
					"nextActions":      agentruntime.JSONValue([]string{"fix wiring immediately"}),
					"confidence":       agentruntime.JSONValue(0.95),
					"needsHumanReview": agentruntime.JSONValue(false),
				},
			},
		}
		score, ok := responseCompletenessScore(agent, run)
		if !ok || score != 1.0 {
			t.Fatalf("expected score 1.0, got %v ok=%v", score, ok)
		}
	})

	// Some fields are empty/null
	t.Run("some fields empty", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"summary":          agentruntime.JSONValue("found hazards"),
					"hazards":          agentruntime.JSONValue([]interface{}{}),
					"overallRiskLevel": agentruntime.JSONValue("high"),
					"nextActions":      agentruntime.JSONValue(""),
					"confidence":       agentruntime.JSONValue(0.95),
					"needsHumanReview": agentruntime.JSONValue(false),
				},
			},
		}
		score, ok := responseCompletenessScore(agent, run)
		if !ok {
			t.Fatal("expected ok=true")
		}
		// 4/6 meaningful: summary, overallRiskLevel, confidence, needsHumanReview
		// hazards is empty array, nextActions is empty string
		if score != 4.0/6.0 {
			t.Fatalf("expected score 4/6, got %v", score)
		}
	})

	// No output at all
	t.Run("no output", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{},
			},
		}
		score, ok := responseCompletenessScore(agent, run)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if score != 0 {
			t.Fatalf("expected score 0, got %v", score)
		}
	})

	// Null values
	t.Run("null values", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"summary":          agentruntime.JSONValue("found hazards"),
					"hazards":          agentruntime.JSONValue(nil),
					"overallRiskLevel": agentruntime.JSONValue(nil),
					"nextActions":      agentruntime.JSONValue(nil),
					"confidence":       agentruntime.JSONValue(nil),
					"needsHumanReview": agentruntime.JSONValue(nil),
				},
			},
		}
		score, ok := responseCompletenessScore(agent, run)
		if !ok {
			t.Fatal("expected ok=true")
		}
		if score != 1.0/6.0 {
			t.Fatalf("expected score 1/6, got %v", score)
		}
	})
}

func TestOutputCoherenceScore(t *testing.T) {
	// Coherent: 1 hazard + low risk
	t.Run("coherent single hazard low risk", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{"title": "wire"}}),
					"overallRiskLevel": agentruntime.JSONValue("low"),
					"confidence":       agentruntime.JSONValue(0.9),
					"needsHumanReview": agentruntime.JSONValue(false),
				},
			},
		}
		score, ok := outputCoherenceScore(run)
		if !ok || score != 1.0 {
			t.Fatalf("expected score 1.0, got %v ok=%v", score, ok)
		}
	})

	// Incoherent: 3 hazards + low risk
	t.Run("incoherent many hazards low risk", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{"title": "a"}, map[string]interface{}{"title": "b"}, map[string]interface{}{"title": "c"}}),
					"overallRiskLevel": agentruntime.JSONValue("low"),
					"confidence":       agentruntime.JSONValue(0.9),
					"needsHumanReview": agentruntime.JSONValue(false),
				},
			},
		}
		score, ok := outputCoherenceScore(run)
		if !ok {
			t.Fatal("expected ok=true")
		}
		// 2/3 passed: hazards-risk check fails, confidence-review passes, hazards-highrisk passes
		if score != 2.0/3.0 {
			t.Fatalf("expected score 2/3, got %v", score)
		}
	})

	// Incoherent: high confidence + needs human review
	t.Run("incoherent high confidence needs review", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{"title": "a"}}),
					"overallRiskLevel": agentruntime.JSONValue("high"),
					"confidence":       agentruntime.JSONValue(0.95),
					"needsHumanReview": agentruntime.JSONValue(true),
				},
			},
		}
		score, ok := outputCoherenceScore(run)
		if !ok {
			t.Fatal("expected ok=true")
		}
		// 2/3 passed: hazards-risk passes, confidence-review fails, hazards-highrisk passes
		if score != 2.0/3.0 {
			t.Fatalf("expected score 2/3, got %v", score)
		}
	})

	// No coherence fields available
	t.Run("no coherence fields", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"summary": agentruntime.JSONValue("something"),
				},
			},
		}
		_, ok := outputCoherenceScore(run)
		if ok {
			t.Fatal("expected ok=false when no coherence fields present")
		}
	})
}

func TestActionableNextStepsScore(t *testing.T) {
	// All actions are specific
	t.Run("all actionable", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"nextActions": agentruntime.JSONValue([]string{
						"Replace all damaged wiring in the electrical room",
						"Install protective covers on exposed junction boxes",
					}),
				},
			},
		}
		score, ok := actionableNextStepsScore(run)
		if !ok || score != 1.0 {
			t.Fatalf("expected score 1.0, got %v ok=%v", score, ok)
		}
	})

	// Some actions are too short
	t.Run("some too short", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"nextActions": agentruntime.JSONValue([]string{
						"fix it",
						"Replace all damaged wiring in the electrical room",
					}),
				},
			},
		}
		score, ok := actionableNextStepsScore(run)
		if !ok || score != 0.5 {
			t.Fatalf("expected score 0.5, got %v ok=%v", score, ok)
		}
	})

	// Empty actions array
	t.Run("empty actions", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"nextActions": agentruntime.JSONValue([]string{}),
				},
			},
		}
		score, ok := actionableNextStepsScore(run)
		if !ok || score != 0 {
			t.Fatalf("expected score 0, got %v ok=%v", score, ok)
		}
	})

	// Missing field
	t.Run("missing field", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{},
			},
		}
		_, ok := actionableNextStepsScore(run)
		if ok {
			t.Fatal("expected ok=false when field missing")
		}
	})
}

func TestConfidenceCalibrationScore(t *testing.T) {
	// High confidence + success = well calibrated
	t.Run("high confidence success", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Phase: string(apiv1alpha1.AgentRunPhaseSucceeded),
				Output: apiv1alpha1.FreeformObject{
					"confidence": agentruntime.JSONValue(0.93),
				},
			},
		}
		score, ok := confidenceCalibrationScore(run)
		if !ok || score != 1.0 {
			t.Fatalf("expected score 1.0, got %v ok=%v", score, ok)
		}
	})

	// Low confidence + failure = well calibrated
	t.Run("low confidence failure", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Phase: string(apiv1alpha1.AgentRunPhaseFailed),
				Output: apiv1alpha1.FreeformObject{
					"confidence": agentruntime.JSONValue(0.15),
				},
			},
		}
		score, ok := confidenceCalibrationScore(run)
		if !ok || score != 1.0 {
			t.Fatalf("expected score 1.0, got %v ok=%v", score, ok)
		}
	})

	// High confidence + failure = poorly calibrated
	t.Run("high confidence failure", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Phase: string(apiv1alpha1.AgentRunPhaseFailed),
				Output: apiv1alpha1.FreeformObject{
					"confidence": agentruntime.JSONValue(0.95),
				},
			},
		}
		score, ok := confidenceCalibrationScore(run)
		if !ok || score != 0.2 {
			t.Fatalf("expected score 0.2, got %v ok=%v", score, ok)
		}
	})

	// Out of range confidence
	t.Run("out of range", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Phase: string(apiv1alpha1.AgentRunPhaseSucceeded),
				Output: apiv1alpha1.FreeformObject{
					"confidence": agentruntime.JSONValue(1.5),
				},
			},
		}
		score, ok := confidenceCalibrationScore(run)
		if !ok || score != 0 {
			t.Fatalf("expected score 0 for out-of-range confidence, got %v ok=%v", score, ok)
		}
	})

	// Medium confidence + success
	t.Run("medium confidence success", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Phase: string(apiv1alpha1.AgentRunPhaseSucceeded),
				Output: apiv1alpha1.FreeformObject{
					"confidence": agentruntime.JSONValue(0.5),
				},
			},
		}
		score, ok := confidenceCalibrationScore(run)
		if !ok || score != 0.7 {
			t.Fatalf("expected score 0.7, got %v ok=%v", score, ok)
		}
	})

	// Missing confidence field
	t.Run("missing field", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{},
			},
		}
		_, ok := confidenceCalibrationScore(run)
		if ok {
			t.Fatal("expected ok=false when confidence missing")
		}
	})
}

func TestBuildEvaluationReport(t *testing.T) {
	evaluation := apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eval-report-test",
			Namespace: "ehs",
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			Thresholds: []apiv1alpha1.EvaluationThresholdSpec{
				{Metric: "run_success", Target: 1.0},
				{Metric: "confidence_calibration", Target: 0.8},
			},
			Gate: apiv1alpha1.EvaluationGateSpec{Mode: "all_blocking"},
		},
	}

	current := evaluatedRunSet{
		AgentRef: "agent-1",
		Revision: "sha256:abc",
		Runs: []apiv1alpha1.AgentRun{
			{
				ObjectMeta: metav1.ObjectMeta{Name: "run-1"},
				Status: apiv1alpha1.AgentRunStatus{
					Phase: string(apiv1alpha1.AgentRunPhaseSucceeded),
				},
			},
			{
				ObjectMeta: metav1.ObjectMeta{Name: "run-2"},
				Status: apiv1alpha1.AgentRunStatus{
					Phase: string(apiv1alpha1.AgentRunPhaseSucceeded),
				},
			},
		},
		Results: []apiv1alpha1.EvaluationMetricStatus{
			{Name: "run_success", Metric: "run_success", Score: 1.0, Threshold: 1.0, Passed: true},
			{Name: "confidence_calibration", Metric: "confidence_calibration", Score: 0.9, Threshold: 0.8, Passed: true, Reason: "well calibrated"},
		},
		Score:      0.95,
		GatePassed: true,
	}

	samples := []evaluationSample{
		{Name: "case-a", Input: apiv1alpha1.FreeformObject{"task": agentruntime.JSONValue("test")}},
		{Name: "case-b", Input: apiv1alpha1.FreeformObject{"task": agentruntime.JSONValue("test")}},
	}

	baseline := &evaluatedRunSet{
		AgentRef:   "agent-baseline",
		Revision:   "sha256:def",
		Score:      0.75,
		GatePassed: false,
	}

	report := buildEvaluationReport(evaluation, current, baseline, samples)

	// Verify report contains expected top-level keys
	for _, key := range []string{"generatedAt", "evaluation", "agent", "overall", "metrics", "samples", "thresholds", "baseline"} {
		if _, ok := report[key]; !ok {
			t.Fatalf("expected report key %q to be present", key)
		}
	}

	// Verify overall section
	overallRaw, ok := report["overall"]
	if !ok {
		t.Fatal("expected overall in report")
	}
	var overall map[string]interface{}
	if err := json.Unmarshal(overallRaw.Raw, &overall); err != nil {
		t.Fatalf("failed to unmarshal overall: %v", err)
	}
	if overall["score"].(float64) != 0.95 {
		t.Fatalf("expected overall score 0.95, got %v", overall["score"])
	}
	if overall["gatePassed"].(bool) != true {
		t.Fatalf("expected gatePassed true, got %v", overall["gatePassed"])
	}
	if int(overall["samples"].(float64)) != 2 {
		t.Fatalf("expected 2 samples, got %v", overall["samples"])
	}

	// Verify metrics section
	metricsRaw, ok := report["metrics"]
	if !ok {
		t.Fatal("expected metrics in report")
	}
	var metrics []map[string]interface{}
	if err := json.Unmarshal(metricsRaw.Raw, &metrics); err != nil {
		t.Fatalf("failed to unmarshal metrics: %v", err)
	}
	if len(metrics) != 2 {
		t.Fatalf("expected 2 metrics, got %d", len(metrics))
	}
	if metrics[0]["metric"] != "run_success" || metrics[0]["passed"].(bool) != true {
		t.Fatalf("unexpected first metric: %v", metrics[0])
	}
	if metrics[1]["reason"] != "well calibrated" {
		t.Fatalf("expected reason in second metric, got %v", metrics[1])
	}

	// Verify samples section
	samplesRaw, ok := report["samples"]
	if !ok {
		t.Fatal("expected samples in report")
	}
	var reportSamples []map[string]interface{}
	if err := json.Unmarshal(samplesRaw.Raw, &reportSamples); err != nil {
		t.Fatalf("failed to unmarshal samples: %v", err)
	}
	if len(reportSamples) != 2 {
		t.Fatalf("expected 2 sample summaries, got %d", len(reportSamples))
	}
	if reportSamples[0]["name"] != "case-a" || reportSamples[0]["succeeded"].(bool) != true {
		t.Fatalf("unexpected first sample: %v", reportSamples[0])
	}

	// Verify baseline section
	baselineRaw, ok := report["baseline"]
	if !ok {
		t.Fatal("expected baseline in report")
	}
	var baselineSection map[string]interface{}
	if err := json.Unmarshal(baselineRaw.Raw, &baselineSection); err != nil {
		t.Fatalf("failed to unmarshal baseline: %v", err)
	}
	if baselineSection["agentRef"] != "agent-baseline" {
		t.Fatalf("expected baseline agentRef, got %v", baselineSection["agentRef"])
	}
	scoreDelta := baselineSection["scoreDelta"].(float64)
	if scoreDelta < 0.19 || scoreDelta > 0.21 {
		t.Fatalf("expected scoreDelta ~0.2, got %v", scoreDelta)
	}
}

func TestBuildEvaluationReportWithoutBaseline(t *testing.T) {
	evaluation := apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "eval-no-baseline",
			Namespace: "default",
		},
		Spec: apiv1alpha1.AgentEvaluationSpec{
			Thresholds: []apiv1alpha1.EvaluationThresholdSpec{
				{Metric: "run_success", Target: 1.0},
			},
		},
	}
	current := evaluatedRunSet{
		AgentRef: "agent-1",
		Revision: "sha256:abc",
		Runs: []apiv1alpha1.AgentRun{
			{ObjectMeta: metav1.ObjectMeta{Name: "run-1"}, Status: apiv1alpha1.AgentRunStatus{Phase: string(apiv1alpha1.AgentRunPhaseSucceeded)}},
		},
		Results:    []apiv1alpha1.EvaluationMetricStatus{{Name: "run_success", Metric: "run_success", Score: 1.0, Passed: true}},
		Score:      1.0,
		GatePassed: true,
	}

	report := buildEvaluationReport(evaluation, current, nil, nil)

	if _, ok := report["baseline"]; ok {
		t.Fatal("expected no baseline key when baseline is nil")
	}
	if _, ok := report["overall"]; !ok {
		t.Fatal("expected overall key")
	}
}

func TestNewEvaluatorsViaMetricScore(t *testing.T) {
	agent := *readyAgent("test-agent", "default", "sha256:test")
	sample := evaluationSample{Name: "test"}

	t.Run("response_completeness via metricScore", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"summary":          agentruntime.JSONValue("found hazards"),
					"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{}}),
					"overallRiskLevel": agentruntime.JSONValue("high"),
					"nextActions":      agentruntime.JSONValue([]string{"fix wiring"}),
					"confidence":       agentruntime.JSONValue(0.9),
					"needsHumanReview": agentruntime.JSONValue(false),
				},
			},
		}
		score, ok := metricScore(agent, run, sample, "response_completeness", nil)
		if !ok || score != 1.0 {
			t.Fatalf("expected 1.0, got %v ok=%v", score, ok)
		}
	})

	t.Run("output_coherence via metricScore", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"hazards":          agentruntime.JSONValue([]interface{}{map[string]interface{}{}}),
					"overallRiskLevel": agentruntime.JSONValue("medium"),
					"confidence":       agentruntime.JSONValue(0.9),
					"needsHumanReview": agentruntime.JSONValue(false),
				},
			},
		}
		score, ok := metricScore(agent, run, sample, "output_coherence", nil)
		if !ok || score != 1.0 {
			t.Fatalf("expected 1.0, got %v ok=%v", score, ok)
		}
	})

	t.Run("actionable_next_steps via metricScore", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Output: apiv1alpha1.FreeformObject{
					"nextActions": agentruntime.JSONValue([]string{"Replace the damaged electrical panel in room 301"}),
				},
			},
		}
		score, ok := metricScore(agent, run, sample, "actionable_next_steps", nil)
		if !ok || score != 1.0 {
			t.Fatalf("expected 1.0, got %v ok=%v", score, ok)
		}
	})

	t.Run("confidence_calibration via metricScore", func(t *testing.T) {
		run := apiv1alpha1.AgentRun{
			Status: apiv1alpha1.AgentRunStatus{
				Phase: string(apiv1alpha1.AgentRunPhaseSucceeded),
				Output: apiv1alpha1.FreeformObject{
					"confidence": agentruntime.JSONValue(0.88),
				},
			},
		}
		score, ok := metricScore(agent, run, sample, "confidence_calibration", nil)
		if !ok || score != 1.0 {
			t.Fatalf("expected 1.0, got %v ok=%v", score, ok)
		}
	})
}

func TestCELMetricScore(t *testing.T) {
	run := apiv1alpha1.AgentRun{
		Status: apiv1alpha1.AgentRunStatus{
			Phase: string(apiv1alpha1.AgentRunPhaseSucceeded),
			Output: apiv1alpha1.FreeformObject{
				"hazards":          agentruntime.JSONValue([]interface{}{"h1", "h2", "h3"}),
				"overallRiskLevel": agentruntime.JSONValue("high"),
				"confidence":       agentruntime.JSONValue(0.9),
			},
		},
	}
	sample := evaluationSample{
		Name:     "test-case",
		Expected: apiv1alpha1.FreeformObject{},
	}

	t.Run("bool expression true", func(t *testing.T) {
		config := apiv1alpha1.FreeformObject{
			"expression": agentruntime.JSONValue("size(output.hazards) > 2 && output.overallRiskLevel != 'low'"),
		}
		score, ok := celMetricScore(run, sample, config)
		if !ok || score != 1.0 {
			t.Fatalf("expected 1.0, got %v ok=%v", score, ok)
		}
	})

	t.Run("bool expression false", func(t *testing.T) {
		config := apiv1alpha1.FreeformObject{
			"expression": agentruntime.JSONValue("output.overallRiskLevel == 'low'"),
		}
		score, ok := celMetricScore(run, sample, config)
		if !ok || score != 0.0 {
			t.Fatalf("expected 0.0, got %v ok=%v", score, ok)
		}
	})

	t.Run("numeric expression", func(t *testing.T) {
		config := apiv1alpha1.FreeformObject{
			"expression": agentruntime.JSONValue("size(output.hazards)"),
		}
		score, ok := celMetricScore(run, sample, config)
		if !ok || score != 3.0 {
			t.Fatalf("expected 3.0, got %v ok=%v", score, ok)
		}
	})

	t.Run("empty config returns false", func(t *testing.T) {
		score, ok := celMetricScore(run, sample, nil)
		if ok {
			t.Fatalf("expected ok=false, got score=%v", score)
		}
	})

	t.Run("missing expression key", func(t *testing.T) {
		config := apiv1alpha1.FreeformObject{
			"other": agentruntime.JSONValue("value"),
		}
		score, ok := celMetricScore(run, sample, config)
		if ok {
			t.Fatalf("expected ok=false, got score=%v", score)
		}
	})

	t.Run("invalid expression returns false", func(t *testing.T) {
		config := apiv1alpha1.FreeformObject{
			"expression": agentruntime.JSONValue("output.???"),
		}
		score, ok := celMetricScore(run, sample, config)
		if ok {
			t.Fatalf("expected ok=false for invalid expression, got score=%v", score)
		}
	})
}
