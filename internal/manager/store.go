package manager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

var ErrNotFound = errors.New("manager record not found")

type WorkspaceRecord struct {
	ID                      string
	TenantID                string
	Slug                    string
	DisplayName             string
	Description             string
	Status                  string
	KubernetesNamespace     string
	KubernetesWorkspaceName string
}

type WorkspaceStore interface {
	GetWorkspace(ctx context.Context, id string) (*WorkspaceRecord, error)
}

type Stores struct {
	Workspaces WorkspaceStore
}

type SQLWorkspaceStore struct {
	DB *sql.DB
}

func NewSQLStores(db *sql.DB) Stores {
	return Stores{
		Workspaces: SQLWorkspaceStore{DB: db},
	}
}

func (s SQLWorkspaceStore) GetWorkspace(ctx context.Context, id string) (*WorkspaceRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	var workspace WorkspaceRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id, tenant_id, slug, display_name, description, status, kubernetes_namespace, kubernetes_workspace_name
FROM workspaces
WHERE id = $1`, id).Scan(
		&workspace.ID,
		&workspace.TenantID,
		&workspace.Slug,
		&workspace.DisplayName,
		&workspace.Description,
		&workspace.Status,
		&workspace.KubernetesNamespace,
		&workspace.KubernetesWorkspaceName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get manager workspace %q: %w", id, err)
	}
	return &workspace, nil
}
