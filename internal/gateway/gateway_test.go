package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
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
