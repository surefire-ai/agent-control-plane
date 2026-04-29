package manager

import "context"

type WorkspaceRecord struct {
	ID                      string
	TenantID                string
	Slug                    string
	DisplayName             string
	KubernetesNamespace     string
	KubernetesWorkspaceName string
}

type WorkspaceStore interface {
	GetWorkspace(ctx context.Context, id string) (*WorkspaceRecord, error)
}

type Stores struct {
	Workspaces WorkspaceStore
}
