package runtime

import (
	"context"
	"encoding/json"
	"testing"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestMockRuntimeReturnsDeterministicOutput(t *testing.T) {
	result, err := NewMockRuntime().Execute(context.Background(), Request{
		Agent: apiv1alpha1.Agent{
			ObjectMeta: metav1.ObjectMeta{Name: "hazard-agent"},
		},
		Run: apiv1alpha1.AgentRun{
			ObjectMeta: metav1.ObjectMeta{Name: "run-1", Namespace: "ehs"},
			Spec: apiv1alpha1.AgentRunSpec{
				Input: apiv1alpha1.FreeformObject{
					"task": JSONValue("identify_hazard"),
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Execute returned error: %v", err)
	}

	var summary string
	if err := json.Unmarshal(result.Output["summary"].Raw, &summary); err != nil {
		t.Fatalf("summary is not a JSON string: %v", err)
	}
	if summary != "Mock execution completed for identify_hazard using hazard-agent." {
		t.Fatalf("unexpected summary: %q", summary)
	}
	if result.Reason != "MockRuntimeSucceeded" {
		t.Fatalf("unexpected reason: %q", result.Reason)
	}
}
