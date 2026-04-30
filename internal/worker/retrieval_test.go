package worker

import (
	"context"
	"testing"

	einoretriever "github.com/cloudwego/eino/components/retriever"

	"github.com/surefire-ai/korus/internal/contract"
)

func TestEinoKnowledgeRetrieverRetrieve(t *testing.T) {
	retriever := EinoKnowledgeRetriever{
		Runtime: contract.WorkerKnowledgeRuntime{DefaultTopK: 2},
		Spec: contract.KnowledgeSpec{
			Name:        "regulations",
			Ref:         "ehs-regulations",
			Description: "法规库",
			Sources: []map[string]interface{}{
				{"name": "国家安全生产法规", "uri": "s3://ehs-kb/regulations/national/"},
			},
		},
	}

	docs, err := retriever.Retrieve(context.Background(), "有限空间 作业 风险", einoretriever.WithTopK(2))
	if err != nil {
		t.Fatalf("Retrieve returned error: %v", err)
	}
	if len(docs) != 2 {
		t.Fatalf("expected 2 docs, got %d", len(docs))
	}
	if docs[0].ID == "" || docs[0].Content == "" {
		t.Fatalf("unexpected doc: %#v", docs[0])
	}
	if docs[0].Score() <= 0 {
		t.Fatalf("expected score on doc: %#v", docs[0])
	}
}

func TestEinoRetrievalInvokerInvoke(t *testing.T) {
	result, err := EinoRetrievalInvoker{}.Invoke(
		context.Background(),
		contract.WorkerKnowledgeRuntime{DefaultTopK: 2},
		contract.KnowledgeSpec{
			Name:        "regulations",
			Ref:         "ehs-regulations",
			Description: "法规库",
			Sources: []map[string]interface{}{
				{"name": "国家安全生产法规", "uri": "s3://ehs-kb/regulations/national/"},
			},
		},
		RequestedRetrievalCall{
			Name:  "regulations",
			Node:  "retrieve_regulations",
			Query: "有限空间 作业 风险",
			TopK:  2,
		},
	)
	if err != nil {
		t.Fatalf("Invoke returned error: %v", err)
	}
	if result.Output["name"] != "regulations" || result.Output["node"] != "retrieve_regulations" {
		t.Fatalf("unexpected output: %#v", result.Output)
	}
	rawResults, _ := result.Output["results"].([]map[string]interface{})
	if len(rawResults) != 2 {
		raw, _ := result.Output["results"].([]interface{})
		if len(raw) != 2 {
			t.Fatalf("expected 2 retrieval results, got %#v", result.Output)
		}
	}
}
