package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGatewayInvokeCreatesAgentRun(t *testing.T) {
	scheme := testScheme(t)
	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(readyAgent()).
		Build()
	server := Server{
		Client: kubeClient,
		Clock: func() time.Time {
			return time.Unix(1713312000, 123)
		},
	}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{
		"input": {
			"task": "identify_hazard",
			"payload": {"site": "factory-a"}
		},
		"execution": {
			"mode": "sync"
		}
	}`))
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, recorder.Code, recorder.Body.String())
	}

	var response InvokeResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	if response.Status != "accepted" {
		t.Fatalf("unexpected response status: %q", response.Status)
	}
	if response.AgentRun.Namespace != "ehs" || response.AgentRun.Name == "" {
		t.Fatalf("unexpected AgentRun reference: %#v", response.AgentRun)
	}

	var run apiv1alpha1.AgentRun
	key := types.NamespacedName{Namespace: "ehs", Name: response.AgentRun.Name}
	if err := kubeClient.Get(context.Background(), key, &run); err != nil {
		t.Fatalf("expected AgentRun to be created: %v", err)
	}
	if run.Spec.AgentRef.Name != "ehs-agent" {
		t.Fatalf("unexpected AgentRef: %#v", run.Spec.AgentRef)
	}
	if run.Spec.WorkspaceRef == nil || run.Spec.WorkspaceRef.Name != "workspace-a" {
		t.Fatalf("unexpected WorkspaceRef: %#v", run.Spec.WorkspaceRef)
	}
	if jsonString(t, run.Spec.Input["task"]) != "identify_hazard" {
		t.Fatalf("unexpected task input: %#v", run.Spec.Input["task"])
	}
	if jsonString(t, run.Spec.Execution["mode"]) != "sync" {
		t.Fatalf("unexpected execution mode: %#v", run.Spec.Execution["mode"])
	}
}

func TestGatewayInvokeRejectsNotReadyAgent(t *testing.T) {
	scheme := testScheme(t)
	agent := readyAgent()
	agent.Status.Conditions = nil
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(agent).Build()
	server := Server{Client: kubeClient}

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke", bytes.NewBufferString(`{}`))
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, recorder.Code, recorder.Body.String())
	}
}

func TestGatewayInvokeRejectsInvalidRequests(t *testing.T) {
	scheme := testScheme(t)
	kubeClient := fake.NewClientBuilder().WithScheme(scheme).WithObjects(readyAgent()).Build()
	server := Server{Client: kubeClient}

	tests := []struct {
		name   string
		method string
		path   string
		body   string
		status int
	}{
		{
			name:   "method",
			method: http.MethodGet,
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke",
			body:   `{}`,
			status: http.StatusMethodNotAllowed,
		},
		{
			name:   "path",
			method: http.MethodPost,
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent",
			body:   `{}`,
			status: http.StatusNotFound,
		},
		{
			name:   "unknown body field",
			method: http.MethodPost,
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke",
			body:   `{"unknown": true}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "trailing body",
			method: http.MethodPost,
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke",
			body:   `{} {}`,
			status: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
			server.Handler().ServeHTTP(recorder, request)
			if recorder.Code != tt.status {
				t.Fatalf("expected status %d, got %d: %s", tt.status, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := apiv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatalf("AddToScheme returned error: %v", err)
	}
	return scheme
}

func readyAgent() *apiv1alpha1.Agent {
	return &apiv1alpha1.Agent{
		ObjectMeta: metav1.ObjectMeta{Name: "ehs-agent", Namespace: "ehs"},
		Status: apiv1alpha1.AgentStatus{
			CompiledRevision: "sha256:agent",
			WorkspaceRef:     "workspace-a",
			CompiledArtifact: apiv1alpha1.FreeformObject{
				"kind": apiextensionsv1.JSON{Raw: []byte(`"AgentCompiledArtifact"`)},
			},
			ConditionedStatus: apiv1alpha1.ConditionedStatus{
				Conditions: []metav1.Condition{
					{
						Type:   readyConditionType,
						Status: metav1.ConditionTrue,
						Reason: "CompilationSucceeded",
					},
				},
			},
		},
	}
}

func jsonString(t *testing.T, value apiextensionsv1.JSON) string {
	t.Helper()
	var output string
	if err := json.Unmarshal(value.Raw, &output); err != nil {
		t.Fatalf("failed to decode JSON string: %v", err)
	}
	return output
}

func TestParseAgentRunPath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantNamespace string
		wantName      string
		wantOk        bool
	}{
		{
			name:          "valid agentrun path",
			path:          "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/my-run-abc",
			wantNamespace: "ehs",
			wantName:      "my-run-abc",
			wantOk:        true,
		},
		{
			name:   "agent invoke path is not agentrun",
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke",
			wantOk: false,
		},
		{
			name:   "missing namespace",
			path:   "/apis/windosx.com/v1alpha1/namespaces//agentruns/my-run",
			wantOk: false,
		},
		{
			name:   "missing name",
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/",
			wantOk: false,
		},
		{
			name:   "wrong prefix",
			path:   "/api/v1/namespaces/ehs/agentruns/my-run",
			wantOk: false,
		},
		{
			name:   "too many segments",
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/my-run/extra",
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, name, ok := parseAgentRunPath(tt.path)
			if ok != tt.wantOk {
				t.Fatalf("parseAgentRunPath(%q) ok = %v, want %v", tt.path, ok, tt.wantOk)
			}
			if ok {
				if ns != tt.wantNamespace {
					t.Errorf("namespace = %q, want %q", ns, tt.wantNamespace)
				}
				if name != tt.wantName {
					t.Errorf("name = %q, want %q", name, tt.wantName)
				}
			}
		})
	}
}

func TestGatewayGetAgentRunStatus(t *testing.T) {
	scheme := testScheme(t)

	// Create an existing AgentRun in the fake store.
	existingRun := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ehs-run-abc",
			Namespace: "ehs",
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
		},
		Status: apiv1alpha1.AgentRunStatus{
			Phase: "Succeeded",
			ConditionedStatus: apiv1alpha1.ConditionedStatus{
				Conditions: []metav1.Condition{
					{
						Type:    "Completed",
						Status:  metav1.ConditionTrue,
						Reason:  "WorkerJobSucceeded",
						Message: "worker completed",
					},
				},
			},
		},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingRun).
		WithStatusSubresource(existingRun).
		Build()

	server := Server{Client: kubeClient}

	tests := []struct {
		name       string
		path       string
		wantStatus int
		wantPhase  string
	}{
		{
			name:       "get existing agentrun",
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/ehs-run-abc",
			wantStatus: http.StatusOK,
			wantPhase:  "Succeeded",
		},
		{
			name:       "get nonexistent agentrun",
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/does-not-exist",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "invalid path structure",
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/",
			wantStatus: http.StatusMethodNotAllowed, // Falls through to invoke handler which rejects GET
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, tt.path, nil)
			server.Handler().ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}

			if tt.wantPhase != "" {
				var resp map[string]interface{}
				if err := json.Unmarshal(recorder.Body.Bytes(), &resp); err != nil {
					t.Fatalf("failed to parse response: %v", err)
				}
				status, _ := resp["status"].(map[string]interface{})
				if status == nil {
					t.Fatal("response missing status field")
				}
				phase, _ := status["phase"].(string)
				if phase != tt.wantPhase {
					t.Errorf("phase = %q, want %q", phase, tt.wantPhase)
				}
			}
		})
	}
}

func TestParseAgentRunCancelPath(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		wantNamespace string
		wantName      string
		wantOk        bool
	}{
		{
			name:          "valid cancel path",
			path:          "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/my-run:cancel",
			wantNamespace: "ehs",
			wantName:      "my-run",
			wantOk:        true,
		},
		{
			name:   "agentrun path without :cancel suffix is not cancel",
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/my-run",
			wantOk: false,
		},
		{
			name:   "agent invoke path is not cancel",
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke",
			wantOk: false,
		},
		{
			name:   "missing namespace in cancel path",
			path:   "/apis/windosx.com/v1alpha1/namespaces//agentruns/my-run:cancel",
			wantOk: false,
		},
		{
			name:   "missing name in cancel path",
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/:cancel",
			wantOk: false,
		},
		{
			name:   "wrong prefix in cancel path",
			path:   "/api/v1/namespaces/ehs/agentruns/my-run:cancel",
			wantOk: false,
		},
		{
			name:   "too many segments in cancel path",
			path:   "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/my-run/extra:cancel",
			wantOk: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ns, name, ok := parseAgentRunCancelPath(tt.path)
			if ok != tt.wantOk {
				t.Fatalf("parseAgentRunCancelPath(%q) ok = %v, want %v", tt.path, ok, tt.wantOk)
			}
			if ok {
				if ns != tt.wantNamespace {
					t.Errorf("namespace = %q, want %q", ns, tt.wantNamespace)
				}
				if name != tt.wantName {
					t.Errorf("name = %q, want %q", name, tt.wantName)
				}
			}
		})
	}
}

func TestGatewayCancelAgentRun(t *testing.T) {
	scheme := testScheme(t)

	tests := []struct {
		name       string
		run        *apiv1alpha1.AgentRun
		wantStatus int
		wantCancel bool
		wantBody   string
	}{
		{
			name: "cancel running agentrun",
			run: &apiv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "running-run",
					Namespace: "ehs",
				},
				Spec: apiv1alpha1.AgentRunSpec{
					AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
				},
				Status: apiv1alpha1.AgentRunStatus{Phase: "Running"},
			},
			wantStatus: http.StatusOK,
			wantCancel: true,
			wantBody:   "cancel_requested",
		},
		{
			name: "cancel pending agentrun",
			run: &apiv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pending-run",
					Namespace: "ehs",
				},
				Spec: apiv1alpha1.AgentRunSpec{
					AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
				},
				Status: apiv1alpha1.AgentRunStatus{Phase: "Pending"},
			},
			wantStatus: http.StatusOK,
			wantCancel: true,
			wantBody:   "cancel_requested",
		},
		{
			name: "cancel already canceled agentrun",
			run: &apiv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "canceled-run",
					Namespace: "ehs",
				},
				Spec: apiv1alpha1.AgentRunSpec{
					AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
				},
				Status: apiv1alpha1.AgentRunStatus{Phase: "Canceled"},
			},
			wantStatus: http.StatusOK,
			wantCancel: false,
			wantBody:   "already canceled",
		},
		{
			name: "cancel already succeeded agentrun",
			run: &apiv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "succeeded-run",
					Namespace: "ehs",
				},
				Spec: apiv1alpha1.AgentRunSpec{
					AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
				},
				Status: apiv1alpha1.AgentRunStatus{Phase: "Succeeded"},
			},
			wantStatus: http.StatusOK,
			wantCancel: false,
			wantBody:   "already succeeded",
		},
		{
			name: "cancel already failed agentrun",
			run: &apiv1alpha1.AgentRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "failed-run",
					Namespace: "ehs",
				},
				Spec: apiv1alpha1.AgentRunSpec{
					AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
				},
				Status: apiv1alpha1.AgentRunStatus{Phase: "Failed"},
			},
			wantStatus: http.StatusOK,
			wantCancel: false,
			wantBody:   "already failed",
		},
		{
			name:       "cancel nonexistent agentrun",
			run:        nil,
			wantStatus: http.StatusNotFound,
			wantBody:   "agentrun not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			builder := fake.NewClientBuilder().WithScheme(scheme)
			runName := "does-not-exist"
			if tt.run != nil {
				builder = builder.WithObjects(tt.run).WithStatusSubresource(tt.run)
				runName = tt.run.Name
			}
			kubeClient := builder.Build()
			server := Server{Client: kubeClient}

			recorder := httptest.NewRecorder()
			path := "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/" + runName + ":cancel"
			request := httptest.NewRequest(http.MethodPost, path, nil)
			server.Handler().ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}

			body := recorder.Body.String()
			if tt.wantBody != "" && !bytes.Contains(recorder.Body.Bytes(), []byte(tt.wantBody)) {
				t.Fatalf("expected body to contain %q, got %s", tt.wantBody, body)
			}

			// Verify cancel field on the object.
			if tt.run != nil && tt.wantCancel {
				var updated apiv1alpha1.AgentRun
				key := types.NamespacedName{Namespace: tt.run.Namespace, Name: tt.run.Name}
				if err := kubeClient.Get(context.Background(), key, &updated); err != nil {
					t.Fatalf("failed to read updated agentrun: %v", err)
				}
				if updated.Spec.Cancel == nil || !*updated.Spec.Cancel {
					t.Fatalf("expected spec.cancel=true, got %v", updated.Spec.Cancel)
				}
			}
		})
	}
}

func TestGatewayCancelAgentRunMethodRouting(t *testing.T) {
	scheme := testScheme(t)

	existingRun := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cancel-test-run",
			Namespace: "ehs",
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
		},
		Status: apiv1alpha1.AgentRunStatus{Phase: "Running"},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(existingRun).
		WithStatusSubresource(existingRun).
		Build()

	server := Server{Client: kubeClient}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "POST cancel sets cancel flag",
			method:     http.MethodPost,
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/cancel-test-run:cancel",
			wantStatus: http.StatusOK,
		},
		{
			name:       "GET cancel path falls through to get handler (404)",
			method:     http.MethodGet,
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/cancel-test-run:cancel",
			wantStatus: http.StatusNotFound,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			server.Handler().ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestGatewayGetAgentRunMethodRouting(t *testing.T) {
	scheme := testScheme(t)

	existingRun := &apiv1alpha1.AgentRun{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-run",
			Namespace: "ehs",
		},
		Spec: apiv1alpha1.AgentRunSpec{
			AgentRef: apiv1alpha1.LocalObjectReference{Name: "ehs-agent"},
		},
		Status: apiv1alpha1.AgentRunStatus{Phase: "Running"},
	}

	kubeClient := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(readyAgent(), existingRun).
		WithStatusSubresource(existingRun).
		Build()

	server := Server{Client: kubeClient}

	tests := []struct {
		name       string
		method     string
		path       string
		wantStatus int
	}{
		{
			name:       "GET agentrun returns status",
			method:     http.MethodGet,
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/test-run",
			wantStatus: http.StatusOK,
		},
		{
			name:       "POST agentrun falls through to invoke (404 no :invoke suffix)",
			method:     http.MethodPost,
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agentruns/test-run",
			wantStatus: http.StatusNotFound,
		},
		{
			name:       "GET agents:invoke returns method not allowed",
			method:     http.MethodGet,
			path:       "/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-agent:invoke",
			wantStatus: http.StatusMethodNotAllowed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(tt.method, tt.path, nil)
			server.Handler().ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tt.wantStatus, recorder.Code, recorder.Body.String())
			}
		})
	}
}
