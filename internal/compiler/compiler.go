package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	apiv1alpha1 "github.com/surefire-ai/agent-control-plane/api/v1alpha1"
	"github.com/surefire-ai/agent-control-plane/internal/contract"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type ReferenceIndex struct {
	Prompts         map[string]struct{}
	PromptTemplates map[string]apiv1alpha1.PromptTemplateSpec
	KnowledgeBases  map[string]struct{}
	KnowledgeSpecs  map[string]apiv1alpha1.KnowledgeBaseSpec
	Tools           map[string]struct{}
	ToolSpecs       map[string]apiv1alpha1.ToolProviderSpec
	MCPServers      map[string]struct{}
	Policies        map[string]struct{}
}

type Result struct {
	Revision string
	Artifact apiv1alpha1.FreeformObject
}

func CompileAgent(agent apiv1alpha1.Agent, refs ReferenceIndex) (Result, error) {
	missing := findMissingReferences(agent, refs)
	if len(missing) > 0 {
		return Result{}, fmt.Errorf("missing references: %v", missing)
	}

	artifact := artifactFor(agent, refs)
	return Result{
		Revision: revisionFor(artifact),
		Artifact: artifact,
	}, nil
}

func artifactFor(agent apiv1alpha1.Agent, refs ReferenceIndex) apiv1alpha1.FreeformObject {
	return apiv1alpha1.FreeformObject{
		"apiVersion":    jsonValue(apiv1alpha1.Group + "/" + apiv1alpha1.Version),
		"kind":          jsonValue(contract.CompiledArtifactKind),
		"schemaVersion": jsonValue(contract.CompiledArtifactSchemaV1),
		"agent": jsonValue(map[string]interface{}{
			"name":       agent.Name,
			"namespace":  agent.Namespace,
			"generation": agent.Generation,
		}),
		"runtime":       jsonValue(runtimeForArtifact(agent.Spec.Runtime)),
		"runner":        jsonValue(runnerForArtifact(agent.Spec, refs)),
		"models":        jsonValue(agent.Spec.Models),
		"identity":      jsonValue(agent.Spec.Identity),
		"promptRefs":    jsonValue(agent.Spec.PromptRefs),
		"knowledgeRefs": jsonValue(agent.Spec.KnowledgeRefs),
		"toolRefs":      jsonValue(agent.Spec.ToolRefs),
		"mcpRefs":       jsonValue(agent.Spec.MCPRefs),
		"policyRef":     jsonValue(agent.Spec.PolicyRef),
		"interfaces":    jsonValue(agent.Spec.Interfaces),
		"memory":        jsonValue(agent.Spec.Memory),
		"graph":         jsonValue(agent.Spec.Graph),
		"observability": jsonValue(agent.Spec.Observability),
	}
}

func runnerForArtifact(spec apiv1alpha1.AgentSpec, refs ReferenceIndex) map[string]interface{} {
	runtime := runtimeForArtifact(spec.Runtime)
	return map[string]interface{}{
		"kind":       "EinoADKRunner",
		"entrypoint": runtime.Entrypoint,
		"graph": map[string]interface{}{
			"stateSchema": spec.Graph.StateSchema,
			"nodes":       spec.Graph.Nodes,
			"edges":       spec.Graph.Edges,
		},
		"prompts": map[string]interface{}{
			"system": promptForArtifact(spec.PromptRefs.System, refs.PromptTemplates),
		},
		"models":    spec.Models,
		"tools":     toolsForArtifact(spec.ToolRefs, refs.ToolSpecs),
		"knowledge": knowledgeForArtifact(spec.KnowledgeRefs, refs.KnowledgeSpecs),
		"output": map[string]interface{}{
			"schema": spec.Interfaces.Output.Schema,
		},
	}
}

func promptForArtifact(name string, prompts map[string]apiv1alpha1.PromptTemplateSpec) map[string]interface{} {
	prompt := map[string]interface{}{
		"name": name,
	}
	if name == "" || prompts == nil {
		return prompt
	}
	spec, ok := prompts[name]
	if !ok {
		return prompt
	}

	variables := make([]map[string]interface{}, 0, len(spec.Variables))
	for _, variable := range spec.Variables {
		variables = append(variables, map[string]interface{}{
			"name":     variable.Name,
			"required": variable.Required,
		})
	}

	prompt["language"] = spec.Language
	prompt["template"] = spec.Template
	prompt["variables"] = variables
	prompt["outputConstraints"] = spec.OutputConstraints
	return prompt
}

func toolsForArtifact(toolRefs []string, specs map[string]apiv1alpha1.ToolProviderSpec) map[string]interface{} {
	if len(toolRefs) == 0 {
		return nil
	}
	tools := make(map[string]interface{}, len(toolRefs))
	for _, name := range toolRefs {
		entry := map[string]interface{}{
			"name": name,
		}
		if spec, ok := specs[name]; ok {
			entry["type"] = spec.Type
			entry["description"] = spec.Description
			entry["schema"] = spec.Schema
			if len(spec.Runtime) > 0 {
				entry["runtime"] = spec.Runtime
			}
			if len(spec.HTTP) > 0 {
				entry["http"] = spec.HTTP
			}
		}
		tools[name] = entry
	}
	return tools
}

func knowledgeForArtifact(bindings []apiv1alpha1.KnowledgeBindingSpec, specs map[string]apiv1alpha1.KnowledgeBaseSpec) map[string]interface{} {
	if len(bindings) == 0 {
		return nil
	}
	knowledge := make(map[string]interface{}, len(bindings))
	for _, binding := range bindings {
		entry := map[string]interface{}{
			"name": binding.Name,
			"ref":  binding.Ref,
		}
		if len(binding.Retrieval) > 0 {
			entry["binding"] = map[string]interface{}{
				"retrieval": binding.Retrieval,
			}
		}
		if spec, ok := specs[binding.Ref]; ok {
			entry["description"] = spec.Description
			entry["sources"] = spec.Sources
			if len(spec.Access) > 0 {
				entry["access"] = spec.Access
			}
			if len(spec.Index) > 0 {
				entry["index"] = spec.Index
			}
			if len(spec.Retrieval) > 0 {
				entry["retrieval"] = spec.Retrieval
			}
			if len(spec.Embedding) > 0 {
				entry["embedding"] = spec.Embedding
			}
		}
		key := binding.Name
		if key == "" {
			key = binding.Ref
		}
		knowledge[key] = entry
	}
	return knowledge
}

func runtimeForArtifact(runtime apiv1alpha1.AgentRuntimeSpec) apiv1alpha1.AgentRuntimeSpec {
	identity := contract.RuntimeIdentityFromSpec(contract.RuntimeSpec{
		Engine:      runtime.Engine,
		RunnerClass: runtime.RunnerClass,
	})
	runtime.Engine = identity.Engine
	runtime.RunnerClass = identity.RunnerClass
	return runtime
}

func findMissingReferences(agent apiv1alpha1.Agent, refs ReferenceIndex) []string {
	var missing []string

	if agent.Spec.PromptRefs.System != "" && !contains(refs.Prompts, agent.Spec.PromptRefs.System) {
		missing = append(missing, "PromptTemplate/"+agent.Spec.PromptRefs.System)
	}
	if agent.Spec.PolicyRef != "" && !contains(refs.Policies, agent.Spec.PolicyRef) {
		missing = append(missing, "AgentPolicy/"+agent.Spec.PolicyRef)
	}
	for _, knowledgeRef := range agent.Spec.KnowledgeRefs {
		if !contains(refs.KnowledgeBases, knowledgeRef.Ref) {
			missing = append(missing, "KnowledgeBase/"+knowledgeRef.Ref)
		}
	}
	for _, toolRef := range agent.Spec.ToolRefs {
		if !contains(refs.Tools, toolRef) {
			missing = append(missing, "ToolProvider/"+toolRef)
		}
	}
	for _, mcpRef := range agent.Spec.MCPRefs {
		if !contains(refs.MCPServers, mcpRef) {
			missing = append(missing, "MCPServer/"+mcpRef)
		}
	}

	sort.Strings(missing)
	return missing
}

func contains(values map[string]struct{}, name string) bool {
	if values == nil {
		return false
	}
	_, ok := values[name]
	return ok
}

func revisionFor(artifact apiv1alpha1.FreeformObject) string {
	raw, err := json.Marshal(artifact)
	if err != nil {
		raw = []byte("{}")
	}
	hash := sha256.Sum256(raw)
	return "sha256:" + hex.EncodeToString(hash[:])
}

func jsonValue(value interface{}) apiextensionsv1.JSON {
	raw, err := json.Marshal(value)
	if err != nil {
		raw = []byte("null")
	}
	return apiextensionsv1.JSON{Raw: raw}
}
