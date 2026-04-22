package controller

import (
	"context"
	"testing"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestBuildReferenceIndexListsNamespaceResources(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := apiv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme returned error: %v", err)
	}

	client := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(
			&apiv1alpha1.PromptTemplate{
				ObjectMeta: metav1.ObjectMeta{Name: "system", Namespace: "ehs"},
				Spec: apiv1alpha1.PromptTemplateSpec{
					Language: "zh-CN",
					Template: "You are a system prompt.",
				},
			},
			&apiv1alpha1.KnowledgeBase{ObjectMeta: metav1.ObjectMeta{Name: "regulations", Namespace: "ehs"}},
			&apiv1alpha1.ToolProvider{ObjectMeta: metav1.ObjectMeta{Name: "vision", Namespace: "ehs"}},
			&apiv1alpha1.Skill{ObjectMeta: metav1.ObjectMeta{Name: "risk-scoring", Namespace: "ehs"}},
			&apiv1alpha1.MCPServer{ObjectMeta: metav1.ObjectMeta{Name: "docs", Namespace: "ehs"}},
			&apiv1alpha1.AgentPolicy{ObjectMeta: metav1.ObjectMeta{Name: "policy", Namespace: "ehs"}},
			&apiv1alpha1.ToolProvider{ObjectMeta: metav1.ObjectMeta{Name: "other-namespace-tool", Namespace: "default"}},
		).
		Build()

	refs, err := BuildReferenceIndex(context.Background(), client, "ehs")
	if err != nil {
		t.Fatalf("BuildReferenceIndex returned error: %v", err)
	}

	assertContains(t, refs.Prompts, "system")
	if refs.PromptTemplates["system"].Template != "You are a system prompt." {
		t.Fatalf("expected prompt template spec to be indexed, got %#v", refs.PromptTemplates["system"])
	}
	assertContains(t, refs.KnowledgeBases, "regulations")
	assertContains(t, refs.Tools, "vision")
	assertContains(t, refs.Skills, "risk-scoring")
	assertContains(t, refs.MCPServers, "docs")
	assertContains(t, refs.Policies, "policy")
	assertNotContains(t, refs.Tools, "other-namespace-tool")
}

func TestSetAgentStatusSetsEndpointAndReadyCondition(t *testing.T) {
	agent := &apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "hazard-agent",
			Namespace:  "ehs",
			Generation: 7,
		},
	}

	artifact := apiv1alpha1.FreeformObject{
		"kind": {},
	}
	setAgentStatus(agent, "Published", "sha256:test", artifact, metav1.Condition{
		Type:               agentReadyCondition,
		Status:             metav1.ConditionTrue,
		Reason:             "CompilationSucceeded",
		Message:            "compiled",
		ObservedGeneration: agent.Generation,
	})

	if agent.Status.Phase != "Published" {
		t.Fatalf("expected phase Published, got %q", agent.Status.Phase)
	}
	if agent.Status.CompiledRevision != "sha256:test" {
		t.Fatalf("expected compiled revision, got %q", agent.Status.CompiledRevision)
	}
	if agent.Status.CompiledArtifact == nil {
		t.Fatal("expected compiled artifact to be set")
	}
	if agent.Status.Endpoint["invoke"] != "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/hazard-agent:invoke" {
		t.Fatalf("unexpected invoke endpoint: %q", agent.Status.Endpoint["invoke"])
	}
	if len(agent.Status.Conditions) != 1 || agent.Status.Conditions[0].Status != metav1.ConditionTrue {
		t.Fatalf("expected one true Ready condition, got %#v", agent.Status.Conditions)
	}
	if agent.Status.Conditions[0].LastTransitionTime.IsZero() {
		t.Fatal("expected Ready condition to have lastTransitionTime")
	}
}

func assertContains(t *testing.T, values map[string]struct{}, key string) {
	t.Helper()
	if _, ok := values[key]; !ok {
		t.Fatalf("expected map to contain %q", key)
	}
}

func assertNotContains(t *testing.T, values map[string]struct{}, key string) {
	t.Helper()
	if _, ok := values[key]; ok {
		t.Fatalf("expected map not to contain %q", key)
	}
}
