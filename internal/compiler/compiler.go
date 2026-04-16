package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"

	apiv1alpha1 "github.com/windosx/agent-control-plane/api/v1alpha1"
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
}

func CompileAgent(agent apiv1alpha1.Agent, refs ReferenceIndex) (Result, error) {
	missing := findMissingReferences(agent, refs)
	if len(missing) > 0 {
		return Result{}, fmt.Errorf("missing references: %v", missing)
	}

	return Result{
		Revision: revisionFor(agent),
	}, nil
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

func revisionFor(agent apiv1alpha1.Agent) string {
	hash := sha256.New()
	hash.Write([]byte(agent.Namespace))
	hash.Write([]byte("/"))
	hash.Write([]byte(agent.Name))
	hash.Write([]byte(fmt.Sprintf("%d", agent.Generation)))
	return "sha256:" + hex.EncodeToString(hash.Sum(nil))
}
