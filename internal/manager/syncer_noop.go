package manager

import "context"

// NoopCRDSyncer is a CRDSyncer that does nothing.
// Used when the Manager is running without Kubernetes access (e.g., local dev, tests).
type NoopCRDSyncer struct{}

func (NoopCRDSyncer) SyncTenant(_ context.Context, _ TenantRecord) error     { return nil }
func (NoopCRDSyncer) DeleteTenant(_ context.Context, _ string) error         { return nil }
func (NoopCRDSyncer) SyncWorkspace(_ context.Context, _ WorkspaceRecord) error { return nil }
func (NoopCRDSyncer) DeleteWorkspace(_ context.Context, _ string) error      { return nil }
func (NoopCRDSyncer) SyncAgent(_ context.Context, _ AgentRecord) error       { return nil }
func (NoopCRDSyncer) DeleteAgent(_ context.Context, _ string) error          { return nil }
func (NoopCRDSyncer) SyncEvaluation(_ context.Context, _ EvaluationRecord) error { return nil }
func (NoopCRDSyncer) DeleteEvaluation(_ context.Context, _ string) error     { return nil }
func (NoopCRDSyncer) SyncProvider(_ context.Context, _ ProviderRecord) error { return nil }
func (NoopCRDSyncer) DeleteProvider(_ context.Context, _ string) error       { return nil }
