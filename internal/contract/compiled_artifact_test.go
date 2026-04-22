package contract

import "testing"

func TestParseCompiledArtifactSupportsPhase1Shape(t *testing.T) {
	artifact, err := ParseCompiledArtifact(`{
		"apiVersion":"windosx.com/v1alpha1",
		"kind":"AgentCompiledArtifact",
		"runtime":{"engine":"eino","runnerClass":"adk","mode":"stateful","checkpointer":{"provider":"postgres"}},
		"policyRef":"ehs-policy"
	}`)
	if err != nil {
		t.Fatalf("ParseCompiledArtifact returned error: %v", err)
	}

	if artifact.Kind != CompiledArtifactKind {
		t.Fatalf("unexpected kind: %q", artifact.Kind)
	}
	if artifact.Runtime.Engine != RuntimeEngineEino {
		t.Fatalf("unexpected runtime: %#v", artifact.Runtime)
	}
	if artifact.RuntimeIdentity().RunnerClass != RunnerClassADK {
		t.Fatalf("unexpected identity: %#v", artifact.RuntimeIdentity())
	}
	if artifact.Summary().PolicyRef != "ehs-policy" {
		t.Fatalf("unexpected summary: %#v", artifact.Summary())
	}
	if _, ok := artifact.Runtime.Extra["checkpointer"]; !ok {
		t.Fatalf("expected runtime extra fields to be preserved: %#v", artifact.Runtime.Extra)
	}
}

func TestParseCompiledArtifactSupportsRunnerShape(t *testing.T) {
	artifact, err := ParseCompiledArtifact(`{
		"apiVersion":"windosx.com/v1alpha1",
		"kind":"AgentCompiledArtifact",
		"schemaVersion":"v1",
		"agent":{"name":"hazard-agent","namespace":"ehs","generation":7},
		"runtime":{"engine":"eino","runnerClass":"adk","entrypoint":"ehs.hazard_identification"},
		"runner":{
			"kind":"EinoADKRunner",
			"entrypoint":"ehs.hazard_identification",
			"prompts":{"system":{"name":"system","language":"zh-CN","template":"hello","variables":[{"name":"risk_matrix_version","required":true}],"outputConstraints":{"format":"json_schema"}}},
			"models":{"planner":{"provider":"openai","model":"gpt-4.1","baseURL":"https://api.openai.com/v1","credentialRef":{"name":"openai-credentials","key":"apiKey"},"temperature":0.1,"maxTokens":4000,"timeoutSeconds":60}},
			"tools":{"vision-inspection-tool":{"name":"vision-inspection-tool","type":"multimodal","description":"图片巡检工具","runtime":{"provider":"internal-runtime"}}},
			"knowledge":{"regulations":{"name":"regulations","ref":"ehs-regulations","description":"法规库","sources":[{"name":"source-a","uri":"s3://bucket/a"}],"binding":{"retrieval":{"topK":5}},"retrieval":{"defaultTopK":5,"defaultScoreThreshold":0.72}}},
			"output":{"schema":{"type":"object"}}
		},
		"policyRef":"ehs-policy"
	}`)
	if err != nil {
		t.Fatalf("ParseCompiledArtifact returned error: %v", err)
	}

	if artifact.SchemaVersion != CompiledArtifactSchemaV1 {
		t.Fatalf("unexpected schema version: %q", artifact.SchemaVersion)
	}
	if artifact.Agent.Generation != 7 {
		t.Fatalf("unexpected agent: %#v", artifact.Agent)
	}
	if artifact.Runner.Kind != "EinoADKRunner" {
		t.Fatalf("unexpected runner: %#v", artifact.Runner)
	}
	if artifact.Runner.Prompts["system"].Language != "zh-CN" {
		t.Fatalf("unexpected prompt: %#v", artifact.Runner.Prompts)
	}
	if len(artifact.Runner.Prompts["system"].Variables) != 1 || artifact.Runner.Prompts["system"].Variables[0].Name != "risk_matrix_version" {
		t.Fatalf("unexpected prompt variables: %#v", artifact.Runner.Prompts)
	}
	if artifact.Runner.Prompts["system"].OutputConstraints["format"] != "json_schema" {
		t.Fatalf("unexpected prompt output constraints: %#v", artifact.Runner.Prompts)
	}
	if artifact.Runner.Models["planner"].MaxTokens != 4000 {
		t.Fatalf("unexpected model: %#v", artifact.Runner.Models)
	}
	if artifact.Runner.Models["planner"].BaseURL != "https://api.openai.com/v1" {
		t.Fatalf("unexpected model baseURL: %#v", artifact.Runner.Models)
	}
	if artifact.Runner.Models["planner"].CredentialRef == nil || artifact.Runner.Models["planner"].CredentialRef.Name != "openai-credentials" {
		t.Fatalf("unexpected model credentialRef: %#v", artifact.Runner.Models)
	}
	if artifact.Runner.Tools["vision-inspection-tool"].Type != "multimodal" {
		t.Fatalf("unexpected tools: %#v", artifact.Runner.Tools)
	}
	if artifact.Runner.Knowledge["regulations"].Ref != "ehs-regulations" || len(artifact.Runner.Knowledge["regulations"].Sources) != 1 {
		t.Fatalf("unexpected knowledge: %#v", artifact.Runner.Knowledge)
	}
}

func TestParseCompiledArtifactRequiresKind(t *testing.T) {
	_, err := ParseCompiledArtifact(`{"runtime":{"engine":"eino"}}`)
	if err == nil {
		t.Fatal("expected missing kind error")
	}
}

func TestParseCompiledArtifactRejectsInvalidJSON(t *testing.T) {
	_, err := ParseCompiledArtifact(`{`)
	if err == nil {
		t.Fatal("expected invalid JSON error")
	}
}
