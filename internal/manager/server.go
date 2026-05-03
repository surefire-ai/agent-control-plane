package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	Config Config
	Stores Stores
	Syncer CRDSyncer
}

type InfoResponse struct {
	Component          string `json:"component"`
	Mode               string `json:"mode"`
	DatabaseConfigured bool   `json:"databaseConfigured"`
	DatabaseDriver     string `json:"databaseDriver,omitempty"`
	DatabaseStatus     string `json:"databaseStatus"`
	MigrateOnStart     bool   `json:"migrateOnStart"`
}

type WorkspaceResponse struct {
	ID                      string `json:"id"`
	TenantID                string `json:"tenantId"`
	Slug                    string `json:"slug"`
	DisplayName             string `json:"displayName"`
	Description             string `json:"description,omitempty"`
	Status                  string `json:"status"`
	KubernetesNamespace     string `json:"kubernetesNamespace,omitempty"`
	KubernetesWorkspaceName string `json:"kubernetesWorkspaceName,omitempty"`
}

type TenantResponse struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Slug           string `json:"slug"`
	DisplayName    string `json:"displayName"`
	Status         string `json:"status"`
	DefaultRegion  string `json:"defaultRegion,omitempty"`
}

type AgentResponse struct {
	ID             string `json:"id"`
	TenantID       string `json:"tenantId"`
	WorkspaceID    string `json:"workspaceId"`
	Slug           string `json:"slug"`
	DisplayName    string `json:"displayName"`
	Description    string `json:"description,omitempty"`
	Status         string `json:"status"`
	Pattern        string `json:"pattern"`
	RuntimeEngine  string `json:"runtimeEngine"`
	RunnerClass    string `json:"runnerClass"`
	ModelProvider  string `json:"modelProvider,omitempty"`
	ModelName      string `json:"modelName,omitempty"`
	LatestRevision string `json:"latestRevision,omitempty"`
}

type EvaluationResponse struct {
	ID               string  `json:"id"`
	TenantID         string  `json:"tenantId"`
	WorkspaceID      string  `json:"workspaceId"`
	AgentID          string  `json:"agentId"`
	Slug             string  `json:"slug"`
	DisplayName      string  `json:"displayName"`
	Description      string  `json:"description,omitempty"`
	Status           string  `json:"status"`
	DatasetName      string  `json:"datasetName"`
	DatasetRevision  string  `json:"datasetRevision,omitempty"`
	BaselineRevision string  `json:"baselineRevision,omitempty"`
	Score            float64 `json:"score"`
	GatePassed       bool    `json:"gatePassed"`
	SamplesTotal     int     `json:"samplesTotal"`
	SamplesEvaluated int     `json:"samplesEvaluated"`
	LatestRunID      string  `json:"latestRunId,omitempty"`
	ReportRef        string  `json:"reportRef,omitempty"`
}

type ProviderResponse struct {
	ID                  string `json:"id"`
	TenantID            string `json:"tenantId"`
	WorkspaceID         string `json:"workspaceId,omitempty"`
	Provider            string `json:"provider"`
	DisplayName         string `json:"displayName"`
	Family              string `json:"family"`
	BaseURL             string `json:"baseUrl,omitempty"`
	CredentialRef       string `json:"credentialRef,omitempty"`
	Status              string `json:"status"`
	Domestic            bool   `json:"domestic"`
	SupportsJSONSchema  bool   `json:"supportsJsonSchema"`
	SupportsToolCalling bool   `json:"supportsToolCalling"`
}

type RunResponse struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenantId"`
	WorkspaceID   string `json:"workspaceId"`
	AgentID       string `json:"agentId"`
	EvaluationID  string `json:"evaluationId,omitempty"`
	AgentRevision string `json:"agentRevision,omitempty"`
	Status        string `json:"status"`
	RuntimeEngine string `json:"runtimeEngine"`
	RunnerClass   string `json:"runnerClass"`
	StartedAt     string `json:"startedAt,omitempty"`
	CompletedAt   string `json:"completedAt,omitempty"`
	Summary       string `json:"summary,omitempty"`
	TraceRef      string `json:"traceRef,omitempty"`
}

type PaginatedWorkspacesResponse struct {
	Workspaces []WorkspaceResponse `json:"workspaces"`
	Page       int                 `json:"page"`
	Limit      int                 `json:"limit"`
	Total      int                 `json:"total"`
}

type PaginatedTenantsResponse struct {
	Tenants []TenantResponse `json:"tenants"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
	Total   int              `json:"total"`
}

type PaginatedAgentsResponse struct {
	Agents []AgentResponse `json:"agents"`
	Page   int             `json:"page"`
	Limit  int             `json:"limit"`
	Total  int             `json:"total"`
}

type PaginatedEvaluationsResponse struct {
	Evaluations []EvaluationResponse `json:"evaluations"`
	Page        int                  `json:"page"`
	Limit       int                  `json:"limit"`
	Total       int                  `json:"total"`
}

type PaginatedProvidersResponse struct {
	Providers []ProviderResponse `json:"providers"`
	Page      int                `json:"page"`
	Limit     int                `json:"limit"`
	Total     int                `json:"total"`
}

type PaginatedRunsResponse struct {
	Runs  []RunResponse `json:"runs"`
	Page  int           `json:"page"`
	Limit int           `json:"limit"`
	Total int           `json:"total"`
}

type CreateWorkspaceRequest struct {
	ID                      string `json:"id"`
	TenantID                string `json:"tenantId"`
	Slug                    string `json:"slug"`
	DisplayName             string `json:"displayName"`
	Description             string `json:"description,omitempty"`
	Status                  string `json:"status,omitempty"`
	KubernetesNamespace     string `json:"kubernetesNamespace,omitempty"`
	KubernetesWorkspaceName string `json:"kubernetesWorkspaceName,omitempty"`
}

type UpdateWorkspaceRequest struct {
	DisplayName             *string `json:"displayName,omitempty"`
	Description             *string `json:"description,omitempty"`
	Status                  *string `json:"status,omitempty"`
	KubernetesNamespace     *string `json:"kubernetesNamespace,omitempty"`
	KubernetesWorkspaceName *string `json:"kubernetesWorkspaceName,omitempty"`
}

type CreateTenantRequest struct {
	ID             string `json:"id"`
	OrganizationID string `json:"organizationId"`
	Slug           string `json:"slug"`
	DisplayName    string `json:"displayName"`
	Status         string `json:"status,omitempty"`
	DefaultRegion  string `json:"defaultRegion,omitempty"`
}

type UpdateTenantRequest struct {
	DisplayName   *string `json:"displayName,omitempty"`
	Status        *string `json:"status,omitempty"`
	DefaultRegion *string `json:"defaultRegion,omitempty"`
}

type CreateAgentRequest struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenantId"`
	WorkspaceID   string `json:"workspaceId"`
	Slug          string `json:"slug"`
	DisplayName   string `json:"displayName"`
	Description   string `json:"description,omitempty"`
	Status        string `json:"status,omitempty"`
	Pattern       string `json:"pattern,omitempty"`
	RuntimeEngine string `json:"runtimeEngine,omitempty"`
	RunnerClass   string `json:"runnerClass,omitempty"`
	ModelProvider string `json:"modelProvider,omitempty"`
	ModelName     string `json:"modelName,omitempty"`
}

type UpdateAgentRequest struct {
	DisplayName   *string `json:"displayName,omitempty"`
	Description   *string `json:"description,omitempty"`
	Status        *string `json:"status,omitempty"`
	Pattern       *string `json:"pattern,omitempty"`
	RuntimeEngine *string `json:"runtimeEngine,omitempty"`
	RunnerClass   *string `json:"runnerClass,omitempty"`
	ModelProvider *string `json:"modelProvider,omitempty"`
	ModelName     *string `json:"modelName,omitempty"`
}

type CreateEvaluationRequest struct {
	ID               string `json:"id"`
	TenantID         string `json:"tenantId"`
	WorkspaceID      string `json:"workspaceId"`
	AgentID          string `json:"agentId"`
	Slug             string `json:"slug"`
	DisplayName      string `json:"displayName"`
	Description      string `json:"description,omitempty"`
	Status           string `json:"status,omitempty"`
	DatasetName      string `json:"datasetName,omitempty"`
	DatasetRevision  string `json:"datasetRevision,omitempty"`
	BaselineRevision string `json:"baselineRevision,omitempty"`
}

type UpdateEvaluationRequest struct {
	DisplayName      *string  `json:"displayName,omitempty"`
	Description      *string  `json:"description,omitempty"`
	Status           *string  `json:"status,omitempty"`
	DatasetName      *string  `json:"datasetName,omitempty"`
	DatasetRevision  *string  `json:"datasetRevision,omitempty"`
	BaselineRevision *string  `json:"baselineRevision,omitempty"`
	Score            *float64 `json:"score,omitempty"`
	GatePassed       *bool    `json:"gatePassed,omitempty"`
	SamplesTotal     *int     `json:"samplesTotal,omitempty"`
	SamplesEvaluated *int     `json:"samplesEvaluated,omitempty"`
	LatestRunID      *string  `json:"latestRunId,omitempty"`
	ReportRef        *string  `json:"reportRef,omitempty"`
}

type CreateProviderRequest struct {
	ID                  string `json:"id"`
	TenantID            string `json:"tenantId"`
	WorkspaceID         string `json:"workspaceId,omitempty"`
	Provider            string `json:"provider"`
	DisplayName         string `json:"displayName"`
	Family              string `json:"family,omitempty"`
	BaseURL             string `json:"baseUrl,omitempty"`
	CredentialRef       string `json:"credentialRef,omitempty"`
	Status              string `json:"status,omitempty"`
	Domestic            bool   `json:"domestic,omitempty"`
	SupportsJSONSchema  bool   `json:"supportsJsonSchema,omitempty"`
	SupportsToolCalling bool   `json:"supportsToolCalling,omitempty"`
}

type UpdateProviderRequest struct {
	DisplayName         *string `json:"displayName,omitempty"`
	Family              *string `json:"family,omitempty"`
	BaseURL             *string `json:"baseUrl,omitempty"`
	CredentialRef       *string `json:"credentialRef,omitempty"`
	Status              *string `json:"status,omitempty"`
	Domestic            *bool   `json:"domestic,omitempty"`
	SupportsJSONSchema  *bool   `json:"supportsJsonSchema,omitempty"`
	SupportsToolCalling *bool   `json:"supportsToolCalling,omitempty"`
}

type CreateRunRequest struct {
	ID            string `json:"id"`
	TenantID      string `json:"tenantId"`
	WorkspaceID   string `json:"workspaceId"`
	AgentID       string `json:"agentId"`
	EvaluationID  string `json:"evaluationId,omitempty"`
	AgentRevision string `json:"agentRevision,omitempty"`
	Status        string `json:"status,omitempty"`
	RuntimeEngine string `json:"runtimeEngine,omitempty"`
	RunnerClass   string `json:"runnerClass,omitempty"`
}

type UpdateRunRequest struct {
	Status      *string `json:"status,omitempty"`
	StartedAt   *string `json:"startedAt,omitempty"`
	CompletedAt *string `json:"completedAt,omitempty"`
	Summary     *string `json:"summary,omitempty"`
	TraceRef    *string `json:"traceRef,omitempty"`
}

func (s Server) syncer() CRDSyncer {
	if s.Syncer == nil {
		return NoopCRDSyncer{}
	}
	return s.Syncer
}

func (s Server) Start(ctx context.Context) error {
	config := s.Config.normalized()
	database, err := OpenDatabase(ctx, config)
	if err != nil {
		return err
	}
	defer func() {
		_ = database.Close()
	}()
	if database != nil && config.AutoMigrate {
		if _, err := database.ApplyBuiltInMigrations(ctx); err != nil {
			return err
		}
	}
	if database != nil && s.Stores.Workspaces == nil {
		s.Stores = NewSQLStores(database.DB)
	}

	server := &http.Server{
		Addr:              config.Addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", config.Addr)
	if err != nil {
		return err
	}

	errCh := make(chan error, 1)
	go func() {
		errCh <- server.Serve(listener)
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			return err
		}
		err := <-errCh
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

func (s Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/readyz", s.handleReady)
	mux.HandleFunc("/api/v1/info", s.handleInfo)
	mux.HandleFunc("/api/v1/workspaces/", s.handleWorkspace)
	mux.HandleFunc("/api/v1/tenants/", s.handleTenant)
	mux.HandleFunc("/api/v1/agents/", s.handleAgent)
	mux.HandleFunc("/api/v1/evaluations/", s.handleEvaluation)
	mux.HandleFunc("/api/v1/providers/", s.handleProvider)
	mux.HandleFunc("/api/v1/runs/", s.handleRun)
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method must be GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s Server) handleReady(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method must be GET")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (s Server) handleInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method must be GET")
		return
	}
	config := s.Config.normalized()
	databaseStatus := "not_configured"
	if config.DatabaseURL != "" {
		databaseStatus = "configured"
	}
	writeJSON(w, http.StatusOK, InfoResponse{
		Component:          "manager",
		Mode:               config.Mode,
		DatabaseConfigured: config.DatabaseURL != "",
		DatabaseDriver:     config.DatabaseDriver,
		DatabaseStatus:     databaseStatus,
		MigrateOnStart:     config.AutoMigrate,
	})
}

func (s Server) handleWorkspace(w http.ResponseWriter, r *http.Request) {
	if s.Stores.Workspaces == nil {
		writeError(w, http.StatusServiceUnavailable, "workspace store is not configured")
		return
	}
	workspaceID := strings.TrimPrefix(r.URL.Path, "/api/v1/workspaces/")
	workspaceID = strings.TrimSpace(workspaceID)

	if workspaceID == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleListWorkspaces(w, r)
		case http.MethodPost:
			s.handleCreateWorkspace(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method must be GET or POST")
		}
		return
	}

	if strings.Contains(workspaceID, "/") {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetWorkspace(w, r, workspaceID)
	case http.MethodPatch:
		s.handleUpdateWorkspace(w, r, workspaceID)
	case http.MethodDelete:
		s.handleDeleteWorkspace(w, r, workspaceID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method must be GET, PATCH, or DELETE")
	}
}

func (s Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationFromQuery(r)
	tenantID := r.URL.Query().Get("tenantId")
	var records []WorkspaceRecord
	var total int
	var err error
	if tenantID != "" {
		records, total, err = s.Stores.Workspaces.ListWorkspacesByTenant(r.Context(), tenantID, page, limit)
	} else {
		records, total, err = s.Stores.Workspaces.ListWorkspaces(r.Context(), page, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}
	workspaces := make([]WorkspaceResponse, 0, len(records))
	for _, rec := range records {
		workspaces = append(workspaces, workspaceResponseFromRecord(rec))
	}
	writeJSON(w, http.StatusOK, PaginatedWorkspacesResponse{
		Workspaces: workspaces,
		Page:       page,
		Limit:      limit,
		Total:      total,
	})
}

func (s Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	var req CreateWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" || req.TenantID == "" || req.Slug == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "id, tenantId, slug, and displayName are required")
		return
	}
	record := WorkspaceRecord{
		ID:                      req.ID,
		TenantID:                req.TenantID,
		Slug:                    req.Slug,
		DisplayName:             req.DisplayName,
		Description:             req.Description,
		Status:                  req.Status,
		KubernetesNamespace:     req.KubernetesNamespace,
		KubernetesWorkspaceName: req.KubernetesWorkspaceName,
	}
	if record.Status == "" {
		record.Status = "active"
	}
	if err := s.Stores.Workspaces.CreateWorkspace(r.Context(), record); err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "workspace already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	if err := s.syncer().SyncWorkspace(r.Context(), record); err != nil {
		log.Printf("syncer: failed to sync workspace %s: %v", record.ID, err)
	}
	writeJSON(w, http.StatusCreated, workspaceResponseFromRecord(record))
}

func (s Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) {
	workspace, err := s.Stores.Workspaces.GetWorkspace(r.Context(), workspaceID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read workspace")
		return
	}
	writeJSON(w, http.StatusOK, workspaceResponseFromRecord(*workspace))
}

func (s Server) handleUpdateWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) {
	var req UpdateWorkspaceRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fields := map[string]string{}
	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.KubernetesNamespace != nil {
		fields["kubernetes_namespace"] = *req.KubernetesNamespace
	}
	if req.KubernetesWorkspaceName != nil {
		fields["kubernetes_workspace_name"] = *req.KubernetesWorkspaceName
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "at least one updatable field must be provided")
		return
	}
	updated, err := s.Stores.Workspaces.UpdateWorkspace(r.Context(), workspaceID, fields)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "workspace not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update workspace")
		return
	}
	if err := s.syncer().SyncWorkspace(r.Context(), *updated); err != nil {
		log.Printf("syncer: failed to sync workspace %s: %v", updated.ID, err)
	}
	writeJSON(w, http.StatusOK, workspaceResponseFromRecord(*updated))
}

func (s Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) {
	if err := s.Stores.Workspaces.DeleteWorkspace(r.Context(), workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	if err := s.syncer().DeleteWorkspace(r.Context(), workspaceID); err != nil {
		log.Printf("syncer: failed to delete workspace %s: %v", workspaceID, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) handleTenant(w http.ResponseWriter, r *http.Request) {
	if s.Stores.Tenants == nil {
		writeError(w, http.StatusServiceUnavailable, "tenant store is not configured")
		return
	}
	tenantID := strings.TrimPrefix(r.URL.Path, "/api/v1/tenants/")
	tenantID = strings.TrimSpace(tenantID)

	if tenantID == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleListTenants(w, r)
		case http.MethodPost:
			s.handleCreateTenant(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method must be GET or POST")
		}
		return
	}

	if strings.Contains(tenantID, "/") {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetTenant(w, r, tenantID)
	case http.MethodPatch:
		s.handleUpdateTenant(w, r, tenantID)
	case http.MethodDelete:
		s.handleDeleteTenant(w, r, tenantID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method must be GET, PATCH, or DELETE")
	}
}

func (s Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if s.Stores.Agents == nil {
		writeError(w, http.StatusServiceUnavailable, "agent store is not configured")
		return
	}
	agentID := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	agentID = strings.TrimSpace(agentID)

	if agentID == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleListAgents(w, r)
		case http.MethodPost:
			s.handleCreateAgent(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method must be GET or POST")
		}
		return
	}

	if strings.Contains(agentID, "/") {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetAgent(w, r, agentID)
	case http.MethodPatch:
		s.handleUpdateAgent(w, r, agentID)
	case http.MethodDelete:
		s.handleDeleteAgent(w, r, agentID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method must be GET, PATCH, or DELETE")
	}
}

func (s Server) handleEvaluation(w http.ResponseWriter, r *http.Request) {
	if s.Stores.Evaluations == nil {
		writeError(w, http.StatusServiceUnavailable, "evaluation store is not configured")
		return
	}
	evaluationID := strings.TrimPrefix(r.URL.Path, "/api/v1/evaluations/")
	evaluationID = strings.TrimSpace(evaluationID)

	if evaluationID == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleListEvaluations(w, r)
		case http.MethodPost:
			s.handleCreateEvaluation(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method must be GET or POST")
		}
		return
	}

	if strings.Contains(evaluationID, "/") {
		writeError(w, http.StatusNotFound, "evaluation not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetEvaluation(w, r, evaluationID)
	case http.MethodPatch:
		s.handleUpdateEvaluation(w, r, evaluationID)
	case http.MethodDelete:
		s.handleDeleteEvaluation(w, r, evaluationID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method must be GET, PATCH, or DELETE")
	}
}

func (s Server) handleProvider(w http.ResponseWriter, r *http.Request) {
	if s.Stores.Providers == nil {
		writeError(w, http.StatusServiceUnavailable, "provider store is not configured")
		return
	}
	providerID := strings.TrimPrefix(r.URL.Path, "/api/v1/providers/")
	providerID = strings.TrimSpace(providerID)

	if providerID == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleListProviders(w, r)
		case http.MethodPost:
			s.handleCreateProvider(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method must be GET or POST")
		}
		return
	}

	if strings.Contains(providerID, "/") {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetProvider(w, r, providerID)
	case http.MethodPatch:
		s.handleUpdateProvider(w, r, providerID)
	case http.MethodDelete:
		s.handleDeleteProvider(w, r, providerID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method must be GET, PATCH, or DELETE")
	}
}

func (s Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if s.Stores.Runs == nil {
		writeError(w, http.StatusServiceUnavailable, "run store is not configured")
		return
	}
	runID := strings.TrimPrefix(r.URL.Path, "/api/v1/runs/")
	runID = strings.TrimSpace(runID)

	if runID == "" {
		switch r.Method {
		case http.MethodGet:
			s.handleListRuns(w, r)
		case http.MethodPost:
			s.handleCreateRun(w, r)
		default:
			writeError(w, http.StatusMethodNotAllowed, "method must be GET or POST")
		}
		return
	}

	if strings.Contains(runID, "/") {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}

	switch r.Method {
	case http.MethodGet:
		s.handleGetRun(w, r, runID)
	case http.MethodPatch:
		s.handleUpdateRun(w, r, runID)
	case http.MethodDelete:
		s.handleDeleteRun(w, r, runID)
	default:
		writeError(w, http.StatusMethodNotAllowed, "method must be GET, PATCH, or DELETE")
	}
}

func (s Server) handleGetAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	agent, err := s.Stores.Agents.GetAgent(r.Context(), agentID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read agent")
		return
	}
	writeJSON(w, http.StatusOK, agentResponseFromRecord(*agent))
}

func (s Server) handleListAgents(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationFromQuery(r)
	tenantID := r.URL.Query().Get("tenantId")
	workspaceID := r.URL.Query().Get("workspaceId")
	var records []AgentRecord
	var total int
	var err error
	switch {
	case workspaceID != "":
		records, total, err = s.Stores.Agents.ListAgentsByWorkspace(r.Context(), workspaceID, page, limit)
	case tenantID != "":
		records, total, err = s.Stores.Agents.ListAgentsByTenant(r.Context(), tenantID, page, limit)
	default:
		records, total, err = s.Stores.Agents.ListAgents(r.Context(), page, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list agents")
		return
	}
	agents := make([]AgentResponse, 0, len(records))
	for _, rec := range records {
		agents = append(agents, agentResponseFromRecord(rec))
	}
	writeJSON(w, http.StatusOK, PaginatedAgentsResponse{
		Agents: agents,
		Page:   page,
		Limit:  limit,
		Total:  total,
	})
}

func (s Server) handleGetEvaluation(w http.ResponseWriter, r *http.Request, evaluationID string) {
	evaluation, err := s.Stores.Evaluations.GetEvaluation(r.Context(), evaluationID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "evaluation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read evaluation")
		return
	}
	writeJSON(w, http.StatusOK, evaluationResponseFromRecord(*evaluation))
}

func (s Server) handleListEvaluations(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationFromQuery(r)
	tenantID := r.URL.Query().Get("tenantId")
	workspaceID := r.URL.Query().Get("workspaceId")
	agentID := r.URL.Query().Get("agentId")
	var records []EvaluationRecord
	var total int
	var err error
	switch {
	case agentID != "":
		records, total, err = s.Stores.Evaluations.ListEvaluationsByAgent(r.Context(), agentID, page, limit)
	case workspaceID != "":
		records, total, err = s.Stores.Evaluations.ListEvaluationsByWorkspace(r.Context(), workspaceID, page, limit)
	case tenantID != "":
		records, total, err = s.Stores.Evaluations.ListEvaluationsByTenant(r.Context(), tenantID, page, limit)
	default:
		records, total, err = s.Stores.Evaluations.ListEvaluations(r.Context(), page, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list evaluations")
		return
	}
	evaluations := make([]EvaluationResponse, 0, len(records))
	for _, rec := range records {
		evaluations = append(evaluations, evaluationResponseFromRecord(rec))
	}
	writeJSON(w, http.StatusOK, PaginatedEvaluationsResponse{
		Evaluations: evaluations,
		Page:        page,
		Limit:       limit,
		Total:       total,
	})
}

func (s Server) handleGetProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	provider, err := s.Stores.Providers.GetProvider(r.Context(), providerID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read provider")
		return
	}
	writeJSON(w, http.StatusOK, providerResponseFromRecord(*provider))
}

func (s Server) handleListProviders(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationFromQuery(r)
	tenantID := r.URL.Query().Get("tenantId")
	workspaceID := r.URL.Query().Get("workspaceId")
	var records []ProviderRecord
	var total int
	var err error
	switch {
	case workspaceID != "":
		records, total, err = s.Stores.Providers.ListProvidersByWorkspace(r.Context(), workspaceID, page, limit)
	case tenantID != "":
		records, total, err = s.Stores.Providers.ListProvidersByTenant(r.Context(), tenantID, page, limit)
	default:
		records, total, err = s.Stores.Providers.ListProviders(r.Context(), page, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list providers")
		return
	}
	providers := make([]ProviderResponse, 0, len(records))
	for _, rec := range records {
		providers = append(providers, providerResponseFromRecord(rec))
	}
	writeJSON(w, http.StatusOK, PaginatedProvidersResponse{
		Providers: providers,
		Page:      page,
		Limit:     limit,
		Total:     total,
	})
}

func (s Server) handleGetRun(w http.ResponseWriter, r *http.Request, runID string) {
	run, err := s.Stores.Runs.GetRun(r.Context(), runID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read run")
		return
	}
	writeJSON(w, http.StatusOK, runResponseFromRecord(*run))
}

func (s Server) handleListRuns(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationFromQuery(r)
	tenantID := r.URL.Query().Get("tenantId")
	workspaceID := r.URL.Query().Get("workspaceId")
	agentID := r.URL.Query().Get("agentId")
	var records []RunRecord
	var total int
	var err error
	switch {
	case agentID != "":
		records, total, err = s.Stores.Runs.ListRunsByAgent(r.Context(), agentID, page, limit)
	case workspaceID != "":
		records, total, err = s.Stores.Runs.ListRunsByWorkspace(r.Context(), workspaceID, page, limit)
	case tenantID != "":
		records, total, err = s.Stores.Runs.ListRunsByTenant(r.Context(), tenantID, page, limit)
	default:
		records, total, err = s.Stores.Runs.ListRuns(r.Context(), page, limit)
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list runs")
		return
	}
	runs := make([]RunResponse, 0, len(records))
	for _, rec := range records {
		runs = append(runs, runResponseFromRecord(rec))
	}
	writeJSON(w, http.StatusOK, PaginatedRunsResponse{
		Runs:  runs,
		Page:  page,
		Limit: limit,
		Total: total,
	})
}

func (s Server) handleGetTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	tenant, err := s.Stores.Tenants.GetTenant(r.Context(), tenantID)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to read tenant")
		return
	}
	writeJSON(w, http.StatusOK, TenantResponse{
		ID:             tenant.ID,
		OrganizationID: tenant.OrganizationID,
		Slug:           tenant.Slug,
		DisplayName:    tenant.DisplayName,
		Status:         tenant.Status,
		DefaultRegion:  tenant.DefaultRegion,
	})
}

func (s Server) handleListTenants(w http.ResponseWriter, r *http.Request) {
	page, limit := paginationFromQuery(r)
	records, total, err := s.Stores.Tenants.ListTenants(r.Context(), page, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list tenants")
		return
	}
	tenants := make([]TenantResponse, 0, len(records))
	for _, rec := range records {
		tenants = append(tenants, TenantResponse{
			ID:             rec.ID,
			OrganizationID: rec.OrganizationID,
			Slug:           rec.Slug,
			DisplayName:    rec.DisplayName,
			Status:         rec.Status,
			DefaultRegion:  rec.DefaultRegion,
		})
	}
	writeJSON(w, http.StatusOK, PaginatedTenantsResponse{
		Tenants: tenants,
		Page:    page,
		Limit:   limit,
		Total:   total,
	})
}

func workspaceResponseFromRecord(rec WorkspaceRecord) WorkspaceResponse {
	return WorkspaceResponse{
		ID:                      rec.ID,
		TenantID:                rec.TenantID,
		Slug:                    rec.Slug,
		DisplayName:             rec.DisplayName,
		Description:             rec.Description,
		Status:                  rec.Status,
		KubernetesNamespace:     rec.KubernetesNamespace,
		KubernetesWorkspaceName: rec.KubernetesWorkspaceName,
	}
}

func agentResponseFromRecord(rec AgentRecord) AgentResponse {
	return AgentResponse{
		ID:             rec.ID,
		TenantID:       rec.TenantID,
		WorkspaceID:    rec.WorkspaceID,
		Slug:           rec.Slug,
		DisplayName:    rec.DisplayName,
		Description:    rec.Description,
		Status:         rec.Status,
		Pattern:        rec.Pattern,
		RuntimeEngine:  rec.RuntimeEngine,
		RunnerClass:    rec.RunnerClass,
		ModelProvider:  rec.ModelProvider,
		ModelName:      rec.ModelName,
		LatestRevision: rec.LatestRevision,
	}
}

func evaluationResponseFromRecord(rec EvaluationRecord) EvaluationResponse {
	return EvaluationResponse{
		ID:               rec.ID,
		TenantID:         rec.TenantID,
		WorkspaceID:      rec.WorkspaceID,
		AgentID:          rec.AgentID,
		Slug:             rec.Slug,
		DisplayName:      rec.DisplayName,
		Description:      rec.Description,
		Status:           rec.Status,
		DatasetName:      rec.DatasetName,
		DatasetRevision:  rec.DatasetRevision,
		BaselineRevision: rec.BaselineRevision,
		Score:            rec.Score,
		GatePassed:       rec.GatePassed,
		SamplesTotal:     rec.SamplesTotal,
		SamplesEvaluated: rec.SamplesEvaluated,
		LatestRunID:      rec.LatestRunID,
		ReportRef:        rec.ReportRef,
	}
}

func providerResponseFromRecord(rec ProviderRecord) ProviderResponse {
	return ProviderResponse{
		ID:                  rec.ID,
		TenantID:            rec.TenantID,
		WorkspaceID:         rec.WorkspaceID,
		Provider:            rec.Provider,
		DisplayName:         rec.DisplayName,
		Family:              rec.Family,
		BaseURL:             rec.BaseURL,
		CredentialRef:       rec.CredentialRef,
		Status:              rec.Status,
		Domestic:            rec.Domestic,
		SupportsJSONSchema:  rec.SupportsJSONSchema,
		SupportsToolCalling: rec.SupportsToolCalling,
	}
}

func runResponseFromRecord(rec RunRecord) RunResponse {
	return RunResponse{
		ID:            rec.ID,
		TenantID:      rec.TenantID,
		WorkspaceID:   rec.WorkspaceID,
		AgentID:       rec.AgentID,
		EvaluationID:  rec.EvaluationID,
		AgentRevision: rec.AgentRevision,
		Status:        rec.Status,
		RuntimeEngine: rec.RuntimeEngine,
		RunnerClass:   rec.RunnerClass,
		StartedAt:     rec.StartedAt,
		CompletedAt:   rec.CompletedAt,
		Summary:       rec.Summary,
		TraceRef:      rec.TraceRef,
	}
}

func tenantResponseFromRecord(rec TenantRecord) TenantResponse {
	return TenantResponse{
		ID:             rec.ID,
		OrganizationID: rec.OrganizationID,
		Slug:           rec.Slug,
		DisplayName:    rec.DisplayName,
		Status:         rec.Status,
		DefaultRegion:  rec.DefaultRegion,
	}
}

func (s Server) handleCreateTenant(w http.ResponseWriter, r *http.Request) {
	var req CreateTenantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" || req.OrganizationID == "" || req.Slug == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "id, organizationId, slug, and displayName are required")
		return
	}
	record := TenantRecord{
		ID:             req.ID,
		OrganizationID: req.OrganizationID,
		Slug:           req.Slug,
		DisplayName:    req.DisplayName,
		Status:         req.Status,
		DefaultRegion:  req.DefaultRegion,
	}
	if record.Status == "" {
		record.Status = "active"
	}
	if err := s.Stores.Tenants.CreateTenant(r.Context(), record); err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "tenant already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create tenant")
		return
	}
	if err := s.syncer().SyncTenant(r.Context(), record); err != nil {
		log.Printf("syncer: failed to sync tenant %s: %v", record.ID, err)
	}
	writeJSON(w, http.StatusCreated, tenantResponseFromRecord(record))
}

func (s Server) handleUpdateTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	var req UpdateTenantRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fields := map[string]string{}
	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.DefaultRegion != nil {
		fields["default_region"] = *req.DefaultRegion
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "at least one updatable field must be provided")
		return
	}
	updated, err := s.Stores.Tenants.UpdateTenant(r.Context(), tenantID, fields)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update tenant")
		return
	}
	if err := s.syncer().SyncTenant(r.Context(), *updated); err != nil {
		log.Printf("syncer: failed to sync tenant %s: %v", updated.ID, err)
	}
	writeJSON(w, http.StatusOK, tenantResponseFromRecord(*updated))
}

func (s Server) handleDeleteTenant(w http.ResponseWriter, r *http.Request, tenantID string) {
	if err := s.Stores.Tenants.DeleteTenant(r.Context(), tenantID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete tenant")
		return
	}
	if err := s.syncer().DeleteTenant(r.Context(), tenantID); err != nil {
		log.Printf("syncer: failed to delete tenant %s: %v", tenantID, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) handleCreateAgent(w http.ResponseWriter, r *http.Request) {
	var req CreateAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" || req.TenantID == "" || req.WorkspaceID == "" || req.Slug == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "id, tenantId, workspaceId, slug, and displayName are required")
		return
	}
	record := AgentRecord{
		ID:            req.ID,
		TenantID:      req.TenantID,
		WorkspaceID:   req.WorkspaceID,
		Slug:          req.Slug,
		DisplayName:   req.DisplayName,
		Description:   req.Description,
		Status:        req.Status,
		Pattern:       req.Pattern,
		RuntimeEngine: req.RuntimeEngine,
		RunnerClass:   req.RunnerClass,
		ModelProvider: req.ModelProvider,
		ModelName:     req.ModelName,
	}
	if record.Status == "" {
		record.Status = "draft"
	}
	if record.Pattern == "" {
		record.Pattern = "react"
	}
	if record.RuntimeEngine == "" {
		record.RuntimeEngine = "eino"
	}
	if record.RunnerClass == "" {
		record.RunnerClass = "adk"
	}
	if err := s.Stores.Agents.CreateAgent(r.Context(), record); err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "agent already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create agent")
		return
	}
	if err := s.syncer().SyncAgent(r.Context(), record); err != nil {
		log.Printf("syncer: failed to sync agent %s: %v", record.ID, err)
	}
	writeJSON(w, http.StatusCreated, agentResponseFromRecord(record))
}

func (s Server) handleUpdateAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	var req UpdateAgentRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fields := map[string]string{}
	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.Pattern != nil {
		fields["pattern"] = *req.Pattern
	}
	if req.RuntimeEngine != nil {
		fields["runtime_engine"] = *req.RuntimeEngine
	}
	if req.RunnerClass != nil {
		fields["runner_class"] = *req.RunnerClass
	}
	if req.ModelProvider != nil {
		fields["model_provider"] = *req.ModelProvider
	}
	if req.ModelName != nil {
		fields["model_name"] = *req.ModelName
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "at least one updatable field must be provided")
		return
	}
	updated, err := s.Stores.Agents.UpdateAgent(r.Context(), agentID, fields)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update agent")
		return
	}
	if err := s.syncer().SyncAgent(r.Context(), *updated); err != nil {
		log.Printf("syncer: failed to sync agent %s: %v", updated.ID, err)
	}
	writeJSON(w, http.StatusOK, agentResponseFromRecord(*updated))
}

func (s Server) handleDeleteAgent(w http.ResponseWriter, r *http.Request, agentID string) {
	if err := s.Stores.Agents.DeleteAgent(r.Context(), agentID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete agent")
		return
	}
	if err := s.syncer().DeleteAgent(r.Context(), agentID); err != nil {
		log.Printf("syncer: failed to delete agent %s: %v", agentID, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) handleCreateEvaluation(w http.ResponseWriter, r *http.Request) {
	var req CreateEvaluationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" || req.TenantID == "" || req.WorkspaceID == "" || req.AgentID == "" || req.Slug == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "id, tenantId, workspaceId, agentId, slug, and displayName are required")
		return
	}
	record := EvaluationRecord{
		ID:               req.ID,
		TenantID:         req.TenantID,
		WorkspaceID:      req.WorkspaceID,
		AgentID:          req.AgentID,
		Slug:             req.Slug,
		DisplayName:      req.DisplayName,
		Description:      req.Description,
		Status:           req.Status,
		DatasetName:      req.DatasetName,
		DatasetRevision:  req.DatasetRevision,
		BaselineRevision: req.BaselineRevision,
	}
	if record.Status == "" {
		record.Status = "pending"
	}
	if err := s.Stores.Evaluations.CreateEvaluation(r.Context(), record); err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "evaluation already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create evaluation")
		return
	}
	if err := s.syncer().SyncEvaluation(r.Context(), record); err != nil {
		log.Printf("syncer: failed to sync evaluation %s: %v", record.ID, err)
	}
	writeJSON(w, http.StatusCreated, evaluationResponseFromRecord(record))
}

func (s Server) handleUpdateEvaluation(w http.ResponseWriter, r *http.Request, evaluationID string) {
	var req UpdateEvaluationRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fields := map[string]string{}
	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.DatasetName != nil {
		fields["dataset_name"] = *req.DatasetName
	}
	if req.DatasetRevision != nil {
		fields["dataset_revision"] = *req.DatasetRevision
	}
	if req.BaselineRevision != nil {
		fields["baseline_revision"] = *req.BaselineRevision
	}
	if req.Score != nil {
		fields["score"] = strconv.FormatFloat(*req.Score, 'f', -1, 64)
	}
	if req.GatePassed != nil {
		fields["gate_passed"] = strconv.FormatBool(*req.GatePassed)
	}
	if req.SamplesTotal != nil {
		fields["samples_total"] = strconv.Itoa(*req.SamplesTotal)
	}
	if req.SamplesEvaluated != nil {
		fields["samples_evaluated"] = strconv.Itoa(*req.SamplesEvaluated)
	}
	if req.LatestRunID != nil {
		fields["latest_run_id"] = *req.LatestRunID
	}
	if req.ReportRef != nil {
		fields["report_ref"] = *req.ReportRef
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "at least one updatable field must be provided")
		return
	}
	updated, err := s.Stores.Evaluations.UpdateEvaluation(r.Context(), evaluationID, fields)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "evaluation not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update evaluation")
		return
	}
	if err := s.syncer().SyncEvaluation(r.Context(), *updated); err != nil {
		log.Printf("syncer: failed to sync evaluation %s: %v", updated.ID, err)
	}
	writeJSON(w, http.StatusOK, evaluationResponseFromRecord(*updated))
}

func (s Server) handleDeleteEvaluation(w http.ResponseWriter, r *http.Request, evaluationID string) {
	if err := s.Stores.Evaluations.DeleteEvaluation(r.Context(), evaluationID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete evaluation")
		return
	}
	if err := s.syncer().DeleteEvaluation(r.Context(), evaluationID); err != nil {
		log.Printf("syncer: failed to delete evaluation %s: %v", evaluationID, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) handleCreateProvider(w http.ResponseWriter, r *http.Request) {
	var req CreateProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" || req.TenantID == "" || req.Provider == "" || req.DisplayName == "" {
		writeError(w, http.StatusBadRequest, "id, tenantId, provider, and displayName are required")
		return
	}
	record := ProviderRecord{
		ID:                  req.ID,
		TenantID:            req.TenantID,
		WorkspaceID:         req.WorkspaceID,
		Provider:            req.Provider,
		DisplayName:         req.DisplayName,
		Family:              req.Family,
		BaseURL:             req.BaseURL,
		CredentialRef:       req.CredentialRef,
		Status:              req.Status,
		Domestic:            req.Domestic,
		SupportsJSONSchema:  req.SupportsJSONSchema,
		SupportsToolCalling: req.SupportsToolCalling,
	}
	if record.Status == "" {
		record.Status = "active"
	}
	if err := s.Stores.Providers.CreateProvider(r.Context(), record); err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "provider already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create provider")
		return
	}
	if err := s.syncer().SyncProvider(r.Context(), record); err != nil {
		log.Printf("syncer: failed to sync provider %s: %v", record.ID, err)
	}
	writeJSON(w, http.StatusCreated, providerResponseFromRecord(record))
}

func (s Server) handleUpdateProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	var req UpdateProviderRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fields := map[string]string{}
	if req.DisplayName != nil {
		fields["display_name"] = *req.DisplayName
	}
	if req.Family != nil {
		fields["family"] = *req.Family
	}
	if req.BaseURL != nil {
		fields["base_url"] = *req.BaseURL
	}
	if req.CredentialRef != nil {
		fields["credential_ref"] = *req.CredentialRef
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.Domestic != nil {
		fields["domestic"] = strconv.FormatBool(*req.Domestic)
	}
	if req.SupportsJSONSchema != nil {
		fields["supports_json_schema"] = strconv.FormatBool(*req.SupportsJSONSchema)
	}
	if req.SupportsToolCalling != nil {
		fields["supports_tool_calling"] = strconv.FormatBool(*req.SupportsToolCalling)
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "at least one updatable field must be provided")
		return
	}
	updated, err := s.Stores.Providers.UpdateProvider(r.Context(), providerID, fields)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "provider not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update provider")
		return
	}
	if err := s.syncer().SyncProvider(r.Context(), *updated); err != nil {
		log.Printf("syncer: failed to sync provider %s: %v", updated.ID, err)
	}
	writeJSON(w, http.StatusOK, providerResponseFromRecord(*updated))
}

func (s Server) handleDeleteProvider(w http.ResponseWriter, r *http.Request, providerID string) {
	if err := s.Stores.Providers.DeleteProvider(r.Context(), providerID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete provider")
		return
	}
	if err := s.syncer().DeleteProvider(r.Context(), providerID); err != nil {
		log.Printf("syncer: failed to delete provider %s: %v", providerID, err)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) handleCreateRun(w http.ResponseWriter, r *http.Request) {
	var req CreateRunRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.ID == "" || req.TenantID == "" || req.WorkspaceID == "" || req.AgentID == "" {
		writeError(w, http.StatusBadRequest, "id, tenantId, workspaceId, and agentId are required")
		return
	}
	record := RunRecord{
		ID:            req.ID,
		TenantID:      req.TenantID,
		WorkspaceID:   req.WorkspaceID,
		AgentID:       req.AgentID,
		EvaluationID:  req.EvaluationID,
		AgentRevision: req.AgentRevision,
		Status:        req.Status,
		RuntimeEngine: req.RuntimeEngine,
		RunnerClass:   req.RunnerClass,
	}
	if record.Status == "" {
		record.Status = "pending"
	}
	if err := s.Stores.Runs.CreateRun(r.Context(), record); err != nil {
		if errors.Is(err, ErrConflict) {
			writeError(w, http.StatusConflict, "run already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create run")
		return
	}
	writeJSON(w, http.StatusCreated, runResponseFromRecord(record))
}

func (s Server) handleUpdateRun(w http.ResponseWriter, r *http.Request, runID string) {
	var req UpdateRunRequest
	if err := decodeJSON(r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fields := map[string]string{}
	if req.Status != nil {
		fields["status"] = *req.Status
	}
	if req.StartedAt != nil {
		fields["started_at"] = *req.StartedAt
	}
	if req.CompletedAt != nil {
		fields["completed_at"] = *req.CompletedAt
	}
	if req.Summary != nil {
		fields["summary"] = *req.Summary
	}
	if req.TraceRef != nil {
		fields["trace_ref"] = *req.TraceRef
	}
	if len(fields) == 0 {
		writeError(w, http.StatusBadRequest, "at least one updatable field must be provided")
		return
	}
	updated, err := s.Stores.Runs.UpdateRun(r.Context(), runID, fields)
	if errors.Is(err, ErrNotFound) {
		writeError(w, http.StatusNotFound, "run not found")
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to update run")
		return
	}
	writeJSON(w, http.StatusOK, runResponseFromRecord(*updated))
}

func (s Server) handleDeleteRun(w http.ResponseWriter, r *http.Request, runID string) {
	if err := s.Stores.Runs.DeleteRun(r.Context(), runID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete run")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func paginationFromQuery(r *http.Request) (page, limit int) {
	page, limit = 1, 20
	if p := r.URL.Query().Get("page"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil && parsed > 0 {
			page = parsed
		}
	}
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	return
}

func decodeJSON(r *http.Request, v any) error {
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		_, _ = fmt.Fprintf(w, `{"error":"failed to encode response"}`)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
