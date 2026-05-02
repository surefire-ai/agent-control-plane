package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/surefire-ai/korus/internal/contract"
)

const (
	defaultReflectionMaxIterations = 3
)

// isReflectionPattern checks if the compiled artifact uses the reflection pattern.
func isReflectionPattern(artifact contract.CompiledArtifact) bool {
	if artifact.Pattern.Type == "reflection" {
		return true
	}
	if p, ok := artifact.Runner.Pattern["type"].(string); ok && p == "reflection" {
		return true
	}
	return false
}

// reflectionStep records one iteration of the reflection loop.
type reflectionStep struct {
	Iteration int    `json:"iteration"`
	Phase     string `json:"phase"`
	Content   string `json:"content"`
}

// executeReflectionLoop runs the generate→critique→revise loop.
//
// The loop:
//  1. Generate: model produces initial output.
//  2. Critique: model evaluates the output and provides feedback.
//  3. Revise: model improves the output based on feedback.
//  4. Repeat critique→revise up to maxIterations.
func (r EinoADKRunner) executeReflectionLoop(
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
	modelCfg, ok := runtimeInfo.Models[modelRef]
	if !ok {
		return contract.WorkerResult{}, false, fmt.Errorf("reflection model %q not found in runtime", modelRef)
	}

	maxIter := artifact.Pattern.MaxIterations
	if maxIter <= 0 {
		maxIter = defaultReflectionMaxIterations
	}

	systemPrompt := artifact.Runner.Prompts["system"]
	if strings.TrimSpace(systemPrompt.Template) == "" {
		return contract.WorkerResult{}, false, FailureReasonError{
			Reason:  "MissingPrompt",
			Message: "system prompt is empty in compiled artifact",
		}
	}

	// Phase 1: Generate initial output.
	generatePrompt := augmentPromptForPhase(systemPrompt, "generate", "")
	genResult, err := r.modelInvoker().Invoke(ctx, modelCfg, modelConfigFor(artifact, modelRef), generatePrompt, request.Config.ParsedRunInput, artifact.Runner.Output)
	if err != nil {
		return contract.WorkerResult{}, false, fmt.Errorf("reflection generate phase failed: %w", err)
	}

	var steps []reflectionStep
	steps = append(steps, reflectionStep{
		Iteration: 1,
		Phase:     "generate",
		Content:   genResult.Content,
	})

	currentOutput := genResult.Content

	// Phase 2..N: Critique → Revise loop.
	for iter := int32(1); iter <= maxIter; iter++ {
		select {
		case <-ctx.Done():
			return contract.WorkerResult{}, false, ctx.Err()
		default:
		}

		// Critique.
		critiquePrompt := augmentPromptForPhase(systemPrompt, "critique", currentOutput)
		critResult, err := r.modelInvoker().Invoke(ctx, modelCfg, modelConfigFor(artifact, modelRef), critiquePrompt, request.Config.ParsedRunInput, nil)
		if err != nil {
			return contract.WorkerResult{}, false, fmt.Errorf("reflection critique phase failed at iteration %d: %w", iter, err)
		}

		steps = append(steps, reflectionStep{
			Iteration: int(iter),
			Phase:     "critique",
			Content:   critResult.Content,
		})

		// Check if critique says output is good enough.
		if isCritiquePositive(critResult.Content) {
			break
		}

		// Revise.
		revisePrompt := augmentPromptForPhase(systemPrompt, "revise", currentOutput+"\n\nCritique: "+critResult.Content)
		revResult, err := r.modelInvoker().Invoke(ctx, modelCfg, modelConfigFor(artifact, modelRef), revisePrompt, request.Config.ParsedRunInput, artifact.Runner.Output)
		if err != nil {
			return contract.WorkerResult{}, false, fmt.Errorf("reflection revise phase failed at iteration %d: %w", iter, err)
		}

		steps = append(steps, reflectionStep{
			Iteration: int(iter),
			Phase:     "revise",
			Content:   revResult.Content,
		})

		currentOutput = revResult.Content
	}

	// Parse final output as JSON if possible.
	output := make(map[string]interface{})
	if err := json.Unmarshal([]byte(currentOutput), &output); err != nil {
		output["text"] = currentOutput
	}
	output["pattern"] = "reflection"
	output["iterations"] = len(steps)

	task := taskFromRunInput(request.Config.ParsedRunInput)
	message := fmt.Sprintf("reflection pattern completed %d steps for task %q", len(steps), task)

	// Build artifacts.
	artifacts := []contract.WorkerArtifact{
		{Name: "reflection-trace", Kind: "json", Inline: map[string]interface{}{"steps": steps}},
	}
	artifacts = append(artifacts, modelArtifacts(genResult)...)

	runtimeInfo.Runner = "EinoADKRunner"

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

// augmentPromptForPhase adds phase-specific instructions to the system prompt.
func augmentPromptForPhase(base contract.PromptSpec, phase string, previousOutput string) contract.PromptSpec {
	var b strings.Builder
	b.WriteString(base.Template)
	b.WriteString("\n\n")

	switch phase {
	case "generate":
		b.WriteString("Generate your best output based on the user's input. Be thorough and precise.")
	case "critique":
		b.WriteString("Review the following output critically. Identify specific issues, errors, or areas for improvement.\n\nOutput to review:\n")
		b.WriteString(previousOutput)
		b.WriteString("\n\nIf the output is already excellent and needs no changes, respond with: {\"status\": \"approved\", \"feedback\": \"No changes needed.\"}")
		b.WriteString("\nOtherwise, respond with: {\"status\": \"needs_revision\", \"feedback\": \"...\"}")
	case "revise":
		b.WriteString("Revise the following output based on the critique. Improve quality, fix errors, and address all feedback.\n\nCurrent output:\n")
		b.WriteString(previousOutput)
	}

	return contract.PromptSpec{
		Name:     base.Name,
		Template: b.String(),
		Language: base.Language,
	}
}

// isCritiquePositive checks if the critique approved the output.
func isCritiquePositive(content string) bool {
	var parsed struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		return false
	}
	return strings.TrimSpace(parsed.Status) == "approved"
}

// modelArtifacts extracts request/response artifacts from a model invocation.
func modelArtifacts(result ModelInvocationResult) []contract.WorkerArtifact {
	if result.RequestBody == nil && result.ResponseBody == nil {
		return nil
	}
	return []contract.WorkerArtifact{
		{Name: "chat-completion-request", Kind: "json", Inline: result.RequestBody},
		{Name: "chat-completion-response", Kind: "json", Inline: result.ResponseBody},
	}
}
