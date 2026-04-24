package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	defaultMaxRequestBytes = int64(1 << 20)
	readyConditionType     = "Ready"
)

type Server struct {
	Addr            string
	Client          client.Client
	MaxRequestBytes int64
	Clock           func() time.Time
}

type InvokeRequest struct {
	Input     apiv1alpha1.FreeformObject `json:"input,omitempty"`
	Execution apiv1alpha1.FreeformObject `json:"execution,omitempty"`
}

type InvokeResponse struct {
	AgentRun AgentRunReference `json:"agentRun"`
	Status   string            `json:"status"`
	Links    map[string]string `json:"links,omitempty"`
}

type AgentRunReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

func (s Server) Start(ctx context.Context) error {
	if s.Client == nil {
		return fmt.Errorf("gateway client is required")
	}
	addr := strings.TrimSpace(s.Addr)
	if addr == "" {
		return nil
	}

	server := &http.Server{
		Addr:              addr,
		Handler:           s.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	listener, err := net.Listen("tcp", addr)
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
	mux.HandleFunc("/apis/"+apiv1alpha1.Group+"/"+apiv1alpha1.Version+"/namespaces/", s.handleInvoke)
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	return mux
}

func (s Server) handleInvoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method must be POST")
		return
	}

	namespace, agentName, ok := parseInvokePath(r.URL.Path)
	if !ok {
		writeError(w, http.StatusNotFound, "invoke endpoint not found")
		return
	}

	var request InvokeRequest
	maxBytes := s.MaxRequestBytes
	if maxBytes <= 0 {
		maxBytes = defaultMaxRequestBytes
	}
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoke request: "+err.Error())
		return
	}
	if err := ensureNoTrailingJSON(decoder); err != nil {
		writeError(w, http.StatusBadRequest, "invalid invoke request: "+err.Error())
		return
	}

	var agent apiv1alpha1.Agent
	agentKey := types.NamespacedName{Namespace: namespace, Name: agentName}
	if err := s.Client.Get(r.Context(), agentKey, &agent); err != nil {
		if apierrors.IsNotFound(err) {
			writeError(w, http.StatusNotFound, "agent not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to read agent")
		return
	}
	if !agentReady(agent) {
		writeError(w, http.StatusConflict, "agent is not ready")
		return
	}

	run := apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runName(agent.Name, s.now()),
			Namespace: namespace,
			Labels: map[string]string{
				"windosx.com/agent": agent.Name,
			},
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef:     apiv1alpha1.LocalObjectReference{Name: agent.Name},
			WorkspaceRef: agentRunWorkspaceRef(agent),
			Input:        copyFreeform(request.Input),
			Execution:    copyFreeform(request.Execution),
		},
	}
	if err := s.Client.Create(r.Context(), &run); err != nil {
		if apierrors.IsAlreadyExists(err) {
			writeError(w, http.StatusConflict, "agentrun name already exists")
			return
		}
		writeError(w, http.StatusInternalServerError, "failed to create agentrun")
		return
	}

	writeJSON(w, http.StatusCreated, InvokeResponse{
		AgentRun: AgentRunReference{Name: run.Name, Namespace: run.Namespace},
		Status:   "accepted",
		Links: map[string]string{
			"self": "/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version +
				"/namespaces/" + run.Namespace + "/agentruns/" + run.Name,
		},
	})
}

func parseInvokePath(path string) (string, string, bool) {
	prefix := "/apis/" + apiv1alpha1.Group + "/" + apiv1alpha1.Version + "/namespaces/"
	if !strings.HasPrefix(path, prefix) {
		return "", "", false
	}
	rest := strings.TrimPrefix(path, prefix)
	parts := strings.Split(rest, "/")
	if len(parts) != 3 || parts[1] != "agents" || !strings.HasSuffix(parts[2], ":invoke") {
		return "", "", false
	}
	namespace, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", false
	}
	agentName, err := url.PathUnescape(strings.TrimSuffix(parts[2], ":invoke"))
	if err != nil {
		return "", "", false
	}
	return namespace, agentName, namespace != "" && agentName != ""
}

func ensureNoTrailingJSON(decoder *json.Decoder) error {
	var extra interface{}
	if err := decoder.Decode(&extra); err == nil {
		return fmt.Errorf("request body must contain a single JSON object")
	} else if err != io.EOF {
		return err
	}
	return nil
}

func agentReady(agent apiv1alpha1.Agent) bool {
	if agent.Status.CompiledRevision == "" || len(agent.Status.CompiledArtifact) == 0 {
		return false
	}
	for _, condition := range agent.Status.Conditions {
		if condition.Type == readyConditionType && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func agentRunWorkspaceRef(agent apiv1alpha1.Agent) *apiv1alpha1.LocalObjectReference {
	if agent.Status.WorkspaceRef != "" {
		return &apiv1alpha1.LocalObjectReference{Name: agent.Status.WorkspaceRef}
	}
	if agent.Spec.WorkspaceRef == nil || agent.Spec.WorkspaceRef.Name == "" {
		return nil
	}
	ref := *agent.Spec.WorkspaceRef
	return &ref
}

func runName(agentName string, now time.Time) string {
	suffix := strconv.FormatInt(now.UTC().UnixNano(), 36)
	prefix := dnsLabelPrefix(agentName + "-run")
	maxPrefixLength := 63 - len(suffix) - 1
	if len(prefix) > maxPrefixLength {
		prefix = strings.TrimRight(prefix[:maxPrefixLength], "-")
	}
	if prefix == "" {
		prefix = "agentrun"
	}
	return prefix + "-" + suffix
}

func dnsLabelPrefix(value string) string {
	var builder strings.Builder
	lastWasDash := false
	for _, char := range strings.ToLower(value) {
		isAllowed := (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')
		if isAllowed {
			builder.WriteRune(char)
			lastWasDash = false
			continue
		}
		if !lastWasDash {
			builder.WriteRune('-')
			lastWasDash = true
		}
	}
	return strings.Trim(builder.String(), "-")
}

func copyFreeform(input apiv1alpha1.FreeformObject) apiv1alpha1.FreeformObject {
	if len(input) == 0 {
		return nil
	}
	output := make(apiv1alpha1.FreeformObject, len(input))
	for key, value := range input {
		output[key] = apiextensionsv1.JSON{Raw: append([]byte(nil), value.Raw...)}
	}
	return output
}

func (s Server) now() time.Time {
	if s.Clock != nil {
		return s.Clock()
	}
	return time.Now()
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]interface{}{
		"error": map[string]string{
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
