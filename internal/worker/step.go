package worker

import (
	"context"
	"fmt"
	"strings"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type RequestedStep struct {
	Node     string
	Sequence []string
	Input    map[string]interface{}
}

func (r EinoADKPlaceholderRunner) invokeRequestedStep(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (map[string]interface{}, []contract.WorkerArtifact, bool, error) {
	step, ok, err := requestedStep(request.Config.ParsedRunInput)
	if err != nil {
		return nil, nil, false, err
	}
	if !ok {
		return nil, nil, false, nil
	}
	if len(step.Sequence) > 0 {
		output, artifacts, err := r.executeStepSequence(ctx, request, runtimeInfo, step)
		return output, artifacts, true, err
	}
	return r.executeGraphStepNode(ctx, request, runtimeInfo, step)
}

func (r EinoADKPlaceholderRunner) executeStepSequence(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, step RequestedStep) (map[string]interface{}, []contract.WorkerArtifact, error) {
	state := initialStepState(request.Config.ParsedRunInput, step.Input)
	steps := make([]map[string]interface{}, 0, len(step.Sequence))
	artifacts := make([]contract.WorkerArtifact, 0, len(step.Sequence))
	for _, nodeName := range step.Sequence {
		nodeStep := RequestedStep{
			Node:  nodeName,
			Input: state,
		}
		output, nodeArtifacts, _, err := r.executeGraphStepNode(ctx, request, runtimeInfo, nodeStep)
		if err != nil {
			return nil, nil, err
		}
		steps = append(steps, output)
		artifacts = append(artifacts, nodeArtifacts...)
		state = mergeStepState(state, output)
	}

	result := map[string]interface{}{
		"sequence":   step.Sequence,
		"steps":      steps,
		"finalState": state,
	}
	if len(step.Sequence) > 0 {
		result["currentNode"] = step.Sequence[len(step.Sequence)-1]
	}
	return result, artifacts, nil
}

func (r EinoADKPlaceholderRunner) executeGraphStepNode(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, step RequestedStep) (map[string]interface{}, []contract.WorkerArtifact, bool, error) {
	node, err := graphNodeByName(request.Artifact, step.Node)
	if err != nil {
		return nil, nil, false, err
	}

	kind, _ := node["kind"].(string)
	switch kind {
	case "llm":
		output, artifacts, err := r.executeStepLLM(ctx, request, runtimeInfo, step, node)
		return output, artifacts, true, err
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

func (r EinoADKPlaceholderRunner) executeStepLLM(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, step RequestedStep, node map[string]interface{}) (map[string]interface{}, []contract.WorkerArtifact, error) {
	modelName, _ := node["modelRef"].(string)
	if strings.TrimSpace(modelName) == "" {
		return nil, nil, FailureReasonError{
			Reason:  "UnknownModel",
			Message: fmt.Sprintf("graph node %q is missing modelRef", step.Node),
		}
	}
	modelConfig, ok := preferredModelConfig(modelName, request.Artifact)
	if !ok {
		return nil, nil, FailureReasonError{
			Reason:  "UnknownModel",
			Message: fmt.Sprintf("unknown model %q for graph node %q", modelName, step.Node),
		}
	}
	modelRuntime, ok := runtimeInfo.Models[modelName]
	if !ok {
		return nil, nil, FailureReasonError{
			Reason:  "UnknownModel",
			Message: fmt.Sprintf("model runtime binding missing for %q", modelName),
		}
	}
	systemPrompt := request.Artifact.Runner.Prompts["system"]
	if strings.TrimSpace(systemPrompt.Template) == "" {
		return nil, nil, FailureReasonError{
			Reason:  "ModelRequestBuildFailed",
			Message: fmt.Sprintf("graph node %q has no usable system prompt", step.Node),
		}
	}
	modelInput := initialStepState(request.Config.ParsedRunInput, step.Input)
	result, err := r.modelInvoker().Invoke(ctx, modelRuntime, modelConfig, systemPrompt, modelInput, request.Artifact.Runner.Output)
	if err != nil {
		return nil, nil, err
	}
	return map[string]interface{}{
			"node":     step.Node,
			"kind":     "llm",
			"model":    modelName,
			"input":    modelInput,
			"result":   result.Parsed,
			"response": result.Content,
		}, []contract.WorkerArtifact{
			{Name: "step-chat-completion-request", Kind: "json", Inline: map[string]interface{}{"node": step.Node, "model": modelName, "request": result.RequestBody}},
			{Name: "step-chat-completion-response", Kind: "json", Inline: map[string]interface{}{"node": step.Node, "model": modelName, "response": result.ResponseBody}},
		}, nil
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
	sequence, err := stringSequence(raw["sequence"])
	if err != nil {
		return RequestedStep{}, false, err
	}
	if strings.TrimSpace(node) == "" && len(sequence) == 0 {
		return RequestedStep{}, false, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: "step.node or step.sequence is required",
		}
	}
	stepInput, _ := raw["input"].(map[string]interface{})
	if stepInput == nil {
		stepInput = map[string]interface{}{}
	}
	return RequestedStep{Node: node, Sequence: sequence, Input: stepInput}, true, nil
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
		Input: initialStepState(request.Config.ParsedRunInput, step.Input),
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
	stepInput := initialStepState(request.Config.ParsedRunInput, step.Input)
	call := RequestedRetrievalCall{
		Node: step.Node,
	}
	if knowledgeRef, _ := node["knowledgeRef"].(string); strings.TrimSpace(knowledgeRef) != "" {
		call.Name = knowledgeRef
	}
	if query, _ := stepInput["query"].(string); strings.TrimSpace(query) != "" {
		call.Query = query
	}
	switch value := stepInput["topK"].(type) {
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

func stringSequence(value interface{}) ([]string, error) {
	if value == nil {
		return nil, nil
	}
	raw, ok := value.([]interface{})
	if !ok {
		return nil, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: "step.sequence must be an array of node names",
		}
	}
	sequence := make([]string, 0, len(raw))
	for _, item := range raw {
		name, _ := item.(string)
		if strings.TrimSpace(name) == "" {
			return nil, FailureReasonError{
				Reason:  "InvalidStepRequest",
				Message: "step.sequence must contain non-empty node names",
			}
		}
		sequence = append(sequence, name)
	}
	return sequence, nil
}

func initialStepState(runInput map[string]interface{}, explicit map[string]interface{}) map[string]interface{} {
	if len(explicit) > 0 {
		return cloneMap(explicit)
	}
	state := cloneMap(runInput)
	delete(state, "step")
	return state
}

func mergeStepState(state map[string]interface{}, output map[string]interface{}) map[string]interface{} {
	merged := cloneMap(state)
	nodeName, _ := output["node"].(string)
	if nodeName != "" {
		merged[nodeName] = output
	}
	merged["lastNode"] = nodeName
	merged["lastOutput"] = output

	if result, _ := output["result"].(map[string]interface{}); len(result) > 0 {
		for key, value := range result {
			merged[key] = value
		}
	}
	if toolOutput, _ := output["output"].(map[string]interface{}); len(toolOutput) > 0 {
		merged["toolOutput"] = toolOutput
	}
	if results, ok := output["results"]; ok {
		merged["retrievalResults"] = results
	}
	return merged
}

func cloneMap(input map[string]interface{}) map[string]interface{} {
	if len(input) == 0 {
		return map[string]interface{}{}
	}
	cloned := make(map[string]interface{}, len(input))
	for key, value := range input {
		cloned[key] = value
	}
	return cloned
}
