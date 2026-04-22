package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	einojsonschema "github.com/eino-contrib/jsonschema"

	einotool "github.com/cloudwego/eino/components/tool"
	einoschema "github.com/cloudwego/eino/schema"

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type HTTPToolInvoker struct {
	Client HTTPDoer
}

type ToolInvoker interface {
	Invoke(ctx context.Context, runtime contract.WorkerToolRuntime, spec contract.ToolSpec, input map[string]interface{}) (ToolInvocationResult, error)
}

type ToolInvocationResult struct {
	Name         string
	RequestBody  map[string]interface{}
	ResponseBody map[string]interface{}
	Output       map[string]interface{}
}

type EinoHTTPTool struct {
	Runtime contract.WorkerToolRuntime
	Spec    contract.ToolSpec
	Invoker HTTPToolInvoker
}

var _ einotool.InvokableTool = (*EinoHTTPTool)(nil)

type EinoToolInvoker struct {
	Client HTTPDoer
}

func (i EinoToolInvoker) Invoke(ctx context.Context, runtime contract.WorkerToolRuntime, spec contract.ToolSpec, input map[string]interface{}) (ToolInvocationResult, error) {
	tool := EinoHTTPTool{
		Runtime: runtime,
		Spec:    spec,
		Invoker: HTTPToolInvoker{Client: i.Client},
	}
	rawInput, err := json.Marshal(input)
	if err != nil {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolRequestBuildFailed",
			Message: fmt.Sprintf("failed to marshal tool input: %v", err),
		}
	}
	result, err := tool.InvokableRun(ctx, string(rawInput))
	if err != nil {
		return ToolInvocationResult{}, err
	}
	var output map[string]interface{}
	if err := json.Unmarshal([]byte(result), &output); err != nil {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolResponseParseFailed",
			Message: fmt.Sprintf("tool response must be valid JSON object: %v", err),
		}
	}
	return ToolInvocationResult{
		Name:         spec.Name,
		RequestBody:  input,
		ResponseBody: output,
		Output:       output,
	}, nil
}

func (i HTTPToolInvoker) Invoke(ctx context.Context, runtime contract.WorkerToolRuntime, spec contract.ToolSpec, input map[string]interface{}) (ToolInvocationResult, error) {
	if !containsString(runtime.Capabilities, "http") || len(spec.HTTP) == 0 {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "UnsupportedToolType",
			Message: fmt.Sprintf("tool %q does not expose an http runtime", spec.Name),
		}
	}

	method := strings.ToUpper(strings.TrimSpace(stringValue(spec.HTTP, "method")))
	if method == "" {
		method = http.MethodPost
	}
	url := strings.TrimSpace(stringValue(spec.HTTP, "url"))
	if url == "" {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolRequestBuildFailed",
			Message: fmt.Sprintf("tool %q is missing http.url", spec.Name),
		}
	}

	rawBody, err := json.Marshal(input)
	if err != nil {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolRequestBuildFailed",
			Message: fmt.Sprintf("failed to marshal tool request: %v", err),
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewReader(rawBody))
	if err != nil {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolRequestBuildFailed",
			Message: fmt.Sprintf("failed to create tool request: %v", err),
		}
	}
	req.Header.Set("Content-Type", "application/json")

	authType := strings.TrimSpace(stringValue(nestedObject(spec.HTTP, "auth"), "type"))
	if authType == "bearerToken" {
		if strings.TrimSpace(runtime.AuthTokenEnv) == "" {
			return ToolInvocationResult{}, FailureReasonError{
				Reason:  "MissingToolCredentials",
				Message: fmt.Sprintf("missing tool credentials for %q", spec.Name),
			}
		}
		token := strings.TrimSpace(lookupEnv(runtime.AuthTokenEnv))
		if token == "" {
			return ToolInvocationResult{}, FailureReasonError{
				Reason:  "MissingToolCredentials",
				Message: fmt.Sprintf("missing tool credentials for %q via %s", spec.Name, runtime.AuthTokenEnv),
			}
		}
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := i.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolCallFailed",
			Message: fmt.Sprintf("tool call failed: %v", err),
		}
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolCallFailed",
			Message: fmt.Sprintf("failed to read tool response: %v", err),
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ToolInvocationResult{}, FailureReasonError{
			Reason:  "ToolCallFailed",
			Message: fmt.Sprintf("tool call returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawResponse))),
		}
	}

	var responseBody map[string]interface{}
	if len(rawResponse) > 0 {
		if err := json.Unmarshal(rawResponse, &responseBody); err != nil {
			return ToolInvocationResult{}, FailureReasonError{
				Reason:  "ToolResponseParseFailed",
				Message: fmt.Sprintf("tool response must be valid JSON object: %v", err),
			}
		}
	}
	if responseBody == nil {
		responseBody = map[string]interface{}{}
	}
	if err := validateToolOutputSchema(responseBody, spec.Schema); err != nil {
		return ToolInvocationResult{}, err
	}
	return ToolInvocationResult{
		Name:         spec.Name,
		RequestBody:  input,
		ResponseBody: responseBody,
		Output:       responseBody,
	}, nil
}

func (t EinoHTTPTool) Info(ctx context.Context) (*einoschema.ToolInfo, error) {
	_ = ctx
	info := &einoschema.ToolInfo{
		Name:  t.Spec.Name,
		Desc:  t.Spec.Description,
		Extra: map[string]any{"type": t.Spec.Type},
	}
	inputSchema := nestedObject(t.Spec.Schema, "input")
	if len(inputSchema) == 0 {
		return info, nil
	}

	schemaBytes, err := json.Marshal(inputSchema)
	if err != nil {
		return nil, FailureReasonError{
			Reason:  "ToolRequestBuildFailed",
			Message: fmt.Sprintf("failed to marshal tool input schema: %v", err),
		}
	}
	var schema einojsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, FailureReasonError{
			Reason:  "ToolRequestBuildFailed",
			Message: fmt.Sprintf("failed to convert tool input schema: %v", err),
		}
	}
	info.ParamsOneOf = einoschema.NewParamsOneOfByJSONSchema(&schema)
	return info, nil
}

func (t EinoHTTPTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...einotool.Option) (string, error) {
	_ = opts
	var input map[string]interface{}
	if strings.TrimSpace(argumentsInJSON) != "" {
		if err := json.Unmarshal([]byte(argumentsInJSON), &input); err != nil {
			return "", FailureReasonError{
				Reason:  "ToolRequestBuildFailed",
				Message: fmt.Sprintf("tool input must be valid JSON object: %v", err),
			}
		}
	}
	if input == nil {
		input = map[string]interface{}{}
	}

	result, err := t.Invoker.Invoke(ctx, t.Runtime, t.Spec, input)
	if err != nil {
		return "", err
	}
	rawOutput, err := json.Marshal(result.Output)
	if err != nil {
		return "", FailureReasonError{
			Reason:  "ToolResponseParseFailed",
			Message: fmt.Sprintf("failed to marshal tool output: %v", err),
		}
	}
	return string(rawOutput), nil
}

func validateToolOutputSchema(result map[string]interface{}, schema map[string]interface{}) error {
	if len(schema) == 0 {
		return nil
	}
	outputSchema := nestedObject(schema, "output")
	if len(outputSchema) == 0 {
		return nil
	}
	return validateSchemaObject(result, outputSchema)
}

func validateSchemaObject(result map[string]interface{}, schema map[string]interface{}) error {
	if schemaType, _ := schema["type"].(string); schemaType != "" && schemaType != "object" {
		return FailureReasonError{
			Reason:  "OutputSchemaValidationFailed",
			Message: fmt.Sprintf("unsupported output schema type %q", schemaType),
		}
	}
	if required, ok := schema["required"].([]interface{}); ok {
		for _, item := range required {
			name, _ := item.(string)
			if name == "" {
				continue
			}
			if _, exists := result[name]; !exists {
				return FailureReasonError{
					Reason:  "OutputSchemaValidationFailed",
					Message: fmt.Sprintf("tool response missing required field %q", name),
				}
			}
		}
	}
	return nil
}

func nestedObject(values map[string]interface{}, key string) map[string]interface{} {
	if len(values) == 0 {
		return nil
	}
	output, _ := values[key].(map[string]interface{})
	return output
}

func stringValue(values map[string]interface{}, key string) string {
	if len(values) == 0 {
		return ""
	}
	output, _ := values[key].(string)
	return output
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
