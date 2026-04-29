package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

type Server struct {
	Config Config
}

type InfoResponse struct {
	Component          string `json:"component"`
	Mode               string `json:"mode"`
	DatabaseConfigured bool   `json:"databaseConfigured"`
	DatabaseDriver     string `json:"databaseDriver,omitempty"`
	DatabaseStatus     string `json:"databaseStatus"`
	MigrateOnStart     bool   `json:"migrateOnStart"`
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
	return mux
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

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		_, _ = fmt.Fprintf(w, `{"error":"failed to encode response"}`)
	}
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
