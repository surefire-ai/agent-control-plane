package manager

import (
	"context"
	"fmt"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultNamespace = "korus-system"

// K8sCRDSyncer syncs Manager database state to Kubernetes CRDs using a
// controller-runtime client. It implements the CRDSyncer interface.
type K8sCRDSyncer struct {
	client client.Client
	scheme *runtime.Scheme
	// store is optional; when set, the syncer can look up related records
	// (e.g., workspace namespace for agents) during sync.
	store *Stores
}

// NewK8sCRDSyncer creates a new K8sCRDSyncer.
func NewK8sCRDSyncer(c client.Client, scheme *runtime.Scheme) *K8sCRDSyncer {
	return &K8sCRDSyncer{client: c, scheme: scheme}
}

// NewK8sCRDSyncerWithStores creates a new K8sCRDSyncer with access to the
// manager stores, enabling cross-record lookups (e.g., resolving a workspace
// namespace when syncing an agent).
func NewK8sCRDSyncerWithStores(c client.Client, scheme *runtime.Scheme, stores *Stores) *K8sCRDSyncer {
	return &K8sCRDSyncer{client: c, scheme: scheme, store: stores}
}

// ---------------------------------------------------------------------------
// Tenant
// ---------------------------------------------------------------------------

func (s *K8sCRDSyncer) SyncTenant(ctx context.Context, rec TenantRecord) error {
	ns := s.tenantNamespace(rec)
	name := rec.ID

	obj := &apiv1alpha1.Tenant{}
	err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, obj)

	if apierrors.IsNotFound(err) {
		obj = &apiv1alpha1.Tenant{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels:    tenantLabels(rec),
			},
			Spec: tenantSpec(rec),
		}
		return s.client.Create(ctx, obj)
	}
	if err != nil {
		return fmt.Errorf("get Tenant %q: %w", name, err)
	}

	obj.Spec = tenantSpec(rec)
	obj.Labels = mergeLabels(obj.Labels, tenantLabels(rec))
	return s.client.Update(ctx, obj)
}

func (s *K8sCRDSyncer) DeleteTenant(ctx context.Context, id string) error {
	obj := &apiv1alpha1.Tenant{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: defaultNamespace},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete Tenant %q: %w", id, err)
	}
	return nil
}

func tenantSpec(rec TenantRecord) apiv1alpha1.TenantSpec {
	return apiv1alpha1.TenantSpec{
		DisplayName: rec.DisplayName,
	}
}

func tenantLabels(rec TenantRecord) map[string]string {
	labels := map[string]string{
		"korus.io/managed-by":  "manager",
		"korus.io/tenant-id":   rec.ID,
		"korus.io/tenant-slug": rec.Slug,
	}
	if rec.OrganizationID != "" {
		labels["korus.io/organization-id"] = rec.OrganizationID
	}
	return labels
}

func (s *K8sCRDSyncer) tenantNamespace(_ TenantRecord) string {
	// Tenants are rendered into the control-plane namespace.
	return defaultNamespace
}

// ---------------------------------------------------------------------------
// Workspace
// ---------------------------------------------------------------------------

func (s *K8sCRDSyncer) SyncWorkspace(ctx context.Context, rec WorkspaceRecord) error {
	ns := workspaceNamespace(rec)
	name := rec.ID

	obj := &apiv1alpha1.Workspace{}
	err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, obj)

	spec := workspaceSpec(rec)
	labels := workspaceLabels(rec)

	if apierrors.IsNotFound(err) {
		obj = &apiv1alpha1.Workspace{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels:    labels,
			},
			Spec: spec,
		}
		s.setTenantOwnerRef(ctx, obj, rec.TenantID, ns)
		return s.client.Create(ctx, obj)
	}
	if err != nil {
		return fmt.Errorf("get Workspace %q: %w", name, err)
	}

	obj.Spec = spec
	obj.Labels = mergeLabels(obj.Labels, labels)
	return s.client.Update(ctx, obj)
}

func (s *K8sCRDSyncer) DeleteWorkspace(ctx context.Context, id string) error {
	obj := &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: defaultNamespace},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete Workspace %q: %w", id, err)
	}
	return nil
}

func workspaceSpec(rec WorkspaceRecord) apiv1alpha1.WorkspaceSpec {
	return apiv1alpha1.WorkspaceSpec{
		TenantRef:   apiv1alpha1.LocalObjectReference{Name: rec.TenantID},
		DisplayName: rec.DisplayName,
		Description: rec.Description,
		Namespace:   rec.KubernetesNamespace,
	}
}

func workspaceLabels(rec WorkspaceRecord) map[string]string {
	return map[string]string{
		"korus.io/managed-by":   "manager",
		"korus.io/workspace-id": rec.ID,
		"korus.io/tenant-id":    rec.TenantID,
	}
}

func workspaceNamespace(rec WorkspaceRecord) string {
	if rec.KubernetesNamespace != "" {
		return rec.KubernetesNamespace
	}
	return defaultNamespace
}

// ---------------------------------------------------------------------------
// Agent
// ---------------------------------------------------------------------------

func (s *K8sCRDSyncer) SyncAgent(ctx context.Context, rec AgentRecord) error {
	ns := s.resolveNamespace(ctx, rec.WorkspaceID)
	name := rec.ID

	obj := &apiv1alpha1.Agent{}
	err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, obj)

	spec := agentSpec(rec)
	labels := agentLabels(rec)

	if apierrors.IsNotFound(err) {
		obj = &apiv1alpha1.Agent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels:    labels,
			},
			Spec: spec,
		}
		s.setWorkspaceOwnerRef(ctx, obj, rec.WorkspaceID, ns)
		return s.client.Create(ctx, obj)
	}
	if err != nil {
		return fmt.Errorf("get Agent %q: %w", name, err)
	}

	obj.Spec = spec
	obj.Labels = mergeLabels(obj.Labels, labels)
	return s.client.Update(ctx, obj)
}

func (s *K8sCRDSyncer) DeleteAgent(ctx context.Context, id string) error {
	obj := &apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: defaultNamespace},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete Agent %q: %w", id, err)
	}
	return nil
}

func agentSpec(rec AgentRecord) apiv1alpha1.AgentSpec {
	spec := apiv1alpha1.AgentSpec{
		Identity: apiv1alpha1.AgentIdentitySpec{
			DisplayName: rec.DisplayName,
			Description: rec.Description,
		},
		Runtime: apiv1alpha1.AgentRuntimeSpec{
			Engine:      rec.RuntimeEngine,
			RunnerClass: rec.RunnerClass,
		},
		WorkspaceRef: &apiv1alpha1.LocalObjectReference{Name: rec.WorkspaceID},
	}

	if rec.Pattern != "" {
		spec.Pattern = &apiv1alpha1.AgentPatternSpec{
			Type: rec.Pattern,
		}
	}

	if rec.ModelProvider != "" || rec.ModelName != "" {
		spec.Models = map[string]apiv1alpha1.ModelSpec{
			"default": {
				Provider: rec.ModelProvider,
				Model:    rec.ModelName,
			},
		}
	}

	return spec
}

func agentLabels(rec AgentRecord) map[string]string {
	return map[string]string{
		"korus.io/managed-by":   "manager",
		"korus.io/agent-id":     rec.ID,
		"korus.io/tenant-id":    rec.TenantID,
		"korus.io/workspace-id": rec.WorkspaceID,
	}
}

// ---------------------------------------------------------------------------
// Evaluation
// ---------------------------------------------------------------------------

func (s *K8sCRDSyncer) SyncEvaluation(ctx context.Context, rec EvaluationRecord) error {
	ns := s.resolveNamespace(ctx, rec.WorkspaceID)
	name := rec.ID

	obj := &apiv1alpha1.AgentEvaluation{}
	err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, obj)

	spec := evaluationSpec(rec)
	labels := evaluationLabels(rec)

	if apierrors.IsNotFound(err) {
		obj = &apiv1alpha1.AgentEvaluation{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels:    labels,
			},
			Spec: spec,
		}
		s.setWorkspaceOwnerRef(ctx, obj, rec.WorkspaceID, ns)
		return s.client.Create(ctx, obj)
	}
	if err != nil {
		return fmt.Errorf("get AgentEvaluation %q: %w", name, err)
	}

	obj.Spec = spec
	obj.Labels = mergeLabels(obj.Labels, labels)
	return s.client.Update(ctx, obj)
}

func (s *K8sCRDSyncer) DeleteEvaluation(ctx context.Context, id string) error {
	obj := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: defaultNamespace},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete AgentEvaluation %q: %w", id, err)
	}
	return nil
}

func evaluationSpec(rec EvaluationRecord) apiv1alpha1.AgentEvaluationSpec {
	return apiv1alpha1.AgentEvaluationSpec{
		AgentRef: apiv1alpha1.LocalObjectReference{Name: rec.AgentID},
		WorkspaceRef: &apiv1alpha1.LocalObjectReference{
			Name: rec.WorkspaceID,
		},
		DatasetRef: apiv1alpha1.EvaluationDatasetReference{
			Name:     rec.DatasetName,
			Revision: rec.DatasetRevision,
		},
		Baseline: &apiv1alpha1.EvaluationBaselineSpec{
			Revision: rec.BaselineRevision,
		},
	}
}

func evaluationLabels(rec EvaluationRecord) map[string]string {
	return map[string]string{
		"korus.io/managed-by":    "manager",
		"korus.io/evaluation-id": rec.ID,
		"korus.io/agent-id":      rec.AgentID,
		"korus.io/tenant-id":     rec.TenantID,
		"korus.io/workspace-id":  rec.WorkspaceID,
	}
}

// ---------------------------------------------------------------------------
// Provider (ToolProvider)
// ---------------------------------------------------------------------------

func (s *K8sCRDSyncer) SyncProvider(ctx context.Context, rec ProviderRecord) error {
	ns := s.resolveNamespace(ctx, rec.WorkspaceID)
	name := rec.ID

	obj := &apiv1alpha1.ToolProvider{}
	err := s.client.Get(ctx, types.NamespacedName{Name: name, Namespace: ns}, obj)

	spec := providerSpec(rec)
	labels := providerLabels(rec)

	if apierrors.IsNotFound(err) {
		obj = &apiv1alpha1.ToolProvider{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: ns,
				Labels:    labels,
			},
			Spec: spec,
		}
		s.setWorkspaceOwnerRef(ctx, obj, rec.WorkspaceID, ns)
		return s.client.Create(ctx, obj)
	}
	if err != nil {
		return fmt.Errorf("get ToolProvider %q: %w", name, err)
	}

	obj.Spec = spec
	obj.Labels = mergeLabels(obj.Labels, labels)
	return s.client.Update(ctx, obj)
}

func (s *K8sCRDSyncer) DeleteProvider(ctx context.Context, id string) error {
	obj := &apiv1alpha1.ToolProvider{
		ObjectMeta: metav1.ObjectMeta{Name: id, Namespace: defaultNamespace},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete ToolProvider %q: %w", id, err)
	}
	return nil
}

func providerSpec(rec ProviderRecord) apiv1alpha1.ToolProviderSpec {
	return apiv1alpha1.ToolProviderSpec{
		Type:        rec.Provider,
		Description: rec.DisplayName,
	}
}

func providerLabels(rec ProviderRecord) map[string]string {
	return map[string]string{
		"korus.io/managed-by":   "manager",
		"korus.io/provider-id":  rec.ID,
		"korus.io/tenant-id":    rec.TenantID,
		"korus.io/workspace-id": rec.WorkspaceID,
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// resolveNamespace looks up the workspace record to determine the target K8s
// namespace. Falls back to defaultNamespace when the store is unavailable or
// the workspace is not found.
func (s *K8sCRDSyncer) resolveNamespace(ctx context.Context, workspaceID string) string {
	if s.store == nil || workspaceID == "" {
		return defaultNamespace
	}
	ws, err := s.store.Workspaces.GetWorkspace(ctx, workspaceID)
	if err != nil || ws.KubernetesNamespace == "" {
		return defaultNamespace
	}
	return ws.KubernetesNamespace
}

// setTenantOwnerRef attempts to set an OwnerReference from obj to the Tenant
// CRD with the given tenant ID. Silently ignored if the Tenant is not found.
func (s *K8sCRDSyncer) setTenantOwnerRef(ctx context.Context, obj client.Object, tenantID, ns string) {
	if tenantID == "" {
		return
	}
	tenant := &apiv1alpha1.Tenant{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: tenantID, Namespace: ns}, tenant); err != nil {
		return // best-effort
	}
	setOwnerRef(obj, tenant.TypeMeta, tenant.ObjectMeta, s.scheme)
}

// setWorkspaceOwnerRef attempts to set an OwnerReference from obj to the
// Workspace CRD with the given workspace ID. Silently ignored if not found.
func (s *K8sCRDSyncer) setWorkspaceOwnerRef(ctx context.Context, obj client.Object, workspaceID, ns string) {
	if workspaceID == "" {
		return
	}
	ws := &apiv1alpha1.Workspace{}
	if err := s.client.Get(ctx, types.NamespacedName{Name: workspaceID, Namespace: ns}, ws); err != nil {
		return // best-effort
	}
	setOwnerRef(obj, ws.TypeMeta, ws.ObjectMeta, s.scheme)
}

// setOwnerRef adds a single OwnerReference to obj based on the owner's
// TypeMeta and ObjectMeta.
func setOwnerRef(obj client.Object, ownerType metav1.TypeMeta, ownerMeta metav1.ObjectMeta, _ *runtime.Scheme) {
	gvk := ownerType.GroupVersionKind()
	blockOwnerDeletion := true
	isController := false
	refs := obj.GetOwnerReferences()
	refs = append(refs, metav1.OwnerReference{
		APIVersion:         gvk.GroupVersion().String(),
		Kind:               gvk.Kind,
		Name:               ownerMeta.Name,
		UID:                ownerMeta.UID,
		BlockOwnerDeletion: &blockOwnerDeletion,
		Controller:         &isController,
	})
	obj.SetOwnerReferences(refs)
}

// mergeLabels copies new labels into existing, preferring new values.
func mergeLabels(existing, new map[string]string) map[string]string {
	if existing == nil {
		existing = make(map[string]string, len(new))
	}
	for k, v := range new {
		existing[k] = v
	}
	return existing
}

// Ensure K8sCRDSyncer satisfies the CRDSyncer interface at compile time.
var _ CRDSyncer = (*K8sCRDSyncer)(nil)
