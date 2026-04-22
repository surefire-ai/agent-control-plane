package worker

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type ExecutedRetrievalInvocation struct {
	Output    map[string]interface{}
	Artifacts []contract.WorkerArtifact
}

type RequestedRetrievalCall struct {
	Name  string
	Node  string
	Query string
	TopK  int
}

func (r EinoADKPlaceholderRunner) invokeRequestedRetrieval(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo) (ExecutedRetrievalInvocation, bool, error) {
	select {
	case <-ctx.Done():
		return ExecutedRetrievalInvocation{}, false, ctx.Err()
	default:
	}

	call, ok, err := requestedRetrievalCall(request.Config.ParsedRunInput)
	if err != nil {
		return ExecutedRetrievalInvocation{}, false, err
	}
	if !ok {
		return ExecutedRetrievalInvocation{}, false, nil
	}
	return r.executeRetrievalCall(ctx, request, runtimeInfo, call)
}

func (r EinoADKPlaceholderRunner) executeRetrievalCall(ctx context.Context, request RunRequest, runtimeInfo contract.WorkerRuntimeInfo, call RequestedRetrievalCall) (ExecutedRetrievalInvocation, bool, error) {
	knowledgeName := call.Name
	nodeName := call.Node
	if nodeName != "" {
		resolved, err := knowledgeNameForGraphNode(request.Artifact, nodeName)
		if err != nil {
			return ExecutedRetrievalInvocation{}, false, err
		}
		knowledgeName = resolved
	}

	spec, ok := request.Artifact.Runner.Knowledge[knowledgeName]
	if !ok {
		return ExecutedRetrievalInvocation{}, false, FailureReasonError{
			Reason:  "UnknownKnowledge",
			Message: fmt.Sprintf("unknown knowledge binding %q", knowledgeName),
		}
	}
	runtime, ok := runtimeInfo.Knowledge[knowledgeName]
	if !ok {
		return ExecutedRetrievalInvocation{}, false, FailureReasonError{
			Reason:  "UnknownKnowledge",
			Message: fmt.Sprintf("knowledge runtime binding missing for %q", knowledgeName),
		}
	}

	topK := call.TopK
	if topK <= 0 {
		topK = int(runtime.DefaultTopK)
	}
	if topK <= 0 {
		topK = 3
	}
	results := retrievalResults(spec, call.Query, topK)
	output := map[string]interface{}{
		"name":    knowledgeName,
		"query":   call.Query,
		"results": results,
		"topK":    topK,
	}
	requestInline := map[string]interface{}{
		"name":  knowledgeName,
		"query": call.Query,
		"topK":  topK,
	}
	responseInline := map[string]interface{}{
		"name":    knowledgeName,
		"results": results,
	}
	if nodeName != "" {
		output["node"] = nodeName
		requestInline["node"] = nodeName
		responseInline["node"] = nodeName
	}
	return ExecutedRetrievalInvocation{
		Output: output,
		Artifacts: []contract.WorkerArtifact{
			{Name: "retrieval-request", Kind: "json", Inline: requestInline},
			{Name: "retrieval-response", Kind: "json", Inline: responseInline},
		},
	}, true, nil
}

func requestedRetrievalCall(input map[string]interface{}) (RequestedRetrievalCall, bool, error) {
	value, ok := input["retrievalCall"]
	if !ok {
		return RequestedRetrievalCall{}, false, nil
	}
	raw, ok := value.(map[string]interface{})
	if !ok {
		return RequestedRetrievalCall{}, false, FailureReasonError{
			Reason:  "InvalidRetrievalCallRequest",
			Message: "retrievalCall must be a JSON object",
		}
	}
	name, _ := raw["name"].(string)
	node, _ := raw["node"].(string)
	query, _ := raw["query"].(string)
	if strings.TrimSpace(name) == "" && strings.TrimSpace(node) == "" {
		return RequestedRetrievalCall{}, false, FailureReasonError{
			Reason:  "InvalidRetrievalCallRequest",
			Message: "retrievalCall.name or retrievalCall.node is required",
		}
	}
	if strings.TrimSpace(query) == "" {
		return RequestedRetrievalCall{}, false, FailureReasonError{
			Reason:  "InvalidRetrievalCallRequest",
			Message: "retrievalCall.query is required",
		}
	}
	topK := 0
	switch value := raw["topK"].(type) {
	case int:
		topK = value
	case int32:
		topK = int(value)
	case int64:
		topK = int(value)
	case float64:
		topK = int(value)
	}
	return RequestedRetrievalCall{Name: name, Node: node, Query: query, TopK: topK}, true, nil
}

func knowledgeNameForGraphNode(artifact contract.CompiledArtifact, nodeName string) (string, error) {
	nodes, ok := artifact.Runner.Graph["nodes"].([]interface{})
	if !ok {
		return "", FailureReasonError{
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
		if name != nodeName {
			continue
		}
		kind, _ := node["kind"].(string)
		if kind != "retrieval" {
			return "", FailureReasonError{
				Reason:  "UnsupportedGraphNode",
				Message: fmt.Sprintf("graph node %q is not a retrieval node", nodeName),
			}
		}
		knowledgeRef, _ := node["knowledgeRef"].(string)
		if strings.TrimSpace(knowledgeRef) == "" {
			return "", FailureReasonError{
				Reason:  "UnknownKnowledge",
				Message: fmt.Sprintf("graph node %q is missing knowledgeRef", nodeName),
			}
		}
		return knowledgeRef, nil
	}
	return "", FailureReasonError{
		Reason:  "UnknownGraphNode",
		Message: fmt.Sprintf("graph node %q was not found", nodeName),
	}
}

func retrievalResults(spec contract.KnowledgeSpec, query string, topK int) []map[string]interface{} {
	if topK <= 0 {
		return nil
	}
	results := make([]map[string]interface{}, 0, topK)
	sources := append([]map[string]interface{}{}, spec.Sources...)
	sort.SliceStable(sources, func(i, j int) bool {
		left, _ := sources[i]["name"].(string)
		right, _ := sources[j]["name"].(string)
		return left < right
	})
	for i := 0; i < topK; i++ {
		result := map[string]interface{}{
			"id":      fmt.Sprintf("%s-%d", strings.ReplaceAll(spec.Ref, " ", "-"), i+1),
			"summary": fmt.Sprintf("retrieved context %d for query %q from %s", i+1, query, spec.Ref),
			"score":   retrievalScore(i),
		}
		if len(sources) > 0 {
			source := sources[i%len(sources)]
			if name, _ := source["name"].(string); name != "" {
				result["sourceName"] = name
			}
			if uri, _ := source["uri"].(string); uri != "" {
				result["sourceURI"] = uri
			}
		}
		results = append(results, result)
	}
	return results
}

func retrievalScore(index int) float64 {
	score := 0.92 - float64(index)*0.07
	if score < 0.1 {
		return 0.1
	}
	return score
}
