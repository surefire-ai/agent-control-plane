package manager

import "context"

// CRDSyncer synchronizes Manager database state to Kubernetes CRDs.
// When the Manager creates, updates, or deletes a resource in its database,
// the syncer ensures the corresponding Kubernetes resource reflects that state.
//
// Implementations:
//   - K8sCRDSyncer: uses a controller-runtime client to manage real CRDs
//   - NoopCRDSyncer: does nothing (used when K8s is not configured)
type CRDSyncer interface {
	SyncTenant(ctx context.Context, tenant TenantRecord) error
	DeleteTenant(ctx context.Context, id string) error

	SyncWorkspace(ctx context.Context, workspace WorkspaceRecord) error
	DeleteWorkspace(ctx context.Context, workspace WorkspaceRecord) error

	SyncAgent(ctx context.Context, agent AgentRecord) error
	DeleteAgent(ctx context.Context, agent AgentRecord) error

	SyncEvaluation(ctx context.Context, eval EvaluationRecord) error
	DeleteEvaluation(ctx context.Context, eval EvaluationRecord) error

	SyncProvider(ctx context.Context, provider ProviderRecord) error
	DeleteProvider(ctx context.Context, provider ProviderRecord) error
}
