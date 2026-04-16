package controller

import (
	"context"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
	"github.com/windosx/agent-control-plane/internal/compiler"
	"k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const agentReadyCondition = "Ready"

type AgentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *AgentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var agent apiv1alpha1.Agent
	if err := r.Get(ctx, req.NamespacedName, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	refs, err := BuildReferenceIndex(ctx, r.Client, req.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	original := agent.DeepCopy()
	previousStatus := agent.Status.DeepCopy()

	result, err := compiler.CompileAgent(agent, refs)
	if err != nil {
		setAgentStatus(&agent, "NotReady", "", metav1.Condition{
			Type:               agentReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "CompilationFailed",
			Message:            err.Error(),
			ObservedGeneration: agent.Generation,
		})
		if equality.Semantic.DeepEqual(previousStatus, &agent.Status) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, r.Status().Patch(ctx, &agent, client.MergeFrom(original))
	}

	setAgentStatus(&agent, string(agent.Spec.Lifecycle.DesiredPhase), result.Revision, metav1.Condition{
		Type:               agentReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "CompilationSucceeded",
		Message:            "agent graph compiled and published",
		ObservedGeneration: agent.Generation,
	})

	if agent.Status.Phase == "" {
		agent.Status.Phase = string(apiv1alpha1.AgentPhaseDraft)
	}

	if equality.Semantic.DeepEqual(previousStatus, &agent.Status) {
		return ctrl.Result{}, nil
	}

	return ctrl.Result{}, r.Status().Patch(ctx, &agent, client.MergeFrom(original))
}

func BuildReferenceIndex(ctx context.Context, reader client.Reader, namespace string) (compiler.ReferenceIndex, error) {
	refs := compiler.ReferenceIndex{
		Prompts:        map[string]struct{}{},
		KnowledgeBases: map[string]struct{}{},
		Tools:          map[string]struct{}{},
		MCPServers:     map[string]struct{}{},
		Policies:       map[string]struct{}{},
	}

	var prompts apiv1alpha1.PromptTemplateList
	if err := reader.List(ctx, &prompts, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range prompts.Items {
		refs.Prompts[item.Name] = struct{}{}
	}

	var knowledgeBases apiv1alpha1.KnowledgeBaseList
	if err := reader.List(ctx, &knowledgeBases, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range knowledgeBases.Items {
		refs.KnowledgeBases[item.Name] = struct{}{}
	}

	var tools apiv1alpha1.ToolProviderList
	if err := reader.List(ctx, &tools, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range tools.Items {
		refs.Tools[item.Name] = struct{}{}
	}

	var mcpServers apiv1alpha1.MCPServerList
	if err := reader.List(ctx, &mcpServers, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range mcpServers.Items {
		refs.MCPServers[item.Name] = struct{}{}
	}

	var policies apiv1alpha1.AgentPolicyList
	if err := reader.List(ctx, &policies, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range policies.Items {
		refs.Policies[item.Name] = struct{}{}
	}

	return refs, nil
}

func (r *AgentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&apiv1alpha1.Agent{}).
		Complete(r)
}

func setAgentStatus(agent *apiv1alpha1.Agent, phase string, revision string, condition metav1.Condition) {
	agent.Status.Phase = phase
	agent.Status.ObservedGeneration = agent.Generation
	agent.Status.CompiledRevision = revision
	agent.Status.Endpoint = map[string]string{
		"invoke": "/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version +
			"/namespaces/" + agent.Namespace + "/agents/" + agent.Name + ":invoke",
	}
	agent.Status.Conditions = mergeCondition(agent.Status.Conditions, condition)
}

func mergeCondition(existing []metav1.Condition, next metav1.Condition) []metav1.Condition {
	result := make([]metav1.Condition, 0, len(existing)+1)
	for _, condition := range existing {
		if condition.Type == next.Type {
			if condition.Status == next.Status &&
				condition.Reason == next.Reason &&
				condition.Message == next.Message &&
				condition.ObservedGeneration == next.ObservedGeneration {
				next.LastTransitionTime = condition.LastTransitionTime
			}
			continue
		}
		result = append(result, condition)
	}
	if next.LastTransitionTime.IsZero() {
		next.LastTransitionTime = metav1.Now()
	}
	result = append(result, next)
	return result
}
