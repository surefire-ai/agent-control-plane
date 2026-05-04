package manager

import (
	"context"
	"database/sql"
	"encoding/json"
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
	Spec           *AgentSpecData
}

// AgentSpecData mirrors the CRD AgentSpec for Manager-side storage.
type AgentSpecData struct {
	Runtime       RuntimeConfig          `json:"runtime,omitempty"`
	Models        map[string]ModelConfig `json:"models,omitempty"`
	Identity      IdentityConfig         `json:"identity,omitempty"`
	Pattern       *PatternConfig         `json:"pattern,omitempty"`
	PromptRefs    PromptRefsConfig       `json:"promptRefs,omitempty"`
	KnowledgeRefs []KnowledgeBinding     `json:"knowledgeRefs,omitempty"`
	ToolRefs      []string               `json:"toolRefs,omitempty"`
	SkillRefs     []SkillBinding         `json:"skillRefs,omitempty"`
	SubAgentRefs  []SubAgentBinding      `json:"subAgentRefs,omitempty"`
	MCPRefs       []string               `json:"mcpRefs,omitempty"`
	PolicyRef     string                 `json:"policyRef,omitempty"`
	Interfaces    InterfaceConfig        `json:"interfaces,omitempty"`
	Graph         *GraphConfig           `json:"graph,omitempty"`
}

type RuntimeConfig struct {
	Engine      string `json:"engine,omitempty"`
	RunnerClass string `json:"runnerClass,omitempty"`
	Mode        string `json:"mode,omitempty"`
	Entrypoint  string `json:"entrypoint,omitempty"`
}

type ModelConfig struct {
	Provider       string  `json:"provider,omitempty"`
	Model          string  `json:"model,omitempty"`
	BaseURL        string  `json:"baseURL,omitempty"`
	CredentialRef  string  `json:"credentialRef,omitempty"`
	Temperature    float64 `json:"temperature,omitempty"`
	MaxTokens      int32   `json:"maxTokens,omitempty"`
	TimeoutSeconds int32   `json:"timeoutSeconds,omitempty"`
}

type IdentityConfig struct {
	DisplayName string `json:"displayName,omitempty"`
	Role        string `json:"role,omitempty"`
	Description string `json:"description,omitempty"`
}

type PatternConfig struct {
	Type             string         `json:"type,omitempty"`
	Version          string         `json:"version,omitempty"`
	ModelRef         string         `json:"modelRef,omitempty"`
	ExecutorModelRef string         `json:"executorModelRef,omitempty"`
	ToolRefs         []string       `json:"toolRefs,omitempty"`
	KnowledgeRefs    []string       `json:"knowledgeRefs,omitempty"`
	MaxIterations    int32          `json:"maxIterations,omitempty"`
	StopWhen         string         `json:"stopWhen,omitempty"`
	Routes           []PatternRoute `json:"routes,omitempty"`
}

type PatternRoute struct {
	Label    string `json:"label"`
	AgentRef string `json:"agentRef,omitempty"`
	ModelRef string `json:"modelRef,omitempty"`
	Default  bool   `json:"default,omitempty"`
}

type PromptRefsConfig struct {
	System string `json:"system,omitempty"`
}

type KnowledgeBinding struct {
	Name           string  `json:"name"`
	Ref            string  `json:"ref"`
	TopK           int32   `json:"topK,omitempty"`
	ScoreThreshold float64 `json:"scoreThreshold,omitempty"`
}

type SkillBinding struct {
	Name string `json:"name"`
	Ref  string `json:"ref"`
}

type SubAgentBinding struct {
	Name      string `json:"name"`
	Ref       string `json:"ref"`
	Namespace string `json:"namespace,omitempty"`
}

type InterfaceConfig struct {
	Input  SchemaConfig `json:"input,omitempty"`
	Output SchemaConfig `json:"output,omitempty"`
}

type SchemaConfig struct {
	Schema map[string]interface{} `json:"schema,omitempty"`
}

type GraphConfig struct {
	Nodes []GraphNode `json:"nodes,omitempty"`
	Edges []GraphEdge `json:"edges,omitempty"`
}

type GraphNode struct {
	Name           string        `json:"name"`
	Kind           string        `json:"kind"`
	ModelRef       string        `json:"modelRef,omitempty"`
	ToolRef        string        `json:"toolRef,omitempty"`
	KnowledgeRef   string        `json:"knowledgeRef,omitempty"`
	AgentRef       string        `json:"agentRef,omitempty"`
	Implementation string        `json:"implementation,omitempty"`
	Position       *NodePosition `json:"position,omitempty"`
}

type NodePosition struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type GraphEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
	When string `json:"when,omitempty"`
}

// agentColumns is the SELECT column list for agents including spec.
const agentColumns = `id, tenant_id, workspace_id, slug, display_name, description, status, pattern, runtime_engine, runner_class, model_provider, model_name, latest_revision, spec`

func scanAgent(scanner interface{ Scan(...any) error }) (AgentRecord, error) {
	var rec AgentRecord
	var specRaw []byte
	err := scanner.Scan(
		&rec.ID, &rec.TenantID, &rec.WorkspaceID, &rec.Slug,
		&rec.DisplayName, &rec.Description, &rec.Status, &rec.Pattern,
		&rec.RuntimeEngine, &rec.RunnerClass, &rec.ModelProvider,
		&rec.ModelName, &rec.LatestRevision, &specRaw,
	)
	if err != nil {
		return rec, err
	}
	if len(specRaw) > 0 {
		rec.Spec = &AgentSpecData{}
		if err := json.Unmarshal(specRaw, rec.Spec); err != nil {
			return rec, fmt.Errorf("unmarshal agent spec: %w", err)
		}
	}
	return rec, nil
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

type RunRecord struct {
	ID            string
	TenantID      string
	WorkspaceID   string
	AgentID       string
	EvaluationID  string
	AgentRevision string
	Status        string
	RuntimeEngine string
	RunnerClass   string
	StartedAt     string
	CompletedAt   string
	Summary       string
	TraceRef      string
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
	CreateTenant(ctx context.Context, tenant TenantRecord) error
	UpdateTenant(ctx context.Context, id string, fields map[string]string) (*TenantRecord, error)
	DeleteTenant(ctx context.Context, id string) error
}

type AgentStore interface {
	GetAgent(ctx context.Context, id string) (*AgentRecord, error)
	ListAgents(ctx context.Context, page, limit int) ([]AgentRecord, int, error)
	ListAgentsByTenant(ctx context.Context, tenantID string, page, limit int) ([]AgentRecord, int, error)
	ListAgentsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]AgentRecord, int, error)
	CreateAgent(ctx context.Context, agent AgentRecord) error
	UpdateAgent(ctx context.Context, id string, fields map[string]string, spec *AgentSpecData) (*AgentRecord, error)
	DeleteAgent(ctx context.Context, id string) error
}

type EvaluationStore interface {
	GetEvaluation(ctx context.Context, id string) (*EvaluationRecord, error)
	ListEvaluations(ctx context.Context, page, limit int) ([]EvaluationRecord, int, error)
	ListEvaluationsByTenant(ctx context.Context, tenantID string, page, limit int) ([]EvaluationRecord, int, error)
	ListEvaluationsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]EvaluationRecord, int, error)
	ListEvaluationsByAgent(ctx context.Context, agentID string, page, limit int) ([]EvaluationRecord, int, error)
	CreateEvaluation(ctx context.Context, evaluation EvaluationRecord) error
	UpdateEvaluation(ctx context.Context, id string, fields map[string]string) (*EvaluationRecord, error)
	DeleteEvaluation(ctx context.Context, id string) error
}

type ProviderStore interface {
	GetProvider(ctx context.Context, id string) (*ProviderRecord, error)
	ListProviders(ctx context.Context, page, limit int) ([]ProviderRecord, int, error)
	ListProvidersByTenant(ctx context.Context, tenantID string, page, limit int) ([]ProviderRecord, int, error)
	ListProvidersByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]ProviderRecord, int, error)
	CreateProvider(ctx context.Context, provider ProviderRecord) error
	UpdateProvider(ctx context.Context, id string, fields map[string]string) (*ProviderRecord, error)
	DeleteProvider(ctx context.Context, id string) error
}

type RunStore interface {
	GetRun(ctx context.Context, id string) (*RunRecord, error)
	ListRuns(ctx context.Context, page, limit int) ([]RunRecord, int, error)
	ListRunsByTenant(ctx context.Context, tenantID string, page, limit int) ([]RunRecord, int, error)
	ListRunsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]RunRecord, int, error)
	ListRunsByAgent(ctx context.Context, agentID string, page, limit int) ([]RunRecord, int, error)
	ListRunsByEvaluation(ctx context.Context, evaluationID string, page, limit int) ([]RunRecord, int, error)
	CreateRun(ctx context.Context, run RunRecord) error
	UpdateRun(ctx context.Context, id string, fields map[string]string) (*RunRecord, error)
	DeleteRun(ctx context.Context, id string) error
}

type Stores struct {
	Workspaces  WorkspaceStore
	Tenants     TenantStore
	Agents      AgentStore
	Evaluations EvaluationStore
	Providers   ProviderStore
	Runs        RunStore
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

type SQLRunStore struct {
	DB *sql.DB
}

func NewSQLStores(db *sql.DB) Stores {
	return Stores{
		Workspaces:  SQLWorkspaceStore{DB: db},
		Tenants:     SQLTenantStore{DB: db},
		Agents:      SQLAgentStore{DB: db},
		Evaluations: SQLEvaluationStore{DB: db},
		Providers:   SQLProviderStore{DB: db},
		Runs:        SQLRunStore{DB: db},
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

var tenantUpdatableColumns = map[string]string{
	"display_name":   "display_name",
	"status":         "status",
	"default_region": "default_region",
}

func (s SQLTenantStore) CreateTenant(ctx context.Context, tenant TenantRecord) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO tenants (id, organization_id, slug, display_name, status, default_region)
	VALUES ($1, $2, $3, $4, $5, $6)`,
		tenant.ID, tenant.OrganizationID, tenant.Slug, tenant.DisplayName,
		tenant.Status, tenant.DefaultRegion,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return fmt.Errorf("create manager tenant %q: %w", tenant.ID, err)
	}
	return nil
}

func (s SQLTenantStore) UpdateTenant(ctx context.Context, id string, fields map[string]string) (*TenantRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	columns := make([]string, 0, len(fields))
	values := make([]any, 0, len(fields)+1)
	values = append(values, id)
	idx := 2
	for key, value := range fields {
		col, ok := tenantUpdatableColumns[key]
		if !ok {
			continue
		}
		columns = append(columns, fmt.Sprintf("%s = $%d", col, idx))
		values = append(values, value)
		idx++
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid fields to update for tenant %q", id)
	}

	query := fmt.Sprintf(`UPDATE tenants SET %s, updated_at = now()
	WHERE id = $1
	RETURNING id, organization_id, slug, display_name, status, COALESCE(default_region, '')`,
		joinStrings(columns, ", "))

	var tenant TenantRecord
	err := s.DB.QueryRowContext(ctx, query, values...).Scan(
		&tenant.ID, &tenant.OrganizationID, &tenant.Slug, &tenant.DisplayName,
		&tenant.Status, &tenant.DefaultRegion,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update manager tenant %q: %w", id, err)
	}
	return &tenant, nil
}

func (s SQLTenantStore) DeleteTenant(ctx context.Context, id string) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM tenants WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete manager tenant %q: %w", id, err)
	}
	return nil
}

func (s SQLAgentStore) GetAgent(ctx context.Context, id string) (*AgentRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	rec, err := scanAgent(s.DB.QueryRowContext(ctx,
		fmt.Sprintf("SELECT %s FROM agents WHERE id = $1", agentColumns), id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get manager agent %q: %w", id, err)
	}
	return &rec, nil
}

func (s SQLAgentStore) ListAgents(ctx context.Context, page, limit int) ([]AgentRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM agents").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager agents: %w", err)
	}
	return s.listAgents(ctx, fmt.Sprintf(`SELECT %s
	FROM agents
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`, agentColumns), total, page, limit)
}

func (s SQLAgentStore) ListAgentsByTenant(ctx context.Context, tenantID string, page, limit int) ([]AgentRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM agents WHERE tenant_id = $1", tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager agents by tenant %q: %w", tenantID, err)
	}
	return s.listAgents(ctx, fmt.Sprintf(`SELECT %s
	FROM agents
	WHERE tenant_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, agentColumns), total, page, limit, tenantID)
}

func (s SQLAgentStore) ListAgentsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]AgentRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM agents WHERE workspace_id = $1", workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager agents by workspace %q: %w", workspaceID, err)
	}
	return s.listAgents(ctx, fmt.Sprintf(`SELECT %s
	FROM agents
	WHERE workspace_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, agentColumns), total, page, limit, workspaceID)
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
		rec, err := scanAgent(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("scan manager agent: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager agents: %w", err)
	}
	return records, total, nil
}

var agentUpdatableColumns = map[string]string{
	"display_name":    "display_name",
	"description":     "description",
	"status":          "status",
	"pattern":         "pattern",
	"runtime_engine":  "runtime_engine",
	"runner_class":    "runner_class",
	"model_provider":  "model_provider",
	"model_name":      "model_name",
	"latest_revision": "latest_revision",
}

func (s SQLAgentStore) CreateAgent(ctx context.Context, agent AgentRecord) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	var specJSON []byte
	if agent.Spec != nil {
		var err error
		specJSON, err = json.Marshal(agent.Spec)
		if err != nil {
			return fmt.Errorf("marshal agent spec: %w", err)
		}
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO agents (id, tenant_id, workspace_id, slug, display_name, description, status, pattern, runtime_engine, runner_class, model_provider, model_name, latest_revision, spec)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
		agent.ID, agent.TenantID, agent.WorkspaceID, agent.Slug,
		agent.DisplayName, agent.Description, agent.Status, agent.Pattern,
		agent.RuntimeEngine, agent.RunnerClass, agent.ModelProvider,
		agent.ModelName, agent.LatestRevision, specJSON,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return fmt.Errorf("create manager agent %q: %w", agent.ID, err)
	}
	return nil
}

func (s SQLAgentStore) UpdateAgent(ctx context.Context, id string, fields map[string]string, spec *AgentSpecData) (*AgentRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	columns := make([]string, 0, len(fields)+1)
	values := make([]any, 0, len(fields)+2)
	values = append(values, id)
	idx := 2
	for key, value := range fields {
		col, ok := agentUpdatableColumns[key]
		if !ok {
			continue
		}
		columns = append(columns, fmt.Sprintf("%s = $%d", col, idx))
		values = append(values, value)
		idx++
	}
	if spec != nil {
		specJSON, err := json.Marshal(spec)
		if err != nil {
			return nil, fmt.Errorf("marshal agent spec: %w", err)
		}
		columns = append(columns, fmt.Sprintf("spec = $%d", idx))
		values = append(values, specJSON)
		idx++
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid fields to update for agent %q", id)
	}

	query := fmt.Sprintf(`UPDATE agents SET %s, updated_at = now()
	WHERE id = $1
	RETURNING %s`,
		joinStrings(columns, ", "), agentColumns)

	rec, err := scanAgent(s.DB.QueryRowContext(ctx, query, values...))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update manager agent %q: %w", id, err)
	}
	return &rec, nil
}

func (s SQLAgentStore) DeleteAgent(ctx context.Context, id string) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM agents WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete manager agent %q: %w", id, err)
	}
	return nil
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

func (s SQLEvaluationStore) ListEvaluationsByAgent(ctx context.Context, agentID string, page, limit int) ([]EvaluationRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM evaluations WHERE agent_id = $1", agentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager evaluations by agent %q: %w", agentID, err)
	}
	return s.listEvaluations(ctx, `SELECT id, tenant_id, workspace_id, agent_id, slug, display_name, description, status, dataset_name, dataset_revision, baseline_revision, score, gate_passed, samples_total, samples_evaluated, latest_run_id, report_ref
	FROM evaluations
	WHERE agent_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, agentID)
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

var evaluationUpdatableColumns = map[string]string{
	"display_name":      "display_name",
	"description":       "description",
	"status":            "status",
	"dataset_name":      "dataset_name",
	"dataset_revision":  "dataset_revision",
	"baseline_revision": "baseline_revision",
	"score":             "score",
	"gate_passed":       "gate_passed",
	"samples_total":     "samples_total",
	"samples_evaluated": "samples_evaluated",
	"latest_run_id":     "latest_run_id",
	"report_ref":        "report_ref",
}

func (s SQLEvaluationStore) CreateEvaluation(ctx context.Context, evaluation EvaluationRecord) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO evaluations (id, tenant_id, workspace_id, agent_id, slug, display_name, description, status, dataset_name, dataset_revision, baseline_revision, score, gate_passed, samples_total, samples_evaluated, latest_run_id, report_ref)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)`,
		evaluation.ID, evaluation.TenantID, evaluation.WorkspaceID, evaluation.AgentID,
		evaluation.Slug, evaluation.DisplayName, evaluation.Description, evaluation.Status,
		evaluation.DatasetName, evaluation.DatasetRevision, evaluation.BaselineRevision,
		evaluation.Score, evaluation.GatePassed, evaluation.SamplesTotal,
		evaluation.SamplesEvaluated, evaluation.LatestRunID, evaluation.ReportRef,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return fmt.Errorf("create manager evaluation %q: %w", evaluation.ID, err)
	}
	return nil
}

func (s SQLEvaluationStore) UpdateEvaluation(ctx context.Context, id string, fields map[string]string) (*EvaluationRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	columns := make([]string, 0, len(fields))
	values := make([]any, 0, len(fields)+1)
	values = append(values, id)
	idx := 2
	for key, value := range fields {
		col, ok := evaluationUpdatableColumns[key]
		if !ok {
			continue
		}
		columns = append(columns, fmt.Sprintf("%s = $%d", col, idx))
		values = append(values, value)
		idx++
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid fields to update for evaluation %q", id)
	}

	query := fmt.Sprintf(`UPDATE evaluations SET %s, updated_at = now()
	WHERE id = $1
	RETURNING id, tenant_id, workspace_id, agent_id, slug, display_name, description, status, dataset_name, dataset_revision, baseline_revision, score, gate_passed, samples_total, samples_evaluated, latest_run_id, report_ref`,
		joinStrings(columns, ", "))

	var evaluation EvaluationRecord
	err := s.DB.QueryRowContext(ctx, query, values...).Scan(
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
		return nil, fmt.Errorf("update manager evaluation %q: %w", id, err)
	}
	return &evaluation, nil
}

func (s SQLEvaluationStore) DeleteEvaluation(ctx context.Context, id string) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM evaluations WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete manager evaluation %q: %w", id, err)
	}
	return nil
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

var providerUpdatableColumns = map[string]string{
	"display_name":          "display_name",
	"family":                "family",
	"base_url":              "base_url",
	"credential_ref":        "credential_ref",
	"status":                "status",
	"domestic":              "domestic",
	"supports_json_schema":  "supports_json_schema",
	"supports_tool_calling": "supports_tool_calling",
}

func (s SQLProviderStore) CreateProvider(ctx context.Context, provider ProviderRecord) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO provider_accounts (id, tenant_id, workspace_id, provider, display_name, family, base_url, credential_ref, status, domestic, supports_json_schema, supports_tool_calling)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
		provider.ID, provider.TenantID, provider.WorkspaceID, provider.Provider,
		provider.DisplayName, provider.Family, provider.BaseURL, provider.CredentialRef,
		provider.Status, provider.Domestic, provider.SupportsJSONSchema, provider.SupportsToolCalling,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return fmt.Errorf("create manager provider %q: %w", provider.ID, err)
	}
	return nil
}

func (s SQLProviderStore) UpdateProvider(ctx context.Context, id string, fields map[string]string) (*ProviderRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	columns := make([]string, 0, len(fields))
	values := make([]any, 0, len(fields)+1)
	values = append(values, id)
	idx := 2
	for key, value := range fields {
		col, ok := providerUpdatableColumns[key]
		if !ok {
			continue
		}
		columns = append(columns, fmt.Sprintf("%s = $%d", col, idx))
		values = append(values, value)
		idx++
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid fields to update for provider %q", id)
	}

	query := fmt.Sprintf(`UPDATE provider_accounts SET %s, updated_at = now()
	WHERE id = $1
	RETURNING id, tenant_id, workspace_id, provider, display_name, family, base_url, credential_ref, status, domestic, supports_json_schema, supports_tool_calling`,
		joinStrings(columns, ", "))

	var provider ProviderRecord
	err := s.DB.QueryRowContext(ctx, query, values...).Scan(
		&provider.ID, &provider.TenantID, &provider.WorkspaceID, &provider.Provider,
		&provider.DisplayName, &provider.Family, &provider.BaseURL, &provider.CredentialRef,
		&provider.Status, &provider.Domestic, &provider.SupportsJSONSchema, &provider.SupportsToolCalling,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update manager provider %q: %w", id, err)
	}
	return &provider, nil
}

func (s SQLProviderStore) DeleteProvider(ctx context.Context, id string) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM provider_accounts WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete manager provider %q: %w", id, err)
	}
	return nil
}

func (s SQLRunStore) GetRun(ctx context.Context, id string) (*RunRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	var run RunRecord
	err := s.DB.QueryRowContext(ctx, `SELECT id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref
	FROM runs
	WHERE id = $1`, id).Scan(
		&run.ID,
		&run.TenantID,
		&run.WorkspaceID,
		&run.AgentID,
		&run.EvaluationID,
		&run.AgentRevision,
		&run.Status,
		&run.RuntimeEngine,
		&run.RunnerClass,
		&run.StartedAt,
		&run.CompletedAt,
		&run.Summary,
		&run.TraceRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get manager run %q: %w", id, err)
	}
	return &run, nil
}

func (s SQLRunStore) ListRuns(ctx context.Context, page, limit int) ([]RunRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM runs").Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager runs: %w", err)
	}
	return s.listRuns(ctx, `SELECT id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref
	FROM runs
	ORDER BY created_at DESC
	LIMIT $1 OFFSET $2`, total, page, limit)
}

func (s SQLRunStore) ListRunsByTenant(ctx context.Context, tenantID string, page, limit int) ([]RunRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM runs WHERE tenant_id = $1", tenantID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager runs by tenant %q: %w", tenantID, err)
	}
	return s.listRuns(ctx, `SELECT id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref
	FROM runs
	WHERE tenant_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, tenantID)
}

func (s SQLRunStore) ListRunsByWorkspace(ctx context.Context, workspaceID string, page, limit int) ([]RunRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM runs WHERE workspace_id = $1", workspaceID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager runs by workspace %q: %w", workspaceID, err)
	}
	return s.listRuns(ctx, `SELECT id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref
	FROM runs
	WHERE workspace_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, workspaceID)
}

func (s SQLRunStore) ListRunsByAgent(ctx context.Context, agentID string, page, limit int) ([]RunRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM runs WHERE agent_id = $1", agentID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager runs by agent %q: %w", agentID, err)
	}
	return s.listRuns(ctx, `SELECT id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref
	FROM runs
	WHERE agent_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, agentID)
}

func (s SQLRunStore) ListRunsByEvaluation(ctx context.Context, evaluationID string, page, limit int) ([]RunRecord, int, error) {
	if s.DB == nil {
		return nil, 0, fmt.Errorf("manager database is required")
	}
	var total int
	if err := s.DB.QueryRowContext(ctx, "SELECT COUNT(*) FROM runs WHERE evaluation_id = $1", evaluationID).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count manager runs by evaluation %q: %w", evaluationID, err)
	}
	return s.listRuns(ctx, `SELECT id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref
	FROM runs
	WHERE evaluation_id = $1
	ORDER BY created_at DESC
	LIMIT $2 OFFSET $3`, total, page, limit, evaluationID)
}

func (s SQLRunStore) listRuns(ctx context.Context, query string, total, page, limit int, filters ...any) ([]RunRecord, int, error) {
	offset := (page - 1) * limit
	args := append([]any{}, filters...)
	args = append(args, limit, offset)
	rows, err := s.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("list manager runs: %w", err)
	}
	defer rows.Close()

	records := make([]RunRecord, 0, limit)
	for rows.Next() {
		var rec RunRecord
		if err := rows.Scan(
			&rec.ID,
			&rec.TenantID,
			&rec.WorkspaceID,
			&rec.AgentID,
			&rec.EvaluationID,
			&rec.AgentRevision,
			&rec.Status,
			&rec.RuntimeEngine,
			&rec.RunnerClass,
			&rec.StartedAt,
			&rec.CompletedAt,
			&rec.Summary,
			&rec.TraceRef,
		); err != nil {
			return nil, 0, fmt.Errorf("scan manager run: %w", err)
		}
		records = append(records, rec)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate manager runs: %w", err)
	}
	return records, total, nil
}

var runUpdatableColumns = map[string]string{
	"status":       "status",
	"started_at":   "started_at",
	"completed_at": "completed_at",
	"summary":      "summary",
	"trace_ref":    "trace_ref",
}

func (s SQLRunStore) CreateRun(ctx context.Context, run RunRecord) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, `INSERT INTO runs (id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		run.ID, run.TenantID, run.WorkspaceID, run.AgentID, run.EvaluationID,
		run.AgentRevision, run.Status, run.RuntimeEngine, run.RunnerClass,
		run.StartedAt, run.CompletedAt, run.Summary, run.TraceRef,
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return ErrConflict
		}
		return fmt.Errorf("create manager run %q: %w", run.ID, err)
	}
	return nil
}

func (s SQLRunStore) UpdateRun(ctx context.Context, id string, fields map[string]string) (*RunRecord, error) {
	if s.DB == nil {
		return nil, fmt.Errorf("manager database is required")
	}
	columns := make([]string, 0, len(fields))
	values := make([]any, 0, len(fields)+1)
	values = append(values, id)
	idx := 2
	for key, value := range fields {
		col, ok := runUpdatableColumns[key]
		if !ok {
			continue
		}
		columns = append(columns, fmt.Sprintf("%s = $%d", col, idx))
		values = append(values, value)
		idx++
	}
	if len(columns) == 0 {
		return nil, fmt.Errorf("no valid fields to update for run %q", id)
	}

	query := fmt.Sprintf(`UPDATE runs SET %s, updated_at = now()
	WHERE id = $1
	RETURNING id, tenant_id, workspace_id, agent_id, evaluation_id, agent_revision, status, runtime_engine, runner_class, started_at, completed_at, summary, trace_ref`,
		joinStrings(columns, ", "))

	var run RunRecord
	err := s.DB.QueryRowContext(ctx, query, values...).Scan(
		&run.ID, &run.TenantID, &run.WorkspaceID, &run.AgentID, &run.EvaluationID,
		&run.AgentRevision, &run.Status, &run.RuntimeEngine, &run.RunnerClass,
		&run.StartedAt, &run.CompletedAt, &run.Summary, &run.TraceRef,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("update manager run %q: %w", id, err)
	}
	return &run, nil
}

func (s SQLRunStore) DeleteRun(ctx context.Context, id string) error {
	if s.DB == nil {
		return fmt.Errorf("manager database is required")
	}
	_, err := s.DB.ExecContext(ctx, "DELETE FROM runs WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("delete manager run %q: %w", id, err)
	}
	return nil
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
