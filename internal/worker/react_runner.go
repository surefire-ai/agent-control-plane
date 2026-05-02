package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/surefire-ai/korus/internal/contract"
)

const (
	defaultReactMaxIterations = 6
	reactFinalAnswerKey       = "final_answer"
	reactActionKey            = "action"
	reactActionInputKey       = "action_input"
)

// reactDecision represents the model's decision in a ReAct iteration.
type reactDecision struct {
	IsFinal     bool
	Action      string
	ActionInput map[string]interface{}
	FinalAnswer interface{}
	RawResponse string
}

// reactStep records one iteration of the ReAct loop.
type reactStep struct {
	Iteration   int                    `json:"iteration"`
	Thought     string                 `json:"thought,omitempty"`
	Action      string                 `json:"action,omitempty"`
	ActionInput map[string]interface{} `json:"action_input,omitempty"`
	Observation interface{}            `json:"observation,omitempty"`
	FinalAnswer interface{}            `json:"final_answer,omitempty"`
}

// isReactPattern checks whether the compiled artifact declares a ReAct pattern.
func isReactPattern(artifact contract.CompiledArtifact) bool {
	if artifact.Pattern.Type == "react" {
		return true
	}
	if p, ok := artifact.Runner.Pattern["type"].(string); ok && p == "react" {
		return true
	}
	return false
}

// reactMaxIterations extracts maxIterations from the artifact pattern config.
func reactMaxIterations(artifact contract.CompiledArtifact) int32 {
	if artifact.Pattern.MaxIterations > 0 {
		return artifact.Pattern.MaxIterations
	}
	if v, ok := artifact.Runner.Pattern["maxIterations"].(float64); ok && v > 0 {
		return int32(v)
	}
	if v, ok := artifact.Runner.Pattern["maxIterations"].(int32); ok && v > 0 {
		return v
	}
	return defaultReactMaxIterations
}

// executeReactLoop runs the iterative ReAct reasoning-action loop.
//
// The loop:
//  1. Calls the model with the augmented system prompt and conversation history.
//  2. Parses the model response to decide: tool call or final answer.
//  3. If tool call: executes the tool, appends observation to history, loops.
//  4. If final answer: returns the result.
//  5. Enforces maxIterations to prevent infinite loops.
func (r EinoADKRunner) executeReactLoop(
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
		return contract.WorkerResult{}, false, nil
	}
	modelRuntime, ok := runtimeInfo.Models[modelRef]
	if !ok {
		return contract.WorkerResult{}, false, nil
	}
	if strings.TrimSpace(modelRuntime.BaseURL) == "" {
		return contract.WorkerResult{}, false, nil
	}

	// Resolve system prompt.
	systemPrompt := artifact.Runner.Prompts["system"]
	if strings.TrimSpace(systemPrompt.Template) == "" {
		return contract.WorkerResult{}, false, nil
	}

	// Build augmented system prompt with tool descriptions and ReAct format.
	augmentedPrompt := buildReactSystemPrompt(systemPrompt, artifact)

	maxIter := reactMaxIterations(artifact)

	// Build initial user message from run input.
	userMsg := userMessageFromInput(request.Config.ParsedRunInput)

	// Conversation history for the ReAct loop.
	messages := []reactMessage{
		{Role: "system", Content: augmentedPrompt},
		{Role: "user", Content: userMsg},
	}

	var steps []reactStep
	var lastResult ModelInvocationResult

	for iter := int32(1); iter <= maxIter; iter++ {
		select {
		case <-ctx.Done():
			return contract.WorkerResult{}, false, ctx.Err()
		default:
		}

		// Call the model.
		result, err := r.callModelForReact(ctx, modelRuntime, modelConfig, messages)
		if err != nil {
			return contract.WorkerResult{}, false, fmt.Errorf("react iteration %d model call failed: %w", iter, err)
		}
		lastResult = result

		// Parse the decision.
		decision := parseReactDecision(result.Content, result.Parsed)

		step := reactStep{
			Iteration: int(iter),
			Thought:   decision.RawResponse,
		}

		if decision.IsFinal {
			step.FinalAnswer = decision.FinalAnswer
			steps = append(steps, step)

			return r.buildReactResult(request, runtimeInfo, steps, decision.FinalAnswer, lastResult), true, nil
		}

		// Execute the tool.
		step.Action = decision.Action
		step.ActionInput = decision.ActionInput

		observation, toolErr := r.executeReactTool(ctx, artifact, runtimeInfo, decision.Action, decision.ActionInput)
		if toolErr != nil {
			observation = map[string]interface{}{
				"error": toolErr.Error(),
			}
		}
		step.Observation = observation
		steps = append(steps, step)

		// Append assistant response and tool observation to conversation.
		messages = append(messages, reactMessage{Role: "assistant", Content: result.Content})
		obsJSON, _ := json.Marshal(observation)
		messages = append(messages, reactMessage{Role: "user", Content: fmt.Sprintf("Observation for %s:\n%s", decision.Action, string(obsJSON))})
	}

	// Max iterations reached — return what we have with the last model response.
	finalAnswer := lastResult.Parsed
	if finalAnswer == nil {
		finalAnswer = map[string]interface{}{
			"raw": lastResult.Content,
		}
	}
	steps = append(steps, reactStep{
		Iteration:   int(maxIter) + 1,
		FinalAnswer: finalAnswer,
		Thought:     "max iterations reached",
	})

	return r.buildReactResult(request, runtimeInfo, steps, finalAnswer, lastResult), true, nil
}

// reactMessage is a simple message struct for the ReAct conversation.
type reactMessage struct {
	Role    string
	Content string
}

// callModelForReact calls the model with the full conversation history.
func (r EinoADKRunner) callModelForReact(
	ctx context.Context,
	modelRuntime contract.WorkerModelRuntime,
	modelConfig contract.ModelConfig,
	messages []reactMessage,
) (ModelInvocationResult, error) {
	invoker := r.modelInvoker()

	// Build a synthetic prompt from conversation history.
	// The ModelInvoker interface expects a single PromptSpec + input map.
	// For ReAct, we encode the full conversation into the system prompt.
	systemContent := messages[0].Content
	var userContent string
	if len(messages) > 1 {
		// Concatenate all non-system messages into the user content.
		var parts []string
		for _, msg := range messages[1:] {
			prefix := "User"
			if msg.Role == "assistant" {
				prefix = "Assistant"
			}
			parts = append(parts, fmt.Sprintf("%s: %s", prefix, msg.Content))
		}
		userContent = strings.Join(parts, "\n\n")
	}

	prompt := contract.PromptSpec{
		Name:     "react-loop",
		Template: systemContent,
	}
	input := map[string]interface{}{
		"task": userContent,
	}

	return invoker.Invoke(ctx, modelRuntime, modelConfig, prompt, input, nil)
}

// buildReactSystemPrompt augments the system prompt with tool descriptions
// and ReAct output format instructions.
func buildReactSystemPrompt(basePrompt contract.PromptSpec, artifact contract.CompiledArtifact) string {
	var b strings.Builder
	b.WriteString(basePrompt.Template)

	// Add tool descriptions.
	tools := artifact.Runner.Tools
	if len(tools) > 0 {
		b.WriteString("\n\n## Available Tools\n\n")
		b.WriteString("You have access to the following tools. Use them when needed:\n\n")
		for name, tool := range tools {
			desc := tool.Description
			if desc == "" {
				desc = name
			}
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", name, desc))
		}
	}

	// Add knowledge context.
	knowledge := artifact.Runner.Knowledge
	if len(knowledge) > 0 {
		b.WriteString("\n\n## Knowledge Sources\n\n")
		for name, kb := range knowledge {
			desc := kb.Description
			if desc == "" {
				desc = name
			}
			b.WriteString(fmt.Sprintf("- **%s**: %s\n", name, desc))
		}
	}

	// Add ReAct format instructions.
	b.WriteString("\n\n## Response Format\n\n")
	b.WriteString("You must respond in one of two JSON formats:\n\n")
	b.WriteString("**To use a tool:**\n")
	b.WriteString("```json\n")
	b.WriteString(`{"action": "tool_name", "action_input": {"key": "value"}}`)
	b.WriteString("\n```\n\n")
	b.WriteString("**When you have the final answer:**\n")
	b.WriteString("```json\n")
	b.WriteString(`{"final_answer": {"summary": "your answer", ...}}`)
	b.WriteString("\n```\n\n")
	b.WriteString("Think step by step. Use tools when you need information. ")
	b.WriteString("When you have enough information, provide the final_answer.")

	return b.String()
}

// userMessageFromInput builds a user message string from the run input.
func userMessageFromInput(input map[string]interface{}) string {
	if task, ok := input["task"].(string); ok {
		if payload, ok := input["payload"].(map[string]interface{}); ok {
			payloadJSON, _ := json.Marshal(payload)
			return fmt.Sprintf("Task: %s\n\nInput:\n%s", task, string(payloadJSON))
		}
		return fmt.Sprintf("Task: %s", task)
	}
	inputJSON, _ := json.Marshal(input)
	return string(inputJSON)
}

// parseReactDecision parses the model's response into a reactDecision.
func parseReactDecision(content string, parsed map[string]interface{}) reactDecision {
	if parsed == nil {
		// Try to extract JSON from the content.
		parsed = extractJSONFromContent(content)
	}

	if parsed == nil {
		return reactDecision{
			IsFinal:     true,
			FinalAnswer: map[string]interface{}{"raw": content},
			RawResponse: content,
		}
	}

	// Check for final_answer.
	if fa, ok := parsed[reactFinalAnswerKey]; ok {
		return reactDecision{
			IsFinal:     true,
			FinalAnswer: fa,
			RawResponse: content,
		}
	}

	// Check for action (tool call).
	if action, ok := parsed[reactActionKey].(string); ok && strings.TrimSpace(action) != "" {
		actionInput, _ := parsed[reactActionInputKey].(map[string]interface{})
		return reactDecision{
			IsFinal:     false,
			Action:      action,
			ActionInput: actionInput,
			RawResponse: content,
		}
	}

	// Fallback: treat as final answer.
	return reactDecision{
		IsFinal:     true,
		FinalAnswer: parsed,
		RawResponse: content,
	}
}

// extractJSONFromContent tries to extract a JSON object from free-form text.
func extractJSONFromContent(content string) map[string]interface{} {
	// Look for JSON code blocks.
	start := strings.Index(content, "```json")
	if start == -1 {
		start = strings.Index(content, "```")
	}
	if start != -1 {
		end := strings.Index(content[start+3:], "```")
		if end != -1 {
			jsonStr := strings.TrimSpace(content[start+3 : start+3+end])
			if strings.HasPrefix(jsonStr, "json\n") {
				jsonStr = jsonStr[5:]
			}
			var result map[string]interface{}
			if json.Unmarshal([]byte(jsonStr), &result) == nil {
				return result
			}
		}
	}

	// Try to find a JSON object in the content.
	braceStart := strings.Index(content, "{")
	braceEnd := strings.LastIndex(content, "}")
	if braceStart != -1 && braceEnd > braceStart {
		var result map[string]interface{}
		if json.Unmarshal([]byte(content[braceStart:braceEnd+1]), &result) == nil {
			return result
		}
	}

	return nil
}

// executeReactTool executes a named tool from the compiled artifact.
func (r EinoADKRunner) executeReactTool(
	ctx context.Context,
	artifact contract.CompiledArtifact,
	runtimeInfo contract.WorkerRuntimeInfo,
	toolName string,
	input map[string]interface{},
) (map[string]interface{}, error) {
	spec, ok := artifact.Runner.Tools[toolName]
	if !ok {
		return nil, fmt.Errorf("tool %q not found in compiled artifact", toolName)
	}
	runtime, ok := runtimeInfo.Tools[toolName]
	if !ok {
		return nil, fmt.Errorf("tool %q not found in runtime bindings", toolName)
	}
	spec.Name = toolName

	if input == nil {
		input = make(map[string]interface{})
	}

	result, err := r.toolInvoker().Invoke(ctx, runtime, spec, input)
	if err != nil {
		return nil, err
	}

	return result.Output, nil
}

// buildReactResult constructs the WorkerResult from the ReAct loop output.
func (r EinoADKRunner) buildReactResult(
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
	steps []reactStep,
	finalAnswer interface{},
	lastModelResult ModelInvocationResult,
) contract.WorkerResult {
	task := taskFromRunInput(request.Config.ParsedRunInput)
	iterationCount := len(steps)

	message := fmt.Sprintf("agent control plane worker completed ReAct loop in %d iteration(s)", iterationCount)
	if task != "" {
		message = fmt.Sprintf("%s for task %q", message, task)
	}

	runtimeInfo.Runner = "EinoADKRunner"

	output := map[string]interface{}{
		"final_answer":   finalAnswer,
		"iterations":     iterationCount,
		"reasoning":      steps,
		"pattern":        "react",
		"max_iterations": reactMaxIterations(request.Artifact),
	}

	// Add model invocation artifacts.
	artifacts := []contract.WorkerArtifact{
		{Name: "react-trace", Kind: "json", Inline: map[string]interface{}{"steps": steps}},
	}
	if lastModelResult.RequestBody != nil {
		artifacts = append(artifacts, contract.WorkerArtifact{
			Name: "chat-completion-request", Kind: "json", Inline: lastModelResult.RequestBody,
		})
	}
	if lastModelResult.ResponseBody != nil {
		artifacts = append(artifacts, contract.WorkerArtifact{
			Name: "chat-completion-response", Kind: "json", Inline: lastModelResult.ResponseBody,
		})
	}

	return contract.WorkerResult{
		Status:           contract.WorkerStatusSucceeded,
		Message:          message,
		Config:           request.Config,
		CompiledArtifact: summarizeArtifact(request.Artifact),
		Output:           output,
		Artifacts:        artifacts,
		Runtime:          &runtimeInfo,
	}
}
