package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	agentruntime "github.com/surefire-ai/agent-control-plane/internal/runtime"
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
	Scheme  *runtime.Scheme
	Clock   func() metav1.Time
	Runtime agentruntime.Runner
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
			setAgentRunFailed(&run, now, explicitRunWorkspace(run), fmt.Sprintf("referenced Agent %q not found", run.Spec.AgentRef.Name))
			return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
		}
		return ctrl.Result{}, err
	}

	workspaceRef, err := resolveAgentRunWorkspace(run, agent)
	if err != nil {
		setAgentRunFailedWithReason(&run, now, workspaceRef, "WorkspaceMismatch", err.Error())
		return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	}

	if !isAgentReady(agent) {
		setAgentRunPending(&run, now, workspaceRef, fmt.Sprintf("waiting for Agent %q to become Ready", agent.Name))
		return ctrl.Result{RequeueAfter: time.Second}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	}

	switch run.Status.Phase {
	case "":
		setAgentRunPending(&run, now, workspaceRef, "AgentRun accepted")
		return ctrl.Result{Requeue: true}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	case string(apiv1alpha1.AgentRunPhasePending):
		setAgentRunRunning(&run, agent, now, workspaceRef)
		return ctrl.Result{Requeue: true}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	case string(apiv1alpha1.AgentRunPhaseRunning):
		result, err := r.runner().Execute(ctx, agentruntime.Request{
			Agent: agent,
			Run:   run,
		})
		if err != nil {
			if errors.Is(err, agentruntime.ErrRuntimeInProgress) {
				setAgentRunRunning(&run, agent, now, workspaceRef)
				return ctrl.Result{RequeueAfter: time.Second}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
			}
			var runtimeFailure agentruntime.Failure
			if errors.As(err, &runtimeFailure) {
				setAgentRunRuntimeFailed(&run, agent, now, workspaceRef, runtimeFailure)
				return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
			}
			setAgentRunFailed(&run, now, workspaceRef, err.Error())
			return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
		}
		setAgentRunSucceeded(&run, agent, now, workspaceRef, result)
		return ctrl.Result{}, r.patchAgentRunStatusIfChanged(ctx, &run, original, previousStatus)
	default:
		setAgentRunFailed(&run, now, workspaceRef, "unsupported AgentRun phase "+run.Status.Phase)
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

func (r *AgentRunReconciler) runner() agentruntime.Runner {
	if r.Runtime != nil {
		return r.Runtime
	}
	runtime := agentruntime.NewMockRuntime()
	return runtime
}

func isTerminalAgentRunPhase(phase string) bool {
	return phase == string(apiv1alpha1.AgentRunPhaseSucceeded) ||
		phase == string(apiv1alpha1.AgentRunPhaseFailed)
}

func isAgentReady(agent apiv1alpha1.Agent) bool {
	if agent.Status.CompiledRevision == "" || len(agent.Status.CompiledArtifact) == 0 {
		return false
	}
	for _, condition := range agent.Status.Conditions {
		if condition.Type == agentReadyCondition && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func explicitRunWorkspace(run apiv1alpha1.AgentRun) string {
	if run.Spec.WorkspaceRef == nil {
		return ""
	}
	return strings.TrimSpace(run.Spec.WorkspaceRef.Name)
}

func resolveAgentRunWorkspace(run apiv1alpha1.AgentRun, agent apiv1alpha1.Agent) (string, error) {
	runWorkspace := explicitRunWorkspace(run)
	agentWorkspace := strings.TrimSpace(agent.Status.WorkspaceRef)
	if agentWorkspace == "" && agent.Spec.WorkspaceRef != nil {
		agentWorkspace = strings.TrimSpace(agent.Spec.WorkspaceRef.Name)
	}
	if runWorkspace != "" && agentWorkspace != "" && runWorkspace != agentWorkspace {
		return runWorkspace, fmt.Errorf("AgentRun workspace %q does not match Agent workspace %q", runWorkspace, agentWorkspace)
	}
	if runWorkspace != "" {
		return runWorkspace, nil
	}
	return agentWorkspace, nil
}

func setAgentRunPending(run *apiv1alpha1.AgentRun, now metav1.Time, workspaceRef string, message string) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhasePending)
	run.Status.WorkspaceRef = workspaceRef
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "Pending",
		Message:            message,
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func setAgentRunRunning(run *apiv1alpha1.AgentRun, agent apiv1alpha1.Agent, now metav1.Time, workspaceRef string) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhaseRunning)
	run.Status.AgentRevision = agent.Status.CompiledRevision
	run.Status.WorkspaceRef = workspaceRef
	run.Status.StartedAt = &now
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionFalse,
		Reason:             "Running",
		Message:            "runtime started",
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func setAgentRunSucceeded(run *apiv1alpha1.AgentRun, agent apiv1alpha1.Agent, now metav1.Time, workspaceRef string, result agentruntime.Result) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhaseSucceeded)
	run.Status.AgentRevision = agent.Status.CompiledRevision
	run.Status.WorkspaceRef = workspaceRef
	run.Status.FinishedAt = &now
	run.Status.Output = result.Output
	run.Status.TraceRef = result.TraceRef
	reason := result.Reason
	if reason == "" {
		reason = "RuntimeSucceeded"
	}
	message := result.Message
	if message == "" {
		message = "runtime completed successfully"
	}
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionTrue,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func setAgentRunFailed(run *apiv1alpha1.AgentRun, now metav1.Time, workspaceRef string, message string) {
	setAgentRunFailedWithReason(run, now, workspaceRef, "Failed", message)
}

func setAgentRunFailedWithReason(run *apiv1alpha1.AgentRun, now metav1.Time, workspaceRef string, reason string, message string) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhaseFailed)
	run.Status.WorkspaceRef = workspaceRef
	run.Status.FinishedAt = &now
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}

func setAgentRunRuntimeFailed(run *apiv1alpha1.AgentRun, agent apiv1alpha1.Agent, now metav1.Time, workspaceRef string, failure agentruntime.Failure) {
	run.Status.Phase = string(apiv1alpha1.AgentRunPhaseFailed)
	run.Status.AgentRevision = agent.Status.CompiledRevision
	run.Status.WorkspaceRef = workspaceRef
	run.Status.FinishedAt = &now
	run.Status.Output = failure.Output
	run.Status.TraceRef = failure.TraceRef
	reason := failure.Reason
	if reason == "" {
		reason = "RuntimeFailed"
	}
	message := failure.Message
	if message == "" {
		message = failure.Error()
	}
	run.Status.Conditions = mergeCondition(run.Status.Conditions, metav1.Condition{
		Type:               agentRunCompletedCondition,
		Status:             metav1.ConditionFalse,
		Reason:             reason,
		Message:            message,
		ObservedGeneration: run.Generation,
		LastTransitionTime: now,
	})
}
