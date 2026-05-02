package contract

import (
	"encoding/json"
	"fmt"
)

const (
	CompiledArtifactKind     = "AgentCompiledArtifact"
	CompiledArtifactSchemaV1 = "v1"
)

type CompiledArtifact struct {
	APIVersion    string                 `json:"apiVersion,omitempty"`
	Kind          string                 `json:"kind,omitempty"`
	SchemaVersion string                 `json:"schemaVersion,omitempty"`
	Agent         ArtifactAgent          `json:"agent,omitempty"`
	Runtime       ArtifactRuntime        `json:"runtime,omitempty"`
	Pattern       ArtifactPattern        `json:"pattern,omitempty"`
	Runner        ArtifactRunner         `json:"runner,omitempty"`
	Models        map[string]ModelConfig `json:"models,omitempty"`
	PolicyRef     string                 `json:"policyRef,omitempty"`
	Raw           map[string]interface{} `json:"-"`
}

type ArtifactAgent struct {
	Name       string `json:"name,omitempty"`
	Namespace  string `json:"namespace,omitempty"`
	Generation int64  `json:"generation,omitempty"`
}

type ArtifactRuntime struct {
	Engine      string                 `json:"engine,omitempty"`
	RunnerClass string                 `json:"runnerClass,omitempty"`
	Mode        string                 `json:"mode,omitempty"`
	Entrypoint  string                 `json:"entrypoint,omitempty"`
	Extra       map[string]interface{} `json:"-"`
}

type ArtifactRunner struct {
	Kind       string                   `json:"kind,omitempty"`
	Entrypoint string                   `json:"entrypoint,omitempty"`
	Pattern    map[string]interface{}   `json:"pattern,omitempty"`
	Graph      map[string]interface{}   `json:"graph,omitempty"`
	Prompts    map[string]PromptSpec    `json:"prompts,omitempty"`
	Models     map[string]ModelConfig   `json:"models,omitempty"`
	Providers  map[string]ProviderSpec  `json:"providers,omitempty"`
	Tools      map[string]ToolSpec      `json:"tools,omitempty"`
	Skills     map[string]SkillSpec     `json:"skills,omitempty"`
	SubAgents  map[string]interface{}   `json:"subAgents,omitempty"`
	Knowledge  map[string]KnowledgeSpec `json:"knowledge,omitempty"`
	Output     map[string]interface{}   `json:"output,omitempty"`
	Extra      map[string]interface{}   `json:"-"`
}

type ProviderSpec struct {
	Name                string `json:"name,omitempty"`
	DisplayName         string `json:"displayName,omitempty"`
	Family              string `json:"family,omitempty"`
	Domestic            bool   `json:"domestic,omitempty"`
	DefaultBaseURL      string `json:"defaultBaseURL,omitempty"`
	SupportsJSONSchema  bool   `json:"supportsJsonSchema,omitempty"`
	SupportsToolCalling bool   `json:"supportsToolCalling,omitempty"`
}

type ArtifactPattern struct {
	Type          string                 `json:"type,omitempty"`
	Version       string                 `json:"version,omitempty"`
	ModelRef      string                 `json:"modelRef,omitempty"`
	ToolRefs      []string               `json:"toolRefs,omitempty"`
	KnowledgeRefs []string               `json:"knowledgeRefs,omitempty"`
	MaxIterations int32                  `json:"maxIterations,omitempty"`
	StopWhen      string                 `json:"stopWhen,omitempty"`
	Routes        []PatternRouteConfig   `json:"routes,omitempty"`
	Expansion     map[string]interface{} `json:"expansion,omitempty"`
}

// PatternRouteConfig defines a route in the router pattern.
type PatternRouteConfig struct {
	Label    string `json:"label,omitempty"`
	AgentRef string `json:"agentRef,omitempty"`
	ModelRef string `json:"modelRef,omitempty"`
	Default  bool   `json:"default,omitempty"`
}

type SkillSpec struct {
	Name          string                      `json:"name,omitempty"`
	Ref           string                      `json:"ref,omitempty"`
	Description   string                      `json:"description,omitempty"`
	PromptRefs    map[string]string           `json:"promptRefs,omitempty"`
	KnowledgeRefs []CompiledSkillKnowledgeRef `json:"knowledgeRefs,omitempty"`
	ToolRefs      []string                    `json:"toolRefs,omitempty"`
	Functions     []string                    `json:"functions,omitempty"`
	Graph         map[string]interface{}      `json:"graph,omitempty"`
}

type CompiledSkillKnowledgeRef struct {
	Name string `json:"name,omitempty"`
	Ref  string `json:"ref,omitempty"`
}

type ToolSpec struct {
	Name        string                 `json:"name,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
	Runtime     map[string]interface{} `json:"runtime,omitempty"`
	HTTP        map[string]interface{} `json:"http,omitempty"`
}

type KnowledgeSpec struct {
	Name        string                   `json:"name,omitempty"`
	Ref         string                   `json:"ref,omitempty"`
	Description string                   `json:"description,omitempty"`
	Sources     []map[string]interface{} `json:"sources,omitempty"`
	Binding     map[string]interface{}   `json:"binding,omitempty"`
	Access      map[string]interface{}   `json:"access,omitempty"`
	Index       map[string]interface{}   `json:"index,omitempty"`
	Retrieval   map[string]interface{}   `json:"retrieval,omitempty"`
	Embedding   map[string]interface{}   `json:"embedding,omitempty"`
}

type PromptSpec struct {
	Name              string                 `json:"name,omitempty"`
	Language          string                 `json:"language,omitempty"`
	Template          string                 `json:"template,omitempty"`
	Variables         []PromptVariableSpec   `json:"variables,omitempty"`
	OutputConstraints map[string]interface{} `json:"outputConstraints,omitempty"`
	Extra             map[string]interface{} `json:"-"`
}

type PromptVariableSpec struct {
	Name     string `json:"name,omitempty"`
	Required bool   `json:"required,omitempty"`
}

type ModelConfig struct {
	Provider       string                 `json:"provider,omitempty"`
	Model          string                 `json:"model,omitempty"`
	BaseURL        string                 `json:"baseURL,omitempty"`
	CredentialRef  *SecretKeyReference    `json:"credentialRef,omitempty"`
	Temperature    float64                `json:"temperature,omitempty"`
	MaxTokens      int32                  `json:"maxTokens,omitempty"`
	TimeoutSeconds int32                  `json:"timeoutSeconds,omitempty"`
	Extra          map[string]interface{} `json:"-"`
}

type SecretKeyReference struct {
	Name string `json:"name,omitempty"`
	Key  string `json:"key,omitempty"`
}

func ParseCompiledArtifact(raw string) (CompiledArtifact, error) {
	var artifact CompiledArtifact
	if err := json.Unmarshal([]byte(raw), &artifact); err != nil {
		return CompiledArtifact{}, fmt.Errorf("AGENT_COMPILED_ARTIFACT must be valid JSON: %w", err)
	}
	if artifact.Kind == "" {
		return CompiledArtifact{}, fmt.Errorf("AGENT_COMPILED_ARTIFACT kind is required")
	}

	if err := json.Unmarshal([]byte(raw), &artifact.Raw); err != nil {
		return CompiledArtifact{}, fmt.Errorf("AGENT_COMPILED_ARTIFACT must be a JSON object: %w", err)
	}
	artifact.Runtime.Extra = extraObject(artifact.Raw, "runtime", "engine", "runnerClass", "mode", "entrypoint")
	artifact.Runner.Extra = extraObject(artifact.Raw, "runner", "kind", "entrypoint", "pattern", "graph", "prompts", "models", "providers", "tools", "skills", "subAgents", "knowledge", "output")
	return artifact, nil
}

func (a CompiledArtifact) RuntimeIdentity() RuntimeIdentity {
	return RuntimeIdentityFromSpec(RuntimeSpec{
		Engine:      a.Runtime.Engine,
		RunnerClass: a.Runtime.RunnerClass,
	})
}

func (a CompiledArtifact) Summary() ArtifactSummary {
	identity := a.RuntimeIdentity()
	return ArtifactSummary{
		APIVersion:    a.APIVersion,
		Kind:          a.Kind,
		RuntimeEngine: identity.Engine,
		RunnerClass:   identity.RunnerClass,
		PolicyRef:     a.PolicyRef,
	}
}

func extraObject(raw map[string]interface{}, key string, knownKeys ...string) map[string]interface{} {
	rawValue, ok := raw[key]
	if !ok {
		return nil
	}
	values, ok := rawValue.(map[string]interface{})
	if !ok {
		return nil
	}

	known := make(map[string]struct{}, len(knownKeys))
	for _, key := range knownKeys {
		known[key] = struct{}{}
	}

	extra := make(map[string]interface{})
	for key, value := range values {
		if _, ok := known[key]; ok {
			continue
		}
		extra[key] = value
	}
	if len(extra) == 0 {
		return nil
	}
	return extra
}
