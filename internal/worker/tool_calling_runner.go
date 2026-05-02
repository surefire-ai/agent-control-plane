package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/surefire-ai/korus/internal/contract"
)

const (
	defaultToolCallingMaxRounds = 3
)

// isToolCallingPattern checks if the compiled artifact uses the tool_calling pattern.
func isToolCallingPattern(artifact contract.CompiledArtifact) bool {
	if artifact.Pattern.Type == "tool_calling" {
		return true
	}
	if p, ok := artifact.Runner.Pattern["type"].(string); ok && p == "tool_calling" {
		return true
	}
	return false
}

// toolCallStep records one round of tool calling.
type toolCallStep struct {
	Round    int                    `json:"round"`
	ToolName string                 `json:"tool_name"`
	Input    map[string]interface{} `json:"input,omitempty"`
	Output   interface{}            `json:"output,omitempty"`
	Error    string                 `json:"error,omitempty"`
}

// executeToolCallingLoop runs the tool_calling pattern: model → tool calls → model.
//
// The loop:
//  1. Calls the model with tool definitions in OpenAI function-calling format.
//  2. If the model returns tool_calls, executes each tool.
//  3. Appends tool results to the conversation and calls the model again.
//  4. Repeats until the model returns a final answer (no tool_calls) or max rounds.
func (r EinoADKRunner) executeToolCallingLoop(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
) (contract.WorkerResult, bool, error) {
	artifact := request.Artifact

	// Resolve model config.
	modelRef := artifact.Pattern.ModelRef
	if modelRef == "" {
		modelRef = "planner"
	}
	modelConfig, ok := preferredModelConfig(modelRef, artifact)
	if !ok {
		return contract.WorkerResult{}, false, FailureReasonError{
			Reason:  "MissingModelConfig",
			Message: fmt.Sprintf("tool_calling pattern: model %q not declared in spec.models", modelRef),
		}
	}
	modelRuntime, ok := runtimeInfo.Models[modelRef]
	if !ok {
		return contract.WorkerResult{}, false, FailureReasonError{
			Reason:  "MissingModelRuntime",
			Message: fmt.Sprintf("tool_calling pattern: model %q not resolved in runtime", modelRef),
		}
	}
	if strings.TrimSpace(modelRuntime.BaseURL) == "" {
		return contract.WorkerResult{}, false, FailureReasonError{
			Reason:  "MissingModelBaseURL",
			Message: fmt.Sprintf("tool_calling pattern: model %q has no baseURL configured", modelRef),
		}
	}

	// Resolve system prompt.
	systemPrompt := artifact.Runner.Prompts["system"]
	if strings.TrimSpace(systemPrompt.Template) == "" {
		return contract.WorkerResult{}, false, FailureReasonError{
			Reason:  "MissingPrompt",
			Message: "system prompt is empty in compiled artifact",
		}
	}

	// Build tool definitions for OpenAI function calling format.
	toolDefs := buildToolDefinitions(artifact, runtimeInfo)

	maxRounds := artifact.Pattern.MaxIterations
	if maxRounds <= 0 {
		maxRounds = defaultToolCallingMaxRounds
	}

	// Build initial messages.
	userMsg := userMessageFromInput(request.Config.ParsedRunInput)
	messages := []toolCallingMessage{
		{Role: "system", Content: systemPrompt.Template},
		{Role: "user", Content: userMsg},
	}

	var steps []toolCallStep
	var lastResult ModelInvocationResult
	round := 0

	for round < int(maxRounds) {
		select {
		case <-ctx.Done():
			return contract.WorkerResult{}, false, ctx.Err()
		default:
		}

		round++

		// Call model with tools.
		result, err := r.callModelWithTools(ctx, modelRuntime, modelConfig, messages, toolDefs)
		if err != nil {
			return contract.WorkerResult{}, false, fmt.Errorf("tool_calling round %d model call failed: %w", round, err)
		}
		lastResult = result

		// Parse response for tool_calls.
		assistantMsg := parseToolCallingResponse(result)

		// If no tool calls, this is the final answer.
		if len(assistantMsg.ToolCalls) == 0 {
			// Parse final output.
			output := make(map[string]interface{})
			if err := json.Unmarshal([]byte(result.Content), &output); err != nil {
				output["text"] = result.Content
			}
			output["pattern"] = "tool_calling"
			output["rounds"] = round
			output["toolSteps"] = len(steps)

			task := taskFromRunInput(request.Config.ParsedRunInput)
			message := fmt.Sprintf("tool_calling pattern completed in %d round(s) for task %q", round, task)

			runtimeInfo.Runner = "EinoADKRunner"

			artifacts := []contract.WorkerArtifact{
				{Name: "tool-calling-trace", Kind: "json", Inline: map[string]interface{}{"steps": steps}},
			}
			artifacts = append(artifacts, modelArtifactsFromResult(result)...)

			return contract.WorkerResult{
				Status:           contract.WorkerStatusSucceeded,
				Message:          message,
				Config:           request.Config,
				CompiledArtifact: summarizeArtifact(artifact),
				Output:           output,
				Artifacts:        artifacts,
				Runtime:          &runtimeInfo,
			}, true, nil
		}

		// Execute tool calls.
		// Append assistant message with tool_calls.
		messages = append(messages, *assistantMsg)

		for _, tc := range assistantMsg.ToolCalls {
			step := toolCallStep{
				Round:    round,
				ToolName: tc.Function.Name,
			}

			// Parse tool input.
			var toolInput map[string]interface{}
			if err := json.Unmarshal([]byte(tc.Function.Arguments), &toolInput); err != nil {
				toolInput = map[string]interface{}{"raw": tc.Function.Arguments}
			}
			step.Input = toolInput

			// Execute tool.
			observation, toolErr := r.executeTool(ctx, artifact, runtimeInfo, tc.Function.Name, toolInput)
			if toolErr != nil {
				step.Error = toolErr.Error()
				step.Output = map[string]interface{}{"error": toolErr.Error()}
			} else {
				step.Output = observation
			}
			steps = append(steps, step)

			// Append tool result message.
			obsJSON, _ := json.Marshal(step.Output)
			messages = append(messages, toolCallingMessage{
				Role:       "tool",
				Content:    string(obsJSON),
				ToolCallID: tc.ID,
			})
		}
	}

	// Max rounds reached — return last model response.
	output := make(map[string]interface{})
	if err := json.Unmarshal([]byte(lastResult.Content), &output); err != nil {
		output["text"] = lastResult.Content
	}
	output["pattern"] = "tool_calling"
	output["rounds"] = round
	output["toolSteps"] = len(steps)
	output["maxRoundsReached"] = true

	task := taskFromRunInput(request.Config.ParsedRunInput)
	message := fmt.Sprintf("tool_calling pattern reached max rounds (%d) for task %q", maxRounds, task)

	runtimeInfo.Runner = "EinoADKRunner"

	artifacts := []contract.WorkerArtifact{
		{Name: "tool-calling-trace", Kind: "json", Inline: map[string]interface{}{"steps": steps}},
	}
	artifacts = append(artifacts, modelArtifactsFromResult(lastResult)...)

	return contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          message,
		Config:           request.Config,
		CompiledArtifact: summarizeArtifact(artifact),
		Output:           output,
		Artifacts:        artifacts,
		Runtime:          &runtimeInfo,
	}, true, nil
}

// toolCallingMessage represents a message in the tool calling conversation.
type toolCallingMessage struct {
	Role       string            `json:"role"`
	Content    string            `json:"content,omitempty"`
	ToolCalls  []toolCallRequest `json:"tool_calls,omitempty"`
	ToolCallID string            `json:"tool_call_id,omitempty"`
}

// toolCallRequest represents a tool call from the model.
type toolCallRequest struct {
	ID       string           `json:"id"`
	Type     string           `json:"type"`
	Function toolCallFunction `json:"function"`
}

// toolCallFunction represents the function call details.
type toolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

// toolDefinition represents an OpenAI tool definition.
type toolDefinition struct {
	Type     string          `json:"type"`
	Function toolFunctionDef `json:"function"`
}

// toolFunctionDef represents a function definition for OpenAI tools.
type toolFunctionDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Parameters  interface{} `json:"parameters,omitempty"`
}

// buildToolDefinitions builds OpenAI-format tool definitions from the compiled artifact.
func buildToolDefinitions(artifact contract.CompiledArtifact, runtimeInfo contract.WorkerRuntimeInfo) []toolDefinition {
	var defs []toolDefinition

	for name, tool := range artifact.Runner.Tools {
		runtime, ok := runtimeInfo.Tools[name]
		if !ok || runtime.Type == "" {
			continue
		}

		def := toolDefinition{
			Type: "function",
			Function: toolFunctionDef{
				Name:        name,
				Description: tool.Description,
			},
		}

		// Use schema if available.
		if tool.Schema != nil {
			def.Function.Parameters = tool.Schema
		}

		defs = append(defs, def)
	}

	return defs
}

// callModelWithTools calls the model with tool definitions.
func (r EinoADKRunner) callModelWithTools(
	ctx context.Context,
	model contract.WorkerModelRuntime,
	config contract.ModelConfig,
	messages []toolCallingMessage,
	tools []toolDefinition,
) (ModelInvocationResult, error) {
	invoker := r.modelInvoker()

	// Build request body manually to include tools.
	requestBody := map[string]interface{}{
		"model":    config.Model,
		"messages": messages,
	}
	if config.Temperature != 0 {
		requestBody["temperature"] = config.Temperature
	}
	if config.MaxTokens > 0 {
		requestBody["max_tokens"] = config.MaxTokens
	}
	if len(tools) > 0 {
		requestBody["tools"] = tools
	}

	// Use the invoker's raw call method if available, otherwise fall back to standard invoke.
	// For now, we'll use a direct approach through the invoker.
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return ModelInvocationResult{}, fmt.Errorf("marshaling tool calling request: %w", err)
	}

	// Call the model through the standard invoker but with a custom prompt that includes tool context.
	// We need to use the raw HTTP approach for tool calling.
	return invoker.InvokeWithBody(ctx, model, config, rawBody, requestBody)
}

// parseToolCallingResponse parses the model response for tool calls.
func parseToolCallingResponse(result ModelInvocationResult) *toolCallingMessage {
	msg := &toolCallingMessage{
		Role:    "assistant",
		Content: result.Content,
	}

	// Check if response has tool_calls in parsed output.
	if result.ResponseBody != nil {
		if choices, ok := result.ResponseBody["choices"].([]interface{}); ok && len(choices) > 0 {
			if choice, ok := choices[0].(map[string]interface{}); ok {
				if message, ok := choice["message"].(map[string]interface{}); ok {
					if toolCalls, ok := message["tool_calls"].([]interface{}); ok {
						for _, tc := range toolCalls {
							if tcMap, ok := tc.(map[string]interface{}); ok {
								tcReq := toolCallRequest{
									ID:   getStringField(tcMap, "id"),
									Type: getStringField(tcMap, "type"),
								}
								if fn, ok := tcMap["function"].(map[string]interface{}); ok {
									tcReq.Function = toolCallFunction{
										Name:      getStringField(fn, "name"),
										Arguments: getStringField(fn, "arguments"),
									}
								}
								msg.ToolCalls = append(msg.ToolCalls, tcReq)
							}
						}
					}
					// Update content from response.
					if content, ok := message["content"].(string); ok && content != "" {
						msg.Content = content
					}
				}
			}
		}
	}

	return msg
}

// getStringField extracts a string field from a map.
func getStringField(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// modelArtifactsFromResult extracts artifacts from a model invocation result.
func modelArtifactsFromResult(result ModelInvocationResult) []contract.WorkerArtifact {
	return []contract.WorkerArtifact{
		{Name: "chat-completion-request", Kind: "json", Inline: result.RequestBody},
		{Name: "chat-completion-response", Kind: "json", Inline: result.ResponseBody},
	}
}
