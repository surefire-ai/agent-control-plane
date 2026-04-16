package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const agentRunCompletedCondition = "Completed"

type AgentRunReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	Clock  func() metav1.Time
}

func (r *AgentRunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var run apiv1alpha1.AgentRun
	if err := r.Get(ctx, req.NamespacedName, &run); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if isTerminalAgentRunPhase(run.Status.Phase) {
		return ctrl.Result{}, nil
	}

	original := run.DeepCopy()
	previousStatus := run.Status.DeepCopy()
	now := r.now()

	var agent apiv1alpha1.Agent
	agentKey := types.NamespacedName{
		Namespace: req.Namespace,
		Name:      run.Spec.AgentRef.Name,
	}
	if err := r.Get(ctx, agentKey, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			setAgentRunFailed(&run, now, fmt.Sprintf("referenced Agent %q not found", run.Spec.AgentRef.Name))
			return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
		}
		return ctrl.Result{}, err
	}

	if !isAgentReady(agent) {
		setAgentRunPending(&run, now, fmt.Sprintf("waiting for Agent %q to become Ready", agent.Name))
		return ctrl.Result{RequeueAfter: time.Second}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	}

	switch run.Status.Phase {
	case "":
		setAgentRunPending(&run, now, "AgentRun accepted")
		return ctrl.Result{Requeue: true}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	case string(apiv1alpha1.AgentRunPhasePending):
		setAgentRunRunning(&run, agent, now)
		return ctrl.Result{Requeue: true}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	case string(apiv1alpha1.AgentRunPhaseRunning):
		output := buildMockAgentRunOutput(run, agent)
		setAgentRunSucceeded(&run, agent, now, output)
		return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	default:
		setAgentRunFailed(&run, now, "unsupported AgentRun phase "+run.Status.Phase)
		return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	}
}

func (r *AgentRunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.AgentRun{}).
		Complete(r)
}

func (r *AgentRunReconciler) patchAgentRunStatusIfChanged(ctx context.Context, run *apiv1alpha1.AgentRun, original *apiv1alpha1.AgentRun, previous *apiv1alpha1.AgentRunStatus) error {
	if equality.Semantic.DeepEqual(previous, &run.Status) {
		return nil
	}
	return r.Status().Patch(ctx, run, client.MergeFrom(original))
}

func (r *AgentRunReconciler) now() metav1.Time {
	if r.Clock != nil {
		return r.Clock()
	}
	return metav1.Now()
}

func isTerminalAgentRunPhase(phase string) bool {
	return phase == string(apiv1alpha1.AgentRunPhaseSucceeded) ||
		phase == string(apiv1alpha1.AgentRunPhaseFailed)
}

func isAgentReady(agent apiv1alpha1.Agent) bool {
	for _, condition := range agent.Status.Conditions {
		if condition.Type == agentReadyCondition && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func setAgentRunPending(run *apiv1alpha1.AgentRun, now metav1.Time, message string) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhasePending)
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "Pending",
		Message:            message,
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func setAgentRunRunning(run *apiv1alpha1.AgentRun, agent apiv1alpha1.Agent, now metav1.Time) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhaseRunning)
	run.Status.AgentRevision = agent.Status.CompiledRevision
	run.Status.StartedAt = &now
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "Running",
		Message:            "mock runtime started",
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func setAgentRunSucceeded(run *apiv1alpha1.AgentRun, agent apiv1alpha1.Agent, now metav1.Time, output apiv1alpha1.FreeformObject) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhaseSucceeded)
	run.Status.AgentRevision = agent.Status.CompiledRevision
	run.Status.FinishedAt = &now
	run.Status.Output = output
	run.Status.TraceRef = apiv1alpha1.FreeformObject{
		"provider": jsonValue("mock"),
		"runId":    jsonValue(run.Namespace + "/" + run.Name),
	}
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "MockRuntimeSucceeded",
		Message:            "mock runtime completed successfully",
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func setAgentRunFailed(run *apiv1alpha1.AgentRun, now metav1.Time, message string) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhaseFailed)
	run.Status.FinishedAt = &now
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "Failed",
		Message:            message,
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func buildMockAgentRunOutput(run apiv1alpha1.AgentRun, agent apiv1alpha1.Agent) apiv1alpha1.FreeformObject {
	task := jsonString(run.Spec.Input, "task")
	if task == "" {
		task = "agent_run"
	}

	summary := strings.TrimSpace(fmt.Sprintf("Mock execution completed for %s using %s.", task, agent.Name))
	return apiv1alpha1.FreeformObject{
		"summary":          jsonValue(summary),
		"hazards":          jsonValue([]interface{}{}),
		"overallRiskLevel": jsonValue("low"),
		"nextActions":      jsonValue([]string{"review mock result before enabling a real runtime"}),
		"confidence":       jsonValue(1.0),
		"needsHumanReview": jsonValue(false),
	}
}

func jsonString(values apiv1alpha1.FreeformObject, key string) string {
	value, ok := values[key]
	if !ok {
		return ""
	}
	var result string
	if err := json.Unmarshal(value.Raw, &result); err != nil {
		return ""
	}
	return result
}

func jsonValue(value interface{}) apiextensionsv1.JSON {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte("null")
	}
	return apiextensionsv1.JSON{Raw: raw}
}
