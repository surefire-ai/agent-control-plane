package manager

import "context"

type devWorkspaceStore struct {
	records    map[string]WorkspaceRecord
	orderedIDs []string
}

func (s devWorkspaceStore) GetWorkspace(_ context.Context, id string) (*WorkspaceRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devWorkspaceStore) ListWorkspaces(_ context.Context, page, limit int) ([]WorkspaceRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []WorkspaceRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]WorkspaceRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func (s devWorkspaceStore) ListWorkspacesByTenant(_ context.Context, tenantID string, page, limit int) ([]WorkspaceRecord, int, error) {
	filtered := make([]WorkspaceRecord, 0)
	for _, id := range s.orderedIDs {
		if s.records[id].TenantID == tenantID {
			filtered = append(filtered, s.records[id])
		}
	}
	total := len(filtered)
	start := (page - 1) * limit
	if start >= total {
		return []WorkspaceRecord{}, total, nil
	}
	end := min(start+limit, total)
	return filtered[start:end], total, nil
}

func (s *devWorkspaceStore) CreateWorkspace(_ context.Context, workspace WorkspaceRecord) error {
	if _, exists := s.records[workspace.ID]; exists {
		return ErrConflict
	}
	s.records[workspace.ID] = workspace
	s.orderedIDs = append(s.orderedIDs, workspace.ID)
	return nil
}

func (s *devWorkspaceStore) UpdateWorkspace(_ context.Context, id string, fields map[string]string) (*WorkspaceRecord, error) {
	rec, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	if v, ok := fields["display_name"]; ok {
		rec.DisplayName = v
	}
	if v, ok := fields["description"]; ok {
		rec.Description = v
	}
	if v, ok := fields["status"]; ok {
		rec.Status = v
	}
	if v, ok := fields["kubernetes_namespace"]; ok {
		rec.KubernetesNamespace = v
	}
	if v, ok := fields["kubernetes_workspace_name"]; ok {
		rec.KubernetesWorkspaceName = v
	}
	s.records[id] = rec
	return &rec, nil
}

func (s *devWorkspaceStore) DeleteWorkspace(_ context.Context, id string) error {
	delete(s.records, id)
	for i, oid := range s.orderedIDs {
		if oid == id {
			s.orderedIDs = append(s.orderedIDs[:i], s.orderedIDs[i+1:]...)
			break
		}
	}
	return nil
}

type devTenantStore struct {
	records    map[string]TenantRecord
	orderedIDs []string
}

func (s devTenantStore) GetTenant(_ context.Context, id string) (*TenantRecord, error) {
	record, ok := s.records[id]
	if !ok {
		return nil, ErrNotFound
	}
	return &record, nil
}

func (s devTenantStore) ListTenants(_ context.Context, page, limit int) ([]TenantRecord, int, error) {
	total := len(s.records)
	start := (page - 1) * limit
	if start >= total {
		return []TenantRecord{}, total, nil
	}
	end := min(start+limit, total)
	result := make([]TenantRecord, 0, end-start)
	for i := start; i < end; i++ {
		result = append(result, s.records[s.orderedIDs[i]])
	}
	return result, total, nil
}

func NewFakeStores() Stores {
	workspaces := &devWorkspaceStore{
		records: map[string]WorkspaceRecord{
			"ws_demo":       {ID: "ws_demo", TenantID: "t_demo", Slug: "demo-ws", DisplayName: "Demo Workspace", Description: "A demo workspace for development", Status: "active", KubernetesNamespace: "demo", KubernetesWorkspaceName: "workspace-demo"},
			"ws_staging":    {ID: "ws_staging", TenantID: "t_demo", Slug: "staging-ws", DisplayName: "Staging Workspace", Status: "active", KubernetesNamespace: "staging"},
			"ws_enterprise": {ID: "ws_enterprise", TenantID: "t_enterprise", Slug: "enterprise-ws", DisplayName: "Enterprise Workspace", Description: "Enterprise customer workspace", Status: "active", KubernetesNamespace: "enterprise", KubernetesWorkspaceName: "workspace-enterprise"},
		},
		orderedIDs: []string{"ws_demo", "ws_staging", "ws_enterprise"},
	}
	tenants := &devTenantStore{
		records: map[string]TenantRecord{
			"t_demo":       {ID: "t_demo", OrganizationID: "org_1", Slug: "demo-tenant", DisplayName: "Demo Tenant", Status: "active", DefaultRegion: "us-east-1"},
			"t_enterprise": {ID: "t_enterprise", OrganizationID: "org_1", Slug: "enterprise-tenant", DisplayName: "Enterprise Tenant", Status: "active", DefaultRegion: "eu-west-1"},
			"t_inactive":   {ID: "t_inactive", OrganizationID: "org_2", Slug: "inactive-tenant", DisplayName: "Inactive Tenant", Status: "inactive"},
		},
		orderedIDs: []string{"t_demo", "t_enterprise", "t_inactive"},
	}
	return Stores{
		Workspaces: workspaces,
		Tenants:    tenants,
	}
}
