package controller

import (
	"context"
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

	setAgentEvaluationReady(&evaluation, req.Namespace, agent.Status.CompiledRevision, baselineRevision)
	return ctrl.Result{}, r.patchAgentEvaluationStatusIfChanged(ctx, &evaluation, original, previousStatus)
}

func (r *AgentEvaluationReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.AgentEvaluation{}).
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
	evaluation.Status.Summary.DatasetRevision = evaluation.Spec.DatasetRef.Revision
	evaluation.Status.Summary.BaselineRevision = baselineRevision
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

func setAgentEvaluationStatus(evaluation *apiv1alpha1.AgentEvaluation, namespace string, phase string, condition metav1.Condition) {
	evaluation.Status.Phase = phase
	evaluation.Status.ObservedGeneration = evaluation.Generation
	evaluation.Status.ReportRef = apiv1alpha1.FreeformObject{
		"provider": jsonValue("kubernetes-status"),
		"report": jsonValue(
			"/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version +
				"/namespaces/" + namespace + "/agentevaluations/" + evaluation.Name + ":report",
		),
	}
	evaluation.Status.Conditions = mergeCondition(evaluation.Status.Conditions, condition)
}

func jsonValue(value string) apiextensionsv1.JSON {
	return apiextensionsv1.JSON{Raw: []byte(fmt.Sprintf("%q", value))}
}
