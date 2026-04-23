package contract

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type WorkerStatus string

const (
	WorkerStatusSucceeded WorkerStatus = "succeeded"
	WorkerStatusFailed    WorkerStatus = "failed"
)

type WorkerResult struct {
	Status           WorkerStatus           `json:"status"`
	Reason           string                 `json:"reason,omitempty"`
	Message          string                 `json:"message"`
	Config           interface{}            `json:"config,omitempty"`
	CompiledArtifact ArtifactSummary        `json:"compiledArtifact,omitempty"`
	Output           map[string]interface{} `json:"output,omitempty"`
	Artifacts        []WorkerArtifact       `json:"artifacts,omitempty"`
	Runtime          *WorkerRuntimeInfo     `json:"runtime,omitempty"`
	StartedAt        time.Time              `json:"startedAt,omitempty"`
}

type WorkerArtifact struct {
	Name   string                 `json:"name,omitempty"`
	Kind   string                 `json:"kind,omitempty"`
	Inline map[string]interface{} `json:"inline,omitempty"`
}

type WorkerRuntimeInfo struct {
	Engine      string                            `json:"engine,omitempty"`
	RunnerClass string                            `json:"runnerClass,omitempty"`
	Runner      string                            `json:"runner,omitempty"`
	Entrypoint  string                            `json:"entrypoint,omitempty"`
	Models      map[string]WorkerModelRuntime     `json:"models,omitempty"`
	Tools       map[string]WorkerToolRuntime      `json:"tools,omitempty"`
	Skills      map[string]WorkerSkillRuntime     `json:"skills,omitempty"`
	Knowledge   map[string]WorkerKnowledgeRuntime `json:"knowledge,omitempty"`
}

type WorkerModelRuntime struct {
	Provider            string `json:"provider,omitempty"`
	ProviderFamily      string `json:"providerFamily,omitempty"`
	ProviderDisplayName string `json:"providerDisplayName,omitempty"`
	SupportsJSONSchema  bool   `json:"supportsJsonSchema,omitempty"`
	Model               string `json:"model,omitempty"`
	BaseURL             string `json:"baseURL,omitempty"`
	APIKeyEnv           string `json:"apiKeyEnv,omitempty"`
	CredentialInjected  bool   `json:"credentialInjected,omitempty"`
}

type WorkerToolRuntime struct {
	Type               string   `json:"type,omitempty"`
	Description        string   `json:"description,omitempty"`
	Capabilities       []string `json:"capabilities,omitempty"`
	AuthTokenEnv       string   `json:"authTokenEnv,omitempty"`
	CredentialInjected bool     `json:"credentialInjected,omitempty"`
}

type WorkerSkillRuntime struct {
	Ref            string            `json:"ref,omitempty"`
	Description    string            `json:"description,omitempty"`
	SystemPrompt   string            `json:"systemPrompt,omitempty"`
	FunctionCount  int               `json:"functionCount,omitempty"`
	Functions      []string          `json:"functions,omitempty"`
	ToolRefs       []string          `json:"toolRefs,omitempty"`
	KnowledgeRefs  []string          `json:"knowledgeRefs,omitempty"`
	DeclaredByName map[string]string `json:"declaredByName,omitempty"`
}

type WorkerKnowledgeRuntime struct {
	Ref            string  `json:"ref,omitempty"`
	Description    string  `json:"description,omitempty"`
	SourceCount    int     `json:"sourceCount,omitempty"`
	RetrievalBound bool    `json:"retrievalBound,omitempty"`
	DefaultTopK    int64   `json:"defaultTopK,omitempty"`
	ScoreThreshold float64 `json:"scoreThreshold,omitempty"`
}

type ArtifactSummary struct {
	APIVersion    string `json:"apiVersion,omitempty"`
	Kind          string `json:"kind,omitempty"`
	RuntimeEngine string `json:"runtimeEngine,omitempty"`
	RunnerClass   string `json:"runnerClass,omitempty"`
	PolicyRef     string `json:"policyRef,omitempty"`
}

func ParseWorkerResult(raw string) (WorkerResult, error) {
	var result WorkerResult
	decoder := json.NewDecoder(strings.NewReader(raw))
	if err := decoder.Decode(&result); err != nil {
		return WorkerResult{}, fmt.Errorf("worker result must be valid JSON: %w", err)
	}
	if err := ensureSingleJSONDocument(decoder); err != nil {
		return WorkerResult{}, err
	}
	if result.Status == "" {
		return WorkerResult{}, fmt.Errorf("worker result status is required")
	}
	return result, nil
}

func WriteWorkerResult(writer io.Writer, result WorkerResult) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	return encoder.Encode(result)
}

func ensureSingleJSONDocument(decoder *json.Decoder) error {
	var trailing interface{}
	if err := decoder.Decode(&trailing); err == nil {
		return fmt.Errorf("worker result must contain a single JSON document")
	} else if err != io.EOF {
		return fmt.Errorf("worker result has invalid trailing data: %w", err)
	}
	return nil
}
