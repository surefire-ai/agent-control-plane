package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const agentEvaluationReadyCondition = "Ready"

type AgentEvaluationReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

type evaluationSample struct {
	Name     string
	Input    apiv1alpha1.FreeformObject
	Expected apiv1alpha1.FreeformObject
}

func (r *AgentEvaluationReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var evaluation apiv1alpha1.AgentEvaluation
	if err := r.Get(ctx, req.NamespacedName, &evaluation); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	original := evaluation.DeepCopy()
	previousStatus := evaluation.Status.DeepCopy()

	agent, err := r.resolveAgent(ctx, req.Namespace, evaluation.Spec.AgentRef.Name)
	if err != nil {
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "AgentReferenceFailed", err.Error())
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}
	if !isAgentReady(*agent) {
		setAgentEvaluationPending(&evaluation, req.Namespace, "WaitingForAgent", fmt.Sprintf("waiting for Agent %q to become Ready", agent.Name))
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}

	baselineRevision, err := r.resolveBaselineRevision(ctx, req.Namespace, evaluation.Spec, agent.Status.CompiledRevision)
	if err != nil {
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "BaselineReferenceFailed", err.Error())
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}

	samples, hasSamples, err := r.evaluationSamples(ctx, req.Namespace, evaluation.Spec)
	if err != nil {
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "InvalidEvaluationRuntime", err.Error())
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}
	if !hasSamples {
		setAgentEvaluationReady(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision)
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}

	runs, created, err := r.ensureEvaluationRuns(ctx, &evaluation, samples)
	if err != nil {
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "EvaluationRunCreateFailed", err.Error())
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}
	if created {
		setAgentEvaluationRunning(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision, runs, "created managed AgentRun set for evaluation execution")
		return ctrl.Result{Requeue: true}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}

	allTerminal := true
	hasFailed := false
	unsupported := ""
	for _, run := range runs {
		switch run.Status.Phase {
		case "", string(apiv1alpha1.AgentRunPhasePending), string(apiv1alpha1.AgentRunPhaseRunning):
			allTerminal = false
		case string(apiv1alpha1.AgentRunPhaseSucceeded):
		case string(apiv1alpha1.AgentRunPhaseFailed):
			hasFailed = true
		default:
			unsupported = fmt.Sprintf("managed AgentRun %q is in unsupported phase %q", run.Name, run.Status.Phase)
		}
	}
	if unsupported != "" {
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "EvaluationRunUnknownPhase", unsupported)
	} else if !allTerminal {
		setAgentEvaluationRunning(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision, runs, "waiting for managed AgentRun set to complete")
	} else if hasFailed {
		setAgentEvaluationFailed(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision, runs)
	} else {
		setAgentEvaluationSucceeded(&evaluation, req.Namespace, agent, baselineRevision, runs, samples)
	}

	return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
}

func (r *AgentEvaluationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.AgentEvaluation{}).
		Owns(&apiv1alpha1.AgentRun{}).
		Complete(r)
}

func (r *AgentEvaluationReconciler) resolveAgent(ctx context.Context, namespace string, name string) (*apiv1alpha1.Agent, error) {
	var agent apiv1alpha1.Agent
	if err := r.Get(ctx, types.NamespacedName{Namespace: namespace, Name: name}, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("referenced Agent %q not found", name)
		}
		return nil, err
	}
	return &agent, nil
}

func (r *AgentEvaluationReconciler) resolveBaselineRevision(ctx context.Context, namespace string, spec apiv1alpha1.AgentEvaluationSpec, defaultRevision string) (string, error) {
	if spec.Baseline == nil {
		return defaultRevision, nil
	}
	if spec.Baseline.Revision != "" {
		return spec.Baseline.Revision, nil
	}
	if spec.Baseline.Reference != "" {
		return spec.Baseline.Reference, nil
	}
	if spec.Baseline.AgentRef == nil || spec.Baseline.AgentRef.Name == "" {
		return defaultRevision, nil
	}
	agent, err := r.resolveAgent(ctx, namespace, spec.Baseline.AgentRef.Name)
	if err != nil {
		return "", err
	}
	if agent.Status.CompiledRevision == "" {
		return "", fmt.Errorf("baseline Agent %q has no compiled revision yet", agent.Name)
	}
	return agent.Status.CompiledRevision, nil
}

func (r *AgentEvaluationReconciler) resolveDataset(ctx context.Context, namespace string, ref apiv1alpha1.EvaluationDatasetReference) (*apiv1alpha1.Dataset, error) {
	datasetNamespace := namespace
	if strings.TrimSpace(ref.Namespace) != "" {
		datasetNamespace = ref.Namespace
	}
	var dataset apiv1alpha1.Dataset
	if err := r.Get(ctx, types.NamespacedName{Namespace: datasetNamespace, Name: ref.Name}, &dataset); err != nil {
		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("referenced Dataset %q not found", ref.Name)
		}
		return nil, err
	}
	if ref.Revision != "" && dataset.Spec.Revision != "" && ref.Revision != dataset.Spec.Revision {
		return nil, fmt.Errorf("referenced Dataset %q revision mismatch: expected %q, got %q", ref.Name, ref.Revision, dataset.Spec.Revision)
	}
	return &dataset, nil
}

func (r *AgentEvaluationReconciler) patchAgentEvaluationStatusIfChanged(ctx context.Context, evaluation *apiv1alpha1.AgentEvaluation, original *apiv1alpha1.AgentEvaluation, previous *apiv1alpha1.AgentEvaluationStatus) error {
	if equality.Semantic.DeepEqual(previous, &evaluation.Status) {
		return nil
	}
	return r.Status().Patch(ctx, evaluation, client.MergeFrom(original))
}

func (r *AgentEvaluationReconciler) ensureEvaluationRuns(ctx context.Context, evaluation *apiv1alpha1.AgentEvaluation, samples []evaluationSample) ([]apiv1alpha1.AgentRun, bool, error) {
	runs := make([]apiv1alpha1.AgentRun, 0, len(samples))
	createdAny := false
	for index, sample := range samples {
		runName := desiredEvaluationRunName(*evaluation, sample.Name)
		key := types.NamespacedName{Namespace: evaluation.Namespace, Name: runName}

		var existing apiv1alpha1.AgentRun
		if err := r.Get(ctx, key, &existing); err == nil {
			runs = append(runs, existing)
			continue
		} else if !apierrors.IsNotFound(err) {
			return nil, false, err
		}

		run := &apiv1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      runName,
				Namespace: evaluation.Namespace,
				Labels: map[string]string{
					"windosx.com/evaluation": evaluation.Name,
				},
				Annotations: map[string]string{
					"windosx.com/evaluation-generation":   fmt.Sprintf("%d", evaluation.Generation),
					"windosx.com/evaluation-sample":       sample.Name,
					"windosx.com/evaluation-sample-index": fmt.Sprintf("%d", index),
				},
			},
			Spec: apiv1alpha1.AgentRunSpec{
				AgentRef: evaluation.Spec.AgentRef,
				Input:    sample.Input,
			},
		}
		if err := controllerutil.SetControllerReference(evaluation, run, r.Scheme); err != nil {
			return nil, false, err
		}
		if err := r.Create(ctx, run); err != nil {
			return nil, false, err
		}
		runs = append(runs, *run)
		createdAny = true
	}
	return runs, createdAny, nil
}

func setAgentEvaluationPending(evaluation *apiv1alpha1.AgentEvaluation, namespace string, reason string, message string) {
	setAgentEvaluationStatus(evaluation, namespace, "Pending", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationNotReady(evaluation *apiv1alpha1.AgentEvaluation, namespace string, reason string, message string) {
	setAgentEvaluationStatus(evaluation, namespace, "NotReady", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationReady(evaluation *apiv1alpha1.AgentEvaluation, namespace string, agentRevision string, baselineRevision string) {
	evaluation.Status.Results = nil
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
	evaluation.Status.Summary.SamplesTotal = 0
	evaluation.Status.Summary.SamplesEvaluated = 0
	evaluation.Status.Summary.Score = 0
	evaluation.Status.LatestRunRef = map[string]string{
		"agentRef":      evaluation.Spec.AgentRef.Name,
		"agentRevision": agentRevision,
	}
	if len(evaluation.Spec.Thresholds) == 0 {
		evaluation.Status.Summary.GatePassed = true
	}
	setAgentEvaluationStatus(evaluation, namespace, "Ready", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "ContractResolved",
		Message:            "evaluation contract resolved and ready for execution",
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationRunning(evaluation *apiv1alpha1.AgentEvaluation, namespace string, agentRevision string, baselineRevision string, runs []apiv1alpha1.AgentRun, message string) {
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
	evaluation.Status.Summary.SamplesTotal = int32(len(runs))
	evaluation.Status.Summary.SamplesEvaluated = countTerminalRuns(runs)
	latest := latestRun(runs)
	evaluation.Status.LatestRunRef = map[string]string{
		"name":          latest.Name,
		"agentRef":      evaluation.Spec.AgentRef.Name,
		"agentRevision": agentRevision,
		"phase":         latest.Status.Phase,
	}
	setAgentEvaluationStatus(evaluation, namespace, "Running", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "EvaluationRunInProgress",
		Message:            message,
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationSucceeded(evaluation *apiv1alpha1.AgentEvaluation, namespace string, agent *apiv1alpha1.Agent, baselineRevision string, runs []apiv1alpha1.AgentRun, samples []evaluationSample) {
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
	evaluation.Status.Summary.SamplesTotal = int32(len(runs))
	evaluation.Status.Summary.SamplesEvaluated = int32(len(runs))
	latest := latestRun(runs)
	evaluation.Status.LatestRunRef = map[string]string{
		"name":          latest.Name,
		"agentRef":      evaluation.Spec.AgentRef.Name,
		"agentRevision": agent.Status.CompiledRevision,
		"phase":         latest.Status.Phase,
	}
	evaluation.Status.Results = buildEvaluationMetricResults(*agent, runs, samples, evaluation.Spec.Thresholds)
	evaluation.Status.Summary.Score = aggregateMetricScore(evaluation.Status.Results)
	evaluation.Status.Summary.GatePassed = gatePassed(evaluation.Spec.Gate, evaluation.Status.Results)
	evaluation.Status.ReportRef = apiv1alpha1.FreeformObject{
		"provider": jsonValue("kubernetes-status"),
		"report": jsonValue(
			"/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version +
				"/namespaces/" + namespace + "/agentevaluations/" + evaluation.Name + ":report",
		),
		"runName":  jsonValue(latest.Name),
		"runCount": jsonAnyValue(len(runs)),
	}
	setAgentEvaluationStatus(evaluation, namespace, "Succeeded", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "EvaluationRunSucceeded",
		Message:            "evaluation executed through managed AgentRun",
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationFailed(evaluation *apiv1alpha1.AgentEvaluation, namespace string, agentRevision string, baselineRevision string, runs []apiv1alpha1.AgentRun) {
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
	evaluation.Status.Summary.SamplesTotal = int32(len(runs))
	evaluation.Status.Summary.SamplesEvaluated = countTerminalRuns(runs)
	evaluation.Status.Results = buildEvaluationMetricResultsForFailures(runs, evaluation.Spec.Thresholds)
	evaluation.Status.Summary.Score = aggregateMetricScore(evaluation.Status.Results)
	evaluation.Status.Summary.GatePassed = false
	latest := latestRun(runs)
	evaluation.Status.LatestRunRef = map[string]string{
		"name":          latest.Name,
		"agentRef":      evaluation.Spec.AgentRef.Name,
		"agentRevision": agentRevision,
		"phase":         latest.Status.Phase,
	}
	setAgentEvaluationStatus(evaluation, namespace, "Failed", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "EvaluationRunFailed",
		Message:            "one or more managed AgentRuns failed",
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationStatus(evaluation *apiv1alpha1.AgentEvaluation, namespace string, phase string, condition metav1.Condition) {
	evaluation.Status.Phase = phase
	evaluation.Status.ObservedGeneration = evaluation.Generation
	if len(evaluation.Status.ReportRef) == 0 {
		evaluation.Status.ReportRef = apiv1alpha1.FreeformObject{
			"provider": jsonValue("kubernetes-status"),
			"report": jsonValue(
				"/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version +
					"/namespaces/" + namespace + "/agentevaluations/" + evaluation.Name + ":report",
			),
		}
	}
	evaluation.Status.Conditions = mergeCondition(evaluation.Status.Conditions, condition)
}

var invalidRunNameChars = regexp.MustCompile(`[^a-z0-9-]+`)

func desiredEvaluationRunName(evaluation apiv1alpha1.AgentEvaluation, sampleName string) string {
	slug := strings.ToLower(sampleName)
	slug = invalidRunNameChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		slug = "sample"
	}
	return fmt.Sprintf("%s-run-g%d-%s", evaluation.Name, evaluation.Generation, slug)
}

func (r *AgentEvaluationReconciler) evaluationSamples(ctx context.Context, namespace string, spec apiv1alpha1.AgentEvaluationSpec) ([]evaluationSample, bool, error) {
	if value, ok := spec.Runtime["samples"]; ok {
		var raw []struct {
			Name     string                 `json:"name"`
			Input    map[string]interface{} `json:"input"`
			Expected map[string]interface{} `json:"expected,omitempty"`
		}
		if err := json.Unmarshal(value.Raw, &raw); err != nil {
			return nil, false, fmt.Errorf("spec.runtime.samples must be a JSON array of {name,input}: %w", err)
		}
		samples := make([]evaluationSample, 0, len(raw))
		for index, item := range raw {
			if len(item.Input) == 0 {
				return nil, false, fmt.Errorf("spec.runtime.samples[%d].input must be a JSON object", index)
			}
			name := item.Name
			if strings.TrimSpace(name) == "" {
				name = fmt.Sprintf("sample-%d", index)
			}
			result := apiv1alpha1.FreeformObject{}
			for key, value := range item.Input {
				result[key] = jsonAnyValue(value)
			}
			expected := apiv1alpha1.FreeformObject{}
			for key, value := range item.Expected {
				expected[key] = jsonAnyValue(value)
			}
			samples = append(samples, evaluationSample{Name: name, Input: result, Expected: expected})
		}
		return samples, len(samples) > 0, nil
	}
	value, ok := spec.Runtime["sampleInput"]
	if ok {
		var input map[string]interface{}
		if err := json.Unmarshal(value.Raw, &input); err != nil {
			return nil, false, fmt.Errorf("spec.runtime.sampleInput must be a JSON object: %w", err)
		}
		result := apiv1alpha1.FreeformObject{}
		for key, item := range input {
			result[key] = jsonAnyValue(item)
		}
		return []evaluationSample{{Name: "sample-0", Input: result}}, true, nil
	}
	if strings.EqualFold(spec.DatasetRef.Kind, "Dataset") || spec.DatasetRef.Kind == "" {
		dataset, err := r.resolveDataset(ctx, namespace, spec.DatasetRef)
		if err != nil {
			return nil, false, err
		}
		samples := make([]evaluationSample, 0, len(dataset.Spec.Samples))
		for index, sample := range dataset.Spec.Samples {
			if len(sample.Input) == 0 {
				return nil, false, fmt.Errorf("dataset %q sample %d has empty input", dataset.Name, index)
			}
			name := sample.Name
			if strings.TrimSpace(name) == "" {
				name = fmt.Sprintf("sample-%d", index)
			}
			samples = append(samples, evaluationSample{Name: name, Input: sample.Input.DeepCopy(), Expected: sample.Expected.DeepCopy()})
		}
		return samples, len(samples) > 0, nil
	}
	return nil, false, nil
}

func buildEvaluationMetricResults(agent apiv1alpha1.Agent, runs []apiv1alpha1.AgentRun, samples []evaluationSample, thresholds []apiv1alpha1.EvaluationThresholdSpec) []apiv1alpha1.EvaluationMetricStatus {
	results := make([]apiv1alpha1.EvaluationMetricStatus, 0, len(thresholds))
	for _, threshold := range thresholds {
		score, ok := aggregateMetric(agent, runs, samples, threshold.Metric)
		result := apiv1alpha1.EvaluationMetricStatus{
			Name:      threshold.Metric,
			Metric:    threshold.Metric,
			Score:     score,
			Threshold: threshold.Target,
			Passed:    false,
		}
		if !ok {
			result.Reason = "metric unavailable in current evaluation runtime"
			results = append(results, result)
			continue
		}
		result.Passed = passesThreshold(score, threshold)
		if !result.Passed {
			result.Reason = fmt.Sprintf("metric %q did not satisfy %s %.4f", threshold.Metric, thresholdOperator(threshold), threshold.Target)
		}
		results = append(results, result)
	}
	return results
}

func buildEvaluationMetricResultsForFailures(runs []apiv1alpha1.AgentRun, thresholds []apiv1alpha1.EvaluationThresholdSpec) []apiv1alpha1.EvaluationMetricStatus {
	results := make([]apiv1alpha1.EvaluationMetricStatus, 0, len(thresholds))
	for _, threshold := range thresholds {
		results = append(results, apiv1alpha1.EvaluationMetricStatus{
			Name:      threshold.Metric,
			Metric:    threshold.Metric,
			Score:     0,
			Threshold: threshold.Target,
			Passed:    false,
			Reason:    "one or more managed AgentRuns failed",
		})
	}
	if len(thresholds) == 0 {
		results = append(results, apiv1alpha1.EvaluationMetricStatus{
			Name:   "run_success",
			Metric: "run_success",
			Score:  0,
			Passed: false,
			Reason: "one or more managed AgentRuns failed",
		})
	}
	return results
}

func aggregateMetric(agent apiv1alpha1.Agent, runs []apiv1alpha1.AgentRun, samples []evaluationSample, metric string) (float64, bool) {
	if len(runs) == 0 {
		return 0, false
	}
	total := 0.0
	seen := 0
	for _, run := range runs {
		sample, _ := sampleForRun(samples, run.Name)
		score, ok := metricScore(agent, run, sample, metric)
		if ok {
			total += score
			seen++
		}
	}
	if seen == 0 {
		return 0, false
	}
	return total / float64(len(runs)), true
}

func metricScore(agent apiv1alpha1.Agent, run apiv1alpha1.AgentRun, sample evaluationSample, metric string) (float64, bool) {
	switch metric {
	case "run_success":
		if run.Status.Phase == string(apiv1alpha1.AgentRunPhaseSucceeded) {
			return 1, true
		}
		return 0, true
	case "schema_validity":
		required := requiredOutputFields(agent)
		if len(required) == 0 {
			required = []string{"summary", "hazards", "overallRiskLevel", "nextActions", "confidence", "needsHumanReview"}
		}
		matched := 0
		for _, field := range required {
			if raw, ok := run.Status.Output[field]; ok && len(raw.Raw) > 0 && string(raw.Raw) != "null" {
				matched++
			}
		}
		return float64(matched) / float64(len(required)), true
	case "risk_level_match":
		return riskLevelMatchScore(run, sample)
	case "hazard_coverage":
		return hazardCoverageScore(run, sample)
	}
	if score, ok := expectedMetricScore(run, sample, metric); ok {
		return score, true
	}
	if value, ok := run.Status.Output[metric]; ok {
		var number float64
		if err := json.Unmarshal(value.Raw, &number); err == nil {
			return number, true
		}
		var boolean bool
		if err := json.Unmarshal(value.Raw, &boolean); err == nil {
			if boolean {
				return 1, true
			}
			return 0, true
		}
	}
	return 0, false
}

var riskOrdinal = map[string]int{
	"low":      0,
	"medium":   1,
	"high":     2,
	"critical": 3,
}

func riskLevelMatchScore(run apiv1alpha1.AgentRun, sample evaluationSample) (float64, bool) {
	expectedValue, ok := sample.Expected["overallRiskLevel"]
	if !ok {
		return 0, false
	}
	actualValue, ok := run.Status.Output["overallRiskLevel"]
	if !ok {
		return 0, false
	}
	expected := strings.ToLower(strings.TrimSpace(jsonString(expectedValue)))
	actual := strings.ToLower(strings.TrimSpace(jsonString(actualValue)))
	expectedOrdinal, okExpected := riskOrdinal[expected]
	actualOrdinal, okActual := riskOrdinal[actual]
	if !okExpected || !okActual {
		return 0, false
	}
	delta := expectedOrdinal - actualOrdinal
	if delta < 0 {
		delta = -delta
	}
	switch delta {
	case 0:
		return 1, true
	case 1:
		return 0.5, true
	default:
		return 0, true
	}
}

func hazardCoverageScore(run apiv1alpha1.AgentRun, sample evaluationSample) (float64, bool) {
	expectedValue, ok := sample.Expected["hazards"]
	if !ok {
		return 0, false
	}
	actualValue, ok := run.Status.Output["hazards"]
	if !ok {
		return 0, false
	}
	expectedHazards, ok := hazardDescriptors(expectedValue)
	if !ok || len(expectedHazards) == 0 {
		return 0, false
	}
	actualHazards, ok := hazardDescriptors(actualValue)
	if !ok {
		return 0, false
	}
	matched := 0
	for _, expected := range expectedHazards {
		if hasHazardMatch(actualHazards, expected) {
			matched++
		}
	}
	return float64(matched) / float64(len(expectedHazards)), true
}

type hazardDescriptor struct {
	Title    string
	Category string
}

func hazardDescriptors(value apiextensionsv1.JSON) ([]hazardDescriptor, bool) {
	var raw []map[string]interface{}
	if err := json.Unmarshal(value.Raw, &raw); err != nil {
		return nil, false
	}
	descriptors := make([]hazardDescriptor, 0, len(raw))
	for _, item := range raw {
		title, _ := item["title"].(string)
		category, _ := item["category"].(string)
		descriptors = append(descriptors, hazardDescriptor{
			Title:    strings.ToLower(strings.TrimSpace(title)),
			Category: strings.ToLower(strings.TrimSpace(category)),
		})
	}
	return descriptors, true
}

func hasHazardMatch(actual []hazardDescriptor, expected hazardDescriptor) bool {
	for _, item := range actual {
		if expected.Category != "" && item.Category == expected.Category {
			return true
		}
		if expected.Title != "" && item.Title == expected.Title {
			return true
		}
	}
	return false
}

func expectedMetricScore(run apiv1alpha1.AgentRun, sample evaluationSample, metric string) (float64, bool) {
	if len(sample.Expected) == 0 {
		return 0, false
	}
	if strings.HasSuffix(metric, "_count") {
		field := strings.TrimSuffix(metric, "_count")
		expectedValue, ok := sample.Expected[metric]
		if !ok {
			return 0, false
		}
		expectedCount, ok := jsonInt(expectedValue)
		if !ok {
			return 0, false
		}
		actualValue, ok := run.Status.Output[field]
		if !ok {
			return 0, false
		}
		actualCount, ok := jsonArrayLen(actualValue)
		if !ok {
			return 0, false
		}
		if actualCount == expectedCount {
			return 1, true
		}
		return 0, true
	}
	expectedValue, ok := sample.Expected[metric]
	if !ok {
		return 0, false
	}
	actualValue, ok := run.Status.Output[metric]
	if !ok {
		return 0, false
	}
	expectedAny := jsonAny(expectedValue)
	actualAny := jsonAny(actualValue)
	if equality.Semantic.DeepEqual(expectedAny, actualAny) {
		return 1, true
	}
	return 0, true
}

func requiredOutputFields(agent apiv1alpha1.Agent) []string {
	if len(agent.Spec.Interfaces.Output.Schema.Raw) == 0 {
		return nil
	}
	var schema struct {
		Required []string `json:"required"`
	}
	if err := json.Unmarshal(agent.Spec.Interfaces.Output.Schema.Raw, &schema); err != nil {
		return nil
	}
	return schema.Required
}

func aggregateMetricScore(results []apiv1alpha1.EvaluationMetricStatus) float64 {
	if len(results) == 0 {
		return 0
	}
	total := 0.0
	for _, result := range results {
		total += result.Score
	}
	return total / float64(len(results))
}

func countTerminalRuns(runs []apiv1alpha1.AgentRun) int32 {
	count := int32(0)
	for _, run := range runs {
		if run.Status.Phase == string(apiv1alpha1.AgentRunPhaseSucceeded) || run.Status.Phase == string(apiv1alpha1.AgentRunPhaseFailed) {
			count++
		}
	}
	return count
}

func latestRun(runs []apiv1alpha1.AgentRun) apiv1alpha1.AgentRun {
	if len(runs) == 0 {
		return apiv1alpha1.AgentRun{}
	}
	return runs[len(runs)-1]
}

func sampleForRun(samples []evaluationSample, runName string) (evaluationSample, bool) {
	for _, sample := range samples {
		if strings.HasSuffix(runName, "-"+sanitizeSampleName(sample.Name)) {
			return sample, true
		}
	}
	return evaluationSample{}, false
}

func sanitizeSampleName(name string) string {
	slug := strings.ToLower(name)
	slug = invalidRunNameChars.ReplaceAllString(slug, "-")
	slug = strings.Trim(slug, "-")
	if slug == "" {
		return "sample"
	}
	return slug
}

func jsonAny(value apiextensionsv1.JSON) interface{} {
	var result interface{}
	if err := json.Unmarshal(value.Raw, &result); err != nil {
		return nil
	}
	return result
}

func jsonInt(value apiextensionsv1.JSON) (int, bool) {
	var number int
	if err := json.Unmarshal(value.Raw, &number); err == nil {
		return number, true
	}
	var floatNumber float64
	if err := json.Unmarshal(value.Raw, &floatNumber); err == nil {
		return int(floatNumber), true
	}
	return 0, false
}

func jsonArrayLen(value apiextensionsv1.JSON) (int, bool) {
	var items []interface{}
	if err := json.Unmarshal(value.Raw, &items); err != nil {
		return 0, false
	}
	return len(items), true
}

func jsonString(value apiextensionsv1.JSON) string {
	var result string
	if err := json.Unmarshal(value.Raw, &result); err != nil {
		return ""
	}
	return result
}

func gatePassed(gate apiv1alpha1.EvaluationGateSpec, results []apiv1alpha1.EvaluationMetricStatus) bool {
	if len(results) == 0 {
		return true
	}
	byMetric := map[string]apiv1alpha1.EvaluationMetricStatus{}
	for _, result := range results {
		byMetric[result.Metric] = result
	}
	for _, required := range gate.Required {
		result, ok := byMetric[required]
		if !ok || !result.Passed {
			return false
		}
	}
	for _, result := range results {
		if !result.Passed && gate.BlockOnFail {
			return false
		}
	}
	return true
}

func passesThreshold(score float64, threshold apiv1alpha1.EvaluationThresholdSpec) bool {
	switch thresholdOperator(threshold) {
	case "lt":
		return score < threshold.Target
	case "lte":
		return score <= threshold.Target
	case "gt":
		return score > threshold.Target
	case "eq":
		return score == threshold.Target
	case "neq":
		return score != threshold.Target
	default:
		return score >= threshold.Target
	}
}

func thresholdOperator(threshold apiv1alpha1.EvaluationThresholdSpec) string {
	if threshold.Operator == "" {
		return "gte"
	}
	return threshold.Operator
}

func jsonValue(value string) apiextensionsv1.JSON {
	return jsonAnyValue(value)
}

func jsonAnyValue(value interface{}) apiextensionsv1.JSON {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte("null")
	}
	return apiextensionsv1.JSON{Raw: raw}
}
