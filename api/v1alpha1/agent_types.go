package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type AgentPhase string

const (
	AgentPhaseDraft     AgentPhase = "Draft"
	AgentPhasePublished AgentPhase = "Published"
)

type AgentSpec struct {
	Lifecycle     AgentLifecycleSpec     `json:"lifecycle,omitempty"`
	Runtime       AgentRuntimeSpec       `json:"runtime,omitempty"`
	Models        map[string]ModelSpec   `json:"models,omitempty"`
	Identity      AgentIdentitySpec      `json:"identity,omitempty"`
	PromptRefs    AgentPromptRefs        `json:"promptRefs,omitempty"`
	KnowledgeRefs []KnowledgeBindingSpec `json:"knowledgeRefs,omitempty"`
	ToolRefs      []string               `json:"toolRefs,omitempty"`
	MCPRefs       []string               `json:"mcpRefs,omitempty"`
	PolicyRef     string                 `json:"policyRef,omitempty"`
	Interfaces    AgentInterfaceSpec     `json:"interfaces,omitempty"`
	Memory        AgentMemorySpec        `json:"memory,omitempty"`
	Graph         AgentGraphSpec         `json:"graph,omitempty"`
	Observability AgentObservabilitySpec `json:"observability,omitempty"`
}

type AgentLifecycleSpec struct {
	DesiredPhase         AgentPhase `json:"desiredPhase,omitempty"`
	RevisionHistoryLimit int32      `json:"revisionHistoryLimit,omitempty"`
}

type AgentRuntimeSpec struct {
	Engine       string         `json:"engine,omitempty"`
	Mode         string         `json:"mode,omitempty"`
	Entrypoint   string         `json:"entrypoint,omitempty"`
	Checkpointer FreeformObject `json:"checkpointer,omitempty"`
	Thread       FreeformObject `json:"thread,omitempty"`
}

type ModelSpec struct {
	Provider       string  `json:"provider,omitempty"`
	Model          string  `json:"model,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
	MaxTokens      int32   `json:"maxTokens,omitempty"`
	TimeoutSeconds int32   `json:"timeoutSeconds,omitempty"`
}

type AgentIdentitySpec struct {
	DisplayName string `json:"displayName,omitempty"`
	Role        string `json:"role,omitempty"`
	Description string `json:"description,omitempty"`
}

type AgentPromptRefs struct {
	System string `json:"system,omitempty"`
}

type KnowledgeBindingSpec struct {
	Name      string         `json:"name"`
	Ref       string         `json:"ref"`
	Retrieval FreeformObject `json:"retrieval,omitempty"`
}

type AgentInterfaceSpec struct {
	Input  SchemaEnvelope `json:"input,omitempty"`
	Output SchemaEnvelope `json:"output,omitempty"`
}

type SchemaEnvelope struct {
	Schema JSONSchema `json:"schema,omitempty"`
}

type AgentMemorySpec struct {
	ShortTerm     FreeformObject `json:"shortTerm,omitempty"`
	LongTerm      FreeformObject `json:"longTerm,omitempty"`
	Summarization FreeformObject `json:"summarization,omitempty"`
}

type AgentGraphSpec struct {
	StateSchema JSONSchema       `json:"stateSchema,omitempty"`
	Nodes       []AgentGraphNode `json:"nodes,omitempty"`
	Edges       []AgentGraphEdge `json:"edges,omitempty"`
}

type AgentGraphNode struct {
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	ModelRef       string `json:"modelRef,omitempty"`
	ToolRef        string `json:"toolRef,omitempty"`
	KnowledgeRef   string `json:"knowledgeRef,omitempty"`
	Implementation string `json:"implementation,omitempty"`
}

type AgentGraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	When string `json:"when,omitempty"`
}

type AgentObservabilitySpec struct {
	Tracing FreeformObject `json:"tracing,omitempty"`
	Logging FreeformObject `json:"logging,omitempty"`
	Metrics FreeformObject `json:"metrics,omitempty"`
}

type AgentStatus struct {
	ConditionedStatus  `json:",inline"`
	Phase              string            `json:"phase,omitempty"`
	ObservedGeneration int64             `json:"observedGeneration,omitempty"`
	CompiledRevision   string            `json:"compiledRevision,omitempty"`
	CompiledArtifact   FreeformObject    `json:"compiledArtifact,omitempty"`
	Endpoint           map[string]string `json:"endpoint,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Agent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   AgentSpec   `json:"spec,omitempty"`
	Status AgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Agent `json:"items"`
}
