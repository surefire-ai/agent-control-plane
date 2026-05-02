package artifact

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	"github.com/surefire-ai/korus/internal/contract"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// smallArtifactThreshold is the size in bytes below which artifacts are
// considered small enough to live inline in the CRD status. Artifacts at or
// below this threshold are skipped to avoid unnecessary ConfigMap overhead.
const smallArtifactThreshold = 1024 // 1 KB

// dnsSafeKeyRe matches characters that are NOT allowed in ConfigMap data keys.
// ConfigMap keys must match [-._a-zA-Z0-9].
var dnsSafeKeyRe = regexp.MustCompile(`[^-._a-zA-Z0-9]`)

// Store is the interface for persisting worker artifacts beyond the lifetime
// of a worker Pod.
type Store interface {
	// Store persists the given artifacts and returns ArtifactRefs that can be
	// recorded in AgentRunStatus. If artifacts are small enough to remain
	// inline in the CRD status, Store returns an empty slice and no error.
	Store(ctx context.Context, runName, namespace string, artifacts []contract.WorkerArtifact) ([]apiv1alpha1.ArtifactRef, error)
}

// ConfigMapArtifactStore persists worker artifacts into a ConfigMap owned by
// the AgentRun. This ensures artifacts survive Pod TTL cleanup.
type ConfigMapArtifactStore struct {
	client client.Client
}

// NewConfigMapArtifactStore creates a new ConfigMap-backed artifact store.
func NewConfigMapArtifactStore(c client.Client) *ConfigMapArtifactStore {
	return &ConfigMapArtifactStore{client: c}
}

// Store serializes the artifacts into a ConfigMap. Each WorkerArtifact is
// stored as a separate data key. If the total serialized size is below
// smallArtifactThreshold, storage is skipped and an empty slice is returned.
func (s *ConfigMapArtifactStore) Store(ctx context.Context, runName, namespace string, artifacts []contract.WorkerArtifact) ([]apiv1alpha1.ArtifactRef, error) {
	if len(artifacts) == 0 {
		return nil, nil
	}

	// Build a sanitized key→value map and check total size.
	data := make(map[string]string, len(artifacts))
	totalSize := 0
	for _, a := range artifacts {
		raw, err := json.Marshal(a)
		if err != nil {
			return nil, fmt.Errorf("marshal artifact %q: %w", a.Name, err)
		}
		key := sanitizeKey(a.Name)
		if key == "" {
			key = fmt.Sprintf("artifact-%d", len(data))
		}
		data[key] = string(raw)
		totalSize += len(key) + len(raw)
	}

	// Skip storage for small artifact payloads.
	if totalSize < smallArtifactThreshold {
		return nil, nil
	}

	cmName := configMapNameForRun(runName)

	// Build the ConfigMap.
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: namespace,
			Labels: map[string]string{
				"app.kubernetes.io/name":       "korus-artifact",
				"app.kubernetes.io/managed-by": "korus",
				"windosx.com/agentrun":         runName,
			},
		},
		Data: data,
	}

	// Try to get the existing ConfigMap; create or update as needed.
	existing := &corev1.ConfigMap{}
	err := s.client.Get(ctx, client.ObjectKeyFromObject(cm), existing)
	if apierrors.IsNotFound(err) {
		if err := s.client.Create(ctx, cm); err != nil {
			return nil, fmt.Errorf("create artifact ConfigMap %q: %w", cmName, err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("get artifact ConfigMap %q: %w", cmName, err)
	} else {
		existing.Data = data
		if err := s.client.Update(ctx, existing); err != nil {
			return nil, fmt.Errorf("update artifact ConfigMap %q: %w", cmName, err)
		}
	}

	// Build refs — one per data key.
	refs := make([]apiv1alpha1.ArtifactRef, 0, len(data))
	for key := range data {
		refs = append(refs, apiv1alpha1.ArtifactRef{
			Namespace: namespace,
			Name:      cmName,
			Key:       key,
		})
	}
	return refs, nil
}

// SetOwnerReference attaches an OwnerReference to the artifact ConfigMap so
// that it is garbage-collected when the AgentRun is deleted. Call this after
// Store with the AgentRun object.
func (s *ConfigMapArtifactStore) SetOwnerReference(ctx context.Context, owner metav1.Object, refs []apiv1alpha1.ArtifactRef) error {
	if len(refs) == 0 {
		return nil
	}
	cmName := refs[0].Name
	namespace := refs[0].Namespace

	var cm corev1.ConfigMap
	if err := s.client.Get(ctx, client.ObjectKey{Name: cmName, Namespace: namespace}, &cm); err != nil {
		return fmt.Errorf("get artifact ConfigMap for owner ref: %w", err)
	}

	// Avoid duplicate owner references.
	for _, ref := range cm.OwnerReferences {
		if ref.UID == owner.GetUID() {
			return nil
		}
	}

	// We need the GVK. Since AgentRun is the only owner type, hardcode it.
	ownerRef := metav1.OwnerReference{
		APIVersion: apiv1alpha1.GroupVersion.String(),
		Kind:       "AgentRun",
		Name:       owner.GetName(),
		UID:        owner.GetUID(),
	}
	cm.OwnerReferences = append(cm.OwnerReferences, ownerRef)
	return s.client.Update(ctx, &cm)
}

// configMapNameForRun generates a DNS-safe ConfigMap name from a run name.
// Format: artifact-<run-prefix>-<hash> (max 63 chars).
func configMapNameForRun(runName string) string {
	hash := sha256.Sum256([]byte(runName))
	suffix := hex.EncodeToString(hash[:])[:10]
	prefix := dnsLabelPrefix("artifact-" + runName)
	maxPrefixLength := 63 - len(suffix) - 1
	if len(prefix) > maxPrefixLength {
		prefix = strings.TrimRight(prefix[:maxPrefixLength], "-")
	}
	if prefix == "" {
		prefix = "artifact"
	}
	return prefix + "-" + suffix
}

// sanitizeKey converts an artifact name into a valid ConfigMap data key.
func sanitizeKey(name string) string {
	sanitized := dnsSafeKeyRe.ReplaceAllString(name, "-")
	sanitized = strings.Trim(sanitized, "-")
	if sanitized == "" {
		return ""
	}
	// Truncate to a reasonable length (253 is the max ConfigMap key length).
	if len(sanitized) > 253 {
		sanitized = sanitized[:253]
	}
	return sanitized
}

// dnsLabelPrefix lowercases and replaces non-DNS characters with dashes.
func dnsLabelPrefix(value string) string {
	var builder strings.Builder
	lastWasDash := false
	for _, char := range strings.ToLower(value) {
		isAllowed := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if isAllowed {
			builder.WriteRune(char)
			lastWasDash = false
			continue
		}
		if !lastWasDash {
			builder.WriteRune('-')
			lastWasDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}
