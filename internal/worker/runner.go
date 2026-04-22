package worker

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type Runner interface {
	Run(ctx context.Context, request RunRequest) (contract.WorkerResult, error)
}

type RunRequest struct {
	Config          Config
	Artifact        contract.CompiledArtifact
	RuntimeIdentity contract.RuntimeIdentity
}

type EinoADKPlaceholderRunner struct {
	Invoker     OpenAICompatibleInvoker
	ToolInvoker HTTPToolInvoker
}

type FailureReasonError struct {
	Reason  string
	Message string
}

func (e FailureReasonError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Reason != "" {
		return e.Reason
	}
	return "worker failed"
}

func (r EinoADKPlaceholderRunner) Run(ctx context.Context, request RunRequest) (contract.WorkerResult, error) {
	select {
	case <-ctx.Done():
		return contract.WorkerResult{}, ctx.Err()
	default:
	}

	runtimeInfo, artifacts, err := runtimeInfoForArtifact(request.Artifact, request.RuntimeIdentity)
	if err != nil {
		return contract.WorkerResult{}, err
	}
	artifacts = append(artifacts, promptPreviewArtifact(request.Artifact, request.Config.ParsedRunInput))

	modelCount := len(runtimeInfo.Models)
	task := taskFromRunInput(request.Config.ParsedRunInput)
	message := "agent control plane worker placeholder completed"
	resultPayload := map[string]interface{}{
		"task":              task,
		"inputKeys":         sortedInputKeys(request.Config.ParsedRunInput),
		"validatedModels":   modelCount,
		"resolvedTools":     len(runtimeInfo.Tools),
		"resolvedKnowledge": len(runtimeInfo.Knowledge),
		"runtimeEntrypoint": runtimeInfo.Entrypoint,
	}
	if modelCount > 0 {
		message = fmt.Sprintf("agent control plane worker placeholder validated %d model binding(s)", modelCount)
	}
	if task != "" {
		message = fmt.Sprintf("%s for task %q", message, task)
	}
	if invocation, ok, err := r.invokePrimaryModel(ctx, request, runtimeInfo); err != nil {
		return contract.WorkerResult{}, err
	} else if ok {
		message = fmt.Sprintf("agent control plane worker executed model %q for task %q", invocation.ModelName, task)
		resultPayload["model"] = invocation.ModelName
		resultPayload["modelResponse"] = invocation.Content
		resultPayload["result"] = invocation.Parsed
		if summary, _ := invocation.Parsed["summary"].(string); strings.TrimSpace(summary) != "" {
			message = summary
		}
		artifacts = append(artifacts, invocation.Artifacts...)
	}
	if toolInvocation, ok, err := r.invokeRequestedTool(ctx, request, runtimeInfo); err != nil {
		return contract.WorkerResult{}, err
	} else if ok {
		resultPayload["toolCall"] = toolInvocation.Output
		artifacts = append(artifacts, toolInvocation.Artifacts...)
	}
	if retrievalInvocation, ok, err := r.invokeRequestedRetrieval(ctx, request, runtimeInfo); err != nil {
		return contract.WorkerResult{}, err
	} else if ok {
		resultPayload["retrievalCall"] = retrievalInvocation.Output
		artifacts = append(artifacts, retrievalInvocation.Artifacts...)
	}
	resultPayload["summary"] = message

	return contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          message,
		Config:           request.Config,
		CompiledArtifact: summarizeArtifact(request.Artifact),
		Output:           resultPayload,
		Artifacts:        artifacts,
		Runtime:          &runtimeInfo,
		StartedAt:        time.Now().UTC(),
	}, nil
}

func runnerFor(identity contract.RuntimeIdentity) (Runner, error) {
	if err := identity.ValidateSupported(); err != nil {
		return nil, err
	}
	return EinoADKPlaceholderRunner{Invoker: OpenAICompatibleInvoker{}}, nil
}

type ModelInvocation struct {
	ModelName string
	Content   string
	Parsed    map[string]interface{}
	Artifacts []contract.WorkerArtifact
}

type ExecutedToolInvocation struct {
	Output    map[string]interface{}
	Artifacts []contract.WorkerArtifact
}

func (r EinoADKPlaceholderRunner) invokePrimaryModel(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (ModelInvocation, bool, error) {
	modelName, modelConfig, ok := primaryModelConfig(request.Artifact)
	if !ok {
		return ModelInvocation{}, false, nil
	}
	modelRuntime, ok := runtimeInfo.Models[modelName]
	if !ok {
		return ModelInvocation{}, false, nil
	}
	if strings.TrimSpace(modelRuntime.BaseURL) == "" {
		return ModelInvocation{}, false, nil
	}

	systemPrompt := request.Artifact.Runner.Prompts["system"]
	if strings.TrimSpace(systemPrompt.Template) == "" {
		return ModelInvocation{}, false, nil
	}

	result, err := r.Invoker.Invoke(ctx, modelRuntime, modelConfig, systemPrompt, request.Config.ParsedRunInput, request.Artifact.Runner.Output)
	if err != nil {
		return ModelInvocation{}, false, err
	}
	return ModelInvocation{
		ModelName: modelName,
		Content:   result.Content,
		Parsed:    result.Parsed,
		Artifacts: []contract.WorkerArtifact{
			{Name: "chat-completion-request", Kind: "json", Inline: result.RequestBody},
			{Name: "chat-completion-response", Kind: "json", Inline: result.ResponseBody},
		},
	}, true, nil
}

func (r EinoADKPlaceholderRunner) invokeRequestedTool(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (ExecutedToolInvocation, bool, error) {
	call, ok, err := requestedToolCall(request.Config.ParsedRunInput)
	if err != nil {
		return ExecutedToolInvocation{}, false, err
	}
	if !ok {
		return ExecutedToolInvocation{}, false, nil
	}
	toolName := call.Name
	nodeName := call.Node
	if nodeName != "" {
		resolved, err := toolNameForGraphNode(request.Artifact, nodeName)
		if err != nil {
			return ExecutedToolInvocation{}, false, err
		}
		toolName = resolved
	}
	spec, ok := request.Artifact.Runner.Tools[toolName]
	if !ok {
		return ExecutedToolInvocation{}, false, FailureReasonError{
			Reason:  "UnknownTool",
			Message: fmt.Sprintf("unknown tool %q", toolName),
		}
	}
	runtime, ok := runtimeInfo.Tools[toolName]
	if !ok {
		return ExecutedToolInvocation{}, false, FailureReasonError{
			Reason:  "UnknownTool",
			Message: fmt.Sprintf("tool runtime binding missing for %q", toolName),
		}
	}
	spec.Name = toolName
	result, err := r.ToolInvoker.Invoke(ctx, runtime, spec, call.Input)
	if err != nil {
		return ExecutedToolInvocation{}, false, err
	}
	output := map[string]interface{}{
		"name":   toolName,
		"input":  call.Input,
		"output": result.Output,
	}
	requestInline := map[string]interface{}{
		"name":  toolName,
		"input": result.RequestBody,
	}
	responseInline := map[string]interface{}{
		"name":   toolName,
		"output": result.ResponseBody,
	}
	if nodeName != "" {
		output["node"] = nodeName
		requestInline["node"] = nodeName
		responseInline["node"] = nodeName
	}
	return ExecutedToolInvocation{
		Output: output,
		Artifacts: []contract.WorkerArtifact{
			{Name: "tool-request", Kind: "json", Inline: requestInline},
			{Name: "tool-response", Kind: "json", Inline: responseInline},
		},
	}, true, nil
}

func runtimeInfoForArtifact(artifact contract.CompiledArtifact, identity contract.RuntimeIdentity) (contract.WorkerRuntimeInfo, []contract.WorkerArtifact, error) {
	info := contract.WorkerRuntimeInfo{
		Engine:      identity.Engine,
		RunnerClass: identity.RunnerClass,
		Runner:      artifact.Runner.Kind,
		Entrypoint:  artifact.Runner.Entrypoint,
		Models:      make(map[string]contract.WorkerModelRuntime, len(artifact.Runner.Models)),
		Tools:       make(map[string]contract.WorkerToolRuntime, len(artifact.Runner.Tools)),
		Knowledge:   make(map[string]contract.WorkerKnowledgeRuntime, len(artifact.Runner.Knowledge)),
	}
	if info.Runner == "" {
		info.Runner = "EinoADKPlaceholderRunner"
	}
	if info.Entrypoint == "" {
		info.Entrypoint = artifact.Runtime.Entrypoint
	}

	modelNames := sortedModelNames(artifact.Runner.Models, artifact.Models)
	for _, name := range modelNames {
		model := artifact.Runner.Models[name]
		if model.Provider == "" && model.Model == "" && model.CredentialRef == nil && model.BaseURL == "" {
			model = artifact.Models[name]
		}

		apiKeyEnv := modelAPIKeyEnvName(name)
		modelRuntime := contract.WorkerModelRuntime{
			Provider:  model.Provider,
			Model:     model.Model,
			BaseURL:   model.BaseURL,
			APIKeyEnv: apiKeyEnv,
		}
		if model.CredentialRef != nil {
			if os.Getenv(apiKeyEnv) == "" {
				return contract.WorkerRuntimeInfo{}, nil, FailureReasonError{
					Reason:  "MissingModelCredentials",
					Message: fmt.Sprintf("missing model credentials for %q via %s", name, apiKeyEnv),
				}
			}
			modelRuntime.CredentialInjected = true
		}
		info.Models[name] = modelRuntime
	}

	artifacts := []contract.WorkerArtifact{
		{
			Name: "runtime-model-bindings",
			Kind: "json",
			Inline: map[string]interface{}{
				"models": info.Models,
			},
		},
	}
	for name, tool := range artifact.Runner.Tools {
		authEnv := toolAuthTokenEnvName(name, tool)
		credentialInjected := authEnv == "" || strings.TrimSpace(os.Getenv(authEnv)) != ""
		info.Tools[name] = contract.WorkerToolRuntime{
			Type:               tool.Type,
			Description:        tool.Description,
			Capabilities:       toolCapabilities(tool),
			AuthTokenEnv:       authEnv,
			CredentialInjected: credentialInjected,
		}
	}
	for name, knowledge := range artifact.Runner.Knowledge {
		info.Knowledge[name] = contract.WorkerKnowledgeRuntime{
			Ref:            knowledge.Ref,
			Description:    knowledge.Description,
			SourceCount:    len(knowledge.Sources),
			RetrievalBound: knowledge.Binding != nil,
			DefaultTopK:    nestedInt64(knowledge.Retrieval, "defaultTopK"),
			ScoreThreshold: nestedFloat64(knowledge.Retrieval, "defaultScoreThreshold"),
		}
	}
	if len(info.Tools) > 0 || len(info.Knowledge) > 0 {
		artifacts = append(artifacts, contract.WorkerArtifact{
			Name: "runtime-dependency-bindings",
			Kind: "json",
			Inline: map[string]interface{}{
				"tools":     info.Tools,
				"knowledge": info.Knowledge,
			},
		})
	}
	return info, artifacts, nil
}

func sortedModelNames(modelSets ...map[string]contract.ModelConfig) []string {
	seen := map[string]struct{}{}
	for _, models := range modelSets {
		for name := range models {
			seen[name] = struct{}{}
		}
	}
	names := make([]string, 0, len(seen))
	for name := range seen {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func primaryModelConfig(artifact contract.CompiledArtifact) (string, contract.ModelConfig, bool) {
	if model, ok := preferredModelConfig("planner", artifact); ok {
		return "planner", model, true
	}
	names := sortedModelNames(artifact.Runner.Models, artifact.Models)
	for _, name := range names {
		model, ok := preferredModelConfig(name, artifact)
		if !ok || model.Model == "" {
			continue
		}
		return name, model, true
	}
	return "", contract.ModelConfig{}, false
}

func preferredModelConfig(name string, artifact contract.CompiledArtifact) (contract.ModelConfig, bool) {
	model := artifact.Runner.Models[name]
	if model.Provider == "" && model.Model == "" && model.CredentialRef == nil && model.BaseURL == "" {
		model = artifact.Models[name]
	}
	if model.Provider == "" && model.Model == "" && model.CredentialRef == nil && model.BaseURL == "" {
		return contract.ModelConfig{}, false
	}
	return model, true
}

func modelAPIKeyEnvName(name string) string {
	return modelEnvPrefix(name) + "_API_KEY"
}

func toolAuthTokenEnvName(name string, tool contract.ToolSpec) string {
	authType := strings.TrimSpace(stringValue(nestedObject(tool.HTTP, "auth"), "type"))
	if authType != "bearerToken" {
		return ""
	}
	return toolEnvPrefix(name) + "_AUTH_TOKEN"
}

func toolCapabilities(tool contract.ToolSpec) []string {
	capabilities := make([]string, 0, 2)
	if len(tool.Runtime) > 0 {
		capabilities = append(capabilities, "runtime")
	}
	if len(tool.HTTP) > 0 {
		capabilities = append(capabilities, "http")
	}
	return capabilities
}

func nestedInt64(values map[string]interface{}, key string) int64 {
	if len(values) == 0 {
		return 0
	}
	switch value := values[key].(type) {
	case int64:
		return value
	case int32:
		return int64(value)
	case int:
		return int64(value)
	case float64:
		return int64(value)
	default:
		return 0
	}
}

func nestedFloat64(values map[string]interface{}, key string) float64 {
	if len(values) == 0 {
		return 0
	}
	switch value := values[key].(type) {
	case float64:
		return value
	case float32:
		return float64(value)
	case int:
		return float64(value)
	case int32:
		return float64(value)
	case int64:
		return float64(value)
	default:
		return 0
	}
}

func taskFromRunInput(input map[string]interface{}) string {
	value, ok := input["task"]
	if !ok {
		return ""
	}
	task, _ := value.(string)
	return task
}

func sortedInputKeys(input map[string]interface{}) []string {
	if len(input) == 0 {
		return nil
	}
	keys := make([]string, 0, len(input))
	for key := range input {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func promptPreviewArtifact(artifact contract.CompiledArtifact, input map[string]interface{}) contract.WorkerArtifact {
	systemPrompt := artifact.Runner.Prompts["system"]
	inline := map[string]interface{}{
		"system": map[string]interface{}{
			"name":              systemPrompt.Name,
			"language":          systemPrompt.Language,
			"template":          systemPrompt.Template,
			"variables":         systemPrompt.Variables,
			"outputConstraints": systemPrompt.OutputConstraints,
		},
		"userInput": input,
	}
	return contract.WorkerArtifact{
		Name:   "prompt-preview",
		Kind:   "json",
		Inline: inline,
	}
}

func modelEnvPrefix(name string) string {
	return envPrefix("MODEL", name)
}

func toolEnvPrefix(name string) string {
	return envPrefix("TOOL", name)
}

func envPrefix(kind string, name string) string {
	var builder strings.Builder
	lastUnderscore := false
	for _, r := range name {
		switch {
		case 'a' <= r && r <= 'z':
			builder.WriteRune(r - ('a' - 'A'))
			lastUnderscore = false
		case 'A' <= r && r <= 'Z', '0' <= r && r <= '9':
			builder.WriteRune(r)
			lastUnderscore = false
		case !lastUnderscore:
			builder.WriteByte('_')
			lastUnderscore = true
		}
	}
	prefix := strings.Trim(builder.String(), "_")
	if prefix == "" {
		prefix = kind
	}
	return kind + "_" + prefix
}

type RequestedToolCall struct {
	Name  string
	Node  string
	Input map[string]interface{}
}

func requestedToolCall(input map[string]interface{}) (RequestedToolCall, bool, error) {
	value, ok := input["toolCall"]
	if !ok {
		return RequestedToolCall{}, false, nil
	}
	raw, ok := value.(map[string]interface{})
	if !ok {
		return RequestedToolCall{}, false, FailureReasonError{
			Reason:  "InvalidToolCallRequest",
			Message: "toolCall must be a JSON object",
		}
	}
	name, _ := raw["name"].(string)
	node, _ := raw["node"].(string)
	if strings.TrimSpace(name) == "" && strings.TrimSpace(node) == "" {
		return RequestedToolCall{}, false, FailureReasonError{
			Reason:  "InvalidToolCallRequest",
			Message: "toolCall.name or toolCall.node is required",
		}
	}
	callInput, _ := raw["input"].(map[string]interface{})
	if callInput == nil {
		callInput = map[string]interface{}{}
	}
	return RequestedToolCall{Name: name, Node: node, Input: callInput}, true, nil
}

func toolNameForGraphNode(artifact contract.CompiledArtifact, nodeName string) (string, error) {
	nodes, ok := artifact.Runner.Graph["nodes"].([]interface{})
	if !ok {
		return "", FailureReasonError{
			Reason:  "UnknownGraphNode",
			Message: fmt.Sprintf("graph node %q was not found", nodeName),
		}
	}
	for _, raw := range nodes {
		node, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := node["name"].(string)
		if name != nodeName {
			continue
		}
		kind, _ := node["kind"].(string)
		if kind != "tool" {
			return "", FailureReasonError{
				Reason:  "UnsupportedGraphNode",
				Message: fmt.Sprintf("graph node %q is not a tool node", nodeName),
			}
		}
		toolRef, _ := node["toolRef"].(string)
		if strings.TrimSpace(toolRef) == "" {
			return "", FailureReasonError{
				Reason:  "UnknownTool",
				Message: fmt.Sprintf("graph node %q is missing toolRef", nodeName),
			}
		}
		return toolRef, nil
	}
	return "", FailureReasonError{
		Reason:  "UnknownGraphNode",
		Message: fmt.Sprintf("graph node %q was not found", nodeName),
	}
}
