package artifact

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/surefire-ai/korus/internal/contract"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestConfigMapArtifactStoreStoresLargeArtifacts(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewConfigMapArtifactStore(kubeClient)

	// Create artifacts that exceed the 1 KB threshold.
	bigValue := strings.Repeat("x", 1050)
	artifacts := []contract.WorkerArtifact{
		{Name: "runtime-model-bindings", Kind: "json", Inline: map[string]interface{}{"data": bigValue}},
	}

	refs, err := store.Store(context.Background(), "run-1", "default", artifacts)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected non-empty refs for large artifacts")
	}
	for _, ref := range refs {
		if ref.Namespace != "default" {
			t.Fatalf("expected namespace 'default', got %q", ref.Namespace)
		}
		if ref.Name == "" {
			t.Fatal("expected non-empty ConfigMap name in ref")
		}
		if ref.Key == "" {
			t.Fatal("expected non-empty key in ref")
		}
	}

	// Verify the ConfigMap was created.
	cmName := refs[0].Name
	var cm corev1.ConfigMap
	if err := kubeClient.Get(context.Background(), types.NamespacedName{
		Namespace: "default",
		Name:      cmName,
	}, &cm); err != nil {
		t.Fatalf("expected ConfigMap to exist: %v", err)
	}
	if len(cm.Data) != len(artifacts) {
		t.Fatalf("expected %d data keys, got %d", len(artifacts), len(cm.Data))
	}
	// Verify labels.
	if cm.Labels["windosx.com/agentrun"] != "run-1" {
		t.Fatalf("expected agentrun label, got %#v", cm.Labels)
	}
}

func TestConfigMapArtifactStoreSkipsSmallArtifacts(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewConfigMapArtifactStore(kubeClient)

	// Small artifacts — well under 1 KB.
	artifacts := []contract.WorkerArtifact{
		{Name: "tiny", Kind: "json", Inline: map[string]interface{}{"x": 1}},
	}

	refs, err := store.Store(context.Background(), "run-2", "default", artifacts)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected empty refs for small artifacts, got %d", len(refs))
	}

	// Verify no ConfigMap was created.
	cmName := configMapNameForRun("run-2")
	var cm corev1.ConfigMap
	err = kubeClient.Get(context.Background(), types.NamespacedName{
		Namespace: "default",
		Name:      cmName,
	}, &cm)
	if err == nil {
		t.Fatal("expected no ConfigMap to be created for small artifacts")
	}
}

func TestConfigMapArtifactStoreSkipsEmptyArtifacts(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewConfigMapArtifactStore(kubeClient)

	refs, err := store.Store(context.Background(), "run-3", "default", nil)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	if len(refs) != 0 {
		t.Fatalf("expected empty refs, got %d", len(refs))
	}
}

func TestConfigMapArtifactStoreSetsOwnerReference(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewConfigMapArtifactStore(kubeClient)

	bigValue := strings.Repeat("y", 1050)
	artifacts := []contract.WorkerArtifact{
		{Name: "output-report", Kind: "json", Inline: map[string]interface{}{"data": bigValue}},
	}

	refs, err := store.Store(context.Background(), "run-4", "default", artifacts)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected non-empty refs")
	}

	owner := &metav1.ObjectMeta{
		Name:      "run-4",
		Namespace: "default",
		UID:       types.UID("run-uid-4"),
	}
	if err := store.SetOwnerReference(context.Background(), owner, refs); err != nil {
		t.Fatalf("SetOwnerReference returned error: %v", err)
	}

	// Verify the ConfigMap now has the owner reference.
	var cm corev1.ConfigMap
	if err := kubeClient.Get(context.Background(), types.NamespacedName{
		Namespace: refs[0].Namespace,
		Name:      refs[0].Name,
	}, &cm); err != nil {
		t.Fatalf("expected ConfigMap: %v", err)
	}
	found := false
	for _, ref := range cm.OwnerReferences {
		if ref.UID == owner.UID && ref.Kind == "AgentRun" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected AgentRun owner reference on ConfigMap, got %#v", cm.OwnerReferences)
	}

	// Calling again should not duplicate the owner reference.
	if err := store.SetOwnerReference(context.Background(), owner, refs); err != nil {
		t.Fatalf("second SetOwnerReference returned error: %v", err)
	}
	if err := kubeClient.Get(context.Background(), types.NamespacedName{
		Namespace: refs[0].Namespace,
		Name:      refs[0].Name,
	}, &cm); err != nil {
		t.Fatalf("expected ConfigMap: %v", err)
	}
	count := 0
	for _, ref := range cm.OwnerReferences {
		if ref.UID == owner.UID {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 owner reference, got %d", count)
	}
}

func TestConfigMapArtifactStoreDataIsDeserializable(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := corev1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	store := NewConfigMapArtifactStore(kubeClient)

	bigValue := strings.Repeat("z", 1050)
	original := contract.WorkerArtifact{
		Name:   "runtime-trace",
		Kind:   "json",
		Inline: map[string]interface{}{"data": bigValue},
	}
	artifacts := []contract.WorkerArtifact{original}

	refs, err := store.Store(context.Background(), "run-5", "default", artifacts)
	if err != nil {
		t.Fatalf("Store returned error: %v", err)
	}
	if len(refs) == 0 {
		t.Fatal("expected non-empty refs")
	}

	var cm corev1.ConfigMap
	if err := kubeClient.Get(context.Background(), types.NamespacedName{
		Namespace: refs[0].Namespace,
		Name:      refs[0].Name,
	}, &cm); err != nil {
		t.Fatalf("expected ConfigMap: %v", err)
	}

	// Deserialize the stored value.
	raw, ok := cm.Data[refs[0].Key]
	if !ok {
		t.Fatalf("expected key %q in ConfigMap data", refs[0].Key)
	}
	var decoded contract.WorkerArtifact
	if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
		t.Fatalf("stored artifact is not valid JSON: %v", err)
	}
	if decoded.Name != original.Name {
		t.Fatalf("expected name %q, got %q", original.Name, decoded.Name)
	}
	if decoded.Kind != original.Kind {
		t.Fatalf("expected kind %q, got %q", original.Kind, decoded.Kind)
	}
}

func TestConfigMapNameForRunIsDNSLabelSafe(t *testing.T) {
	name := configMapNameForRun("Run_With_A_Very_Long_Name_That_Should_Be_Shortened_Before_It_Becomes_A_ConfigMap_Name")
	if len(name) > 63 {
		t.Fatalf("expected ConfigMap name to fit DNS label length, got %d: %s", len(name), name)
	}
	if strings.Contains(name, "_") {
		t.Fatalf("expected DNS-safe ConfigMap name, got %q", name)
	}
	if !strings.HasPrefix(name, "artifact-") {
		t.Fatalf("expected ConfigMap name to start with 'artifact-', got %q", name)
	}
}

func TestSanitizeKey(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"runtime-model-bindings", "runtime-model-bindings"},
		{"prompt_preview", "prompt_preview"},
		{"invalid key!@#", "invalid-key"},
		{"", ""},
		{"...---...", "...---..."},
	}
	for _, tt := range tests {
		got := sanitizeKey(tt.input)
		if got != tt.want {
			t.Errorf("sanitizeKey(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
