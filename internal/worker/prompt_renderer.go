package worker

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/surefire-ai/korus/internal/contract"
)

// renderPrompt resolves a prompt template with variables from the run input.
// It supports Go text/template syntax in the template string.
// If the template has no template delimiters, it is returned as-is with
// variable values appended as context.
func renderPrompt(prompt contract.PromptSpec, input map[string]interface{}) (string, error) {
	if strings.TrimSpace(prompt.Template) == "" {
		return "", nil
	}

	vars := promptVariables(prompt, input)

	// Check if the template contains Go template delimiters.
	if !strings.Contains(prompt.Template, "{{") {
		return prompt.Template, nil
	}

	tmpl, err := template.New("prompt").Parse(prompt.Template)
	if err != nil {
		return "", FailureReasonError{
			Reason:  "PromptRenderFailed",
			Message: fmt.Sprintf("failed to parse prompt template: %v", err),
		}
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", FailureReasonError{
			Reason:  "PromptRenderFailed",
			Message: fmt.Sprintf("failed to render prompt template: %v", err),
		}
	}
	return buf.String(), nil
}

// promptVariables builds a variable map from the prompt spec and run input.
// Declared variables are extracted from the input. Undeclared input keys
// are available under an "input" key.
func promptVariables(prompt contract.PromptSpec, input map[string]interface{}) map[string]interface{} {
	vars := make(map[string]interface{}, len(prompt.Variables)+1)

	// Map declared variables from input.
	for _, v := range prompt.Variables {
		if value, ok := input[v.Name]; ok {
			vars[v.Name] = value
		}
	}

	// Always make the full input available.
	vars["input"] = input

	return vars
}

// renderUserMessage builds the user-facing message from run input.
// For structured input, it serializes to JSON. For simple string input,
// it returns the string directly.
func renderUserMessage(input map[string]interface{}) string {
	if len(input) == 0 {
		return "{}"
	}

	// If there's a single "message" or "query" key with a string value, use it directly.
	for _, key := range []string{"message", "query", "text", "prompt"} {
		if value, ok := input[key].(string); ok && strings.TrimSpace(value) != "" && len(input) == 1 {
			return value
		}
	}

	raw, err := json.Marshal(input)
	if err != nil {
		return "{}"
	}
	return string(raw)
}
