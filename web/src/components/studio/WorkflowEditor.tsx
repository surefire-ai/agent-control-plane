import { useState, useCallback } from "react";
import { useTranslation } from "react-i18next";
import { Plus, X, GripVertical, ChevronDown, ChevronRight } from "lucide-react";
import type { GraphNode, GraphEdge, GraphConfig } from "@/types/api";
import { Input } from "@/components/shared/Input";
import { Button } from "@/components/shared/Button";

interface WorkflowEditorProps {
  graph: GraphConfig;
  onChange: (graph: GraphConfig) => void;
}

const NODE_KINDS = ["model", "tool", "agent", "knowledge", "custom"] as const;

function kindColor(kind: string): string {
  switch (kind) {
    case "model":
      return "bg-blue-100 text-blue-700 border-blue-300";
    case "tool":
      return "bg-emerald-100 text-emerald-700 border-emerald-300";
    case "agent":
      return "bg-purple-100 text-purple-700 border-purple-300";
    case "knowledge":
      return "bg-amber-100 text-amber-700 border-amber-300";
    default:
      return "bg-zinc-100 text-zinc-600 border-zinc-300";
  }
}

function FieldLabel({ label }: { label: string }) {
  return <label className="mb-1.5 block text-xs font-medium text-zinc-600">{label}</label>;
}

// ─── Node Editor ────────────────────────────────────────────────────────────

function NodeRow({
  node,
  onUpdate,
  onRemove,
}: {
  node: GraphNode;
  index: number;
  onUpdate: (updates: Partial<GraphNode>) => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();
  const [expanded, setExpanded] = useState(false);

  return (
    <div className="rounded-lg border border-zinc-200 bg-white">
      <div className="flex items-center gap-2 px-3 py-2.5">
        <GripVertical className="h-4 w-4 shrink-0 text-zinc-300" />
        <button
          type="button"
          onClick={() => setExpanded(!expanded)}
          className="shrink-0 text-zinc-400 hover:text-zinc-600"
        >
          {expanded ? <ChevronDown className="h-4 w-4" /> : <ChevronRight className="h-4 w-4" />}
        </button>
        <span className={`inline-flex items-center rounded px-2 py-0.5 text-[11px] font-medium border ${kindColor(node.kind)}`}>
          {node.kind}
        </span>
        <span className="flex-1 truncate text-sm font-medium text-zinc-800">
          {node.name || `(${t("studio.workflow.unnamed")})`}
        </span>
        <button
          type="button"
          onClick={onRemove}
          className="rounded p-1 text-zinc-400 hover:bg-rose-50 hover:text-rose-600"
        >
          <X className="h-4 w-4" />
        </button>
      </div>

      {expanded && (
        <div className="border-t border-zinc-100 px-4 py-3">
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <div>
              <FieldLabel label={t("studio.workflow.nodeName")} />
              <Input
                value={node.name}
                onChange={(e) => onUpdate({ name: e.target.value })}
                placeholder="e.g. classify_intent"
              />
            </div>
            <div>
              <FieldLabel label={t("studio.workflow.nodeKind")} />
              <select
                value={node.kind}
                onChange={(e) => onUpdate({ kind: e.target.value })}
                className="w-full rounded-md border border-zinc-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500"
              >
                {NODE_KINDS.map((k) => (
                  <option key={k} value={k}>{k}</option>
                ))}
              </select>
            </div>
            {(node.kind === "model" || node.kind === "custom") && (
              <div>
                <FieldLabel label={t("studio.workflow.modelRef")} />
                <Input
                  value={node.modelRef ?? ""}
                  onChange={(e) => onUpdate({ modelRef: e.target.value })}
                  placeholder="e.g. default-model"
                />
              </div>
            )}
            {node.kind === "tool" && (
              <div>
                <FieldLabel label={t("studio.workflow.toolRef")} />
                <Input
                  value={node.toolRef ?? ""}
                  onChange={(e) => onUpdate({ toolRef: e.target.value })}
                  placeholder="e.g. web-search"
                />
              </div>
            )}
            {node.kind === "knowledge" && (
              <div>
                <FieldLabel label={t("studio.workflow.knowledgeRef")} />
                <Input
                  value={node.knowledgeRef ?? ""}
                  onChange={(e) => onUpdate({ knowledgeRef: e.target.value })}
                  placeholder="e.g. docs-kb"
                />
              </div>
            )}
            {node.kind === "agent" && (
              <div>
                <FieldLabel label={t("studio.workflow.agentRef")} />
                <Input
                  value={node.agentRef ?? ""}
                  onChange={(e) => onUpdate({ agentRef: e.target.value })}
                  placeholder="e.g. support-agent"
                />
              </div>
            )}
            {node.kind === "custom" && (
              <div className="sm:col-span-2">
                <FieldLabel label={t("studio.workflow.implementation")} />
                <Input
                  value={node.implementation ?? ""}
                  onChange={(e) => onUpdate({ implementation: e.target.value })}
                  placeholder="e.g. pkg/custom.MyNode"
                />
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}

// ─── Edge Editor ────────────────────────────────────────────────────────────

function EdgeRow({
  edge,
  index,
  nodeNames,
  onUpdate,
  onRemove,
}: {
  edge: GraphEdge;
  index: number;
  nodeNames: string[];
  onUpdate: (updates: Partial<GraphEdge>) => void;
  onRemove: () => void;
}) {
  const { t } = useTranslation();

  return (
    <div className="flex items-center gap-2 rounded-lg border border-zinc-200 bg-white px-3 py-2.5">
      <span className="shrink-0 text-xs font-medium text-zinc-400 w-5">#{index + 1}</span>
      <select
        value={edge.from}
        onChange={(e) => onUpdate({ from: e.target.value })}
        className="flex-1 rounded-md border border-zinc-300 bg-white px-2 py-1.5 text-sm shadow-sm focus:border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500"
      >
        <option value="">{t("studio.workflow.selectFrom")}</option>
        {nodeNames.map((n) => (
          <option key={n} value={n}>{n}</option>
        ))}
      </select>
      <span className="shrink-0 text-zinc-400">→</span>
      <select
        value={edge.to}
        onChange={(e) => onUpdate({ to: e.target.value })}
        className="flex-1 rounded-md border border-zinc-300 bg-white px-2 py-1.5 text-sm shadow-sm focus:border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500"
      >
        <option value="">{t("studio.workflow.selectTo")}</option>
        {nodeNames.map((n) => (
          <option key={n} value={n}>{n}</option>
        ))}
      </select>
      <Input
        value={edge.when ?? ""}
        onChange={(e) => onUpdate({ when: e.target.value })}
        placeholder={t("studio.workflow.whenPlaceholder")}
        className="w-36 text-xs"
      />
      <button
        type="button"
        onClick={onRemove}
        className="shrink-0 rounded p-1 text-zinc-400 hover:bg-rose-50 hover:text-rose-600"
      >
        <X className="h-4 w-4" />
      </button>
    </div>
  );
}

// ─── Main Workflow Editor ───────────────────────────────────────────────────

export function WorkflowEditor({ graph, onChange }: WorkflowEditorProps) {
  const { t } = useTranslation();
  const nodes = graph.nodes ?? [];
  const edges = graph.edges ?? [];
  const nodeNames = nodes.map((n) => n.name).filter(Boolean);

  const updateNodes = useCallback((updater: (prev: GraphNode[]) => GraphNode[]) => {
    onChange({ ...graph, nodes: updater(nodes) });
  }, [graph, nodes, onChange]);

  const updateEdges = useCallback((updater: (prev: GraphEdge[]) => GraphEdge[]) => {
    onChange({ ...graph, edges: updater(edges) });
  }, [graph, edges, onChange]);

  const addNode = useCallback(() => {
    updateNodes((prev) => [...prev, { name: "", kind: "model" }]);
  }, [updateNodes]);

  const removeNode = useCallback((index: number) => {
    const nodeName = nodes[index]?.name;
    updateNodes((prev) => prev.filter((_, i) => i !== index));
    // Also remove edges referencing this node
    if (nodeName) {
      updateEdges((prev) => prev.filter((e) => e.from !== nodeName && e.to !== nodeName));
    }
  }, [nodes, updateNodes, updateEdges]);

  const updateNode = useCallback((index: number, updates: Partial<GraphNode>) => {
    const oldName = nodes[index]?.name;
    const newName = updates.name;
    updateNodes((prev) => prev.map((n, i) => (i === index ? { ...n, ...updates } : n)));
    // If name changed, update edge references
    if (oldName && newName && oldName !== newName) {
      updateEdges((prev) =>
        prev.map((e) => ({
          ...e,
          from: e.from === oldName ? newName : e.from,
          to: e.to === oldName ? newName : e.to,
        }))
      );
    }
  }, [nodes, updateNodes, updateEdges]);

  const addEdge = useCallback(() => {
    updateEdges((prev) => [...prev, { from: "", to: "" }]);
  }, [updateEdges]);

  const removeEdge = useCallback((index: number) => {
    updateEdges((prev) => prev.filter((_, i) => i !== index));
  }, [updateEdges]);

  const updateEdge = useCallback((index: number, updates: Partial<GraphEdge>) => {
    updateEdges((prev) => prev.map((e, i) => (i === index ? { ...e, ...updates } : e)));
  }, [updateEdges]);

  return (
    <div className="space-y-6">
      {/* Nodes Section */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <div>
            <h4 className="text-sm font-semibold text-zinc-800">{t("studio.workflow.nodes")}</h4>
            <p className="text-xs text-zinc-500">{t("studio.workflow.nodesDesc")}</p>
          </div>
          <Button variant="secondary" size="sm" onClick={addNode} type="button">
            <Plus className="mr-1 h-3.5 w-3.5" />
            {t("studio.workflow.addNode")}
          </Button>
        </div>
        {nodes.length === 0 && (
          <div className="rounded-md border border-dashed border-zinc-300 p-6 text-center">
            <p className="text-sm text-zinc-400">{t("studio.workflow.noNodes")}</p>
          </div>
        )}
        <div className="space-y-2">
          {nodes.map((node, i) => (
            <NodeRow
              key={i}
              node={node}
              index={i}
              onUpdate={(updates) => updateNode(i, updates)}
              onRemove={() => removeNode(i)}
            />
          ))}
        </div>
      </div>

      {/* Edges Section */}
      <div>
        <div className="mb-3 flex items-center justify-between">
          <div>
            <h4 className="text-sm font-semibold text-zinc-800">{t("studio.workflow.edges")}</h4>
            <p className="text-xs text-zinc-500">{t("studio.workflow.edgesDesc")}</p>
          </div>
          <Button variant="secondary" size="sm" onClick={addEdge} type="button" disabled={nodeNames.length < 2}>
            <Plus className="mr-1 h-3.5 w-3.5" />
            {t("studio.workflow.addEdge")}
          </Button>
        </div>
        {edges.length === 0 && (
          <div className="rounded-md border border-dashed border-zinc-300 p-6 text-center">
            <p className="text-sm text-zinc-400">{t("studio.workflow.noEdges")}</p>
          </div>
        )}
        <div className="space-y-2">
          {edges.map((edge, i) => (
            <EdgeRow
              key={i}
              edge={edge}
              index={i}
              nodeNames={nodeNames}
              onUpdate={(updates) => updateEdge(i, updates)}
              onRemove={() => removeEdge(i)}
            />
          ))}
        </div>
      </div>
    </div>
  );
}
