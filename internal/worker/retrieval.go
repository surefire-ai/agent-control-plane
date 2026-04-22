package worker

import (
	"context"
	"fmt"
	"sort"
	"strings"

	einoretriever "github.com/cloudwego/eino/components/retriever"
	einoschema "github.com/cloudwego/eino/schema"

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

type RetrievalInvoker interface {
	Invoke(ctx context.Context, runtime contract.WorkerKnowledgeRuntime, spec contract.KnowledgeSpec, call RequestedRetrievalCall) (ExecutedRetrievalInvocation, error)
}

type EinoKnowledgeRetriever struct {
	Runtime contract.WorkerKnowledgeRuntime
	Spec    contract.KnowledgeSpec
}

var _ einoretriever.Retriever = (*EinoKnowledgeRetriever)(nil)

type EinoRetrievalInvoker struct{}

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
	result, err := r.retrievalInvoker().Invoke(ctx, runtime, spec, RequestedRetrievalCall{
		Name:  knowledgeName,
		Node:  nodeName,
		Query: call.Query,
		TopK:  topK,
	})
	if err != nil {
		return ExecutedRetrievalInvocation{}, false, err
	}
	return result, true, nil
}

func (i EinoRetrievalInvoker) Invoke(ctx context.Context, runtime contract.WorkerKnowledgeRuntime, spec contract.KnowledgeSpec, call RequestedRetrievalCall) (ExecutedRetrievalInvocation, error) {
	retriever := EinoKnowledgeRetriever{
		Runtime: runtime,
		Spec:    spec,
	}
	docs, err := retriever.Retrieve(ctx, call.Query, einoretriever.WithTopK(call.TopK))
	if err != nil {
		return ExecutedRetrievalInvocation{}, err
	}
	results := retrievalResultsFromDocuments(docs)
	output := map[string]interface{}{
		"name":    call.Name,
		"query":   call.Query,
		"results": results,
		"topK":    call.TopK,
	}
	requestInline := map[string]interface{}{
		"name":  call.Name,
		"query": call.Query,
		"topK":  call.TopK,
	}
	responseInline := map[string]interface{}{
		"name":    call.Name,
		"results": results,
	}
	if call.Node != "" {
		output["node"] = call.Node
		requestInline["node"] = call.Node
		responseInline["node"] = call.Node
	}
	return ExecutedRetrievalInvocation{
		Output: output,
		Artifacts: []contract.WorkerArtifact{
			{Name: "retrieval-request", Kind: "json", Inline: requestInline},
			{Name: "retrieval-response", Kind: "json", Inline: responseInline},
		},
	}, nil
}

func (r EinoKnowledgeRetriever) Retrieve(ctx context.Context, query string, opts ...einoretriever.Option) ([]*einoschema.Document, error) {
	_ = ctx
	defaultTopK := int(r.Runtime.DefaultTopK)
	options := einoretriever.GetCommonOptions(&einoretriever.Options{TopK: &defaultTopK}, opts...)
	topK := defaultTopK
	if options.TopK != nil && *options.TopK > 0 {
		topK = *options.TopK
	}
	if topK <= 0 {
		topK = 3
	}
	results := retrievalResults(r.Spec, query, topK)
	docs := make([]*einoschema.Document, 0, len(results))
	for _, result := range results {
		doc := &einoschema.Document{
			ID:      stringMapValue(result, "id"),
			Content: stringMapValue(result, "summary"),
			MetaData: map[string]any{
				"sourceName": stringMapValue(result, "sourceName"),
				"sourceURI":  stringMapValue(result, "sourceURI"),
			},
		}
		if score, ok := result["score"].(float64); ok {
			doc = doc.WithScore(score)
		}
		docs = append(docs, doc)
	}
	return docs, nil
}

func retrievalResultsFromDocuments(docs []*einoschema.Document) []map[string]interface{} {
	results := make([]map[string]interface{}, 0, len(docs))
	for _, doc := range docs {
		if doc == nil {
			continue
		}
		result := map[string]interface{}{
			"id":      doc.ID,
			"summary": doc.Content,
			"score":   doc.Score(),
		}
		if doc.MetaData != nil {
			if sourceName, _ := doc.MetaData["sourceName"].(string); sourceName != "" {
				result["sourceName"] = sourceName
			}
			if sourceURI, _ := doc.MetaData["sourceURI"].(string); sourceURI != "" {
				result["sourceURI"] = sourceURI
			}
		}
		results = append(results, result)
	}
	return results
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

func stringMapValue(values map[string]interface{}, key string) string {
	if len(values) == 0 {
		return ""
	}
	value, _ := values[key].(string)
	return value
}
