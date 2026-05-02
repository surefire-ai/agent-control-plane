package worker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/surefire-ai/korus/internal/contract"
)

// EinoADKRunner is the real Eino-based runner that executes agents using
// Eino's compose.Graph orchestration. It replaces the placeholder runner
// for production use.
//
// The runner supports two execution modes:
//  1. Graph mode: when the compiled artifact defines a graph, it builds an
//     Eino compose.Graph and executes it.
//  2. Single-model mode: when no graph is defined, it falls back to a
//     direct model invocation (the EHS happy path).
type EinoADKRunner struct {
	Invoker          ModelInvoker
	ToolInvoker      ToolInvoker
	RetrievalInvoker RetrievalInvoker
}

func (r EinoADKRunner) modelInvoker() ModelInvoker {
	if r.Invoker != nil {
		return r.Invoker
	}
	return EinoOpenAIInvoker{}
}

func (r EinoADKRunner) toolInvoker() ToolInvoker {
	if r.ToolInvoker != nil {
		return r.ToolInvoker
	}
	return EinoToolInvoker{}
}

func (r EinoADKRunner) retrievalInvoker() RetrievalInvoker {
	if r.RetrievalInvoker != nil {
		return r.RetrievalInvoker
	}
	return EinoRetrievalInvoker{}
}

// patternHandler is a function that attempts to handle a specific pattern.
// Returns (result, true, nil) if the pattern matched and was executed,
// (zero, false, nil) if the pattern doesn't apply, or an error.
type patternHandler func(ctx context.Context, r EinoADKRunner, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (contract.WorkerResult, bool, error)

// patternRegistry defines the order in which pattern handlers are tried.
// Earlier entries take precedence. New patterns should be appended here.
var patternRegistry = []struct {
	name    string
	check   func(contract.CompiledArtifact) bool
	handler patternHandler
}{
	{"react", isReactPattern, func(ctx context.Context, r EinoADKRunner, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (contract.WorkerResult, bool, error) {
		return r.executeReactLoop(ctx, request, runtimeInfo)
	}},
	{"router", isRouterPattern, func(ctx context.Context, r EinoADKRunner, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (contract.WorkerResult, bool, error) {
		return r.executeRouterLoop(ctx, request, runtimeInfo)
	}},
	{"reflection", isReflectionPattern, func(ctx context.Context, r EinoADKRunner, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (contract.WorkerResult, bool, error) {
		return r.executeReflectionLoop(ctx, request, runtimeInfo)
	}},
	{"tool_calling", isToolCallingPattern, func(ctx context.Context, r EinoADKRunner, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (contract.WorkerResult, bool, error) {
		return r.executeToolCallingLoop(ctx, request, runtimeInfo)
	}},
	{"plan_execute", isPlanExecutePattern, func(ctx context.Context, r EinoADKRunner, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (contract.WorkerResult, bool, error) {
		return r.executePlanExecuteLoop(ctx, request, runtimeInfo)
	}},
	{"workflow", isWorkflowPattern, func(ctx context.Context, r EinoADKRunner, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (contract.WorkerResult, bool, error) {
		return r.executeWorkflow(ctx, request, runtimeInfo)
	}},
}

func (r EinoADKRunner) Run(ctx context.Context, request RunRequest) (contract.WorkerResult, error) {
	select {
	case <-ctx.Done():
		return contract.WorkerResult{}, ctx.Err()
	default:
	}

	startedAt := time.Now().UTC()

	runtimeInfo, artifacts, err := runtimeInfoForArtifact(request.Artifact, request.RuntimeIdentity)
	if err != nil {
		return contract.WorkerResult{}, err
	}
	artifacts = append(artifacts, promptPreviewArtifact(request.Artifact, request.Config.ParsedRunInput))

	// Try registered pattern handlers in order.
	for _, ph := range patternRegistry {
		if !ph.check(request.Artifact) {
			continue
		}
		if result, ok, err := ph.handler(ctx, r, request, runtimeInfo); err != nil {
			return contract.WorkerResult{}, err
		} else if ok {
			result.StartedAt = startedAt
			result.Artifacts = append(result.Artifacts, artifacts...)
			return result, nil
		}
	}

	// Try graph-based execution next.
	if result, ok, err := r.tryGraphExecution(ctx, request, runtimeInfo); err != nil {
		return contract.WorkerResult{}, err
	} else if ok {
		result.StartedAt = startedAt
		result.Artifacts = append(result.Artifacts, artifacts...)
		return result, nil
	}

	// Fall back to single-model execution (EHS happy path).
	result, err := r.executeSingleModel(ctx, request, runtimeInfo)
	if err != nil {
		return contract.WorkerResult{}, err
	}
	result.StartedAt = startedAt
	result.Artifacts = append(result.Artifacts, artifacts...)
	return result, nil
}

// tryGraphExecution attempts to build and execute an Eino compose.Graph from
// the compiled artifact's graph definition. Returns (result, true, nil) on
// success, (zero, false, nil) if no graph is available, or an error.
func (r EinoADKRunner) tryGraphExecution(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
) (contract.WorkerResult, bool, error) {
	graphDef := request.Artifact.Runner.Graph
	nodesRaw, ok := graphDef["nodes"].([]interface{})
	if !ok || len(nodesRaw) == 0 {
		return contract.WorkerResult{}, false, nil
	}

	runnable, err := buildGraph(ctx, request.Artifact, runtimeInfo, r.modelInvoker(), r.toolInvoker(), r.retrievalInvoker())
	if err != nil {
		return contract.WorkerResult{}, false, fmt.Errorf("building eino graph: %w", err)
	}
	if runnable == nil {
		return contract.WorkerResult{}, false, nil
	}

	// Prepare initial state from run input.
	initialState := make(graphState, len(request.Config.ParsedRunInput)+2)
	for k, v := range request.Config.ParsedRunInput {
		initialState[k] = v
	}
	initialState["_agent"] = request.Config.AgentName
	initialState["_run"] = request.Config.AgentRunName

	// Execute the graph.
	output, err := runnable.Invoke(ctx, initialState)
	if err != nil {
		return contract.WorkerResult{}, false, fmt.Errorf("executing eino graph: %w", err)
	}

	// Build the result from graph output.
	message := "agent control plane worker completed via eino graph"
	if summary, ok := output["summary"].(string); ok && strings.TrimSpace(summary) != "" {
		message = summary
	}

	task := taskFromRunInput(request.Config.ParsedRunInput)
	if task != "" {
		message = fmt.Sprintf("%s for task %q", message, task)
	}

	runtimeInfo.Runner = "EinoADKRunner"

	return contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          message,
		Config:           request.Config,
		CompiledArtifact: summarizeArtifact(request.Artifact),
		Output:           output,
		Runtime:          &runtimeInfo,
	}, true, nil
}

// executeSingleModel runs the EHS happy path: a single model invocation
// with the system prompt and user input, producing structured JSON output.
// If no model is configured or the model has no base URL, it returns a
// success result with diagnostic information (no model call is made).
func (r EinoADKRunner) executeSingleModel(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
) (contract.WorkerResult, error) {
	task := taskFromRunInput(request.Config.ParsedRunInput)
	inputKeys := sortedInputKeys(request.Config.ParsedRunInput)
	modelCount := len(runtimeInfo.Models)

	message := "agent control plane worker completed"
	if modelCount > 0 {
		message = fmt.Sprintf("agent control plane worker validated %d model binding(s)", modelCount)
	}
	if task != "" {
		message = fmt.Sprintf("%s for task %q", message, task)
	}

	modelName, modelConfig, ok := primaryModelConfig(request.Artifact)
	if !ok {
		// No model configured — return diagnostic result.
		runtimeInfo.Runner = "EinoADKRunner"
		return contract.WorkerResult{
			Status:           contract.WorkerStatusSucceeded,
			Message:          message,
			Config:           request.Config,
			CompiledArtifact: summarizeArtifact(request.Artifact),
			Output: map[string]interface{}{
				"task":              task,
				"inputKeys":         inputKeys,
				"validatedModels":   modelCount,
				"resolvedTools":     len(runtimeInfo.Tools),
				"resolvedSkills":    len(runtimeInfo.Skills),
				"resolvedKnowledge": len(runtimeInfo.Knowledge),
				"runtimeEntrypoint": runtimeInfo.Entrypoint,
				"summary":           message,
			},
			Runtime: &runtimeInfo,
		}, nil
	}

	modelRuntime, ok := runtimeInfo.Models[modelName]
	if !ok {
		// Model not in runtime — return diagnostic result.
		runtimeInfo.Runner = "EinoADKRunner"
		return contract.WorkerResult{
			Status:           contract.WorkerStatusSucceeded,
			Message:          message,
			Config:           request.Config,
			CompiledArtifact: summarizeArtifact(request.Artifact),
			Output: map[string]interface{}{
				"task":              task,
				"inputKeys":         inputKeys,
				"validatedModels":   modelCount,
				"resolvedTools":     len(runtimeInfo.Tools),
				"resolvedSkills":    len(runtimeInfo.Skills),
				"resolvedKnowledge": len(runtimeInfo.Knowledge),
				"runtimeEntrypoint": runtimeInfo.Entrypoint,
				"summary":           message,
			},
			Runtime: &runtimeInfo,
		}, nil
	}
	if strings.TrimSpace(modelRuntime.BaseURL) == "" {
		// No base URL — return diagnostic result (model binding validated but not invoked).
		runtimeInfo.Runner = "EinoADKRunner"
		return contract.WorkerResult{
			Status:           contract.WorkerStatusSucceeded,
			Message:          message,
			Config:           request.Config,
			CompiledArtifact: summarizeArtifact(request.Artifact),
			Output: map[string]interface{}{
				"task":              task,
				"inputKeys":         inputKeys,
				"validatedModels":   modelCount,
				"resolvedTools":     len(runtimeInfo.Tools),
				"resolvedSkills":    len(runtimeInfo.Skills),
				"resolvedKnowledge": len(runtimeInfo.Knowledge),
				"runtimeEntrypoint": runtimeInfo.Entrypoint,
				"summary":           message,
			},
			Runtime: &runtimeInfo,
		}, nil
	}

	systemPrompt := request.Artifact.Runner.Prompts["system"]
	if strings.TrimSpace(systemPrompt.Template) == "" {
		return contract.WorkerResult{}, FailureReasonError{
			Reason:  "MissingPrompt",
			Message: "system prompt is empty in compiled artifact",
		}
	}

	if modelRuntime.ProviderFamily != "" && modelRuntime.ProviderFamily != "openai-compatible" {
		return contract.WorkerResult{}, FailureReasonError{
			Reason:  "UnsupportedModelProvider",
			Message: fmt.Sprintf("provider %q family %q is not wired yet", modelRuntime.Provider, modelRuntime.ProviderFamily),
		}
	}

	result, err := r.modelInvoker().Invoke(ctx, modelRuntime, modelConfig, systemPrompt, request.Config.ParsedRunInput, request.Artifact.Runner.Output)
	if err != nil {
		return contract.WorkerResult{}, err
	}

	task = taskFromRunInput(request.Config.ParsedRunInput)
	message = fmt.Sprintf("agent control plane worker executed model %q for task %q", modelName, task)
	if summary, _ := result.Parsed["summary"].(string); strings.TrimSpace(summary) != "" {
		message = summary
	}

	runtimeInfo.Runner = "EinoADKRunner"

	output := make(map[string]interface{}, len(result.Parsed)+4)
	for k, v := range result.Parsed {
		output[k] = v
	}
	output["model"] = modelName
	output["modelResponse"] = result.Content

	// Build model invocation artifacts (request/response).
	modelArtifacts := []contract.WorkerArtifact{
		{Name: "chat-completion-request", Kind: "json", Inline: result.RequestBody},
		{Name: "chat-completion-response", Kind: "json", Inline: result.ResponseBody},
	}

	return contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          message,
		Config:           request.Config,
		CompiledArtifact: summarizeArtifact(request.Artifact),
		Output:           output,
		Artifacts:        modelArtifacts,
		Runtime:          &runtimeInfo,
	}, nil
}
