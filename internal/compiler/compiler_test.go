package compiler

import (
	"encoding/json"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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
	if jsonString(t, result.Artifact["kind"]) != "AgentCompiledArtifact" {
		t.Fatalf("expected compiled artifact kind, got %#v", result.Artifact["kind"])
	}
	if jsonString(t, result.Artifact["policyRef"]) != "ehs-default-safety-policy" {
		t.Fatalf("expected policy ref in artifact, got %#v", result.Artifact["policyRef"])
	}
	runtime := runtimeArtifact(t, result.Artifact["runtime"])
	if runtime.Engine != "eino" {
		t.Fatalf("expected default runtime engine, got %#v", runtime)
	}
	if runtime.RunnerClass != "adk" {
		t.Fatalf("expected default runner class, got %#v", runtime)
	}
}

func TestCompileAgentRevisionChangesWhenArtifactChanges(t *testing.T) {
	refs := ReferenceIndex{
		Prompts:        set("ehs-hazard-identification-system"),
		KnowledgeBases: set("ehs-regulations", "ehs-hazard-cases"),
		Tools:          set("vision-inspection-tool", "rectify-ticket-api"),
		MCPServers:     set("ehs-docs-mcp"),
		Policies:       set("ehs-default-safety-policy"),
	}
	first, err := CompileAgent(testAgent(), refs)
	if err != nil {
		t.Fatalf("CompileAgent returned error: %v", err)
	}
	agent := testAgent()
	agent.Spec.Runtime.RunnerClass = "custom"
	second, err := CompileAgent(agent, refs)
	if err != nil {
		t.Fatalf("CompileAgent returned error: %v", err)
	}
	if first.Revision == second.Revision {
		t.Fatalf("expected revision to change when compiled artifact changes: %q", first.Revision)
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

func jsonString(t *testing.T, value apiextensionsv1.JSON) string {
	t.Helper()
	var output string
	if err := json.Unmarshal(value.Raw, &output); err != nil {
		t.Fatalf("failed to decode JSON string: %v", err)
	}
	return output
}

func runtimeArtifact(t *testing.T, value apiextensionsv1.JSON) apiv1alpha1.AgentRuntimeSpec {
	t.Helper()
	var output apiv1alpha1.AgentRuntimeSpec
	if err := json.Unmarshal(value.Raw, &output); err != nil {
		t.Fatalf("failed to decode runtime artifact: %v", err)
	}
	return output
}
