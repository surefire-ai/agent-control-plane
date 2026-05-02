package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/surefire-ai/korus/internal/contract"
)

const (
	defaultRouterMaxRetries = 2
	routerClassificationKey = "classification"
	defaultGatewayURL       = "http://korus-gateway.korus-system.svc.cluster.local:8082"
)

// isRouterPattern checks if the compiled artifact uses the router pattern.
func isRouterPattern(artifact contract.CompiledArtifact) bool {
	if artifact.Pattern.Type == "router" {
		return true
	}
	if p, ok := artifact.Runner.Pattern["type"].(string); ok && p == "router" {
		return true
	}
	return false
}

// routerClassifierPrompt builds the system prompt for the classifier model.
func routerClassifierPrompt(routes []contract.PatternRouteConfig) string {
	var b strings.Builder
	b.WriteString("You are a task classifier. Given the user's input, classify it into exactly one of the following categories:\n\n")
	for _, route := range routes {
		b.WriteString(fmt.Sprintf("- %s\n", route.Label))
	}
	b.WriteString("\nRespond with a JSON object: {\"classification\": \"<label>\"}\n")
	b.WriteString("Do not include any other text.")
	return b.String()
}

// routerClassification represents the classifier model's response.
type routerClassification struct {
	Classification string `json:"classification"`
}

// executeRouterLoop runs the router pattern: classify → route.
//
// The loop:
//  1. Calls the classifier model with the task input.
//  2. Parses the classification label.
//  3. Finds the matching route.
//  4. If the route targets a SubAgent: invokes it via the gateway.
//  5. If the route targets a model: invokes the model directly.
//  6. Falls back to the default route if no match is found.
func (r EinoADKRunner) executeRouterLoop(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
) (contract.WorkerResult, bool, error) {
	artifact := request.Artifact
	routes := artifact.Pattern.Routes
	if len(routes) == 0 {
		return contract.WorkerResult{}, false, nil
	}

	// Resolve classifier model.
	classifierModelRef := artifact.Pattern.ModelRef
	if classifierModelRef == "" {
		classifierModelRef = "classifier"
	}
	classifierModel, ok := runtimeInfo.Models[classifierModelRef]
	if !ok {
		return contract.WorkerResult{}, false, fmt.Errorf("router classifier model %q not found in runtime", classifierModelRef)
	}

	// Build classifier prompt.
	classifierPrompt := routerClassifierPrompt(routes)

	// Call the classifier model.
	classifierResult, err := r.modelInvoker().Invoke(ctx, classifierModel, modelConfigFor(artifact, classifierModelRef), promptSpecFromTemplate(classifierPrompt), request.Config.ParsedRunInput, nil)
	if err != nil {
		return contract.WorkerResult{}, false, fmt.Errorf("router classification failed: %w", err)
	}

	// Parse classification.
	var classification routerClassification
	if err := json.Unmarshal([]byte(classifierResult.Content), &classification); err != nil {
		// Try to extract from parsed output.
		if label, ok := classifierResult.Parsed[routerClassificationKey].(string); ok {
			classification.Classification = label
		} else {
			return contract.WorkerResult{}, false, fmt.Errorf("router classifier returned invalid JSON: %s", classifierResult.Content)
		}
	}

	classLabel := strings.TrimSpace(classification.Classification)
	if classLabel == "" {
		return contract.WorkerResult{}, false, fmt.Errorf("router classifier returned empty classification")
	}

	// Find matching route.
	var matchedRoute *contract.PatternRouteConfig
	var defaultRoute *contract.PatternRouteConfig
	for i := range routes {
		if routes[i].Label == classLabel {
			matchedRoute = &routes[i]
		}
		if routes[i].Default {
			defaultRoute = &routes[i]
		}
	}
	if matchedRoute == nil {
		matchedRoute = defaultRoute
	}
	if matchedRoute == nil {
		return contract.WorkerResult{}, false, fmt.Errorf("router: no route matched classification %q and no default route", classLabel)
	}

	// Execute the matched route.
	result, err := r.executeRoute(ctx, request, runtimeInfo, matchedRoute, classLabel)
	if err != nil {
		return contract.WorkerResult{}, false, err
	}

	// Add routing metadata to output.
	if result.Output == nil {
		result.Output = make(map[string]interface{})
	}
	result.Output[routerClassificationKey] = classLabel
	result.Output["matchedRoute"] = matchedRoute.Label
	result.Output["pattern"] = "router"

	task := taskFromRunInput(request.Config.ParsedRunInput)
	result.Message = fmt.Sprintf("router pattern classified %q → route %q for task %q", classLabel, matchedRoute.Label, task)

	return result, true, nil
}

// executeRoute runs the matched route: either invoke a SubAgent or call a model.
func (r EinoADKRunner) executeRoute(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
	route *contract.PatternRouteConfig,
	classLabel string,
) (contract.WorkerResult, error) {
	if strings.TrimSpace(route.AgentRef) != "" {
		return r.executeRouteAgent(ctx, request, runtimeInfo, route)
	}
	return r.executeRouteModel(ctx, request, runtimeInfo, route)
}

// executeRouteAgent invokes a SubAgent via the gateway.
func (r EinoADKRunner) executeRouteAgent(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
	route *contract.PatternRouteConfig,
) (contract.WorkerResult, error) {
	// Resolve SubAgent binding from compiled artifact.
	subAgentName := route.AgentRef
	subAgentNamespace := request.Artifact.Agent.Namespace

	if binding, ok := request.Artifact.Runner.SubAgents[route.AgentRef]; ok {
		if m, ok := binding.(map[string]interface{}); ok {
			if ref, ok := m["ref"].(string); ok && ref != "" {
				subAgentName = ref
			}
			if ns, ok := m["namespace"].(string); ok && ns != "" {
				subAgentNamespace = ns
			}
		}
	}

	gatewayURL := gatewayURL()
	invokeURL := fmt.Sprintf("%s/apis/windosx.com/v1alpha1/namespaces/%s/agents/%s:invoke",
		gatewayURL, subAgentNamespace, subAgentName)

	payload := map[string]interface{}{
		"input": request.Config.ParsedRunInput,
		"execution": map[string]interface{}{
			"mode":   "sync",
			"source": fmt.Sprintf("router-%s", route.Label),
		},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return contract.WorkerResult{}, fmt.Errorf("marshaling SubAgent request: %w", err)
	}

	httpClient := &http.Client{Timeout: 120 * time.Second}
	resp, err := httpClient.Post(invokeURL, "application/json", bytes.NewReader(body))
	if err != nil {
		return contract.WorkerResult{}, fmt.Errorf("invoking SubAgent %q: %w", subAgentName, err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return contract.WorkerResult{}, fmt.Errorf("reading SubAgent response: %w", err)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		return contract.WorkerResult{}, fmt.Errorf("SubAgent %q returned HTTP %d: %s", subAgentName, resp.StatusCode, string(respBody))
	}

	var invokeResp map[string]interface{}
	if err := json.Unmarshal(respBody, &invokeResp); err != nil {
		return contract.WorkerResult{}, fmt.Errorf("parsing SubAgent response: %w", err)
	}

	// If async (201 Created or 202 Accepted), poll for the AgentRun result.
	if resp.StatusCode == http.StatusCreated || resp.StatusCode == http.StatusAccepted {
		agentRunRef, ok := invokeResp["agentRun"].(map[string]interface{})
		if !ok {
			return contract.WorkerResult{}, fmt.Errorf("SubAgent %q: accepted but no agentRun ref", subAgentName)
		}
		agentRunName, _ := agentRunRef["name"].(string)
		agentRunNamespace, _ := agentRunRef["namespace"].(string)
		if agentRunName == "" || agentRunNamespace == "" {
			return contract.WorkerResult{}, fmt.Errorf("SubAgent %q: incomplete agentRun ref", subAgentName)
		}

		result, err := pollAgentRun(ctx, httpClient, gatewayURL, agentRunNamespace, agentRunName)
		if err != nil {
			return contract.WorkerResult{}, fmt.Errorf("SubAgent %q polling: %w", subAgentName, err)
		}
		return result, nil
	}

	return contract.WorkerResult{
		Status:  contract.WorkerStatusSucceeded,
		Message: fmt.Sprintf("router route %q invoked SubAgent %q", route.Label, route.AgentRef),
		Output:  invokeResp,
		Runtime: &runtimeInfo,
	}, nil
}

// pollAgentRun polls the AgentRun status until it completes or times out.
func pollAgentRun(ctx context.Context, httpClient *http.Client, gatewayURL, namespace, name string) (contract.WorkerResult, error) {
	statusURL := fmt.Sprintf("%s/apis/windosx.com/v1alpha1/namespaces/%s/agentruns/%s",
		gatewayURL, namespace, name)

	deadline := time.After(110 * time.Second)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return contract.WorkerResult{}, ctx.Err()
		case <-deadline:
			return contract.WorkerResult{}, fmt.Errorf("timeout waiting for AgentRun %s/%s", namespace, name)
		case <-ticker.C:
			resp, err := httpClient.Get(statusURL)
			if err != nil {
				continue
			}
			body, err := io.ReadAll(resp.Body)
			resp.Body.Close()
			if err != nil {
				continue
			}

			var agentRun map[string]interface{}
			if err := json.Unmarshal(body, &agentRun); err != nil {
				continue
			}

			status, _ := agentRun["status"].(map[string]interface{})
			if status == nil {
				continue
			}

			phase, _ := status["phase"].(string)
			switch phase {
			case "Succeeded":
				output, _ := status["output"].(map[string]interface{})
				if output == nil {
					output = make(map[string]interface{})
				}
				return contract.WorkerResult{
					Status:  contract.WorkerStatusSucceeded,
					Message: fmt.Sprintf("SubAgent AgentRun %s completed", name),
					Output:  output,
				}, nil
			case "Failed":
				conditions, _ := status["conditions"].([]interface{})
				errMsg := "unknown error"
				if len(conditions) > 0 {
					if cond, ok := conditions[0].(map[string]interface{}); ok {
						errMsg, _ = cond["message"].(string)
					}
				}
				return contract.WorkerResult{}, fmt.Errorf("SubAgent AgentRun %s failed: %s", name, errMsg)
			case "Canceled":
				return contract.WorkerResult{}, fmt.Errorf("SubAgent AgentRun %s was canceled", name)
			}
			// Pending/Running - continue polling
		}
	}
}

// executeRouteModel invokes a model directly.
func (r EinoADKRunner) executeRouteModel(
	ctx context.Context,
	request RunRequest,
	runtimeInfo contract.WorkerRuntimeInfo,
	route *contract.PatternRouteConfig,
) (contract.WorkerResult, error) {
	modelRef := route.ModelRef
	if modelRef == "" {
		modelRef = "planner"
	}

	modelCfg, ok := runtimeInfo.Models[modelRef]
	if !ok {
		return contract.WorkerResult{}, fmt.Errorf("router route %q: model %q not found in runtime", route.Label, modelRef)
	}

	systemPrompt := request.Artifact.Runner.Prompts["system"]
	if strings.TrimSpace(systemPrompt.Template) == "" {
		return contract.WorkerResult{}, FailureReasonError{
			Reason:  "MissingPrompt",
			Message: "system prompt is empty in compiled artifact",
		}
	}

	result, err := r.modelInvoker().Invoke(ctx, modelCfg, modelConfigFor(request.Artifact, modelRef), systemPrompt, request.Config.ParsedRunInput, request.Artifact.Runner.Output)
	if err != nil {
		return contract.WorkerResult{}, fmt.Errorf("router route %q model %q failed: %w", route.Label, modelRef, err)
	}

	output := make(map[string]interface{}, len(result.Parsed)+4)
	for k, v := range result.Parsed {
		output[k] = v
	}
	output["model"] = modelRef
	output["modelResponse"] = result.Content

	return contract.WorkerResult{
		Status:  contract.WorkerStatusSucceeded,
		Message: fmt.Sprintf("router route %q executed model %q", route.Label, modelRef),
		Output:  output,
		Artifacts: []contract.WorkerArtifact{
			{Name: "chat-completion-request", Kind: "json", Inline: result.RequestBody},
			{Name: "chat-completion-response", Kind: "json", Inline: result.ResponseBody},
		},
		Runtime: &runtimeInfo,
	}, nil
}

// modelConfigFor extracts a model config from the artifact.
func modelConfigFor(artifact contract.CompiledArtifact, modelRef string) contract.ModelConfig {
	if cfg, ok := artifact.Models[modelRef]; ok {
		return cfg
	}
	return contract.ModelConfig{}
}

// promptSpecFromTemplate creates a PromptSpec from a template string.
func promptSpecFromTemplate(template string) contract.PromptSpec {
	return contract.PromptSpec{
		Template: template,
	}
}

// gatewayURL returns the gateway URL from env or default.
func gatewayURL() string {
	if url := os.Getenv("KORUS_GATEWAY_URL"); url != "" {
		return url
	}
	return defaultGatewayURL
}
