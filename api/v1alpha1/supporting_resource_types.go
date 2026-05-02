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

type DatasetSpec struct {
	Description string              `json:"description,omitempty"`
	Revision    string              `json:"revision,omitempty"`
	Samples     []DatasetSampleSpec `json:"samples,omitempty"`
}

type DatasetSampleSpec struct {
	Name     string            `json:"name"`
	Input    FreeformObject    `json:"input,omitempty"`
	Expected FreeformObject    `json:"expected,omitempty"`
	Labels   map[string]string `json:"labels,omitempty"`
}

type SkillSpec struct {
	Description   string                 `json:"description,omitempty"`
	PromptRefs    AgentPromptRefs        `json:"promptRefs,omitempty"`
	KnowledgeRefs []KnowledgeBindingSpec `json:"knowledgeRefs,omitempty"`
	ToolRefs      []string               `json:"toolRefs,omitempty"`
	Functions     []string               `json:"functions,omitempty"`
	Graph         SkillGraphSpec         `json:"graph,omitempty"`
}

type SkillGraphSpec struct {
	Nodes []AgentGraphNode `json:"nodes,omitempty"`
	Edges []AgentGraphEdge `json:"edges,omitempty"`
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

type TenantSpec struct {
	DisplayName string         `json:"displayName,omitempty"`
	Description string         `json:"description,omitempty"`
	Profile     FreeformObject `json:"profile,omitempty"`
	Governance  FreeformObject `json:"governance,omitempty"`
	Provider    FreeformObject `json:"provider,omitempty"`
}

type TenantStatus struct {
	ConditionedStatus  `json:",inline"`
	Phase              string `json:"phase,omitempty"`
	ObservedGeneration int64  `json:"observedGeneration,omitempty"`
	WorkspaceCount     int32  `json:"workspaceCount,omitempty"`
}

type WorkspaceSpec struct {
	TenantRef      LocalObjectReference        `json:"tenantRef"`
	DisplayName    string                      `json:"displayName,omitempty"`
	Description    string                      `json:"description,omitempty"`
	Namespace      string                      `json:"namespace,omitempty"`
	PolicyRef      string                      `json:"policyRef,omitempty"`
	ProviderPolicy WorkspaceProviderPolicySpec `json:"providerPolicy,omitempty"`
	Provider       FreeformObject              `json:"provider,omitempty"`
	Governance     FreeformObject              `json:"governance,omitempty"`
}

type WorkspaceProviderPolicySpec struct {
	DefaultProvider  string                         `json:"defaultProvider,omitempty"`
	AllowedProviders []string                       `json:"allowedProviders,omitempty"`
	Bindings         []WorkspaceProviderBindingSpec `json:"bindings,omitempty"`
}

type WorkspaceProviderBindingSpec struct {
	Provider      string              `json:"provider"`
	BaseURL       string              `json:"baseURL,omitempty"`
	CredentialRef *SecretKeyReference `json:"credentialRef,omitempty"`
}

type WorkspaceStatus struct {
	ConditionedStatus  `json:",inline"`
	Phase              string            `json:"phase,omitempty"`
	ObservedGeneration int64             `json:"observedGeneration,omitempty"`
	TenantRef          string            `json:"tenantRef,omitempty"`
	Namespace          string            `json:"namespace,omitempty"`
	Endpoint           map[string]string `json:"endpoint,omitempty"`
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
	AgentRef              LocalObjectReference  `json:"agentRef"`
	WorkspaceRef          *LocalObjectReference `json:"workspaceRef,omitempty"`
	Input                 FreeformObject        `json:"input,omitempty"`
	Execution             FreeformObject        `json:"execution,omitempty"`
	ActiveDeadlineSeconds *int64                `json:"activeDeadlineSeconds,omitempty"`
	MaxRetries            *int32                `json:"maxRetries,omitempty"`
	RetryBackoffSeconds   *int64                `json:"retryBackoffSeconds,omitempty"`
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
	WorkspaceRef      string         `json:"workspaceRef,omitempty"`
	RetryCount        int32          `json:"retryCount,omitempty"`
	LastFailureReason string         `json:"lastFailureReason,omitempty"`
	ArtifactRefs      []ArtifactRef  `json:"artifactRefs,omitempty"`
}

// ArtifactRef identifies an external resource (e.g. ConfigMap data key)
// that stores durable worker artifacts. Namespace and Name identify the
// ConfigMap; Key is the data key within it.
type ArtifactRef struct {
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Key       string `json:"key,omitempty"`
}

type AgentRunPhase string

const (
	AgentRunPhasePending   AgentRunPhase = "Pending"
	AgentRunPhaseRunning   AgentRunPhase = "Running"
	AgentRunPhaseSucceeded AgentRunPhase = "Succeeded"
	AgentRunPhaseFailed    AgentRunPhase = "Failed"
	AgentRunPhaseCanceled  AgentRunPhase = "Canceled"
	AgentRunPhaseRetrying  AgentRunPhase = "Retrying"
)

type AgentEvaluationSpec struct {
	AgentRef     LocalObjectReference       `json:"agentRef"`
	WorkspaceRef *LocalObjectReference      `json:"workspaceRef,omitempty"`
	Baseline     *EvaluationBaselineSpec    `json:"baseline,omitempty"`
	DatasetRef   EvaluationDatasetReference `json:"datasetRef"`
	Evaluators   []EvaluationEvaluatorSpec  `json:"evaluators,omitempty"`
	Thresholds   []EvaluationThresholdSpec  `json:"thresholds,omitempty"`
	Gate         EvaluationGateSpec         `json:"gate,omitempty"`
	Reporting    EvaluationReportingSpec    `json:"reporting,omitempty"`
	Runtime      FreeformObject             `json:"runtime,omitempty"`
}

type EvaluationBaselineSpec struct {
	AgentRef  *LocalObjectReference `json:"agentRef,omitempty"`
	Revision  string                `json:"revision,omitempty"`
	Reference string                `json:"reference,omitempty"`
}

type EvaluationDatasetReference struct {
	Kind      string `json:"kind,omitempty"`
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
	Revision  string `json:"revision,omitempty"`
}

type EvaluationEvaluatorSpec struct {
	Name   string         `json:"name"`
	Type   string         `json:"type"`
	Metric string         `json:"metric,omitempty"`
	Weight float64        `json:"weight,omitempty"`
	Config FreeformObject `json:"config,omitempty"`
}

type EvaluationThresholdSpec struct {
	Metric   string  `json:"metric"`
	Operator string  `json:"operator,omitempty"`
	Target   float64 `json:"target"`
	Blocking bool    `json:"blocking,omitempty"`
}

type EvaluationGateSpec struct {
	Mode        string   `json:"mode,omitempty"`
	Required    []string `json:"required,omitempty"`
	BlockOnFail bool     `json:"blockOnFail,omitempty"`
}

type EvaluationReportingSpec struct {
	Formats []string          `json:"formats,omitempty"`
	Sinks   []string          `json:"sinks,omitempty"`
	Labels  map[string]string `json:"labels,omitempty"`
}

type AgentEvaluationStatus struct {
	ConditionedStatus  `json:",inline"`
	Phase              string                      `json:"phase,omitempty"`
	ObservedGeneration int64                       `json:"observedGeneration,omitempty"`
	WorkspaceRef       string                      `json:"workspaceRef,omitempty"`
	LatestRunRef       map[string]string           `json:"latestRunRef,omitempty"`
	Summary            EvaluationSummaryStatus     `json:"summary,omitempty"`
	Comparison         *EvaluationComparisonStatus `json:"comparison,omitempty"`
	Results            []EvaluationMetricStatus    `json:"results,omitempty"`
	ReportRef          FreeformObject              `json:"reportRef,omitempty"`
}

type EvaluationSummaryStatus struct {
	DatasetRevision  string  `json:"datasetRevision,omitempty"`
	BaselineRevision string  `json:"baselineRevision,omitempty"`
	SamplesTotal     int32   `json:"samplesTotal,omitempty"`
	SamplesEvaluated int32   `json:"samplesEvaluated,omitempty"`
	Score            float64 `json:"score,omitempty"`
	GatePassed       bool    `json:"gatePassed,omitempty"`
}

type EvaluationMetricStatus struct {
	Name      string  `json:"name,omitempty"`
	Metric    string  `json:"metric,omitempty"`
	Score     float64 `json:"score,omitempty"`
	Threshold float64 `json:"threshold,omitempty"`
	Passed    bool    `json:"passed,omitempty"`
	Reason    string  `json:"reason,omitempty"`
}

type EvaluationComparisonStatus struct {
	BaselineAgentRef   string  `json:"baselineAgentRef,omitempty"`
	CurrentScore       float64 `json:"currentScore,omitempty"`
	BaselineScore      float64 `json:"baselineScore,omitempty"`
	ScoreDelta         float64 `json:"scoreDelta,omitempty"`
	CurrentGatePassed  bool    `json:"currentGatePassed,omitempty"`
	BaselineGatePassed bool    `json:"baselineGatePassed,omitempty"`
}

type ResourceStatus struct {
	ConditionedStatus `json:",inline"`
	Phase             string `json:"phase,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Tenant struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              TenantSpec   `json:"spec,omitempty"`
	Status            TenantStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type TenantList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Tenant `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Workspace struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              WorkspaceSpec   `json:"spec,omitempty"`
	Status            WorkspaceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type WorkspaceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Workspace `json:"items"`
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
type Dataset struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              DatasetSpec    `json:"spec,omitempty"`
	Status            ResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type DatasetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Dataset `json:"items"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
type Skill struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              SkillSpec      `json:"spec,omitempty"`
	Status            ResourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type SkillList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Skill `json:"items"`
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
	Spec              AgentEvaluationSpec   `json:"spec,omitempty"`
	Status            AgentEvaluationStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true
type AgentEvaluationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []AgentEvaluation `json:"items"`
}
