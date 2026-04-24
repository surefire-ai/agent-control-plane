package controller

import (
	"context"
	"fmt"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	"github.com/surefire-ai/agent-control-plane/internal/compiler"
	"github.com/surefire-ai/agent-control-plane/internal/providers"
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

	workspaceName, workspace, err := resolveWorkspaceScope(ctx, r.Client, req.Namespace, agent.Spec.WorkspaceRef)
	if err != nil {
		setAgentStatus(&agent, "NotReady", workspaceName, "", nil, metav1.Condition{
			Type:               agentReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "WorkspaceReferenceFailed",
			Message:            err.Error(),
			ObservedGeneration: agent.Generation,
		})
		if equality.Semantic.DeepEqual(previousStatus, &agent.Status) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, r.Status().Patch(ctx, &agent, client.MergeFrom(original))
	}
	effectiveAgent, err := agentWithWorkspaceDefaults(agent, workspace)
	if err != nil {
		setAgentStatus(&agent, "NotReady", workspaceName, "", nil, metav1.Condition{
			Type:               agentReadyCondition,
			Status:             metav1.ConditionFalse,
			Reason:             "WorkspacePolicyRejected",
			Message:            err.Error(),
			ObservedGeneration: agent.Generation,
		})
		if equality.Semantic.DeepEqual(previousStatus, &agent.Status) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, r.Status().Patch(ctx, &agent, client.MergeFrom(original))
	}

	result, err := compiler.CompileAgent(effectiveAgent, refs)
	if err != nil {
		setAgentStatus(&agent, "NotReady", workspaceName, "", nil, metav1.Condition{
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

	setAgentStatus(&agent, string(agent.Spec.Lifecycle.DesiredPhase), workspaceName, result.Revision, result.Artifact, metav1.Condition{
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
		Prompts:         map[string]struct{}{},
		PromptTemplates: map[string]apiv1alpha1.PromptTemplateSpec{},
		KnowledgeBases:  map[string]struct{}{},
		KnowledgeSpecs:  map[string]apiv1alpha1.KnowledgeBaseSpec{},
		Tools:           map[string]struct{}{},
		ToolSpecs:       map[string]apiv1alpha1.ToolProviderSpec{},
		Skills:          map[string]struct{}{},
		SkillSpecs:      map[string]apiv1alpha1.SkillSpec{},
		MCPServers:      map[string]struct{}{},
		Policies:        map[string]struct{}{},
	}

	var prompts apiv1alpha1.PromptTemplateList
	if err := reader.List(ctx, &prompts, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range prompts.Items {
		refs.Prompts[item.Name] = struct{}{}
		refs.PromptTemplates[item.Name] = item.Spec
	}

	var knowledgeBases apiv1alpha1.KnowledgeBaseList
	if err := reader.List(ctx, &knowledgeBases, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range knowledgeBases.Items {
		refs.KnowledgeBases[item.Name] = struct{}{}
		refs.KnowledgeSpecs[item.Name] = item.Spec
	}

	var tools apiv1alpha1.ToolProviderList
	if err := reader.List(ctx, &tools, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range tools.Items {
		refs.Tools[item.Name] = struct{}{}
		refs.ToolSpecs[item.Name] = item.Spec
	}

	var skills apiv1alpha1.SkillList
	if err := reader.List(ctx, &skills, client.InNamespace(namespace)); err != nil {
		return compiler.ReferenceIndex{}, err
	}
	for _, item := range skills.Items {
		refs.Skills[item.Name] = struct{}{}
		refs.SkillSpecs[item.Name] = item.Spec
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

func setAgentStatus(agent *apiv1alpha1.Agent, phase string, workspaceRef string, revision string, artifact apiv1alpha1.FreeformObject, condition metav1.Condition) {
	agent.Status.Phase = phase
	agent.Status.ObservedGeneration = agent.Generation
	agent.Status.WorkspaceRef = workspaceRef
	agent.Status.CompiledRevision = revision
	agent.Status.CompiledArtifact = artifact
	agent.Status.Endpoint = map[string]string{
		"invoke": "/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version +
			"/namespaces/" + agent.Namespace + "/agents/" + agent.Name + ":invoke",
	}
	agent.Status.Conditions = mergeCondition(agent.Status.Conditions, condition)
}

func resolveWorkspaceScope(ctx context.Context, reader client.Reader, namespace string, ref *apiv1alpha1.LocalObjectReference) (string, *apiv1alpha1.Workspace, error) {
	if ref == nil || ref.Name == "" {
		return "", nil, nil
	}
	var workspace apiv1alpha1.Workspace
	if err := reader.Get(ctx, client.ObjectKey{Namespace: namespace, Name: ref.Name}, &workspace); err != nil {
		if apierrors.IsNotFound(err) {
			return ref.Name, nil, fmt.Errorf("referenced Workspace %q not found", ref.Name)
		}
		return ref.Name, nil, err
	}
	if workspace.Status.Phase != "Ready" {
		return workspace.Name, &workspace, fmt.Errorf("referenced Workspace %q is not Ready", workspace.Name)
	}
	return workspace.Name, &workspace, nil
}

func agentWithWorkspaceDefaults(agent apiv1alpha1.Agent, workspace *apiv1alpha1.Workspace) (apiv1alpha1.Agent, error) {
	if workspace == nil {
		return agent, nil
	}
	effective := agent.DeepCopy()
	if effective.Spec.PolicyRef == "" && workspace.Spec.PolicyRef != "" {
		effective.Spec.PolicyRef = workspace.Spec.PolicyRef
	}
	if err := validateWorkspaceProviders(*effective, *workspace); err != nil {
		return apiv1alpha1.Agent{}, err
	}
	return *effective, nil
}

func validateWorkspaceProviders(agent apiv1alpha1.Agent, workspace apiv1alpha1.Workspace) error {
	allowed := map[string]struct{}{}
	for _, providerName := range workspace.Spec.ProviderPolicy.AllowedProviders {
		normalized := providers.Normalize(providerName)
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	if len(allowed) == 0 {
		return nil
	}
	for name, model := range agent.Spec.Models {
		providerName := providers.Normalize(model.Provider)
		if _, ok := allowed[providerName]; !ok {
			return fmt.Errorf("model %q uses provider %q outside Workspace %q allowedProviders", name, model.Provider, workspace.Name)
		}
	}
	return nil
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
