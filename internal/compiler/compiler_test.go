package compiler

import (
	"encoding/json"
	"strings"
	"testing"

	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	"github.com/surefire-ai/agent-control-plane/internal/contract"
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
	if jsonString(t, result.Artifact["schemaVersion"]) != contract.CompiledArtifactSchemaV1 {
		t.Fatalf("expected schema version in artifact, got %#v", result.Artifact["schemaVersion"])
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
	runner := runnerArtifact(t, result.Artifact["runner"])
	if runner.Kind != "EinoADKRunner" {
		t.Fatalf("expected Eino runner artifact, got %#v", runner)
	}
	if runner.Entrypoint != "ehs.hazard_identification" {
		t.Fatalf("expected runner entrypoint, got %#v", runner)
	}
	if runner.Prompts["system"].Name != "ehs-hazard-identification-system" {
		t.Fatalf("expected system prompt in runner artifact, got %#v", runner.Prompts)
	}
	if runner.Models["planner"].Provider != "openai" {
		t.Fatalf("expected planner model in runner artifact, got %#v", runner.Models)
	}
	if runner.Models["planner"].BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("expected planner base URL in runner artifact, got %#v", runner.Models)
	}
	if runner.Models["planner"].CredentialRef == nil || runner.Models["planner"].CredentialRef.Name != "openai-credentials" {
		t.Fatalf("expected planner credential ref in runner artifact, got %#v", runner.Models)
	}
	if runner.Output == nil {
		t.Fatalf("expected output schema in runner artifact, got %#v", runner)
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

func TestCompileAgentArtifactCanBeDecodedByContract(t *testing.T) {
	result, err := CompileAgent(testAgent(), ReferenceIndex{
		Prompts:        set("ehs-hazard-identification-system"),
		KnowledgeBases: set("ehs-regulations", "ehs-hazard-cases"),
		Tools:          set("vision-inspection-tool", "rectify-ticket-api"),
		MCPServers:     set("ehs-docs-mcp"),
		Policies:       set("ehs-default-safety-policy"),
	})
	if err != nil {
		t.Fatalf("CompileAgent returned error: %v", err)
	}

	raw, err := json.Marshal(result.Artifact)
	if err != nil {
		t.Fatalf("failed to marshal artifact: %v", err)
	}
	artifact, err := contract.ParseCompiledArtifact(string(raw))
	if err != nil {
		t.Fatalf("compiled artifact did not decode through contract: %v", err)
	}
	if artifact.Runner.Kind != "EinoADKRunner" {
		t.Fatalf("unexpected runner: %#v", artifact.Runner)
	}
	if artifact.RuntimeIdentity().RunnerClass != contract.RunnerClassADK {
		t.Fatalf("unexpected runtime identity: %#v", artifact.RuntimeIdentity())
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
			Runtime: apiv1alpha1.AgentRuntimeSpec{
				Entrypoint: "ehs.hazard_identification",
			},
			Models: map[string]apiv1alpha1.ModelSpec{
				"planner": {
					Provider:       "openai",
					Model:          "gpt-4.1",
					BaseURL:        "https://api.openai.com/v1",
					CredentialRef:  &apiv1alpha1.SecretKeyReference{Name: "openai-credentials", Key: "apiKey"},
					Temperature:    0.1,
					MaxTokens:      4000,
					TimeoutSeconds: 60,
				},
			},
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
			Interfaces: apiv1alpha1.AgentInterfaceSpec{
				Output: apiv1alpha1.SchemaEnvelope{
					Schema: apiv1alpha1.JSONSchema{Raw: []byte(`{"type":"object"}`)},
				},
			},
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

func runnerArtifact(t *testing.T, value apiextensionsv1.JSON) contract.ArtifactRunner {
	t.Helper()
	var output contract.ArtifactRunner
	if err := json.Unmarshal(value.Raw, &output); err != nil {
		t.Fatalf("failed to decode runner artifact: %v", err)
	}
	return output
}
