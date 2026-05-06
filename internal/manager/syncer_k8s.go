package manager

import (
	"context"
	"encoding/json"
	"fmt"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
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

// SetStores attaches manager stores after server startup has initialized them.
func (s *K8sCRDSyncer) SetStores(stores *Stores) {
	s.store = stores
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

func (s *K8sCRDSyncer) DeleteWorkspace(ctx context.Context, rec WorkspaceRecord) error {
	obj := &apiv1alpha1.Workspace{
		ObjectMeta: metav1.ObjectMeta{Name: rec.ID, Namespace: workspaceNamespace(rec)},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete Workspace %q: %w", rec.ID, err)
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

func (s *K8sCRDSyncer) DeleteAgent(ctx context.Context, rec AgentRecord) error {
	obj := &apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: rec.ID, Namespace: s.resolveNamespace(ctx, rec.WorkspaceID)},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete Agent %q: %w", rec.ID, err)
	}
	return nil
}

func agentSpec(rec AgentRecord) apiv1alpha1.AgentSpec {
	// If a full spec is stored, use it to build the CRD spec.
	if rec.Spec != nil {
		return agentSpecFromData(rec)
	}
	// Fallback: build a minimal spec from the flat record fields.
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

// agentSpecFromData builds a full AgentSpec from the stored AgentSpecData,
// overlaying record-level identity and workspace fields.
func agentSpecFromData(rec AgentRecord) apiv1alpha1.AgentSpec {
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

	d := rec.Spec

	if d.Runtime.Engine != "" {
		spec.Runtime.Engine = d.Runtime.Engine
	}
	if d.Runtime.RunnerClass != "" {
		spec.Runtime.RunnerClass = d.Runtime.RunnerClass
	}
	if d.Runtime.Mode != "" {
		spec.Runtime.Mode = d.Runtime.Mode
	}
	if d.Runtime.Entrypoint != "" {
		spec.Runtime.Entrypoint = d.Runtime.Entrypoint
	}

	if d.Identity.DisplayName != "" {
		spec.Identity.DisplayName = d.Identity.DisplayName
	}
	if d.Identity.Description != "" {
		spec.Identity.Description = d.Identity.Description
	}
	if d.Identity.Role != "" {
		spec.Identity.Role = d.Identity.Role
	}

	if len(d.Models) > 0 {
		spec.Models = make(map[string]apiv1alpha1.ModelSpec, len(d.Models))
		for k, m := range d.Models {
			ms := apiv1alpha1.ModelSpec{
				Provider:       m.Provider,
				Model:          m.Model,
				BaseURL:        m.BaseURL,
				Temperature:    m.Temperature,
				MaxTokens:      m.MaxTokens,
				TimeoutSeconds: m.TimeoutSeconds,
			}
			if m.CredentialRef != nil && m.CredentialRef.Name != "" && m.CredentialRef.Key != "" {
				ms.CredentialRef = &apiv1alpha1.SecretKeyReference{Name: m.CredentialRef.Name, Key: m.CredentialRef.Key}
			}
			spec.Models[k] = ms
		}
	} else if rec.ModelProvider != "" || rec.ModelName != "" {
		spec.Models = map[string]apiv1alpha1.ModelSpec{
			"default": {
				Provider: rec.ModelProvider,
				Model:    rec.ModelName,
			},
		}
	}

	if d.Pattern != nil {
		pat := &apiv1alpha1.AgentPatternSpec{
			Type:             d.Pattern.Type,
			Version:          d.Pattern.Version,
			ModelRef:         d.Pattern.ModelRef,
			ExecutorModelRef: d.Pattern.ExecutorModelRef,
			ToolRefs:         d.Pattern.ToolRefs,
			KnowledgeRefs:    d.Pattern.KnowledgeRefs,
			MaxIterations:    d.Pattern.MaxIterations,
			StopWhen:         d.Pattern.StopWhen,
		}
		if len(d.Pattern.Routes) > 0 {
			pat.Routes = make([]apiv1alpha1.PatternRoute, len(d.Pattern.Routes))
			for i, r := range d.Pattern.Routes {
				pat.Routes[i] = apiv1alpha1.PatternRoute{
					Label:    r.Label,
					AgentRef: r.AgentRef,
					ModelRef: r.ModelRef,
					Default:  r.Default,
				}
			}
		}
		spec.Pattern = pat
	} else if rec.Pattern != "" {
		spec.Pattern = &apiv1alpha1.AgentPatternSpec{
			Type: rec.Pattern,
		}
	}

	if d.PromptRefs.System != "" {
		spec.PromptRefs = apiv1alpha1.AgentPromptRefs{
			System: d.PromptRefs.System,
		}
	}

	if len(d.KnowledgeRefs) > 0 {
		spec.KnowledgeRefs = make([]apiv1alpha1.KnowledgeBindingSpec, len(d.KnowledgeRefs))
		for i, k := range d.KnowledgeRefs {
			spec.KnowledgeRefs[i] = knowledgeBindingSpec(k)
		}
	}

	spec.ToolRefs = d.ToolRefs

	if len(d.SkillRefs) > 0 {
		spec.SkillRefs = make([]apiv1alpha1.SkillBindingSpec, len(d.SkillRefs))
		for i, s := range d.SkillRefs {
			spec.SkillRefs[i] = apiv1alpha1.SkillBindingSpec{
				Name: s.Name,
				Ref:  s.Ref,
			}
		}
	}

	if len(d.SubAgentRefs) > 0 {
		spec.SubAgentRefs = make([]apiv1alpha1.SubAgentBindingSpec, len(d.SubAgentRefs))
		for i, s := range d.SubAgentRefs {
			spec.SubAgentRefs[i] = apiv1alpha1.SubAgentBindingSpec{
				Name:      s.Name,
				Ref:       s.Ref,
				Namespace: s.Namespace,
			}
		}
	}

	spec.MCPRefs = d.MCPRefs
	spec.PolicyRef = d.PolicyRef

	if d.Graph != nil {
		graph := apiv1alpha1.AgentGraphSpec{}
		if len(d.Graph.Nodes) > 0 {
			graph.Nodes = make([]apiv1alpha1.AgentGraphNode, len(d.Graph.Nodes))
			for i, n := range d.Graph.Nodes {
				graph.Nodes[i] = apiv1alpha1.AgentGraphNode{
					Name:           n.Name,
					Kind:           n.Kind,
					ModelRef:       n.ModelRef,
					ToolRef:        n.ToolRef,
					KnowledgeRef:   n.KnowledgeRef,
					AgentRef:       n.AgentRef,
					Implementation: n.Implementation,
				}
			}
		}
		if len(d.Graph.Edges) > 0 {
			graph.Edges = make([]apiv1alpha1.AgentGraphEdge, len(d.Graph.Edges))
			for i, e := range d.Graph.Edges {
				graph.Edges[i] = apiv1alpha1.AgentGraphEdge{
					From: e.From,
					To:   e.To,
					When: e.When,
				}
			}
		}
		spec.Graph = graph
	}

	if len(d.Interfaces.Input.Schema) > 0 || len(d.Interfaces.Output.Schema) > 0 {
		iface := apiv1alpha1.AgentInterfaceSpec{}
		if len(d.Interfaces.Input.Schema) > 0 {
			b, err := json.Marshal(d.Interfaces.Input.Schema)
			if err == nil {
				schema := apiv1alpha1.JSONSchema{Raw: b}
				iface.Input = apiv1alpha1.SchemaEnvelope{Schema: schema}
			}
		}
		if len(d.Interfaces.Output.Schema) > 0 {
			b, err := json.Marshal(d.Interfaces.Output.Schema)
			if err == nil {
				schema := apiv1alpha1.JSONSchema{Raw: b}
				iface.Output = apiv1alpha1.SchemaEnvelope{Schema: schema}
			}
		}
		spec.Interfaces = iface
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

func (s *K8sCRDSyncer) DeleteEvaluation(ctx context.Context, rec EvaluationRecord) error {
	obj := &apiv1alpha1.AgentEvaluation{
		ObjectMeta: metav1.ObjectMeta{Name: rec.ID, Namespace: s.resolveNamespace(ctx, rec.WorkspaceID)},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete AgentEvaluation %q: %w", rec.ID, err)
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

func (s *K8sCRDSyncer) DeleteProvider(ctx context.Context, rec ProviderRecord) error {
	obj := &apiv1alpha1.ToolProvider{
		ObjectMeta: metav1.ObjectMeta{Name: rec.ID, Namespace: s.resolveNamespace(ctx, rec.WorkspaceID)},
	}
	if err := s.client.Delete(ctx, obj); err != nil && !apierrors.IsNotFound(err) {
		return fmt.Errorf("delete ToolProvider %q: %w", rec.ID, err)
	}
	return nil
}

func knowledgeBindingSpec(binding KnowledgeBinding) apiv1alpha1.KnowledgeBindingSpec {
	spec := apiv1alpha1.KnowledgeBindingSpec{
		Name: binding.Name,
		Ref:  binding.Ref,
	}
	retrieval := apiv1alpha1.FreeformObject{}
	if binding.TopK > 0 {
		retrieval["topK"] = jsonValue(binding.TopK)
	}
	if binding.ScoreThreshold > 0 {
		retrieval["scoreThreshold"] = jsonValue(binding.ScoreThreshold)
	}
	if len(retrieval) > 0 {
		spec.Retrieval = retrieval
	}
	return spec
}

func providerSpec(rec ProviderRecord) apiv1alpha1.ToolProviderSpec {
	spec := apiv1alpha1.ToolProviderSpec{
		Type:        rec.Provider,
		Description: rec.DisplayName,
	}

	runtime := apiv1alpha1.FreeformObject{}
	if rec.Family != "" {
		runtime["family"] = jsonValue(rec.Family)
	}
	if rec.Domestic {
		runtime["domestic"] = jsonValue(true)
	}
	capabilities := map[string]bool{}
	if rec.SupportsJSONSchema {
		capabilities["jsonSchema"] = true
	}
	if rec.SupportsToolCalling {
		capabilities["toolCalling"] = true
	}
	if len(capabilities) > 0 {
		runtime["capabilities"] = jsonValue(capabilities)
	}
	if len(runtime) > 0 {
		spec.Runtime = runtime
	}

	http := apiv1alpha1.FreeformObject{}
	if rec.BaseURL != "" {
		http["baseURL"] = jsonValue(rec.BaseURL)
	}
	if rec.CredentialRef != "" {
		http["credentialRef"] = jsonValue(rec.CredentialRef)
	}
	if len(http) > 0 {
		spec.HTTP = http
	}

	return spec
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
	if s.store == nil || s.store.Workspaces == nil || workspaceID == "" {
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

func jsonValue(value interface{}) apiextensionsv1.JSON {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte("null")
	}
	return apiextensionsv1.JSON{Raw: raw}
}

// Ensure K8sCRDSyncer satisfies the CRDSyncer interface at compile time.
var _ CRDSyncer = (*K8sCRDSyncer)(nil)
