package compiler

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	apiv1alpha1 "github.com/surefire-ai/korus/api/v1alpha1"
	"github.com/surefire-ai/korus/internal/contract"
	"github.com/surefire-ai/korus/internal/providers"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

type ReferenceIndex struct {
	Prompts         map[string]struct{}
	PromptTemplates map[string]apiv1alpha1.PromptTemplateSpec
	KnowledgeBases  map[string]struct{}
	KnowledgeSpecs  map[string]apiv1alpha1.KnowledgeBaseSpec
	Tools           map[string]struct{}
	ToolSpecs       map[string]apiv1alpha1.ToolProviderSpec
	Skills          map[string]struct{}
	SkillSpecs      map[string]apiv1alpha1.SkillSpec
	SubAgents       map[string]struct{}
	// SubAgentRefs maps agent name → its SubAgentRef list, for cycle detection.
	SubAgentRefs map[string][]apiv1alpha1.SubAgentBindingSpec
	MCPServers   map[string]struct{}
	Policies     map[string]struct{}
}

type Result struct {
	Revision string
	Artifact apiv1alpha1.FreeformObject
}

func CompileAgent(agent apiv1alpha1.Agent, refs ReferenceIndex) (Result, error) {
	if err := validatePattern(agent.Spec); err != nil {
		return Result{}, err
	}
	if err := validateModelProviders(agent.Spec.Models); err != nil {
		return Result{}, err
	}
	missing := findMissingReferences(agent, refs)
	if len(missing) > 0 {
		return Result{}, fmt.Errorf("missing references: %v", missing)
	}
	if err := DetectSubAgentCycles(agent.Name, agent.Spec.SubAgentRefs, refs.SubAgentRefs); err != nil {
		return Result{}, err
	}
	if err := validateSkillGraphMerges(agent.Spec, refs.SkillSpecs); err != nil {
		return Result{}, err
	}

	artifact := artifactFor(agent, refs)
	return Result{
		Revision: revisionFor(artifact),
		Artifact: artifact,
	}, nil
}

func artifactFor(agent apiv1alpha1.Agent, refs ReferenceIndex) apiv1alpha1.FreeformObject {
	normalizedModels := providers.NormalizeModels(agent.Spec.Models)
	pattern := patternForArtifact(agent.Spec)
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
		"pattern":       jsonValue(pattern),
		"runner":        jsonValue(runnerForArtifact(agent.Spec, refs, normalizedModels)),
		"models":        jsonValue(normalizedModels),
		"identity":      jsonValue(agent.Spec.Identity),
		"patternSpec":   jsonValue(agent.Spec.Pattern),
		"promptRefs":    jsonValue(agent.Spec.PromptRefs),
		"knowledgeRefs": jsonValue(agent.Spec.KnowledgeRefs),
		"toolRefs":      jsonValue(agent.Spec.ToolRefs),
		"skillRefs":     jsonValue(agent.Spec.SkillRefs),
		"mcpRefs":       jsonValue(agent.Spec.MCPRefs),
		"policyRef":     jsonValue(agent.Spec.PolicyRef),
		"interfaces":    jsonValue(agent.Spec.Interfaces),
		"memory":        jsonValue(agent.Spec.Memory),
		"graph":         jsonValue(agent.Spec.Graph),
		"observability": jsonValue(agent.Spec.Observability),
	}
}

func runnerForArtifact(spec apiv1alpha1.AgentSpec, refs ReferenceIndex, normalizedModels map[string]apiv1alpha1.ModelSpec) map[string]interface{} {
	runtime := runtimeForArtifact(spec.Runtime)
	resolvedPromptRefs := resolvedPromptRefs(spec, refs.SkillSpecs)
	resolvedToolRefs := resolvedToolRefs(spec.ToolRefs, spec.Pattern, spec.SkillRefs, refs.SkillSpecs)
	resolvedKnowledgeRefs := resolvedKnowledgeRefs(spec.KnowledgeRefs, spec.Pattern, spec.SkillRefs, refs.SkillSpecs)
	resolvedGraph := resolvedGraph(spec, refs.SkillSpecs, resolvedToolRefs, resolvedKnowledgeRefs)
	return map[string]interface{}{
		"kind":       "EinoADKRunner",
		"entrypoint": runtime.Entrypoint,
		"pattern":    patternForArtifact(spec),
		"graph": map[string]interface{}{
			"stateSchema": spec.Graph.StateSchema,
			"nodes":       resolvedGraph.Nodes,
			"edges":       resolvedGraph.Edges,
		},
		"prompts": map[string]interface{}{
			"system": promptForArtifact(resolvedPromptRefs.System, refs.PromptTemplates),
		},
		"models":    normalizedModels,
		"providers": providers.CatalogForModels(normalizedModels),
		"tools":     toolsForArtifact(resolvedToolRefs, refs.ToolSpecs),
		"skills":    skillsForArtifact(spec.SkillRefs, refs.SkillSpecs),
		"subAgents": subAgentsForArtifact(spec.SubAgentRefs),
		"knowledge": knowledgeForArtifact(resolvedKnowledgeRefs, refs.KnowledgeSpecs),
		"output": map[string]interface{}{
			"schema": spec.Interfaces.Output.Schema,
		},
	}
}

func validateModelProviders(models map[string]apiv1alpha1.ModelSpec) error {
	for name, model := range models {
		providerName := providers.Normalize(model.Provider)
		if providerName == "" {
			return fmt.Errorf("model %q provider is required", name)
		}
		if _, ok := providers.Lookup(providerName); !ok {
			return fmt.Errorf("model %q uses unsupported provider %q", name, model.Provider)
		}
	}
	return nil
}

func patternForArtifact(spec apiv1alpha1.AgentSpec) map[string]interface{} {
	if spec.Pattern == nil {
		return nil
	}
	pattern := map[string]interface{}{
		"type":          spec.Pattern.Type,
		"version":       spec.Pattern.Version,
		"modelRef":      spec.Pattern.ModelRef,
		"toolRefs":      spec.Pattern.ToolRefs,
		"knowledgeRefs": spec.Pattern.KnowledgeRefs,
		"maxIterations": spec.Pattern.MaxIterations,
		"stopWhen":      spec.Pattern.StopWhen,
	}
	if len(spec.Pattern.Routes) > 0 {
		routes := make([]map[string]interface{}, 0, len(spec.Pattern.Routes))
		for _, r := range spec.Pattern.Routes {
			route := map[string]interface{}{
				"label":   r.Label,
				"default": r.Default,
			}
			if r.AgentRef != "" {
				route["agentRef"] = r.AgentRef
			}
			if r.ModelRef != "" {
				route["modelRef"] = r.ModelRef
			}
			routes = append(routes, route)
		}
		pattern["routes"] = routes
	}
	if expansion := patternExpansionMetadata(spec); expansion != nil {
		pattern["expansion"] = expansion
	}
	return pattern
}

func resolvedPromptRefs(spec apiv1alpha1.AgentSpec, skills map[string]apiv1alpha1.SkillSpec) apiv1alpha1.AgentPromptRefs {
	if strings.TrimSpace(spec.PromptRefs.System) != "" {
		return spec.PromptRefs
	}
	for _, binding := range spec.SkillRefs {
		skill, ok := skills[binding.Ref]
		if !ok {
			continue
		}
		if strings.TrimSpace(skill.PromptRefs.System) == "" {
			continue
		}
		return skill.PromptRefs
	}
	return spec.PromptRefs
}

func resolvedToolRefs(explicit []string, pattern *apiv1alpha1.AgentPatternSpec, skillBindings []apiv1alpha1.SkillBindingSpec, skills map[string]apiv1alpha1.SkillSpec) []string {
	seen := map[string]struct{}{}
	resolved := make([]string, 0, len(explicit))
	appendToolRef := func(name string) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if _, ok := seen[name]; ok {
			return
		}
		seen[name] = struct{}{}
		resolved = append(resolved, name)
	}

	for _, name := range explicit {
		appendToolRef(name)
	}
	if pattern != nil {
		for _, name := range pattern.ToolRefs {
			appendToolRef(name)
		}
	}
	for _, binding := range skillBindings {
		skill, ok := skills[binding.Ref]
		if !ok {
			continue
		}
		for _, name := range skill.ToolRefs {
			appendToolRef(name)
		}
	}
	return resolved
}

func resolvedKnowledgeRefs(explicit []apiv1alpha1.KnowledgeBindingSpec, pattern *apiv1alpha1.AgentPatternSpec, skillBindings []apiv1alpha1.SkillBindingSpec, skills map[string]apiv1alpha1.SkillSpec) []apiv1alpha1.KnowledgeBindingSpec {
	seen := map[string]struct{}{}
	resolved := make([]apiv1alpha1.KnowledgeBindingSpec, 0, len(explicit))
	appendKnowledgeRef := func(binding apiv1alpha1.KnowledgeBindingSpec) {
		key := strings.TrimSpace(binding.Name)
		if key == "" {
			key = strings.TrimSpace(binding.Ref)
		}
		if key == "" {
			return
		}
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		resolved = append(resolved, binding)
	}

	for _, binding := range explicit {
		appendKnowledgeRef(binding)
	}
	if pattern != nil {
		for _, ref := range pattern.KnowledgeRefs {
			appendKnowledgeRef(apiv1alpha1.KnowledgeBindingSpec{Name: ref, Ref: ref})
		}
	}
	for _, skillBinding := range skillBindings {
		skill, ok := skills[skillBinding.Ref]
		if !ok {
			continue
		}
		for _, binding := range skill.KnowledgeRefs {
			appendKnowledgeRef(binding)
		}
	}
	return resolved
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

func skillsForArtifact(bindings []apiv1alpha1.SkillBindingSpec, specs map[string]apiv1alpha1.SkillSpec) map[string]interface{} {
	if len(bindings) == 0 {
		return nil
	}
	skills := make(map[string]interface{}, len(bindings))
	for _, binding := range bindings {
		key := binding.Name
		if key == "" {
			key = binding.Ref
		}
		entry := map[string]interface{}{
			"name": binding.Name,
			"ref":  binding.Ref,
		}
		if spec, ok := specs[binding.Ref]; ok {
			entry["description"] = spec.Description
			entry["promptRefs"] = spec.PromptRefs
			entry["knowledgeRefs"] = spec.KnowledgeRefs
			entry["toolRefs"] = spec.ToolRefs
			entry["functions"] = spec.Functions
			if len(spec.Graph.Nodes) > 0 || len(spec.Graph.Edges) > 0 {
				entry["graph"] = map[string]interface{}{
					"nodes": spec.Graph.Nodes,
					"edges": spec.Graph.Edges,
				}
			}
		}
		skills[key] = entry
	}
	return skills
}

func subAgentsForArtifact(bindings []apiv1alpha1.SubAgentBindingSpec) map[string]interface{} {
	if len(bindings) == 0 {
		return nil
	}
	subAgents := make(map[string]interface{}, len(bindings))
	for _, binding := range bindings {
		key := binding.Name
		if key == "" {
			key = binding.Ref
		}
		entry := map[string]interface{}{
			"name":      binding.Name,
			"ref":       binding.Ref,
			"namespace": binding.Namespace,
		}
		subAgents[key] = entry
	}
	return subAgents
}

func resolvedGraph(spec apiv1alpha1.AgentSpec, skills map[string]apiv1alpha1.SkillSpec, toolRefs []string, knowledgeRefs []apiv1alpha1.KnowledgeBindingSpec) apiv1alpha1.AgentGraphSpec {
	graph := apiv1alpha1.AgentGraphSpec{
		StateSchema: spec.Graph.StateSchema,
		Nodes:       append([]apiv1alpha1.AgentGraphNode(nil), spec.Graph.Nodes...),
		Edges:       append([]apiv1alpha1.AgentGraphEdge(nil), spec.Graph.Edges...),
	}
	if len(graph.Nodes) == 0 && len(graph.Edges) == 0 && spec.Pattern != nil {
		graph = expandPatternGraph(spec.Pattern, graph, toolRefs, knowledgeRefs)
	}
	if len(spec.SkillRefs) == 0 {
		return graph
	}

	skillNodes := make([]apiv1alpha1.AgentGraphNode, 0)
	skillEdges := make([]apiv1alpha1.AgentGraphEdge, 0)
	for _, binding := range spec.SkillRefs {
		skill, ok := skills[binding.Ref]
		if !ok {
			continue
		}
		skillNodes = append(skillNodes, skill.Graph.Nodes...)
		skillEdges = append(skillEdges, skill.Graph.Edges...)
	}
	graph.Nodes = append(skillNodes, graph.Nodes...)
	graph.Edges = append(skillEdges, graph.Edges...)
	return graph
}

func expandPatternGraph(pattern *apiv1alpha1.AgentPatternSpec, graph apiv1alpha1.AgentGraphSpec, toolRefs []string, knowledgeRefs []apiv1alpha1.KnowledgeBindingSpec) apiv1alpha1.AgentGraphSpec {
	if pattern == nil {
		return graph
	}
	switch strings.TrimSpace(pattern.Type) {
	case "react":
		return reactPatternGraph(pattern, graph, toolRefs, knowledgeRefs)
	case "router":
		return routerPatternGraph(pattern, graph)
	default:
		return graph
	}
}

func routerPatternGraph(pattern *apiv1alpha1.AgentPatternSpec, graph apiv1alpha1.AgentGraphSpec) apiv1alpha1.AgentGraphSpec {
	modelRef := strings.TrimSpace(pattern.ModelRef)
	if modelRef == "" {
		modelRef = "classifier"
	}

	nodes := make([]apiv1alpha1.AgentGraphNode, 0, len(pattern.Routes)+2)
	edges := make([]apiv1alpha1.AgentGraphEdge, 0, len(pattern.Routes)*2+2)

	// Classify node.
	nodes = append(nodes, apiv1alpha1.AgentGraphNode{
		Name:     "classify",
		Kind:     "llm",
		ModelRef: modelRef,
	})
	edges = append(edges, apiv1alpha1.AgentGraphEdge{From: "START", To: "classify"})

	// Route nodes.
	for _, route := range pattern.Routes {
		label := strings.TrimSpace(route.Label)
		if label == "" {
			continue
		}
		nodeName := "route_" + strings.ReplaceAll(label, "-", "_")

		if strings.TrimSpace(route.AgentRef) != "" {
			nodes = append(nodes, apiv1alpha1.AgentGraphNode{
				Name:     nodeName,
				Kind:     "agent",
				AgentRef: route.AgentRef,
			})
		} else {
			routeModelRef := strings.TrimSpace(route.ModelRef)
			if routeModelRef == "" {
				routeModelRef = modelRef
			}
			nodes = append(nodes, apiv1alpha1.AgentGraphNode{
				Name:     nodeName,
				Kind:     "llm",
				ModelRef: routeModelRef,
			})
		}

		when := fmt.Sprintf("classification == %q", label)
		if route.Default {
			when = "default"
		}
		edges = append(edges, apiv1alpha1.AgentGraphEdge{
			From: "classify",
			To:   nodeName,
			When: when,
		})
		edges = append(edges, apiv1alpha1.AgentGraphEdge{
			From: nodeName,
			To:   "END",
		})
	}

	graph.Nodes = nodes
	graph.Edges = edges
	return graph
}

func reactPatternGraph(pattern *apiv1alpha1.AgentPatternSpec, graph apiv1alpha1.AgentGraphSpec, toolRefs []string, knowledgeRefs []apiv1alpha1.KnowledgeBindingSpec) apiv1alpha1.AgentGraphSpec {
	modelRef := strings.TrimSpace(pattern.ModelRef)
	if modelRef == "" {
		modelRef = "planner"
	}

	nodes := make([]apiv1alpha1.AgentGraphNode, 0, len(toolRefs)+len(knowledgeRefs)+2)
	edges := make([]apiv1alpha1.AgentGraphEdge, 0, len(toolRefs)+len(knowledgeRefs)+3)

	previous := "START"
	for _, binding := range knowledgeRefs {
		bindingName := knowledgeBindingName(binding)
		if bindingName == "" {
			continue
		}
		nodeName := patternKnowledgeNodeName(bindingName)
		nodes = append(nodes, apiv1alpha1.AgentGraphNode{
			Name:         nodeName,
			Kind:         "retrieval",
			KnowledgeRef: bindingName,
		})
		edges = append(edges, apiv1alpha1.AgentGraphEdge{From: previous, To: nodeName})
		previous = nodeName
	}

	nodes = append(nodes, apiv1alpha1.AgentGraphNode{Name: "reason", Kind: "llm", ModelRef: modelRef})
	edges = append(edges, apiv1alpha1.AgentGraphEdge{From: previous, To: "reason"})

	for _, toolRef := range toolRefs {
		toolRef = strings.TrimSpace(toolRef)
		if toolRef == "" {
			continue
		}
		nodeName := patternToolNodeName(toolRef)
		nodes = append(nodes, apiv1alpha1.AgentGraphNode{
			Name:    nodeName,
			Kind:    "tool",
			ToolRef: toolRef,
		})
		edges = append(edges, apiv1alpha1.AgentGraphEdge{From: "reason", To: nodeName})
	}

	nodes = append(nodes, apiv1alpha1.AgentGraphNode{Name: "finalize", Kind: "llm", ModelRef: modelRef})
	if len(toolRefs) == 0 {
		edges = append(edges, apiv1alpha1.AgentGraphEdge{From: "reason", To: "finalize"})
	} else {
		for _, toolRef := range toolRefs {
			toolRef = strings.TrimSpace(toolRef)
			if toolRef == "" {
				continue
			}
			edges = append(edges, apiv1alpha1.AgentGraphEdge{From: patternToolNodeName(toolRef), To: "finalize"})
		}
		edges = append(edges, apiv1alpha1.AgentGraphEdge{From: "reason", To: "finalize"})
	}
	edges = append(edges, apiv1alpha1.AgentGraphEdge{From: "finalize", To: "END"})

	graph.Nodes = nodes
	graph.Edges = edges
	return graph
}

func patternToolNodeName(toolRef string) string {
	return "tool_" + strings.ReplaceAll(strings.TrimSpace(toolRef), "-", "_")
}

func patternKnowledgeNodeName(bindingName string) string {
	return "retrieve_" + strings.ReplaceAll(strings.TrimSpace(bindingName), "-", "_")
}

func knowledgeBindingName(binding apiv1alpha1.KnowledgeBindingSpec) string {
	name := strings.TrimSpace(binding.Name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(binding.Ref)
}

func patternExpansionMetadata(spec apiv1alpha1.AgentSpec) map[string]interface{} {
	if spec.Pattern == nil {
		return nil
	}
	if len(spec.Graph.Nodes) > 0 || len(spec.Graph.Edges) > 0 {
		return map[string]interface{}{
			"mode": "explicit_graph",
		}
	}
	return map[string]interface{}{
		"mode":      "preset_graph",
		"preset":    spec.Pattern.Type,
		"graphOnly": true,
	}
}

func validatePattern(spec apiv1alpha1.AgentSpec) error {
	if spec.Pattern == nil {
		return nil
	}
	patternType := strings.TrimSpace(spec.Pattern.Type)
	if patternType == "" {
		return fmt.Errorf("pattern.type is required when spec.pattern is set")
	}
	if len(spec.Graph.Nodes) > 0 || len(spec.Graph.Edges) > 0 {
		return fmt.Errorf("spec.pattern cannot be used together with explicit spec.graph")
	}
	modelRef := strings.TrimSpace(spec.Pattern.ModelRef)
	if modelRef != "" {
		if _, ok := spec.Models[modelRef]; !ok {
			return fmt.Errorf("pattern.modelRef %q is not declared under spec.models", modelRef)
		}
	}
	switch patternType {
	case "react":
		return nil
	case "router":
		return validateRouterPattern(spec.Pattern)
	default:
		return fmt.Errorf("pattern.type %q is not supported yet", patternType)
	}
}

func validateRouterPattern(pattern *apiv1alpha1.AgentPatternSpec) error {
	if len(pattern.Routes) == 0 {
		return fmt.Errorf("router pattern requires at least one route")
	}
	hasDefault := false
	for i, route := range pattern.Routes {
		if strings.TrimSpace(route.Label) == "" {
			return fmt.Errorf("router route[%d]: label is required", i)
		}
		if strings.TrimSpace(route.AgentRef) == "" && strings.TrimSpace(route.ModelRef) == "" {
			return fmt.Errorf("router route[%d] %q: agentRef or modelRef is required", i, route.Label)
		}
		if route.Default {
			hasDefault = true
		}
	}
	if !hasDefault {
		return fmt.Errorf("router pattern requires at least one route with default=true")
	}
	return nil
}

func validateSkillGraphMerges(spec apiv1alpha1.AgentSpec, skills map[string]apiv1alpha1.SkillSpec) error {
	seen := map[string]string{}
	recordNode := func(name string, source string) error {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil
		}
		if previous, ok := seen[name]; ok {
			return fmt.Errorf("duplicate graph node %q declared by %s and %s", name, previous, source)
		}
		seen[name] = source
		return nil
	}

	for _, node := range spec.Graph.Nodes {
		if err := recordNode(node.Name, "Agent.spec.graph"); err != nil {
			return err
		}
	}
	for _, binding := range spec.SkillRefs {
		skill, ok := skills[binding.Ref]
		if !ok {
			continue
		}
		sourceName := binding.Name
		if strings.TrimSpace(sourceName) == "" {
			sourceName = binding.Ref
		}
		source := fmt.Sprintf("Skill/%s", sourceName)
		for _, node := range skill.Graph.Nodes {
			if err := recordNode(node.Name, source); err != nil {
				return err
			}
		}
	}
	return nil
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
	if agent.Spec.Pattern != nil {
		for _, toolRef := range agent.Spec.Pattern.ToolRefs {
			if !contains(refs.Tools, toolRef) {
				missing = append(missing, "ToolProvider/"+toolRef)
			}
		}
		for _, knowledgeRef := range agent.Spec.Pattern.KnowledgeRefs {
			if !contains(refs.KnowledgeBases, knowledgeRef) {
				missing = append(missing, "KnowledgeBase/"+knowledgeRef)
			}
		}
	}
	for _, skillRef := range agent.Spec.SkillRefs {
		if !contains(refs.Skills, skillRef.Ref) {
			missing = append(missing, "Skill/"+skillRef.Ref)
		}
	}
	for _, mcpRef := range agent.Spec.MCPRefs {
		if !contains(refs.MCPServers, mcpRef) {
			missing = append(missing, "MCPServer/"+mcpRef)
		}
	}
	for _, subAgentRef := range agent.Spec.SubAgentRefs {
		if !contains(refs.SubAgents, subAgentRef.Ref) {
			missing = append(missing, "Agent/"+subAgentRef.Ref)
		}
		// Self-reference check.
		if subAgentRef.Ref == agent.Name {
			missing = append(missing, fmt.Sprintf("Agent/%s: self-reference in subAgentRefs", agent.Name))
		}
	}

	// Validate graph nodes with kind=agent have a matching subAgentRef.
	subAgentNames := make(map[string]struct{})
	for _, ref := range agent.Spec.SubAgentRefs {
		subAgentNames[ref.Name] = struct{}{}
	}
	for _, node := range agent.Spec.Graph.Nodes {
		if node.Kind == "agent" {
			if strings.TrimSpace(node.AgentRef) == "" {
				missing = append(missing, fmt.Sprintf("graph node %q: kind=agent requires agentRef", node.Name))
			} else if _, ok := subAgentNames[node.AgentRef]; !ok {
				missing = append(missing, fmt.Sprintf("graph node %q: agentRef %q not found in subAgentRefs", node.Name, node.AgentRef))
			}
		}
	}

	// Validate router pattern routes reference valid SubAgents.
	if agent.Spec.Pattern != nil && agent.Spec.Pattern.Type == "router" {
		for _, route := range agent.Spec.Pattern.Routes {
			if ref := strings.TrimSpace(route.AgentRef); ref != "" {
				if _, ok := subAgentNames[ref]; !ok {
					missing = append(missing, fmt.Sprintf("router route %q: agentRef %q not found in subAgentRefs", route.Label, ref))
				}
			}
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

// DetectSubAgentCycles checks for cycles in the SubAgent dependency graph.
// It performs a DFS from the given agent and returns an error if a cycle is found.
func DetectSubAgentCycles(agentName string, agentRefs []apiv1alpha1.SubAgentBindingSpec, index map[string][]apiv1alpha1.SubAgentBindingSpec) error {
	visited := make(map[string]bool)
	return dfsVisit(agentName, agentRefs, index, visited)
}

func dfsVisit(current string, refs []apiv1alpha1.SubAgentBindingSpec, index map[string][]apiv1alpha1.SubAgentBindingSpec, visited map[string]bool) error {
	visited[current] = true
	defer delete(visited, current)

	for _, ref := range refs {
		if visited[ref.Ref] {
			return fmt.Errorf("SubAgent cycle detected: %s → %s", current, ref.Ref)
		}
		childRefs, ok := index[ref.Ref]
		if !ok {
			continue
		}
		if err := dfsVisit(ref.Ref, childRefs, index, visited); err != nil {
			return err
		}
	}
	return nil
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
