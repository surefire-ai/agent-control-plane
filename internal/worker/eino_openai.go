package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	einojsonschema "github.com/eino-contrib/jsonschema"

	einoopenai "github.com/cloudwego/eino-ext/components/model/openai"
	einoschema "github.com/cloudwego/eino/schema"

	"github.com/surefire-ai/korus/internal/contract"
)

type EinoOpenAIInvoker struct {
	Client *http.Client
}

func (i EinoOpenAIInvoker) Invoke(ctx context.Context, model contract.WorkerModelRuntime, config contract.ModelConfig, prompt contract.PromptSpec, input map[string]interface{}, output map[string]interface{}) (ModelInvocationResult, error) {
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

	responseFormat, err := einoResponseFormat(output)
	if err != nil {
		return ModelInvocationResult{}, err
	}

	var temperature *float32
	if config.Temperature != 0 {
		value := float32(config.Temperature)
		temperature = &value
	}
	var maxTokens *int
	if config.MaxTokens > 0 {
		value := int(config.MaxTokens)
		maxTokens = &value
	}

	chatModel, err := einoopenai.NewChatModel(ctx, &einoopenai.ChatModelConfig{
		APIKey:         apiKey,
		BaseURL:        baseURL,
		Model:          config.Model,
		HTTPClient:     i.Client,
		Temperature:    temperature,
		MaxTokens:      maxTokens,
		ResponseFormat: responseFormat,
	})
	if err != nil {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelRequestBuildFailed",
			Message: fmt.Sprintf("failed to initialize Eino chat model: %v", err),
		}
	}

	requestBody := chatCompletionRequest(config, prompt, input, output)
	message, err := chatModel.Generate(ctx, []*einoschema.Message{
		einoschema.SystemMessage(prompt.Template),
		einoschema.UserMessage(userContent(input)),
	})
	if err != nil {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelCallFailed",
			Message: fmt.Sprintf("model call failed: %v", err),
		}
	}
	if strings.TrimSpace(message.Content) == "" {
		return ModelInvocationResult{}, FailureReasonError{
			Reason:  "ModelResponseParseFailed",
			Message: "model response did not include a message content",
		}
	}

	parsed, err := parseModelContent(message.Content)
	if err != nil {
		return ModelInvocationResult{}, err
	}
	if err := validateOutputSchema(parsed, output); err != nil {
		return ModelInvocationResult{}, err
	}

	return ModelInvocationResult{
		RequestBody: requestBody,
		ResponseBody: map[string]interface{}{
			"message": map[string]interface{}{
				"role":    string(message.Role),
				"content": message.Content,
			},
		},
		Content: strings.TrimSpace(message.Content),
		Parsed:  parsed,
	}, nil
}

func einoResponseFormat(output map[string]interface{}) (*einoopenai.ChatCompletionResponseFormat, error) {
	if len(output) == 0 {
		return nil, nil
	}
	rawSchema, ok := output["schema"]
	if !ok {
		return nil, nil
	}

	schemaBytes, err := json.Marshal(rawSchema)
	if err != nil {
		return nil, FailureReasonError{
			Reason:  "ModelRequestBuildFailed",
			Message: fmt.Sprintf("failed to marshal output schema: %v", err),
		}
	}

	var schema einojsonschema.Schema
	if err := json.Unmarshal(schemaBytes, &schema); err != nil {
		return nil, FailureReasonError{
			Reason:  "ModelRequestBuildFailed",
			Message: fmt.Sprintf("failed to convert output schema: %v", err),
		}
	}

	return &einoopenai.ChatCompletionResponseFormat{
		Type: einoopenai.ChatCompletionResponseFormatTypeJSONSchema,
		JSONSchema: &einoopenai.ChatCompletionResponseFormatJSONSchema{
			Name:        "agent_output",
			Description: "structured agent output",
			Strict:      false,
			JSONSchema:  &schema,
		},
	}, nil
}
