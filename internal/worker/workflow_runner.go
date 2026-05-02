package worker

import (
	"context"

	"github.com/surefire-ai/korus/internal/contract"
)

// isWorkflowPattern checks if the compiled artifact uses the workflow pattern.
func isWorkflowPattern(artifact contract.CompiledArtifact) bool {
	if artifact.Pattern.Type == "workflow" {
		return true
	}
	if p, ok := artifact.Runner.Pattern["type"].(string); ok && p == "workflow" {
		return true
	}
	return false
}

// executeWorkflow runs the workflow pattern: deterministic graph execution.
//
// The workflow pattern is syntactic sugar over the graph execution engine.
// When spec.pattern.type == "workflow", the compiler copies spec.graph into
// the runner artifact. At runtime, this handler delegates to tryGraphExecution.
//
// Returns (result, true, nil) if the graph was executed, (zero, false, nil)
// if no graph is available (should not happen for a valid workflow pattern).
func (r EinoADKRunner) executeWorkflow(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
) (contract.WorkerResult, bool, error) {
	// Workflow pattern delegates entirely to the graph execution engine.
	// The graph should have been compiled into the artifact by the compiler.
	return r.tryGraphExecution(ctx, request, runtimeInfo)
}
