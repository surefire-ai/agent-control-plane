package manager

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type Server struct {
	Config Config
	Stores Stores
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
	writeJSON(w, http.StatusOK, workspaceResponseFromRecord(*updated))
}

func (s Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request, workspaceID string) {
	if err := s.Stores.Workspaces.DeleteWorkspace(r.Context(), workspaceID); err != nil {
		writeError(w, http.StatusInternalServerError, "failed to delete workspace")
		return
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
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method must be GET")
			return
		}
		s.handleListTenants(w, r)
		return
	}

	if strings.Contains(tenantID, "/") {
		writeError(w, http.StatusNotFound, "tenant not found")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method must be GET")
		return
	}
	s.handleGetTenant(w, r, tenantID)
}

func (s Server) handleAgent(w http.ResponseWriter, r *http.Request) {
	if s.Stores.Agents == nil {
		writeError(w, http.StatusServiceUnavailable, "agent store is not configured")
		return
	}
	agentID := strings.TrimPrefix(r.URL.Path, "/api/v1/agents/")
	agentID = strings.TrimSpace(agentID)

	if agentID == "" {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method must be GET")
			return
		}
		s.handleListAgents(w, r)
		return
	}

	if strings.Contains(agentID, "/") {
		writeError(w, http.StatusNotFound, "agent not found")
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method must be GET")
		return
	}
	s.handleGetAgent(w, r, agentID)
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
