package providers

import (
	"sort"
	"strings"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
)

const (
	FamilyOpenAICompatible = "openai-compatible"
	FamilyAnthropicNative  = "anthropic-native"
)

type Spec struct {
	Name                string `json:"name,omitempty"`
	DisplayName         string `json:"displayName,omitempty"`
	Family              string `json:"family,omitempty"`
	Domestic            bool   `json:"domestic,omitempty"`
	DefaultBaseURL      string `json:"defaultBaseURL,omitempty"`
	SupportsJSONSchema  bool   `json:"supportsJsonSchema,omitempty"`
	SupportsToolCalling bool   `json:"supportsToolCalling,omitempty"`
}

var catalog = map[string]Spec{
	"openai": {
		Name:                "openai",
		DisplayName:         "OpenAI",
		Family:              FamilyOpenAICompatible,
		DefaultBaseURL:      "https://api.openai.com/v1",
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"azure-openai": {
		Name:                "azure-openai",
		DisplayName:         "Azure OpenAI",
		Family:              FamilyOpenAICompatible,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"deepseek": {
		Name:                "deepseek",
		DisplayName:         "DeepSeek",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"qwen": {
		Name:                "qwen",
		DisplayName:         "Qwen",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"moonshot": {
		Name:                "moonshot",
		DisplayName:         "Moonshot",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"doubao": {
		Name:                "doubao",
		DisplayName:         "Doubao",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"glm": {
		Name:                "glm",
		DisplayName:         "GLM",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"baichuan": {
		Name:                "baichuan",
		DisplayName:         "Baichuan",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"minimax": {
		Name:                "minimax",
		DisplayName:         "MiniMax",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"siliconflow": {
		Name:                "siliconflow",
		DisplayName:         "SiliconFlow",
		Family:              FamilyOpenAICompatible,
		Domestic:            true,
		SupportsJSONSchema:  true,
		SupportsToolCalling: true,
	},
	"anthropic": {
		Name:                "anthropic",
		DisplayName:         "Anthropic",
		Family:              FamilyAnthropicNative,
		SupportsJSONSchema:  false,
		SupportsToolCalling: true,
	},
}

func Normalize(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func Lookup(name string) (Spec, bool) {
	spec, ok := catalog[Normalize(name)]
	return spec, ok
}

func KnownProviders() []string {
	names := make([]string, 0, len(catalog))
	for name := range catalog {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func NormalizeModels(models map[string]apiv1alpha1.ModelSpec) map[string]apiv1alpha1.ModelSpec {
	if len(models) == 0 {
		return nil
	}
	out := make(map[string]apiv1alpha1.ModelSpec, len(models))
	for name, model := range models {
		normalized := model
		normalized.Provider = Normalize(model.Provider)
		out[name] = normalized
	}
	return out
}

func CatalogForModels(models map[string]apiv1alpha1.ModelSpec) map[string]Spec {
	result := map[string]Spec{}
	for _, model := range models {
		spec, ok := Lookup(model.Provider)
		if !ok {
			continue
		}
		result[spec.Name] = spec
	}
	if len(result) == 0 {
		return nil
	}
	return result
}
