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

	"github.com/surefire-ai/agent-control-plane/internal/contract"
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type OpenAICompatibleInvoker struct {
	Client HTTPDoer
}

type ChatCompletionRequest struct {
	Model          string                  `json:"model"`
	Messages       []ChatCompletionMessage `json:"messages"`
	Temperature    *float64                `json:"temperature,omitempty"`
	MaxTokens      *int32                  `json:"max_tokens,omitempty"`
	ResponseFormat map[string]interface{}  `json:"response_format,omitempty"`
}

type ChatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatCompletionResponse struct {
	ID      string                 `json:"id,omitempty"`
	Model   string                 `json:"model,omitempty"`
	Choices []ChatCompletionChoice `json:"choices,omitempty"`
}

type ChatCompletionChoice struct {
	Index   int                   `json:"index,omitempty"`
	Message ChatCompletionMessage `json:"message,omitempty"`
}

type ModelInvocationResult struct {
	RequestBody  map[string]interface{}
	ResponseBody map[string]interface{}
	Content      string
}

var lookupEnv = os.Getenv

func (i OpenAICompatibleInvoker) Invoke(ctx context.Context, model contract.WorkerModelRuntime, config contract.ModelConfig, prompt contract.PromptSpec, input map[string]interface{}, output map[string]interface{}) (ModelInvocationResult, error) {
	baseURL := strings.TrimRight(model.BaseURL, "/")
	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model.APIKeyEnv == "" {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "MissingModelCredentials",
			Message: fmt.Sprintf("missing model credentials for %q", config.Model),
		}
	}

	apiKey := strings.TrimSpace(lookupEnv(model.APIKeyEnv))
	if apiKey == "" {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "MissingModelCredentials",
			Message: fmt.Sprintf("missing model credentials for %q via %s", config.Model, model.APIKeyEnv),
		}
	}

	requestBody := chatCompletionRequest(config, prompt, input, output)
	rawBody, err := json.Marshal(requestBody)
	if err != nil {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelRequestBuildFailed",
			Message: fmt.Sprintf("failed to marshal model request: %v", err),
		}
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/chat/completions", bytes.NewReader(rawBody))
	if err != nil {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelRequestBuildFailed",
			Message: fmt.Sprintf("failed to create model request: %v", err),
		}
	}
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	client := i.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(httpReq)
	if err != nil {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelCallFailed",
			Message: fmt.Sprintf("model call failed: %v", err),
		}
	}
	defer resp.Body.Close()

	rawResponse, err := io.ReadAll(resp.Body)
	if err != nil {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelCallFailed",
			Message: fmt.Sprintf("failed to read model response: %v", err),
		}
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelCallFailed",
			Message: fmt.Sprintf("model call returned status %d: %s", resp.StatusCode, strings.TrimSpace(string(rawResponse))),
		}
	}

	var response ChatCompletionResponse
	if err := json.Unmarshal(rawResponse, &response); err != nil {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelResponseParseFailed",
			Message: fmt.Sprintf("failed to parse model response: %v", err),
		}
	}
	if len(response.Choices) == 0 || strings.TrimSpace(response.Choices[0].Message.Content) == "" {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelResponseParseFailed",
			Message: "model response did not include a message content",
		}
	}

	var responseBody map[string]interface{}
	if err := json.Unmarshal(rawResponse, &responseBody); err != nil {
		responseBody = map[string]interface{}{}
	}
	return ModelInvocationResult{
		RequestBody:  requestBody,
		ResponseBody: responseBody,
		Content:      strings.TrimSpace(response.Choices[0].Message.Content),
	}, nil
}

func chatCompletionRequest(config contract.ModelConfig, prompt contract.PromptSpec, input map[string]interface{}, output map[string]interface{}) map[string]interface{} {
	request := map[string]interface{}{
		"model": config.Model,
		"messages": []map[string]interface{}{
			{
				"role":    "system",
				"content": prompt.Template,
			},
			{
				"role":    "user",
				"content": userContent(input),
			},
		},
	}
	if config.Temperature != 0 {
		request["temperature"] = config.Temperature
	}
	if config.MaxTokens > 0 {
		request["max_tokens"] = config.MaxTokens
	}
	if len(output) > 0 {
		if schema, ok := output["schema"]; ok {
			request["response_format"] = map[string]interface{}{
				"type": "json_schema",
				"json_schema": map[string]interface{}{
					"name":   "agent_output",
					"schema": schema,
				},
			}
		}
	}
	return request
}

func userContent(input map[string]interface{}) string {
	if len(input) == 0 {
		return "{}"
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
