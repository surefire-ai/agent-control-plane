package compiler

import (
	"strings"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
)

func TestCompileAgentReturnsRevisionWhenReferencesExist(t *testing.T) {
	agent := testAgent()
	refs := ReferenceIndex{
		Prompts:        set("ehs-hazard-identification-system"),
		KnowledgeBases: set("ehs-regulations", "ehs-hazard-cases"),
		Tools:          set("vision-inspection-tool", "rectify-ticket-api"),
		MCPServers:     set("ehs-docs-mcp"),
		Policies:       set("ehs-default-safety-policy"),
	}

	result, err := CompileAgent(agent, refs)
	if err != nil {
		t.Fatalf("CompileAgent returned error: %v", err)
	}

	if !strings.HasPrefix(result.Revision, "sha256:") {
		t.Fatalf("expected sha256 revision, got %q", result.Revision)
	}
}

func TestCompileAgentReportsMissingReferences(t *testing.T) {
	_, err := CompileAgent(testAgent(), ReferenceIndex{})
	if err == nil {
		t.Fatal("expected missing reference error")
	}

	message := err.Error()
	for _, expected := range []string{
		"PromptTemplate/ehs-hazard-identification-system",
		"AgentPolicy/ehs-default-safety-policy",
		"KnowledgeBase/ehs-regulations",
		"ToolProvider/vision-inspection-tool",
		"MCPServer/ehs-docs-mcp",
	} {
		if !strings.Contains(message, expected) {
			t.Fatalf("expected error to contain %q, got %q", expected, message)
		}
	}
}

func testAgent() apiv1alpha1.Agent {
	return apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:       "ehs-hazard-identification-agent",
			Namespace:  "ehs",
			Generation: 3,
		},
		Spec: apiv1alpha1.AgentSpec{
			PromptRefs: apiv1alpha1.AgentPromptRefs{
				System: "ehs-hazard-identification-system",
			},
			KnowledgeRefs: []apiv1alpha1.KnowledgeBindingSpec{
				{Name: "regulations", Ref: "ehs-regulations"},
				{Name: "cases", Ref: "ehs-hazard-cases"},
			},
			ToolRefs:  []string{"vision-inspection-tool", "rectify-ticket-api"},
			MCPRefs:   []string{"ehs-docs-mcp"},
			PolicyRef: "ehs-default-safety-policy",
		},
	}
}

func set(values ...string) map[string]struct{} {
	result := make(map[string]struct{}, len(values))
	for _, value := range values {
		result[value] = struct{}{}
	}
	return result
}
