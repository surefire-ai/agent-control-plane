package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/surefire-ai/korus/internal/contract"
)

const (
	defaultPlanExecuteMaxSteps = 5
)

// isPlanExecutePattern checks if the compiled artifact uses the plan_execute pattern.
func isPlanExecutePattern(artifact contract.CompiledArtifact) bool {
	if artifact.Pattern.Type == "plan_execute" {
		return true
	}
	if p, ok := artifact.Runner.Pattern["type"].(string); ok && p == "plan_execute" {
		return true
	}
	return false
}

// planStep represents one step in a plan.
type planStep struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Action      string `json:"action,omitempty"`
}

// planStepResult records the execution result of one plan step.
type planStepResult struct {
	Step    planStep    `json:"step"`
	Output  interface{} `json:"output,omitempty"`
	Error   string      `json:"error,omitempty"`
	Success bool        `json:"success"`
}

// executePlanExecuteLoop runs the plan_execute pattern: plan → execute each step.
//
// The loop:
//  1. Call the planner model to produce a JSON array of plan steps.
//  2. For each step, call the executor model (or execute a tool) to complete it.
//  3. Collect step results and return the final output.
func (r EinoADKRunner) executePlanExecuteLoop(
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
			Message: fmt.Sprintf("plan_execute pattern: model %q not declared in spec.models", modelRef),
		}
	}
	modelRuntime, ok := runtimeInfo.Models[modelRef]
	if !ok {
		return contract.WorkerResult{}, false, FailureReasonError{
			Reason:  "MissingModelRuntime",
			Message: fmt.Sprintf("plan_execute pattern: model %q not resolved in runtime", modelRef),
		}
	}
	if strings.TrimSpace(modelRuntime.BaseURL) == "" {
		return contract.WorkerResult{}, false, FailureReasonError{
			Reason:  "MissingModelBaseURL",
			Message: fmt.Sprintf("plan_execute pattern: model %q has no baseURL configured", modelRef),
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

	maxSteps := artifact.Pattern.MaxIterations
	if maxSteps <= 0 {
		maxSteps = defaultPlanExecuteMaxSteps
	}

	// Phase 1: Plan — call the planner model to generate steps.
	// Use InvokeWithBody because the planner returns a JSON array, not an object.
	planPrompt := augmentPlanPrompt(systemPrompt, "plan", maxSteps)
	planReqBody := map[string]interface{}{
		"model": modelConfig.Model,
		"messages": []map[string]interface{}{
			{"role": "system", "content": planPrompt.Template},
			{"role": "user", "content": userMessageFromInput(request.Config.ParsedRunInput)},
		},
	}
	if modelConfig.Temperature != 0 {
		planReqBody["temperature"] = modelConfig.Temperature
	}
	if modelConfig.MaxTokens > 0 {
		planReqBody["max_tokens"] = modelConfig.MaxTokens
	}
	planRawBody, err := json.Marshal(planReqBody)
	if err != nil {
		return contract.WorkerResult{}, false, fmt.Errorf("plan_execute: marshaling plan request: %w", err)
	}
	planResult, err := r.modelInvoker().InvokeWithBody(ctx, modelRuntime, modelConfig, planRawBody, planReqBody)
	if err != nil {
		return contract.WorkerResult{}, false, fmt.Errorf("plan_execute planning phase failed: %w", err)
	}

	// Parse the plan steps from the model response.
	steps, err := parsePlanSteps(planResult.Content, planResult.Parsed)
	if err != nil {
		return contract.WorkerResult{}, false, fmt.Errorf("plan_execute: failed to parse plan steps: %w", err)
	}
	if len(steps) == 0 {
		return contract.WorkerResult{}, false, fmt.Errorf("plan_execute: planner returned empty plan")
	}

	// Cap steps to maxSteps.
	if len(steps) > int(maxSteps) {
		steps = steps[:maxSteps]
	}

	// Phase 2: Execute each step.
	var stepResults []planStepResult
	var lastOutput string

	for i, step := range steps {
		select {
		case <-ctx.Done():
			return contract.WorkerResult{}, false, ctx.Err()
		default:
		}

		// Build execution prompt with context from previous steps.
		execPrompt := augmentPlanPrompt(systemPrompt, "execute", maxSteps)
		stepInput := buildStepInput(request.Config.ParsedRunInput, step, stepResults)

		execResult, execErr := r.modelInvoker().Invoke(ctx, modelRuntime, modelConfig, execPrompt, stepInput, nil)

		result := planStepResult{
			Step: step,
		}
		if execErr != nil {
			result.Error = execErr.Error()
			result.Success = false
		} else {
			result.Output = execResult.Content
			result.Success = true
			lastOutput = execResult.Content
		}
		stepResults = append(stepResults, result)

		// Stop on failure.
		if execErr != nil {
			break
		}

		_ = i // step index available for future use
	}

	// Build final output.
	output := make(map[string]interface{})
	if err := json.Unmarshal([]byte(lastOutput), &output); err != nil {
		output["text"] = lastOutput
	}
	output["pattern"] = "plan_execute"
	output["planSteps"] = len(steps)
	output["executedSteps"] = len(stepResults)

	// Check for failures.
	failedCount := 0
	for _, sr := range stepResults {
		if !sr.Success {
			failedCount++
		}
	}
	if failedCount > 0 {
		output["failedSteps"] = failedCount
	}

	task := taskFromRunInput(request.Config.ParsedRunInput)
	message := fmt.Sprintf("plan_execute pattern completed %d/%d steps for task %q", len(stepResults)-failedCount, len(steps), task)

	runtimeInfo.Runner = "EinoADKRunner"

	artifacts := []contract.WorkerArtifact{
		{Name: "plan-execute-trace", Kind: "json", Inline: map[string]interface{}{"steps": stepResults}},
	}
	artifacts = append(artifacts, modelArtifactsFromResult(planResult)...)

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

// augmentPlanPrompt adds plan or execute phase instructions to the system prompt.
func augmentPlanPrompt(base contract.PromptSpec, phase string, maxSteps int32) contract.PromptSpec {
	var b strings.Builder
	b.WriteString(base.Template)
	b.WriteString("\n\n")

	switch phase {
	case "plan":
		b.WriteString(fmt.Sprintf("You are in the PLANNING phase. Analyze the user's request and create a step-by-step plan.\n"))
		b.WriteString(fmt.Sprintf("Respond with a JSON array of steps (max %d). Each step must have:\n", maxSteps))
		b.WriteString(`- "id": a short identifier (e.g., "step-1", "step-2")
- "description": what this step accomplishes
- "action": the specific action to take

Example: [{"id":"step-1","description":"Analyze the input","action":"extract key facts"},{"id":"step-2","description":"Generate output","action":"produce final answer"}]

Respond ONLY with the JSON array. No other text.`)
	case "execute":
		b.WriteString("You are in the EXECUTION phase. Execute the given step thoroughly and produce a concrete result.\n")
		b.WriteString("Focus only on this step. Provide your output as a JSON object.\n")
		b.WriteString("If the step requires tool use, respond with the appropriate tool call format.")
	}

	return contract.PromptSpec{
		Name:     base.Name,
		Template: b.String(),
		Language: base.Language,
	}
}

// parsePlanSteps extracts plan steps from the model response.
func parsePlanSteps(content string, parsed map[string]interface{}) ([]planStep, error) {
	// Try to extract from parsed output first.
	if stepsRaw, ok := parsed["steps"]; ok {
		if stepsArr, ok := stepsRaw.([]interface{}); ok {
			return convertToSteps(stepsArr)
		}
	}

	// Try direct JSON array parse.
	var steps []planStep
	if err := json.Unmarshal([]byte(content), &steps); err == nil {
		return steps, nil
	}

	// Try extracting JSON array from text.
	if start := strings.Index(content, "["); start >= 0 {
		if end := strings.LastIndex(content, "]"); end > start {
			if err := json.Unmarshal([]byte(content[start:end+1]), &steps); err == nil {
				return steps, nil
			}
		}
	}

	return nil, fmt.Errorf("could not parse plan steps from response: %s", content[:min(len(content), 200)])
}

// convertToSteps converts raw interface slice to planStep slice.
func convertToSteps(raw []interface{}) ([]planStep, error) {
	var steps []planStep
	for _, item := range raw {
		itemMap, ok := item.(map[string]interface{})
		if !ok {
			continue
		}
		step := planStep{
			ID:          getStringField(itemMap, "id"),
			Description: getStringField(itemMap, "description"),
			Action:      getStringField(itemMap, "action"),
		}
		if step.ID == "" && step.Description == "" {
			continue
		}
		steps = append(steps, step)
	}
	return steps, nil
}

// buildStepInput builds the input for executing a single plan step.
func buildStepInput(originalInput map[string]interface{}, step planStep, previousResults []planStepResult) map[string]interface{} {
	input := make(map[string]interface{}, len(originalInput)+3)
	for k, v := range originalInput {
		input[k] = v
	}
	input["_plan_step"] = map[string]interface{}{
		"id":          step.ID,
		"description": step.Description,
		"action":      step.Action,
	}
	if len(previousResults) > 0 {
		var prevOutputs []map[string]interface{}
		for _, pr := range previousResults {
			prevOutputs = append(prevOutputs, map[string]interface{}{
				"step_id": pr.Step.ID,
				"success": pr.Success,
				"output":  pr.Output,
			})
		}
		input["_previous_steps"] = prevOutputs
	}
	return input
}
