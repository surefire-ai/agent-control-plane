package worker

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/compose"

	"github.com/surefire-ai/korus/internal/contract"
)

// graphState is the shared state type flowing through the Eino graph.
// Each node receives the current state, processes it, and returns the
// updated state.
type graphState = map[string]interface{}

// buildGraph converts a compiled artifact's graph definition into an Eino
// compose.Graph. It maps artifact node kinds (llm, tool, retrieval, function)
// to Eino lambda nodes and wires edges including conditional branches.
//
// The graph input and output are both graphState (map[string]interface{}).
func buildGraph(
	ctx context.Context,
	artifact contract.CompiledArtifact,
	runtimeInfo contract.WorkerRuntimeInfo,
	modelInvoker ModelInvoker,
	toolInvoker ToolInvoker,
	retrievalInvoker RetrievalInvoker,
) (compose.Runnable[graphState, graphState], error) {
	graphDef := artifact.Runner.Graph
	if len(graphDef) == 0 {
		return nil, nil
	}

	nodesRaw, ok := graphDef["nodes"].([]interface{})
	if !ok || len(nodesRaw) == 0 {
		return nil, nil
	}

	edgesRaw, _ := graphDef["edges"].([]interface{})
	stateSchema, _ := graphDef["stateSchema"].(map[string]interface{})
	_ = stateSchema

	g := compose.NewGraph[graphState, graphState]()

	// Add a passthrough START node so we can branch from it.
	// The compose.START constant is the built-in entry point.

	// Register each artifact node as an Eino lambda node.
	for _, raw := range nodesRaw {
		node, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := node["name"].(string)
		if strings.TrimSpace(name) == "" {
			continue
		}

		kind, _ := node["kind"].(string)
		lambda, err := buildNodeLambda(kind, node, artifact, runtimeInfo, modelInvoker, toolInvoker, retrievalInvoker)
		if err != nil {
			return nil, fmt.Errorf("building lambda for node %q: %w", name, err)
		}
		if lambda == nil {
			// Skip unsupported node kinds with a passthrough.
			lambda = compose.InvokableLambda(func(ctx context.Context, state graphState) (graphState, error) {
				out := copyState(state)
				out[name] = map[string]interface{}{
					"skipped": true,
					"kind":    kind,
					"reason":  fmt.Sprintf("node kind %q not yet wired", kind),
				}
				return out, nil
			})
		}

		if err := g.AddLambdaNode(name, lambda); err != nil {
			return nil, fmt.Errorf("adding node %q: %w", name, err)
		}
	}

	// Wire edges.
	if err := wireEdges(g, edgesRaw); err != nil {
		return nil, fmt.Errorf("wiring edges: %w", err)
	}

	// Auto-wire START/END if edges don't reference them.
	if err := autoWireEdges(g, nodesRaw, edgesRaw); err != nil {
		return nil, fmt.Errorf("auto-wiring edges: %w", err)
	}

	// Compile the graph.
	runner, err := g.Compile(ctx)
	if err != nil {
		return nil, fmt.Errorf("compiling graph: %w", err)
	}
	return runner, nil
}

// buildNodeLambda creates an Eino Lambda for a single artifact graph node.
func buildNodeLambda(
	kind string,
	node map[string]interface{},
	artifact contract.CompiledArtifact,
	runtimeInfo contract.WorkerRuntimeInfo,
	modelInvoker ModelInvoker,
	toolInvoker ToolInvoker,
	retrievalInvoker RetrievalInvoker,
) (*compose.Lambda, error) {
	switch kind {
	case "llm":
		return buildLLMLambda(node, artifact, runtimeInfo, modelInvoker)
	case "tool":
		return buildToolLambda(node, artifact, runtimeInfo, toolInvoker)
	case "retrieval":
		return buildRetrievalLambda(node, artifact, runtimeInfo, retrievalInvoker)
	case "function":
		return buildFunctionLambda(node, artifact)
	default:
		return nil, nil
	}
}

// buildLLMLambda creates a Lambda that invokes a model for an LLM node.
func buildLLMLambda(
	node map[string]interface{},
	artifact contract.CompiledArtifact,
	runtimeInfo contract.WorkerRuntimeInfo,
	modelInvoker ModelInvoker,
) (*compose.Lambda, error) {
	nodeName, _ := node["name"].(string)
	modelName, _ := node["modelRef"].(string)
	if strings.TrimSpace(modelName) == "" {
		return nil, fmt.Errorf("llm node %q missing modelRef", nodeName)
	}

	promptRef, _ := node["promptRef"].(string)
	if strings.TrimSpace(promptRef) == "" {
		promptRef = "system"
	}

	return compose.InvokableLambda(func(ctx context.Context, state graphState) (graphState, error) {
		modelConfig, ok := preferredModelConfig(modelName, artifact)
		if !ok {
			return nil, FailureReasonError{
				Reason:  "UnknownModel",
				Message: fmt.Sprintf("unknown model %q for node %q", modelName, nodeName),
			}
		}
		modelRuntime, ok := runtimeInfo.Models[modelName]
		if !ok {
			return nil, FailureReasonError{
				Reason:  "UnknownModel",
				Message: fmt.Sprintf("model runtime binding missing for %q", modelName),
			}
		}
		if strings.TrimSpace(modelRuntime.BaseURL) == "" {
			return nil, FailureReasonError{
				Reason:  "MissingModelConfig",
				Message: fmt.Sprintf("model %q has no base URL configured", modelName),
			}
		}

		systemPrompt := artifact.Runner.Prompts[promptRef]
		if strings.TrimSpace(systemPrompt.Template) == "" {
			return nil, FailureReasonError{
				Reason:  "MissingPrompt",
				Message: fmt.Sprintf("prompt %q is empty for node %q", promptRef, nodeName),
			}
		}

		result, err := modelInvoker.Invoke(ctx, modelRuntime, modelConfig, systemPrompt, state, artifact.Runner.Output)
		if err != nil {
			return nil, err
		}

		out := copyState(state)
		out[nodeName] = map[string]interface{}{
			"kind":     "llm",
			"model":    modelName,
			"response": result.Content,
			"result":   result.Parsed,
		}
		// Merge parsed result into top-level state for downstream nodes.
		for k, v := range result.Parsed {
			out[k] = v
		}
		return out, nil
	}), nil
}

// buildToolLambda creates a Lambda that invokes a tool for a tool node.
func buildToolLambda(
	node map[string]interface{},
	artifact contract.CompiledArtifact,
	runtimeInfo contract.WorkerRuntimeInfo,
	toolInvoker ToolInvoker,
) (*compose.Lambda, error) {
	nodeName, _ := node["name"].(string)
	toolRef, _ := node["toolRef"].(string)
	if strings.TrimSpace(toolRef) == "" {
		return nil, fmt.Errorf("tool node %q missing toolRef", nodeName)
	}

	return compose.InvokableLambda(func(ctx context.Context, state graphState) (graphState, error) {
		spec, ok := artifact.Runner.Tools[toolRef]
		if !ok {
			return nil, FailureReasonError{
				Reason:  "UnknownTool",
				Message: fmt.Sprintf("unknown tool %q for node %q", toolRef, nodeName),
			}
		}
		runtime, ok := runtimeInfo.Tools[toolRef]
		if !ok {
			return nil, FailureReasonError{
				Reason:  "UnknownTool",
				Message: fmt.Sprintf("tool runtime binding missing for %q", toolRef),
			}
		}
		spec.Name = toolRef

		// Build tool input from state.
		toolInput := make(map[string]interface{})
		if inputMapping, ok := node["inputMapping"].(map[string]interface{}); ok {
			for targetKey, sourceKey := range inputMapping {
				if sk, ok := sourceKey.(string); ok {
					if val, exists := state[sk]; exists {
						toolInput[targetKey] = val
					}
				}
			}
		}
		if len(toolInput) == 0 {
			toolInput = state
		}

		result, err := toolInvoker.Invoke(ctx, runtime, spec, toolInput)
		if err != nil {
			return nil, err
		}

		out := copyState(state)
		out[nodeName] = map[string]interface{}{
			"kind":   "tool",
			"tool":   toolRef,
			"input":  toolInput,
			"output": result.Output,
		}
		return out, nil
	}), nil
}

// buildRetrievalLambda creates a Lambda that invokes a retriever for a retrieval node.
func buildRetrievalLambda(
	node map[string]interface{},
	artifact contract.CompiledArtifact,
	runtimeInfo contract.WorkerRuntimeInfo,
	retrievalInvoker RetrievalInvoker,
) (*compose.Lambda, error) {
	nodeName, _ := node["name"].(string)
	knowledgeRef, _ := node["knowledgeRef"].(string)
	if strings.TrimSpace(knowledgeRef) == "" {
		return nil, fmt.Errorf("retrieval node %q missing knowledgeRef", nodeName)
	}

	return compose.InvokableLambda(func(ctx context.Context, state graphState) (graphState, error) {
		spec, ok := artifact.Runner.Knowledge[knowledgeRef]
		if !ok {
			return nil, FailureReasonError{
				Reason:  "UnknownKnowledge",
				Message: fmt.Sprintf("unknown knowledge %q for node %q", knowledgeRef, nodeName),
			}
		}
		runtime, ok := runtimeInfo.Knowledge[knowledgeRef]
		if !ok {
			return nil, FailureReasonError{
				Reason:  "UnknownKnowledge",
				Message: fmt.Sprintf("knowledge runtime binding missing for %q", knowledgeRef),
			}
		}

		query := defaultRetrievalQuery(state)
		topK := int(runtime.DefaultTopK)
		if topK <= 0 {
			topK = 3
		}

		call := RequestedRetrievalCall{
			Name:  knowledgeRef,
			Node:  nodeName,
			Query: query,
			TopK:  topK,
		}

		result, err := retrievalInvoker.Invoke(ctx, runtime, spec, call)
		if err != nil {
			return nil, err
		}

		out := copyState(state)
		out[nodeName] = result.Output
		return out, nil
	}), nil
}

// buildFunctionLambda creates a Lambda that executes a builtin skill function.
func buildFunctionLambda(
	node map[string]interface{},
	artifact contract.CompiledArtifact,
) (*compose.Lambda, error) {
	nodeName, _ := node["name"].(string)
	implementation, _ := node["implementation"].(string)
	if strings.TrimSpace(implementation) == "" {
		return nil, fmt.Errorf("function node %q missing implementation", nodeName)
	}

	return compose.InvokableLambda(func(ctx context.Context, state graphState) (graphState, error) {
		_, _, fn, err := resolveBuiltinSkillFunction(implementation)
		if err != nil {
			return nil, err
		}
		output := fn(state)

		out := copyState(state)
		out[nodeName] = map[string]interface{}{
			"kind":           "function",
			"implementation": implementation,
			"result":         output,
		}
		// Merge function output into top-level state.
		for k, v := range output {
			out[k] = v
		}
		return out, nil
	}), nil
}

// wireEdges adds edges to the graph from the compiled artifact's edge list.
// Unconditional edges are added directly. Conditional edges use Eino branches.
// START/END node names are normalized to compose.START/compose.END.
func wireEdges(g *compose.Graph[graphState, graphState], edgesRaw []interface{}) error {
	if len(edgesRaw) == 0 {
		return nil
	}

	// Group edges by source node.
	type edge struct {
		to    string
		when  string
	}
	edgesBySource := make(map[string][]edge)
	for _, raw := range edgesRaw {
		e, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		from := normalizeNodeName(e["from"])
		to := normalizeNodeName(e["to"])
		when, _ := e["when"].(string)
		if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
			continue
		}
		edgesBySource[from] = append(edgesBySource[from], edge{to: to, when: when})
	}

	for from, edges := range edgesBySource {
		var unconditional []string
		var conditional []edge

		for _, e := range edges {
			if strings.TrimSpace(e.when) == "" {
				unconditional = append(unconditional, e.to)
			} else {
				conditional = append(conditional, e)
			}
		}

		// If there's only one unconditional edge, add it directly.
		if len(conditional) == 0 && len(unconditional) == 1 {
			if err := g.AddEdge(from, unconditional[0]); err != nil {
				return fmt.Errorf("adding edge %s -> %s: %w", from, unconditional[0], err)
			}
			continue
		}

		// If there are conditional edges, use a branch.
		if len(conditional) > 0 {
			// Build the endNodes map for the branch.
			endNodes := make(map[string]bool)
			for _, ce := range conditional {
				endNodes[ce.to] = true
			}
			for _, to := range unconditional {
				endNodes[to] = true
			}

			branch := compose.NewGraphBranch(func(ctx context.Context, state graphState) (string, error) {
				for _, ce := range conditional {
					match, err := edgeConditionMatches(ce.when, nil, state)
					if err != nil {
						return "", err
					}
					if match {
						return ce.to, nil
					}
				}
				// Fall back to first unconditional edge.
				if len(unconditional) > 0 {
					return unconditional[0], nil
				}
				return compose.END, nil
			}, endNodes)
			if err := g.AddBranch(from, branch); err != nil {
				return fmt.Errorf("adding branch from %s: %w", from, err)
			}
			continue
		}

		// Multiple unconditional edges — add them all (parallel execution).
		for _, to := range unconditional {
			if err := g.AddEdge(from, to); err != nil {
				return fmt.Errorf("adding edge %s -> %s: %w", from, to, err)
			}
		}
	}

	// Ensure all leaf nodes (no outgoing edges) connect to END.
	allNodes := make(map[string]bool)
	for from := range edgesBySource {
		allNodes[from] = true
		for _, e := range edgesBySource[from] {
			allNodes[e.to] = true
		}
	}
	hasOutgoing := make(map[string]bool)
	for from, edges := range edgesBySource {
		if len(edges) > 0 {
			hasOutgoing[from] = true
		}
	}
	for node := range allNodes {
		if !hasOutgoing[node] && node != compose.END {
			if err := g.AddEdge(node, compose.END); err != nil {
				return fmt.Errorf("adding terminal edge %s -> END: %w", node, err)
			}
		}
	}

	return nil
}

// normalizeNodeName maps artifact node names to Eino graph node names.
// "START" and "start" both map to compose.START.
// "END" and "end" both map to compose.END.
func normalizeNodeName(raw interface{}) string {
	name, _ := raw.(string)
	switch strings.TrimSpace(name) {
	case "START", compose.START:
		return compose.START
	case "END", compose.END:
		return compose.END
	default:
		return name
	}
}

// autoWireEdges adds automatic START → first-node → END edges when the
// artifact graph has nodes but no edges, or when edges don't reference START.
func autoWireEdges(g *compose.Graph[graphState, graphState], nodesRaw []interface{}, edgesRaw []interface{}) error {
	// Collect all node names.
	nodeNames := make([]string, 0, len(nodesRaw))
	for _, raw := range nodesRaw {
		node, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		name, _ := node["name"].(string)
		if strings.TrimSpace(name) != "" {
			nodeNames = append(nodeNames, name)
		}
	}
	if len(nodeNames) == 0 {
		return nil
	}

	// Check if any edge references START.
	hasStartEdge := false
	for _, raw := range edgesRaw {
		e, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		from := normalizeNodeName(e["from"])
		if from == compose.START {
			hasStartEdge = true
			break
		}
	}

	// If no edges reference START, wire START → first node.
	if !hasStartEdge {
		if err := g.AddEdge(compose.START, nodeNames[0]); err != nil {
			return fmt.Errorf("auto-wiring START -> %s: %w", nodeNames[0], err)
		}
	}

	// Check if any edge references END.
	hasEndEdge := false
	for _, raw := range edgesRaw {
		e, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}
		to := normalizeNodeName(e["to"])
		if to == compose.END {
			hasEndEdge = true
			break
		}
	}

	// If no edges reference END, wire last node → END.
	if !hasEndEdge {
		// Find leaf nodes (nodes with no outgoing edges).
		outgoing := make(map[string]bool)
		for _, raw := range edgesRaw {
			e, ok := raw.(map[string]interface{})
			if !ok {
				continue
			}
			from := normalizeNodeName(e["from"])
			outgoing[from] = true
		}
		for _, name := range nodeNames {
			if !outgoing[name] {
				if err := g.AddEdge(name, compose.END); err != nil {
					return fmt.Errorf("auto-wiring %s -> END: %w", name, err)
				}
			}
		}
	}

	return nil
}

// copyState creates a shallow copy of the state map.
func copyState(state graphState) graphState {
	out := make(graphState, len(state)+1)
	for k, v := range state {
		out[k] = v
	}
	return out
}

// defaultRetrievalQuery is defined in step.go and reused here.

// resolveConditionPath reuses the existing edge condition logic from step.go.
