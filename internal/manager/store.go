package manager

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrNotFound = errors.New("manager record not found")
var ErrConflict = errors.New("manager record already exists")

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

type TenantRecord struct {
	ID             string
	OrganizationID string
	Slug           string
	DisplayName    string
	Status         string
	DefaultRegion  string
}

type AgentRecord struct {
	ID             string
	TenantID       string
	WorkspaceID    string
	Slug           string
	DisplayName    string
	Description    string
	Status         string
	Pattern        string
	RuntimeEngine  string
	RunnerClass    string
	ModelProvider  string
	ModelName      string
	LatestRevision string
}

type EvaluationRecord struct {
	ID               string
	TenantID         string
	WorkspaceID      string
	AgentID          string
	Slug             string
	DisplayName      string
	Description      string
	Status           string
	DatasetName      string
	DatasetRevision  string
	BaselineRevision string
	Score            float64
	GatePassed       bool
	SamplesTotal     int
	SamplesEvaluated int
	LatestRunID      string
	ReportRef        string
}

type ProviderRecord struct {
	ID                  string
	TenantID            string
	WorkspaceID         string
	Provider            string
	DisplayName         string
	Family              string
	BaseURL             string
	CredentialRef       string
	Status              string
	Domestic            bool
	SupportsJSONSchema  bool
	SupportsToolCalling bool
}

type WorkspaceStore interface {
	GetWorkspace(ctx context.Context, id string) (*WorkspaceRecord, error)
	ListWorkspaces(ctx context.Context, page, limit int) ([]WorkspaceRecord, int, error)
	ListWorkspacesByTenant(ctx context.Context, tenantID string, page, limit int) ([]WorkspaceRecord, int, error)
	CreateWorkspace(ctx context.Context, workspace WorkspaceRecord) error
	UpdateWorkspace(ctx context.Context, id string, fields map[string]string) (*WorkspaceRecord, error)
	DeleteWorkspace(ctx context.Context, id string) error
}

type TenantStore interface {
	GetTenant(ctx context.Context, id string) (*TenantRecord, error)
	ListTenants(ctx context.Context, page, limit int) ([]TenantRecord, int, error)
}

type AgentStore interface {
	GetAgent(ctx context.Context, id string) (*AgentRecord, error)
	ListAgents(ctx context.Context, page, limit int) ([]AgentRecord, int, error)
	ListAgentsByTenant(ctx context.Context, tenantID string, page, limit int) ([]AgentRecord, int, error)
	ListAgentsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]AgentRecord, int, error)
}

type EvaluationStore interface {
	GetEvaluation(ctx context.Context, id string) (*EvaluationRecord, error)
	ListEvaluations(ctx context.Context, page, limit int) ([]EvaluationRecord, int, error)
	ListEvaluationsByTenant(ctx context.Context, tenantID string, page, limit int) ([]EvaluationRecord, int, error)
	ListEvaluationsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]EvaluationRecord, int, error)
}

type ProviderStore interface {
	GetProvider(ctx context.Context, id string) (*ProviderRecord, error)
	ListProviders(ctx context.Context, page, limit int) ([]ProviderRecord, int, error)
	ListProvidersByTenant(ctx context.Context, tenantID string, page, limit int) ([]ProviderRecord, int, error)
	ListProvidersByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]ProviderRecord, int, error)
}

type Stores struct {
	Workspaces  WorkspaceStore
	Tenants     TenantStore
	Agents      AgentStore
	Evaluations EvaluationStore
	Providers   ProviderStore
}

type SQLWorkspaceStore struct {
	DB *sql.DB
}

type SQLTenantStore struct {
	DB *sql.DB
}

type SQLAgentStore struct {
	DB *sql.DB
}

type SQLEvaluationStore struct {
	DB *sql.DB
}

type SQLProviderStore struct {
	DB *sql.DB
}

func NewSQLStores(db *sql.DB) Stores {
	return Stores{
		Workspaces:  SQLWorkspaceStore{DB: db},
		Tenants:     SQLTenantStore{DB: db},
		Agents:      SQLAgentStore{DB: db},
		Evaluations: SQLEvaluationStore{DB: db},
		Providers:   SQLProviderStore{DB: db},
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

func (s SQLWorkspaceStore) ListWorkspaces(ctx context.Context, page, limit int) ([]WorkspaceRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspaces").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager workspaces: %w", err)
	}
	offset := (page - 1) * limit
	rows, err := s.DB.QueryContext(ctx, `SELECT id, tenant_id, slug, display_name, description, status, kubernetes_namespace, kubernetes_workspace_name
	FROM workspaces
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list manager workspaces: %w", err)
	}
	defer rows.Close()

	records := make([]WorkspaceRecord, 0, limit)
	for rows.Next() {
		var rec WorkspaceRecord
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Slug, &rec.DisplayName, &rec.Description, &rec.Status, &rec.KubernetesNamespace, &rec.KubernetesWorkspaceName); err != nil {
			return nil, 0, fmt.Errorf("scan manager workspace: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager workspaces: %w", err)
	}
	return records, total, nil
}

func (s SQLWorkspaceStore) ListWorkspacesByTenant(ctx context.Context, tenantID string, page, limit int) ([]WorkspaceRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM workspaces WHERE tenant_id = $1", tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager workspaces by tenant %q: %w", tenantID, err)
	}
	offset := (page - 1) * limit
	rows, err := s.DB.QueryContext(ctx, `SELECT id, tenant_id, slug, display_name, description, status, kubernetes_namespace, kubernetes_workspace_name
	FROM workspaces
	WHERE tenant_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, tenantID, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list manager workspaces by tenant %q: %w", tenantID, err)
	}
	defer rows.Close()

	records := make([]WorkspaceRecord, 0, limit)
	for rows.Next() {
		var rec WorkspaceRecord
		if err := rows.Scan(&rec.ID, &rec.TenantID, &rec.Slug, &rec.DisplayName, &rec.Description, &rec.Status, &rec.KubernetesNamespace, &rec.KubernetesWorkspaceName); err != nil {
			return nil, 0, fmt.Errorf("scan manager workspace: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager workspaces: %w", err)
	}
	return records, total, nil
}

func (s SQLWorkspaceStore) CreateWorkspace(ctx context.Context, workspace WorkspaceRecord) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO workspaces (id, tenant_id, slug, display_name, description, status, kubernetes_namespace, kubernetes_workspace_name)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		workspace.ID, workspace.TenantID, workspace.Slug, workspace.DisplayName,
		workspace.Description, workspace.Status, workspace.KubernetesNamespace, workspace.KubernetesWorkspaceName,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return fmt.Errorf("create manager workspace %q: %w", workspace.ID, err)
	}
	return nil
}

var workspaceUpdatableColumns = map[string]string{
	"display_name":              "display_name",
	"description":               "description",
	"status":                    "status",
	"kubernetes_namespace":      "kubernetes_namespace",
	"kubernetes_workspace_name": "kubernetes_workspace_name",
}

func (s SQLWorkspaceStore) UpdateWorkspace(ctx context.Context, id string, fields map[string]string) (*WorkspaceRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	columns := make([]string, 0, len(fields))
	values := make([]any, 0, len(fields)+1)
	values = append(values, id)
	idx := 2
	for key, value := range fields {
		col, ok := workspaceUpdatableColumns[key]
		if !ok {
			continue
		}
		columns = append(columns, fmt.Sprintf("%s = $%d", col, idx))
		values = append(values, value)
		idx++
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid fields to update for workspace %q", id)
	}

	query := fmt.Sprintf(`UPDATE workspaces SET %s, updated_at = now()
	WHERE id = $1
	RETURNING id, tenant_id, slug, display_name, description, status, kubernetes_namespace, kubernetes_workspace_name`,
		joinStrings(columns, ", "))

	var workspace WorkspaceRecord
	err := s.DB.QueryRowContext(ctx, query, values...).Scan(
		&workspace.ID, &workspace.TenantID, &workspace.Slug, &workspace.DisplayName,
		&workspace.Description, &workspace.Status, &workspace.KubernetesNamespace, &workspace.KubernetesWorkspaceName,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update manager workspace %q: %w", id, err)
	}
	return &workspace, nil
}

func (s SQLWorkspaceStore) DeleteWorkspace(ctx context.Context, id string) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM workspaces WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete manager workspace %q: %w", id, err)
	}
	return nil
}

func (s SQLTenantStore) GetTenant(ctx context.Context, id string) (*TenantRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	var tenant TenantRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id, organization_id, slug, display_name, status, COALESCE(default_region, '') FROM tenants WHERE id = $1`, id).Scan(
		&tenant.ID, &tenant.OrganizationID, &tenant.Slug, &tenant.DisplayName, &tenant.Status, &tenant.DefaultRegion,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get manager tenant %q: %w", id, err)
	}
	return &tenant, nil
}

func (s SQLTenantStore) ListTenants(ctx context.Context, page, limit int) ([]TenantRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM tenants").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager tenants: %w", err)
	}
	offset := (page - 1) * limit
	rows, err := s.DB.QueryContext(ctx, `SELECT id, organization_id, slug, display_name, status, COALESCE(default_region, '')
	FROM tenants
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list manager tenants: %w", err)
	}
	defer rows.Close()

	records := make([]TenantRecord, 0, limit)
	for rows.Next() {
		var rec TenantRecord
		if err := rows.Scan(&rec.ID, &rec.OrganizationID, &rec.Slug, &rec.DisplayName, &rec.Status, &rec.DefaultRegion); err != nil {
			return nil, 0, fmt.Errorf("scan manager tenant: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager tenants: %w", err)
	}
	return records, total, nil
}

func (s SQLAgentStore) GetAgent(ctx context.Context, id string) (*AgentRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	var agent AgentRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id, tenant_id, workspace_id, slug, display_name, description, status, pattern, runtime_engine, runner_class, model_provider, model_name, latest_revision
	FROM agents
	WHERE id = $1`, id).Scan(
		&agent.ID,
		&agent.TenantID,
		&agent.WorkspaceID,
		&agent.Slug,
		&agent.DisplayName,
		&agent.Description,
		&agent.Status,
		&agent.Pattern,
		&agent.RuntimeEngine,
		&agent.RunnerClass,
		&agent.ModelProvider,
		&agent.ModelName,
		&agent.LatestRevision,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get manager agent %q: %w", id, err)
	}
	return &agent, nil
}

func (s SQLAgentStore) ListAgents(ctx context.Context, page, limit int) ([]AgentRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM agents").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager agents: %w", err)
	}
	return s.listAgents(ctx, `SELECT id, tenant_id, workspace_id, slug, display_name, description, status, pattern, runtime_engine, runner_class, model_provider, model_name, latest_revision
	FROM agents
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`, total, page, limit)
}

func (s SQLAgentStore) ListAgentsByTenant(ctx context.Context, tenantID string, page, limit int) ([]AgentRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM agents WHERE tenant_id = $1", tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager agents by tenant %q: %w", tenantID, err)
	}
	return s.listAgents(ctx, `SELECT id, tenant_id, workspace_id, slug, display_name, description, status, pattern, runtime_engine, runner_class, model_provider, model_name, latest_revision
	FROM agents
	WHERE tenant_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, tenantID)
}

func (s SQLAgentStore) ListAgentsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]AgentRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM agents WHERE workspace_id = $1", workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager agents by workspace %q: %w", workspaceID, err)
	}
	return s.listAgents(ctx, `SELECT id, tenant_id, workspace_id, slug, display_name, description, status, pattern, runtime_engine, runner_class, model_provider, model_name, latest_revision
	FROM agents
	WHERE workspace_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, workspaceID)
}

func (s SQLAgentStore) listAgents(ctx context.Context, query string, total, page, limit int, filters ...any) ([]AgentRecord, int, error) {
	offset := (page - 1) * limit
	args := append([]any{}, filters...)
	args = append(args, limit, offset)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list manager agents: %w", err)
	}
	defer rows.Close()

	records := make([]AgentRecord, 0, limit)
	for rows.Next() {
		var rec AgentRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.TenantID,
			&rec.WorkspaceID,
			&rec.Slug,
			&rec.DisplayName,
			&rec.Description,
			&rec.Status,
			&rec.Pattern,
			&rec.RuntimeEngine,
			&rec.RunnerClass,
			&rec.ModelProvider,
			&rec.ModelName,
			&rec.LatestRevision,
		); err != nil {
			return nil, 0, fmt.Errorf("scan manager agent: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager agents: %w", err)
	}
	return records, total, nil
}

func (s SQLEvaluationStore) GetEvaluation(ctx context.Context, id string) (*EvaluationRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	var evaluation EvaluationRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id, tenant_id, workspace_id, agent_id, slug, display_name, description, status, dataset_name, dataset_revision, baseline_revision, score, gate_passed, samples_total, samples_evaluated, latest_run_id, report_ref
	FROM evaluations
	WHERE id = $1`, id).Scan(
		&evaluation.ID, &evaluation.TenantID, &evaluation.WorkspaceID, &evaluation.AgentID,
		&evaluation.Slug, &evaluation.DisplayName, &evaluation.Description, &evaluation.Status,
		&evaluation.DatasetName, &evaluation.DatasetRevision, &evaluation.BaselineRevision,
		&evaluation.Score, &evaluation.GatePassed, &evaluation.SamplesTotal,
		&evaluation.SamplesEvaluated, &evaluation.LatestRunID, &evaluation.ReportRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get manager evaluation %q: %w", id, err)
	}
	return &evaluation, nil
}

func (s SQLEvaluationStore) ListEvaluations(ctx context.Context, page, limit int) ([]EvaluationRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM evaluations").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager evaluations: %w", err)
	}
	return s.listEvaluations(ctx, `SELECT id, tenant_id, workspace_id, agent_id, slug, display_name, description, status, dataset_name, dataset_revision, baseline_revision, score, gate_passed, samples_total, samples_evaluated, latest_run_id, report_ref
	FROM evaluations
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`, total, page, limit)
}

func (s SQLEvaluationStore) ListEvaluationsByTenant(ctx context.Context, tenantID string, page, limit int) ([]EvaluationRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM evaluations WHERE tenant_id = $1", tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager evaluations by tenant %q: %w", tenantID, err)
	}
	return s.listEvaluations(ctx, `SELECT id, tenant_id, workspace_id, agent_id, slug, display_name, description, status, dataset_name, dataset_revision, baseline_revision, score, gate_passed, samples_total, samples_evaluated, latest_run_id, report_ref
	FROM evaluations
	WHERE tenant_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, tenantID)
}

func (s SQLEvaluationStore) ListEvaluationsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]EvaluationRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM evaluations WHERE workspace_id = $1", workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager evaluations by workspace %q: %w", workspaceID, err)
	}
	return s.listEvaluations(ctx, `SELECT id, tenant_id, workspace_id, agent_id, slug, display_name, description, status, dataset_name, dataset_revision, baseline_revision, score, gate_passed, samples_total, samples_evaluated, latest_run_id, report_ref
	FROM evaluations
	WHERE workspace_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, workspaceID)
}

func (s SQLEvaluationStore) listEvaluations(ctx context.Context, query string, total, page, limit int, filters ...any) ([]EvaluationRecord, int, error) {
	offset := (page - 1) * limit
	args := append([]any{}, filters...)
	args = append(args, limit, offset)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list manager evaluations: %w", err)
	}
	defer rows.Close()

	records := make([]EvaluationRecord, 0, limit)
	for rows.Next() {
		var rec EvaluationRecord
		if err := rows.Scan(
			&rec.ID, &rec.TenantID, &rec.WorkspaceID, &rec.AgentID, &rec.Slug,
			&rec.DisplayName, &rec.Description, &rec.Status, &rec.DatasetName,
			&rec.DatasetRevision, &rec.BaselineRevision, &rec.Score, &rec.GatePassed,
			&rec.SamplesTotal, &rec.SamplesEvaluated, &rec.LatestRunID, &rec.ReportRef,
		); err != nil {
			return nil, 0, fmt.Errorf("scan manager evaluation: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager evaluations: %w", err)
	}
	return records, total, nil
}

func (s SQLProviderStore) GetProvider(ctx context.Context, id string) (*ProviderRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	var provider ProviderRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id, tenant_id, workspace_id, provider, display_name, family, base_url, credential_ref, status, domestic, supports_json_schema, supports_tool_calling
	FROM provider_accounts
	WHERE id = $1`, id).Scan(
		&provider.ID,
		&provider.TenantID,
		&provider.WorkspaceID,
		&provider.Provider,
		&provider.DisplayName,
		&provider.Family,
		&provider.BaseURL,
		&provider.CredentialRef,
		&provider.Status,
		&provider.Domestic,
		&provider.SupportsJSONSchema,
		&provider.SupportsToolCalling,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get manager provider %q: %w", id, err)
	}
	return &provider, nil
}

func (s SQLProviderStore) ListProviders(ctx context.Context, page, limit int) ([]ProviderRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM provider_accounts").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager providers: %w", err)
	}
	return s.listProviders(ctx, `SELECT id, tenant_id, workspace_id, provider, display_name, family, base_url, credential_ref, status, domestic, supports_json_schema, supports_tool_calling
	FROM provider_accounts
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`, total, page, limit)
}

func (s SQLProviderStore) ListProvidersByTenant(ctx context.Context, tenantID string, page, limit int) ([]ProviderRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM provider_accounts WHERE tenant_id = $1", tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager providers by tenant %q: %w", tenantID, err)
	}
	return s.listProviders(ctx, `SELECT id, tenant_id, workspace_id, provider, display_name, family, base_url, credential_ref, status, domestic, supports_json_schema, supports_tool_calling
	FROM provider_accounts
	WHERE tenant_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, tenantID)
}

func (s SQLProviderStore) ListProvidersByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]ProviderRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM provider_accounts WHERE workspace_id = $1", workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager providers by workspace %q: %w", workspaceID, err)
	}
	return s.listProviders(ctx, `SELECT id, tenant_id, workspace_id, provider, display_name, family, base_url, credential_ref, status, domestic, supports_json_schema, supports_tool_calling
	FROM provider_accounts
	WHERE workspace_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, workspaceID)
}

func (s SQLProviderStore) listProviders(ctx context.Context, query string, total, page, limit int, filters ...any) ([]ProviderRecord, int, error) {
	offset := (page - 1) * limit
	args := append([]any{}, filters...)
	args = append(args, limit, offset)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list manager providers: %w", err)
	}
	defer rows.Close()

	records := make([]ProviderRecord, 0, limit)
	for rows.Next() {
		var rec ProviderRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.TenantID,
			&rec.WorkspaceID,
			&rec.Provider,
			&rec.DisplayName,
			&rec.Family,
			&rec.BaseURL,
			&rec.CredentialRef,
			&rec.Status,
			&rec.Domestic,
			&rec.SupportsJSONSchema,
			&rec.SupportsToolCalling,
		); err != nil {
			return nil, 0, fmt.Errorf("scan manager provider: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager providers: %w", err)
	}
	return records, total, nil
}

func joinStrings(values []string, separator string) string {
	if len(values) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString(values[0])
	for i := 1; i < len(values); i++ {
		b.WriteString(separator)
		b.WriteString(values[i])
	}
	return b.String()
}
