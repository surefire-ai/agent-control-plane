package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
	"github.com/windosx/agent-control-plane/internal/contract"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type ReferenceIndex struct {
	Prompts        map[string]struct{}
	KnowledgeBases map[string]struct{}
	Tools          map[string]struct{}
	MCPServers     map[string]struct{}
	Policies       map[string]struct{}
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

	artifact := artifactFor(agent)
	return Result{
		Revision: revisionFor(artifact),
		Artifact: artifact,
	}, nil
}

func artifactFor(agent apiv1alpha1.Agent) apiv1alpha1.FreeformObject {
	return apiv1alpha1.FreeformObject{
		"apiVersion": jsonValue(apiv1alpha1.Group + "/" + apiv1alpha1.Version),
		"kind":       jsonValue("AgentCompiledArtifact"),
		"agent": jsonValue(map[string]interface{}{
			"name":       agent.Name,
			"namespace":  agent.Namespace,
			"generation": agent.Generation,
		}),
		"runtime":       jsonValue(runtimeForArtifact(agent.Spec.Runtime)),
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

func runtimeForArtifact(runtime apiv1alpha1.AgentRuntimeSpec) apiv1alpha1.AgentRuntimeSpec {
	if runtime.Engine == "" {
		runtime.Engine = contract.RuntimeEngineEino
	}
	if runtime.RunnerClass == "" {
		runtime.RunnerClass = contract.RunnerClassADK
	}
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
