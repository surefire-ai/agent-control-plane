package manager

import (
	"context"
	"encoding/json"
	"math"
	"testing"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestAgentSpecFromDataPreservesModelCredentialKey(t *testing.T) {
	spec := agentSpecFromData(AgentRecord{
		ID:            "agent-1",
		WorkspaceID:   "workspace-1",
		DisplayName:   "Agent One",
		Description:   "test agent",
		RuntimeEngine: "eino",
		RunnerClass:   "adk",
		Spec: &AgentSpecData{
			Models: map[string]ModelConfig{
				"planner": {
					Provider: "openai",
					Model:    "gpt-4o-mini",
					CredentialRef: &SecretKeyReference{
						Name: "openai-credentials",
						Key:  "apiKey",
					},
				},
			},
		},
	})

	model, ok := spec.Models["planner"]
	if !ok {
		t.Fatalf("expected planner model to be synced")
	}
	if model.CredentialRef == nil {
		t.Fatalf("expected credentialRef to be synced")
	}
	if model.CredentialRef.Name != "openai-credentials" {
		t.Fatalf("credentialRef.name = %q; want %q", model.CredentialRef.Name, "openai-credentials")
	}
	if model.CredentialRef.Key != "apiKey" {
		t.Fatalf("credentialRef.key = %q; want %q", model.CredentialRef.Key, "apiKey")
	}
}

func TestModelConfigCredentialRefUnmarshalLegacyString(t *testing.T) {
	var cfg ModelConfig
	if err := json.Unmarshal([]byte(`{"credentialRef":"legacy-secret"}`), &cfg); err != nil {
		t.Fatalf("unmarshal legacy credentialRef: %v", err)
	}
	if cfg.CredentialRef == nil {
		t.Fatalf("expected legacy credentialRef to be decoded")
	}
	if cfg.CredentialRef.Name != "legacy-secret" {
		t.Fatalf("credentialRef.name = %q; want %q", cfg.CredentialRef.Name, "legacy-secret")
	}
	if cfg.CredentialRef.Key != "" {
		t.Fatalf("credentialRef.key = %q; want empty", cfg.CredentialRef.Key)
	}
}

func TestAgentSpecFromDataOmitsIncompleteModelCredentialRef(t *testing.T) {
	spec := agentSpecFromData(AgentRecord{
		ID:            "agent-1",
		WorkspaceID:   "workspace-1",
		DisplayName:   "Agent One",
		RuntimeEngine: "eino",
		RunnerClass:   "adk",
		Spec: &AgentSpecData{
			Models: map[string]ModelConfig{
				"legacy": {
					Provider:      "openai",
					Model:         "gpt-4o-mini",
					CredentialRef: &SecretKeyReference{Name: "legacy-secret"},
				},
				"key-only": {
					Provider:      "openai",
					Model:         "gpt-4o-mini",
					CredentialRef: &SecretKeyReference{Key: "apiKey"},
				},
			},
		},
	})

	for name, model := range spec.Models {
		if model.CredentialRef != nil {
			t.Fatalf("model %q credentialRef = %#v; want omitted for incomplete Secret key reference", name, model.CredentialRef)
		}
	}
}

func TestAgentSpecFromDataPreservesKnowledgeRetrievalParameters(t *testing.T) {
	spec := agentSpecFromData(AgentRecord{
		ID:            "agent-1",
		WorkspaceID:   "workspace-1",
		DisplayName:   "Agent One",
		RuntimeEngine: "eino",
		RunnerClass:   "adk",
		Spec: &AgentSpecData{
			KnowledgeRefs: []KnowledgeBinding{
				{
					Name:           "docs",
					Ref:            "kb-docs",
					TopK:           8,
					ScoreThreshold: 0.72,
				},
				{
					Name: "faq",
					Ref:  "kb-faq",
				},
			},
		},
	})

	if len(spec.KnowledgeRefs) != 2 {
		t.Fatalf("knowledgeRefs length = %d; want 2", len(spec.KnowledgeRefs))
	}
	first := spec.KnowledgeRefs[0]
	if first.Name != "docs" || first.Ref != "kb-docs" {
		t.Fatalf("first knowledge binding = %#v; want docs/kb-docs", first)
	}
	assertFreeformFloat(t, first.Retrieval, "topK", 8)
	assertFreeformFloat(t, first.Retrieval, "scoreThreshold", 0.72)
	if len(spec.KnowledgeRefs[1].Retrieval) != 0 {
		t.Fatalf("second retrieval = %#v; want empty", spec.KnowledgeRefs[1].Retrieval)
	}
}

func TestProviderSpecPreservesManagerProviderFields(t *testing.T) {
	spec := providerSpec(ProviderRecord{
		Provider:            "openai",
		DisplayName:         "OpenAI",
		Family:              "openai-compatible",
		BaseURL:             "https://api.example.test/v1",
		CredentialRef:       "provider-credentials",
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	})

	if spec.Type != "openai" {
		t.Fatalf("type = %q; want %q", spec.Type, "openai")
	}
	if spec.Description != "OpenAI" {
		t.Fatalf("description = %q; want %q", spec.Description, "OpenAI")
	}
	assertFreeformString(t, spec.Runtime, "family", "openai-compatible")
	assertFreeformBool(t, spec.Runtime, "domestic", true)
	assertFreeformString(t, spec.HTTP, "baseURL", "https://api.example.test/v1")
	assertFreeformString(t, spec.HTTP, "credentialRef", "provider-credentials")

	var capabilities map[string]bool
	if err := json.Unmarshal(spec.Runtime["capabilities"].Raw, &capabilities); err != nil {
		t.Fatalf("unmarshal runtime.capabilities: %v", err)
	}
	if !capabilities["jsonSchema"] {
		t.Fatalf("runtime.capabilities.jsonSchema = false; want true")
	}
	if !capabilities["toolCalling"] {
		t.Fatalf("runtime.capabilities.toolCalling = false; want true")
	}
}

func TestProviderSpecOmitsEmptyFreeformSections(t *testing.T) {
	spec := providerSpec(ProviderRecord{
		Provider:    "custom",
		DisplayName: "Custom Provider",
	})

	if len(spec.Runtime) != 0 {
		t.Fatalf("runtime = %#v; want empty", spec.Runtime)
	}
	if len(spec.HTTP) != 0 {
		t.Fatalf("http = %#v; want empty", spec.HTTP)
	}
}

func TestK8sCRDSyncerDeleteAgentUsesWorkspaceNamespace(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	if err := apiv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("add scheme: %v", err)
	}

	agent := &apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "agent-1",
			Namespace: "demo-ns",
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	stores := Stores{
		Workspaces: &fakeWorkspaceStore{
			records: map[string]WorkspaceRecord{
				"ws_1": {ID: "ws_1", KubernetesNamespace: "demo-ns"},
			},
			orderedIDs: []string{"ws_1"},
		},
	}
	syncer := NewK8sCRDSyncer(c, scheme)
	syncer.SetStores(&stores)

	if err := syncer.DeleteAgent(ctx, AgentRecord{ID: "agent-1", WorkspaceID: "ws_1"}); err != nil {
		t.Fatalf("delete agent: %v", err)
	}

	var deleted apiv1alpha1.Agent
	err := c.Get(ctx, types.NamespacedName{Name: "agent-1", Namespace: "demo-ns"}, &deleted)
	if !apierrors.IsNotFound(err) {
		t.Fatalf("expected agent to be deleted from workspace namespace, got err=%v object=%#v", err, deleted)
	}
}

func assertFreeformString(t *testing.T, values map[string]apiextensionsv1.JSON, key, want string) {
	t.Helper()
	value, ok := values[key]
	if !ok {
		t.Fatalf("missing freeform key %q", key)
	}
	var got string
	if err := json.Unmarshal(value.Raw, &got); err != nil {
		t.Fatalf("unmarshal freeform key %q: %v", key, err)
	}
	if got != want {
		t.Fatalf("freeform key %q = %q; want %q", key, got, want)
	}
}

func assertFreeformFloat(t *testing.T, values map[string]apiextensionsv1.JSON, key string, want float64) {
	t.Helper()
	value, ok := values[key]
	if !ok {
		t.Fatalf("missing freeform key %q", key)
	}
	var got float64
	if err := json.Unmarshal(value.Raw, &got); err != nil {
		t.Fatalf("unmarshal freeform key %q: %v", key, err)
	}
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("freeform key %q = %f; want %f", key, got, want)
	}
}

func assertFreeformBool(t *testing.T, values map[string]apiextensionsv1.JSON, key string, want bool) {
	t.Helper()
	value, ok := values[key]
	if !ok {
		t.Fatalf("missing freeform key %q", key)
	}
	var got bool
	if err := json.Unmarshal(value.Raw, &got); err != nil {
		t.Fatalf("unmarshal freeform key %q: %v", key, err)
	}
	if got != want {
		t.Fatalf("freeform key %q = %t; want %t", key, got, want)
	}
}
