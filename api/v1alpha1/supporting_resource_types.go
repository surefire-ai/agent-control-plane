package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

type PromptTemplateSpec struct {
	Version           string               `json:"version,omitempty"`
	Language          string               `json:"language,omitempty"`
	Type              string               `json:"type,omitempty"`
	Description       string               `json:"description,omitempty"`
	Template          string               `json:"template,omitempty"`
	Variables         []PromptVariableSpec `json:"variables,omitempty"`
	OutputConstraints FreeformObject       `json:"outputConstraints,omitempty"`
}

type PromptVariableSpec struct {
	Name     string `json:"name"`
	Required bool   `json:"required,omitempty"`
}

type KnowledgeBaseSpec struct {
	Description string         `json:"description,omitempty"`
	Embedding   FreeformObject `json:"embedding,omitempty"`
	Index       FreeformObject `json:"index,omitempty"`
	Ingestion   FreeformObject `json:"ingestion,omitempty"`
	Retrieval   FreeformObject `json:"retrieval,omitempty"`
	Sources     []NamedURI     `json:"sources,omitempty"`
	Access      FreeformObject `json:"access,omitempty"`
}

type NamedURI struct {
	Name string `json:"name"`
	URI  string `json:"uri"`
}

type ToolProviderSpec struct {
	Type        string         `json:"type,omitempty"`
	Description string         `json:"description,omitempty"`
	Schema      ToolSchemaSpec `json:"schema,omitempty"`
	Runtime     FreeformObject `json:"runtime,omitempty"`
	HTTP        FreeformObject `json:"http,omitempty"`
}

type ToolSchemaSpec struct {
	Input  JSONSchema `json:"input,omitempty"`
	Output JSONSchema `json:"output,omitempty"`
}

type MCPServerSpec struct {
	Transport      string         `json:"transport,omitempty"`
	URL            string         `json:"url,omitempty"`
	TimeoutSeconds int32          `json:"timeoutSeconds,omitempty"`
	Capabilities   []string       `json:"capabilities,omitempty"`
	Auth           FreeformObject `json:"auth,omitempty"`
	HealthCheck    FreeformObject `json:"healthCheck,omitempty"`
}

type AgentPolicySpec struct {
	HumanInTheLoop FreeformObject `json:"humanInTheLoop,omitempty"`
	Guardrails     FreeformObject `json:"guardrails,omitempty"`
	Budgets        FreeformObject `json:"budgets,omitempty"`
	AllowedModels  []string       `json:"allowedModels,omitempty"`
	AllowedTools   []string       `json:"allowedTools,omitempty"`
	Security       FreeformObject `json:"security,omitempty"`
	Resilience     FreeformObject `json:"resilience,omitempty"`
}

type AgentRunSpec struct {
	AgentRef  LocalObjectReference `json:"agentRef"`
	Input     FreeformObject       `json:"input,omitempty"`
	Execution FreeformObject       `json:"execution,omitempty"`
}

type AgentRunStatus struct {
	ConditionedStatus `json:",inline"`
	Phase             string         `json:"phase,omitempty"`
	StartedAt         *metav1.Time   `json:"startedAt,omitempty"`
	FinishedAt        *metav1.Time   `json:"finishedAt,omitempty"`
	Output            FreeformObject `json:"output,omitempty"`
	TraceRef          FreeformObject `json:"traceRef,omitempty"`
	Ticket            FreeformObject `json:"ticket,omitempty"`
	AgentRevision     string         `json:"agentRevision,omitempty"`
}

type AgentRunPhase string

const (
	AgentRunPhasePending   AgentRunPhase = "Pending"
	AgentRunPhaseRunning   AgentRunPhase = "Running"
	AgentRunPhaseSucceeded AgentRunPhase = "Succeeded"
	AgentRunPhaseFailed    AgentRunPhase = "Failed"
)

type AgentEvaluationSpec struct {
	AgentRef   LocalObjectReference `json:"agentRef"`
	DatasetRef map[string]string    `json:"datasetRef,omitempty"`
	Evaluators []map[string]string  `json:"evaluators,omitempty"`
	Thresholds map[string]float64   `json:"thresholds,omitempty"`
}

type ResourceStatus struct {
	ConditionedStatus `json:",inline"`
	Phase             string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type PromptTemplate struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PromptTemplateSpec `json:"spec,omitempty"`
	Status            ResourceStatus     `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type PromptTemplateList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []PromptTemplate `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type KnowledgeBase struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              KnowledgeBaseSpec `json:"spec,omitempty"`
	Status            ResourceStatus    `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type KnowledgeBaseList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []KnowledgeBase `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type ToolProvider struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              ToolProviderSpec `json:"spec,omitempty"`
	Status            ResourceStatus   `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type ToolProviderList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ToolProvider `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type MCPServer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              MCPServerSpec  `json:"spec,omitempty"`
	Status            ResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type MCPServerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MCPServer `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AgentPolicy struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AgentPolicySpec `json:"spec,omitempty"`
	Status            ResourceStatus  `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AgentPolicyList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentPolicy `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AgentRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AgentRunSpec   `json:"spec,omitempty"`
	Status            AgentRunStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AgentRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentRun `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type AgentEvaluation struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              AgentEvaluationSpec `json:"spec,omitempty"`
	Status            ResourceStatus      `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AgentEvaluationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentEvaluation `json:"items"`
}
