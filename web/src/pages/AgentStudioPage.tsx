import { useState, useEffect, useCallback, useRef } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { useDocumentTitle } from "@/hooks/useDocumentTitle";
import { useAgent, useUpdateAgent } from "@/api/agents";
import type {
  AgentSpecData,
  PatternConfig,
  ModelConfig,
  KnowledgeBinding,
  SkillBinding,
  SubAgentBinding,
  GraphConfig,
} from "@/types/api";
import { PageHeader } from "@/components/shared/PageHeader";
import { Button } from "@/components/shared/Button";
import { LoadingSkeleton } from "@/components/shared/LoadingSkeleton";
import { ErrorAlert } from "@/components/shared/ErrorAlert";
import { PatternSelector } from "@/components/studio/PatternSelector";
import { PatternConfigForm } from "@/components/studio/PatternConfigForm";
import { ModelConfigForm } from "@/components/studio/ModelConfigForm";
import { BindingPanel, StringArrayBindingPanel } from "@/components/studio/BindingPanel";
import { GraphPreview } from "@/components/studio/GraphPreview";
import { WorkflowCanvas } from "@/components/studio/flow/WorkflowCanvas";
import { Input } from "@/components/shared/Input";

type TabKey = "pattern" | "models" | "preview";

function defaultSpec(): AgentSpecData {
  return {
    pattern: { type: "react", modelRef: "" },
    models: {},
    toolRefs: [],
    knowledgeRefs: [],
    skillRefs: [],
    subAgentRefs: [],
    mcpRefs: [],
  };
}

export function AgentStudioPage() {
  const { t } = useTranslation();
  const { tenantId, agentId } = useParams<{ tenantId: string; agentId: string }>();
  const navigate = useNavigate();
  const { data: agent, isLoading, isError, error, refetch } = useAgent(agentId);
  const saveMutation = useUpdateAgent();

  const [activeTab, setActiveTab] = useState<TabKey>("pattern");
  const [spec, setSpec] = useState<AgentSpecData>(defaultSpec());
  const [saveStatus, setSaveStatus] = useState<"idle" | "saving" | "saved" | "error">("idle");

  // Ref for workflow validation function
  const workflowValidateRef = useRef<(() => string[]) | null>(null);

  useDocumentTitle(agent?.displayName ? `${agent.displayName} — Studio` : t("studio.title"));

  // Initialize spec from agent data
  useEffect(() => {
    if (agent?.spec) {
      setSpec({
        ...defaultSpec(),
        ...agent.spec,
        pattern: agent.spec.pattern ?? { type: agent.pattern ?? "react", modelRef: "" },
        models: agent.spec.models ?? {},
        toolRefs: agent.spec.toolRefs ?? [],
        knowledgeRefs: agent.spec.knowledgeRefs ?? [],
        skillRefs: agent.spec.skillRefs ?? [],
        subAgentRefs: agent.spec.subAgentRefs ?? [],
        mcpRefs: agent.spec.mcpRefs ?? [],
        graph: agent.spec.graph ?? { nodes: [], edges: [] },
      });
    } else if (agent) {
      setSpec({
        ...defaultSpec(),
        pattern: { type: agent.pattern ?? "react", modelRef: "" },
      });
    }
  }, [agent]);

  const handleSave = useCallback(() => {
    if (!agentId) return;

    // Run workflow validation if in workflow mode
    if (spec.pattern?.type === "workflow" && workflowValidateRef.current) {
      const errors = workflowValidateRef.current();
      if (errors.length > 0) {
        setSaveStatus("error");
        return; // Validation errors shown in canvas
      }
    }

    setSaveStatus("saving");
    const patchData = {
      pattern: spec.pattern?.type,
      spec,
    };
    saveMutation.mutate(
      { id: agentId, ...patchData },
      {
        onSuccess: () => {
          setSaveStatus("saved");
          setTimeout(() => {
            navigate(`/tenants/${tenantId}/agents/${agentId}`);
          }, 800);
        },
        onError: () => {
          setSaveStatus("error");
        },
      }
    );
  }, [agentId, spec, saveMutation, tenantId, navigate]);

  const handleGraphChange = useCallback((graph: GraphConfig) => {
    setSpec((prev) => ({ ...prev, graph }));
  }, []);

  const handlePatternSelect = useCallback((pattern: string) => {
    setSpec((prev) => ({
      ...prev,
      pattern: { ...prev.pattern, type: pattern, modelRef: prev.pattern?.modelRef },
    }));
  }, []);

  const handlePatternConfigChange = useCallback((patternConfig: PatternConfig) => {
    setSpec((prev) => ({ ...prev, pattern: patternConfig }));
  }, []);

  const handleModelsChange = useCallback((models: Record<string, ModelConfig>) => {
    setSpec((prev) => ({ ...prev, models }));
  }, []);

  const handleToolRefsChange = useCallback((toolRefs: string[]) => {
    setSpec((prev) => ({ ...prev, toolRefs }));
  }, []);

  const handleMcpRefsChange = useCallback((mcpRefs: string[]) => {
    setSpec((prev) => ({ ...prev, mcpRefs }));
  }, []);

  const handleKnowledgeAdd = useCallback(() => {
    setSpec((prev) => ({ ...prev, knowledgeRefs: [...(prev.knowledgeRefs ?? []), { name: "", ref: "" }] }));
  }, []);

  const handleKnowledgeRemove = useCallback((index: number) => {
    setSpec((prev) => ({ ...prev, knowledgeRefs: (prev.knowledgeRefs ?? []).filter((_, i) => i !== index) }));
  }, []);

  const handleKnowledgeChange = useCallback((index: number, field: keyof KnowledgeBinding, value: string | number) => {
    setSpec((prev) => ({
      ...prev,
      knowledgeRefs: (prev.knowledgeRefs ?? []).map((k, i) => i === index ? { ...k, [field]: value } : k),
    }));
  }, []);

  const handleSkillAdd = useCallback(() => {
    setSpec((prev) => ({ ...prev, skillRefs: [...(prev.skillRefs ?? []), { name: "", ref: "" }] }));
  }, []);

  const handleSkillRemove = useCallback((index: number) => {
    setSpec((prev) => ({ ...prev, skillRefs: (prev.skillRefs ?? []).filter((_, i) => i !== index) }));
  }, []);

  const handleSkillChange = useCallback((index: number, field: keyof SkillBinding, value: string) => {
    setSpec((prev) => ({
      ...prev,
      skillRefs: (prev.skillRefs ?? []).map((s, i) => i === index ? { ...s, [field]: value } : s),
    }));
  }, []);

  const handleSubAgentAdd = useCallback(() => {
    setSpec((prev) => ({ ...prev, subAgentRefs: [...(prev.subAgentRefs ?? []), { name: "", ref: "" }] }));
  }, []);

  const handleSubAgentRemove = useCallback((index: number) => {
    setSpec((prev) => ({ ...prev, subAgentRefs: (prev.subAgentRefs ?? []).filter((_, i) => i !== index) }));
  }, []);

  const handleSubAgentChange = useCallback((index: number, field: keyof SubAgentBinding, value: string) => {
    setSpec((prev) => ({
      ...prev,
      subAgentRefs: (prev.subAgentRefs ?? []).map((s, i) => i === index ? { ...s, [field]: value } : s),
    }));
  }, []);

  if (isLoading) return <LoadingSkeleton />;

  if (isError) {
    return (
      <ErrorAlert
        message={error instanceof Error ? error.message : t("agent.loadError")}
        onRetry={() => refetch()}
      />
    );
  }

  if (!agent) {
    return <ErrorAlert message={t("agent.notFound")} />;
  }

  const isWorkflow = spec.pattern?.type === "workflow";

  const tabVariants: Record<TabKey, "primary" | "secondary"> = {
    pattern: activeTab === "pattern" ? "primary" : "secondary",
    models: activeTab === "models" ? "primary" : "secondary",
    preview: activeTab === "preview" ? "primary" : "secondary",
  };

  return (
    <div>
      <PageHeader title={t("studio.title")} subtitle={t("studio.subtitle")} />

      {/* Toolbar */}
      <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between">
        <div className="flex gap-2">
          <Button variant={tabVariants.pattern} onClick={() => setActiveTab("pattern")}>
            {isWorkflow ? t("studio.tabs.workflow") : t("studio.tabs.pattern")}
          </Button>
          <Button variant={tabVariants.models} onClick={() => setActiveTab("models")}>
            {t("studio.tabs.models")}
          </Button>
          <Button variant={tabVariants.preview} onClick={() => setActiveTab("preview")}>
            {t("studio.tabs.preview")}
          </Button>
        </div>
        <div className="flex gap-2">
          <Button variant="secondary" onClick={() => navigate(`/tenants/${tenantId}/agents/${agentId}`)}>
            {t("studio.cancel")}
          </Button>
          <Button onClick={handleSave} disabled={saveStatus === "saving" || saveStatus === "saved"}>
            {saveStatus === "saving" && t("studio.saving")}
            {saveStatus === "saved" && t("studio.saved")}
            {saveStatus === "error" && t("studio.saveError")}
            {saveStatus === "idle" && t("studio.save")}
          </Button>
        </div>
      </div>

      {/* Tab Content */}
      <div className="rounded-lg border border-zinc-200 bg-white p-6">
        {activeTab === "pattern" && (
          <div>
            <PatternSelector selected={spec.pattern?.type ?? "react"} onSelect={handlePatternSelect} />
            {isWorkflow ? (
              <WorkflowCanvas
                graph={spec.graph ?? { nodes: [], edges: [] }}
                onChange={handleGraphChange}
                onValidateRef={workflowValidateRef}
              />
            ) : (
              <PatternConfigForm
                pattern={spec.pattern?.type ?? "react"}
                config={spec.pattern ?? {}}
                onChange={handlePatternConfigChange}
                graph={spec.graph}
                onGraphChange={handleGraphChange}
              />
            )}
          </div>
        )}

        {activeTab === "models" && (
          <div className="space-y-8">
            <ModelConfigForm models={spec.models ?? {}} onChange={handleModelsChange} />
            <hr className="border-zinc-200" />
            <StringArrayBindingPanel
              title={t("studio.bindings.tools")}
              description={t("studio.bindings.toolsDesc")}
              addLabel={t("studio.bindings.addTool")}
              items={spec.toolRefs ?? []}
              onChange={handleToolRefsChange}
              placeholder="e.g. web-search, calculator"
            />
            <hr className="border-zinc-200" />
            <BindingPanel
              title={t("studio.bindings.knowledge")}
              description={t("studio.bindings.knowledgeDesc")}
              addLabel={t("studio.bindings.addKnowledge")}
              items={spec.knowledgeRefs ?? []}
              onAdd={handleKnowledgeAdd}
              onRemove={handleKnowledgeRemove}
              renderItem={(item, index) => {
                const kb = item as KnowledgeBinding;
                return (
                  <div className="flex gap-2">
                    <Input value={kb.name} placeholder="Name" onChange={(e) => handleKnowledgeChange(index, "name", e.target.value)} />
                    <Input value={kb.ref} placeholder="Ref" onChange={(e) => handleKnowledgeChange(index, "ref", e.target.value)} />
                  </div>
                );
              }}
            />
            <hr className="border-zinc-200" />
            <BindingPanel
              title={t("studio.bindings.skills")}
              description={t("studio.bindings.skillsDesc")}
              addLabel={t("studio.bindings.addSkill")}
              items={spec.skillRefs ?? []}
              onAdd={handleSkillAdd}
              onRemove={handleSkillRemove}
              renderItem={(item, index) => {
                const skill = item as SkillBinding;
                return (
                  <div className="flex gap-2">
                    <Input value={skill.name} placeholder="Name" onChange={(e) => handleSkillChange(index, "name", e.target.value)} />
                    <Input value={skill.ref} placeholder="Ref" onChange={(e) => handleSkillChange(index, "ref", e.target.value)} />
                  </div>
                );
              }}
            />
            <hr className="border-zinc-200" />
            <BindingPanel
              title={t("studio.bindings.subAgents")}
              description={t("studio.bindings.subAgentsDesc")}
              addLabel={t("studio.bindings.addSubAgent")}
              items={spec.subAgentRefs ?? []}
              onAdd={handleSubAgentAdd}
              onRemove={handleSubAgentRemove}
              renderItem={(item, index) => {
                const sa = item as SubAgentBinding;
                return (
                  <div className="flex gap-2">
                    <Input value={sa.name} placeholder="Name" onChange={(e) => handleSubAgentChange(index, "name", e.target.value)} />
                    <Input value={sa.ref} placeholder="Ref" onChange={(e) => handleSubAgentChange(index, "ref", e.target.value)} />
                  </div>
                );
              }}
            />
            <hr className="border-zinc-200" />
            <StringArrayBindingPanel
              title="MCP Servers"
              description="Bind MCP server references"
              addLabel="Add MCP Server"
              items={spec.mcpRefs ?? []}
              onChange={handleMcpRefsChange}
              placeholder="e.g. filesystem-mcp"
            />
          </div>
        )}

        {activeTab === "preview" && <GraphPreview spec={spec} />}
      </div>
    </div>
  );
}
