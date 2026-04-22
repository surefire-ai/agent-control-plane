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
	Auto     bool
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
	if step.Auto {
		output, artifacts, err := r.executeAutoStepSequence(ctx, request, runtimeInfo, step)
		return output, artifacts, true, err
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

func (r EinoADKPlaceholderRunner) executeAutoStepSequence(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, step RequestedStep) (map[string]interface{}, []contract.WorkerArtifact, error) {
	edges, ok := request.Artifact.Runner.Graph["edges"].([]interface{})
	if !ok || len(edges) == 0 {
		return nil, nil, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: "graph does not expose edges for automatic execution",
		}
	}

	state := initialStepState(request.Config.ParsedRunInput, step.Input)
	sequence := make([]string, 0, len(edges))
	steps := make([]map[string]interface{}, 0, len(edges))
	artifacts := make([]contract.WorkerArtifact, 0, len(edges))
	current := "START"
	visited := map[string]struct{}{}

	for stepCount := 0; stepCount < len(edges)+1; stepCount++ {
		next, done, err := nextAutoEdge(edges, current, request.Config.ParsedRunInput, state)
		if err != nil {
			return nil, nil, err
		}
		if done {
			result := map[string]interface{}{
				"sequence":   sequence,
				"steps":      steps,
				"finalState": state,
			}
			if len(sequence) > 0 {
				result["currentNode"] = sequence[len(sequence)-1]
			}
			return result, artifacts, nil
		}
		if _, seen := visited[next]; seen {
			return nil, nil, FailureReasonError{
				Reason:  "InvalidStepRequest",
				Message: fmt.Sprintf("automatic graph execution detected a cycle at %q", next),
			}
		}
		visited[next] = struct{}{}
		sequence = append(sequence, next)
		nodeStep := RequestedStep{Node: next, Input: state}
		output, nodeArtifacts, _, err := r.executeGraphStepNode(ctx, request, runtimeInfo, nodeStep)
		if err != nil {
			return nil, nil, err
		}
		steps = append(steps, output)
		artifacts = append(artifacts, nodeArtifacts...)
		state = mergeStepState(state, output)
		current = next
	}

	return nil, nil, FailureReasonError{
		Reason:  "InvalidStepRequest",
		Message: "automatic graph execution exceeded the edge walk limit",
	}
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
	case "function":
		output, artifacts, err := r.executeStepFunction(ctx, request, runtimeInfo, step, node)
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
	auto, _ := raw["auto"].(bool)
	if strings.TrimSpace(node) == "" && len(sequence) == 0 && !auto {
		return RequestedStep{}, false, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: "step.node, step.sequence, or step.auto is required",
		}
	}
	stepInput, _ := raw["input"].(map[string]interface{})
	if stepInput == nil {
		stepInput = map[string]interface{}{}
	}
	return RequestedStep{Node: node, Sequence: sequence, Auto: auto, Input: stepInput}, true, nil
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

func (r EinoADKPlaceholderRunner) executeStepFunction(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, step RequestedStep, node map[string]interface{}) (map[string]interface{}, []contract.WorkerArtifact, error) {
	_ = ctx
	_ = runtimeInfo
	implementation, _ := node["implementation"].(string)
	if strings.TrimSpace(implementation) == "" {
		return nil, nil, FailureReasonError{
			Reason:  "UnsupportedGraphNode",
			Message: fmt.Sprintf("graph node %q is missing implementation", step.Node),
		}
	}
	stepInput := initialStepState(request.Config.ParsedRunInput, step.Input)
	bindingName, declaredSkill, err := resolveDeclaredSkillFunction(request.Artifact, implementation)
	if err != nil {
		return nil, nil, err
	}
	skillName, functionName, fn, err := resolveBuiltinSkillFunction(implementation)
	if err != nil {
		return nil, nil, err
	}
	output := fn(stepInput)
	return map[string]interface{}{
			"node":           step.Node,
			"kind":           "function",
			"implementation": implementation,
			"skillBinding":   bindingName,
			"skillRef":       declaredSkill.Ref,
			"skill":          skillName,
			"function":       functionName,
			"input":          stepInput,
			"result":         output,
		}, []contract.WorkerArtifact{
			{Name: "step-function-result", Kind: "json", Inline: map[string]interface{}{"node": step.Node, "implementation": implementation, "skillBinding": bindingName, "skillRef": declaredSkill.Ref, "skill": skillName, "function": functionName, "result": output}},
		}, nil
}

func scoreRiskByMatrix(state map[string]interface{}) map[string]interface{} {
	hazards := hazardsFromState(state)
	overall := highestRiskLevel(hazards)
	if overall == "" {
		if existing, _ := state["overallRiskLevel"].(string); strings.TrimSpace(existing) != "" {
			overall = existing
		} else {
			overall = "low"
		}
	}
	return map[string]interface{}{
		"overallRiskLevel": overall,
		"hazards":          hazards,
		"riskScored":       true,
	}
}

func hazardsFromState(state map[string]interface{}) []interface{} {
	if hazards, ok := state["hazards"].([]interface{}); ok {
		return hazards
	}
	if identifyHazards, ok := state["identify_hazards"].(map[string]interface{}); ok {
		if result, ok := identifyHazards["result"].(map[string]interface{}); ok {
			if hazards, ok := result["hazards"].([]interface{}); ok {
				return hazards
			}
		}
	}
	return nil
}

func highestRiskLevel(hazards []interface{}) string {
	best := ""
	bestRank := -1
	for _, raw := range hazards {
		hazard, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		level, _ := hazard["riskLevel"].(string)
		rank := riskRank(level)
		if rank > bestRank {
			best = level
			bestRank = rank
		}
	}
	return best
}

func riskRank(level string) int {
	switch strings.TrimSpace(level) {
	case "critical":
		return 3
	case "high":
		return 2
	case "medium":
		return 1
	case "low":
		return 0
	default:
		return -1
	}
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

func nextAutoEdge(edges []interface{}, from string, runInput map[string]interface{}, state map[string]interface{}) (string, bool, error) {
	var matchedConditional []string
	var matchedDefault []string
	for _, raw := range edges {
		edge, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		edgeFrom, _ := edge["from"].(string)
		if edgeFrom != from {
			continue
		}
		to, _ := edge["to"].(string)
		if strings.TrimSpace(to) == "" {
			continue
		}
		when, _ := edge["when"].(string)
		if strings.TrimSpace(when) == "" {
			matchedDefault = append(matchedDefault, to)
			continue
		}
		match, err := edgeConditionMatches(when, runInput, state)
		if err != nil {
			return "", false, err
		}
		if match {
			matchedConditional = append(matchedConditional, to)
		}
	}

	selected := matchedConditional
	if len(selected) == 0 {
		selected = matchedDefault
	}
	if len(selected) == 0 {
		return "", false, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: fmt.Sprintf("automatic graph execution found no eligible edge from %q", from),
		}
	}
	if len(selected) > 1 {
		return "", false, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: fmt.Sprintf("automatic graph execution found multiple eligible edges from %q", from),
		}
	}
	if selected[0] == "END" {
		return "", true, nil
	}
	return selected[0], false, nil
}

func edgeConditionMatches(expression string, runInput map[string]interface{}, state map[string]interface{}) (bool, error) {
	clauses := strings.Split(expression, "&&")
	for _, clause := range clauses {
		match, err := evaluateEdgeClause(strings.TrimSpace(clause), runInput, state)
		if err != nil {
			return false, err
		}
		if !match {
			return false, nil
		}
	}
	return true, nil
}

func evaluateEdgeClause(clause string, runInput map[string]interface{}, state map[string]interface{}) (bool, error) {
	switch {
	case strings.Contains(clause, " != null"):
		path := strings.TrimSpace(strings.TrimSuffix(clause, " != null"))
		value, ok := resolveConditionPath(path, runInput, state)
		return ok && value != nil, nil
	case strings.HasPrefix(clause, "len(") && strings.Contains(clause, ") > "):
		parts := strings.SplitN(clause, ") > ", 2)
		path := strings.TrimPrefix(parts[0], "len(")
		value, ok := resolveConditionPath(strings.TrimSpace(path), runInput, state)
		if !ok || value == nil {
			return false, nil
		}
		minimum := strings.TrimSpace(parts[1])
		switch minimum {
		case "0":
			switch typed := value.(type) {
			case []interface{}:
				return len(typed) > 0, nil
			case []string:
				return len(typed) > 0, nil
			}
		}
		return false, nil
	case strings.Contains(clause, " in ["):
		parts := strings.SplitN(clause, " in ", 2)
		path := strings.TrimSpace(parts[0])
		value, ok := resolveConditionPath(path, runInput, state)
		if !ok {
			return false, nil
		}
		target, _ := value.(string)
		options := parseStringList(parts[1])
		for _, option := range options {
			if target == option {
				return true, nil
			}
		}
		return false, nil
	default:
		return false, FailureReasonError{
			Reason:  "InvalidStepRequest",
			Message: fmt.Sprintf("unsupported edge condition %q", clause),
		}
	}
}

func resolveConditionPath(path string, runInput map[string]interface{}, state map[string]interface{}) (interface{}, bool) {
	context := cloneMap(state)
	context["input"] = runInput
	parts := strings.Split(path, ".")
	var current interface{} = context
	for _, part := range parts {
		values, ok := current.(map[string]interface{})
		if !ok {
			return nil, false
		}
		current, ok = values[part]
		if !ok {
			return nil, false
		}
	}
	return current, true
}

func parseStringList(raw string) []string {
	trimmed := strings.TrimSpace(raw)
	trimmed = strings.TrimPrefix(trimmed, "[")
	trimmed = strings.TrimSuffix(trimmed, "]")
	if strings.TrimSpace(trimmed) == "" {
		return nil
	}
	parts := strings.Split(trimmed, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		value = strings.Trim(value, "'")
		value = strings.Trim(value, "\"")
		if value != "" {
			values = append(values, value)
		}
	}
	return values
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
		if nodeName == "review_output" {
			merged["finalResponse"] = result
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
