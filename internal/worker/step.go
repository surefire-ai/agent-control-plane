package worker

import (
	"context"
	"fmt"
	"strings"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type RequestedStep struct {
	Node  string
	Input map[string]interface{}
}

func (r EinoADKPlaceholderRunner) invokeRequestedStep(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (map[string]interface{}, []contract.WorkerArtifact, bool, error) {
	step, ok, err := requestedStep(request.Config.ParsedRunInput)
	if err != nil {
		return nil, nil, false, err
	}
	if !ok {
		return nil, nil, false, nil
	}

	node, err := graphNodeByName(request.Artifact, step.Node)
	if err != nil {
		return nil, nil, false, err
	}

	kind, _ := node["kind"].(string)
	switch kind {
	case "tool":
		output, artifacts, err := r.executeStepTool(ctx, request, runtimeInfo, step, node)
		return output, artifacts, true, err
	case "retrieval":
		output, artifacts, err := r.executeStepRetrieval(ctx, request, runtimeInfo, step, node)
		return output, artifacts, true, err
	default:
		return nil, nil, false, FailureReasonError{
			Reason:  "UnsupportedGraphNode",
			Message: fmt.Sprintf("graph step %q with kind %q is not supported yet", step.Node, kind),
		}
	}
}

func requestedStep(input map[string]interface{}) (RequestedStep, bool, error) {
	value, ok := input["step"]
	if !ok {
		return RequestedStep{}, false, nil
	}
	raw, ok := value.(map[string]interface{})
	if !ok {
		return RequestedStep{}, false, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: "step must be a JSON object",
		}
	}
	node, _ := raw["node"].(string)
	if strings.TrimSpace(node) == "" {
		return RequestedStep{}, false, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: "step.node is required",
		}
	}
	stepInput, _ := raw["input"].(map[string]interface{})
	if stepInput == nil {
		stepInput = map[string]interface{}{}
	}
	return RequestedStep{Node: node, Input: stepInput}, true, nil
}

func graphNodeByName(artifact contract.CompiledArtifact, nodeName string) (map[string]interface{}, error) {
	nodes, ok := artifact.Runner.Graph["nodes"].([]interface{})
	if !ok {
		return nil, FailureReasonError{
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
		if name == nodeName {
			return node, nil
		}
	}
	return nil, FailureReasonError{
		Reason:  "UnknownGraphNode",
		Message: fmt.Sprintf("graph node %q was not found", nodeName),
	}
}

func (r EinoADKPlaceholderRunner) executeStepTool(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, step RequestedStep, node map[string]interface{}) (map[string]interface{}, []contract.WorkerArtifact, error) {
	call := RequestedToolCall{
		Node:  step.Node,
		Input: step.Input,
	}
	if toolRef, _ := node["toolRef"].(string); strings.TrimSpace(toolRef) != "" {
		call.Name = toolRef
	}
	result, ok, err := r.executeToolCall(ctx, request, runtimeInfo, call)
	if err != nil || !ok {
		return nil, nil, err
	}
	return result.Output, result.Artifacts, nil
}

func (r EinoADKPlaceholderRunner) executeStepRetrieval(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, step RequestedStep, node map[string]interface{}) (map[string]interface{}, []contract.WorkerArtifact, error) {
	call := RequestedRetrievalCall{
		Node: step.Node,
	}
	if knowledgeRef, _ := node["knowledgeRef"].(string); strings.TrimSpace(knowledgeRef) != "" {
		call.Name = knowledgeRef
	}
	if query, _ := step.Input["query"].(string); strings.TrimSpace(query) != "" {
		call.Query = query
	}
	switch value := step.Input["topK"].(type) {
	case int:
		call.TopK = value
	case int32:
		call.TopK = int(value)
	case int64:
		call.TopK = int(value)
	case float64:
		call.TopK = int(value)
	}
	result, ok, err := r.executeRetrievalCall(ctx, request, runtimeInfo, call)
	if err != nil || !ok {
		return nil, nil, err
	}
	return result.Output, result.Artifacts, nil
}
