package runtime

import (
	"context"
	"fmt"
	"strings"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
)

type MockRuntime struct{}

func NewMockRuntime() MockRuntime {
	return MockRuntime{}
}

func (r MockRuntime) Execute(ctx context.Context, request Request) (Result, error) {
	task := JSONString(request.Run.Spec.Input, "task")
	if task == "" {
		task = "agent_run"
	}

	summary := strings.TrimSpace(fmt.Sprintf(
		"Mock execution completed for %s using %s.",
		task,
		request.Agent.Name,
	))

	return Result{
		Output: apiv1alpha1.FreeformObject{
			"summary":          JSONValue(summary),
			"hazards":          JSONValue([]interface{}{}),
			"overallRiskLevel": JSONValue("low"),
			"nextActions":      JSONValue([]string{"review mock result before enabling a real runtime"}),
			"confidence":       JSONValue(1.0),
			"needsHumanReview": JSONValue(false),
		},
		TraceRef: apiv1alpha1.FreeformObject{
			"provider": JSONValue("mock"),
			"runId":    JSONValue(request.Run.Namespace + "/" + request.Run.Name),
		},
		Reason:  "MockRuntimeSucceeded",
		Message: "mock runtime completed successfully",
	}, nil
}
