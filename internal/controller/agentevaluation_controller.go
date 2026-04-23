package controller

import (
	"context"
	"encoding/json"
	"fmt"

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

	input, hasInput, err := evaluationRunInput(evaluation.Spec)
	if err != nil {
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "InvalidEvaluationRuntime", err.Error())
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}
	if !hasInput {
		setAgentEvaluationReady(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision)
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}

	run, created, err := r.ensureEvaluationRun(ctx, &evaluation, input)
	if err != nil {
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "EvaluationRunCreateFailed", err.Error())
		return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}
	if created {
		setAgentEvaluationRunning(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision, run.Name, "created managed AgentRun for evaluation execution")
		return ctrl.Result{Requeue: true}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
	}

	switch run.Status.Phase {
	case "", string(apiv1alpha1.AgentRunPhasePending), string(apiv1alpha1.AgentRunPhaseRunning):
		setAgentEvaluationRunning(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision, run.Name, "waiting for managed AgentRun to complete")
	case string(apiv1alpha1.AgentRunPhaseSucceeded):
		setAgentEvaluationSucceeded(&evaluation, req.Namespace, agent, baselineRevision, run)
	case string(apiv1alpha1.AgentRunPhaseFailed):
		setAgentEvaluationFailed(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision, run)
	default:
		setAgentEvaluationNotReady(&evaluation, req.Namespace, "EvaluationRunUnknownPhase", fmt.Sprintf("managed AgentRun %q is in unsupported phase %q", run.Name, run.Status.Phase))
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

func (r *AgentEvaluationReconciler) patchAgentEvaluationStatusIfChanged(ctx context.Context, evaluation *apiv1alpha1.AgentEvaluation, original *apiv1alpha1.AgentEvaluation, previous *apiv1alpha1.AgentEvaluationStatus) error {
	if equality.Semantic.DeepEqual(previous, &evaluation.Status) {
		return nil
	}
	return r.Status().Patch(ctx, evaluation, client.MergeFrom(original))
}

func (r *AgentEvaluationReconciler) ensureEvaluationRun(ctx context.Context, evaluation *apiv1alpha1.AgentEvaluation, input apiv1alpha1.FreeformObject) (*apiv1alpha1.AgentRun, bool, error) {
	runName := desiredEvaluationRunName(*evaluation)
	key := types.NamespacedName{Namespace: evaluation.Namespace, Name: runName}

	var existing apiv1alpha1.AgentRun
	if err := r.Get(ctx, key, &existing); err == nil {
		return &existing, false, nil
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
				"windosx.com/evaluation-generation": fmt.Sprintf("%d", evaluation.Generation),
			},
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: evaluation.Spec.AgentRef,
			Input:    input,
		},
	}
	if err := controllerutil.SetControllerReference(evaluation, run, r.Scheme); err != nil {
		return nil, false, err
	}
	if err := r.Create(ctx, run); err != nil {
		return nil, false, err
	}
	return run, true, nil
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

func setAgentEvaluationRunning(evaluation *apiv1alpha1.AgentEvaluation, namespace string, agentRevision string, baselineRevision string, runName string, message string) {
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
	evaluation.Status.Summary.SamplesTotal = 1
	evaluation.Status.Summary.SamplesEvaluated = 0
	evaluation.Status.LatestRunRef = map[string]string{
		"name":          runName,
		"agentRef":      evaluation.Spec.AgentRef.Name,
		"agentRevision": agentRevision,
		"phase":         string(apiv1alpha1.AgentRunPhaseRunning),
	}
	setAgentEvaluationStatus(evaluation, namespace, "Running", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "EvaluationRunInProgress",
		Message:            message,
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationSucceeded(evaluation *apiv1alpha1.AgentEvaluation, namespace string, agent *apiv1alpha1.Agent, baselineRevision string, run *apiv1alpha1.AgentRun) {
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
	evaluation.Status.Summary.SamplesTotal = 1
	evaluation.Status.Summary.SamplesEvaluated = 1
	evaluation.Status.LatestRunRef = map[string]string{
		"name":          run.Name,
		"agentRef":      evaluation.Spec.AgentRef.Name,
		"agentRevision": agent.Status.CompiledRevision,
		"phase":         run.Status.Phase,
	}
	evaluation.Status.Results = buildEvaluationMetricResults(*agent, *run, evaluation.Spec.Thresholds)
	evaluation.Status.Summary.Score = aggregateMetricScore(evaluation.Status.Results)
	evaluation.Status.Summary.GatePassed = gatePassed(evaluation.Spec.Gate, evaluation.Status.Results)
	evaluation.Status.ReportRef = apiv1alpha1.FreeformObject{
		"provider": jsonValue("kubernetes-status"),
		"report": jsonValue(
			"/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version +
				"/namespaces/" + namespace + "/agentevaluations/" + evaluation.Name + ":report",
		),
		"runName": jsonValue(run.Name),
	}
	setAgentEvaluationStatus(evaluation, namespace, "Succeeded", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "EvaluationRunSucceeded",
		Message:            "evaluation executed through managed AgentRun",
		ObservedGeneration: evaluation.Generation,
	})
}

func setAgentEvaluationFailed(evaluation *apiv1alpha1.AgentEvaluation, namespace string, agentRevision string, baselineRevision string, run *apiv1alpha1.AgentRun) {
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
	evaluation.Status.Summary.SamplesTotal = 1
	evaluation.Status.Summary.SamplesEvaluated = 1
	evaluation.Status.Summary.Score = 0
	evaluation.Status.Summary.GatePassed = false
	evaluation.Status.Results = []apiv1alpha1.EvaluationMetricStatus{
		{
			Name:   "run_success",
			Metric: "run_success",
			Score:  0,
			Passed: false,
			Reason: "managed AgentRun failed",
		},
	}
	evaluation.Status.LatestRunRef = map[string]string{
		"name":          run.Name,
		"agentRef":      evaluation.Spec.AgentRef.Name,
		"agentRevision": agentRevision,
		"phase":         run.Status.Phase,
	}
	setAgentEvaluationStatus(evaluation, namespace, "Failed", metav1.Condition{
		Type:               agentEvaluationReadyCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "EvaluationRunFailed",
		Message:            fmt.Sprintf("managed AgentRun %q failed", run.Name),
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

func desiredEvaluationRunName(evaluation apiv1alpha1.AgentEvaluation) string {
	return fmt.Sprintf("%s-run-g%d", evaluation.Name, evaluation.Generation)
}

func evaluationRunInput(spec apiv1alpha1.AgentEvaluationSpec) (apiv1alpha1.FreeformObject, bool, error) {
	value, ok := spec.Runtime["sampleInput"]
	if !ok {
		return nil, false, nil
	}
	var input map[string]interface{}
	if err := json.Unmarshal(value.Raw, &input); err != nil {
		return nil, false, fmt.Errorf("spec.runtime.sampleInput must be a JSON object: %w", err)
	}
	result := apiv1alpha1.FreeformObject{}
	for key, item := range input {
		result[key] = jsonAnyValue(item)
	}
	return result, true, nil
}

func buildEvaluationMetricResults(agent apiv1alpha1.Agent, run apiv1alpha1.AgentRun, thresholds []apiv1alpha1.EvaluationThresholdSpec) []apiv1alpha1.EvaluationMetricStatus {
	results := make([]apiv1alpha1.EvaluationMetricStatus, 0, len(thresholds))
	for _, threshold := range thresholds {
		score, ok := metricScore(agent, run, threshold.Metric)
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

func metricScore(agent apiv1alpha1.Agent, run apiv1alpha1.AgentRun, metric string) (float64, bool) {
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
