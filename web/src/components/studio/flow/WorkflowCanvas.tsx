import { useState, useCallback, useMemo, useRef, useEffect } from "react";
import {
  ReactFlow,
  Background,
  Controls,
  MiniMap,
  addEdge,
  useNodesState,
  useEdgesState,
  type Connection,
  type Edge,
  type Node,
  type NodeTypes,
  type OnNodesChange,
  type OnEdgesChange,
  BackgroundVariant,
  MarkerType,
} from "@xyflow/react";
import "@xyflow/react/dist/style.css";
import dagre from "dagre";
import type { GraphConfig, GraphNode, GraphEdge } from "@/types/api";
import { WorkflowNode, type WorkflowNodeData } from "./WorkflowNode";
import { NodePalette } from "./NodePalette";
import { NodeConfigPanel } from "./NodeConfigPanel";
import { useTranslation } from "react-i18next";

interface WorkflowCanvasProps {
  graph: GraphConfig;
  onChange: (graph: GraphConfig) => void;
  /** Called by parent to trigger validation before save */
  onValidateRef?: React.MutableRefObject<(() => string[]) | null>;
}

// ─── Conversion ───────────────────────────────────────────────────

function toFlow(graph: GraphConfig): { rfNodes: Node<WorkflowNodeData>[]; rfEdges: Edge[] } {
  const rfNodes: Node<WorkflowNodeData>[] = (graph.nodes ?? []).map((n, i) => ({
    id: n.name || `node-${i}`,
    type: "workflow" as const,
    position: n.position ?? { x: 250 + (i % 3) * 220, y: 80 + Math.floor(i / 3) * 140 },
    data: {
      label: n.name,
      kind: (n.kind as WorkflowNodeData["kind"]) ?? "model",
      modelRef: n.modelRef,
      toolRef: n.toolRef,
      knowledgeRef: n.knowledgeRef,
      agentRef: n.agentRef,
      implementation: n.implementation,
    },
  }));

  const rfEdges: Edge[] = (graph.edges ?? []).map((e, i) => ({
    id: `edge-${e.from}-${e.to}-${i}`,
    source: e.from,
    target: e.to,
    label: e.when || undefined,
    animated: !!e.when,
    style: { stroke: "#94a3b8", strokeWidth: 2 },
    markerEnd: { type: MarkerType.ArrowClosed, color: "#94a3b8" },
    labelStyle: { fontSize: 11, fill: "#64748b", fontWeight: 500 },
    labelBgStyle: { fill: "#f8fafc", stroke: "#e2e8f0", strokeWidth: 1 },
    labelBgPadding: [4, 2] as [number, number],
    labelBgBorderRadius: 4,
  }));

  return { rfNodes, rfEdges };
}

function toGraph(rfNodes: Node<WorkflowNodeData>[], rfEdges: Edge[]): GraphConfig {
  const graphNodes: GraphNode[] = rfNodes.map((n) => ({
    name: n.data.label || n.id,
    kind: n.data.kind,
    modelRef: n.data.modelRef || undefined,
    toolRef: n.data.toolRef || undefined,
    knowledgeRef: n.data.knowledgeRef || undefined,
    agentRef: n.data.agentRef || undefined,
    implementation: n.data.implementation || undefined,
    position: n.position,
  }));

  const graphEdges: GraphEdge[] = rfEdges.map((e) => ({
    from: e.source,
    to: e.target,
    when: e.label ? String(e.label) : undefined,
  }));

  return { nodes: graphNodes, edges: graphEdges };
}

// ─── Dagre Auto-Layout ────────────────────────────────────────────

function getLayoutedElements(nodes: Node<WorkflowNodeData>[], edges: Edge[], direction: "LR" | "TB" = "LR") {
  const dagreGraph = new dagre.graphlib.Graph();
  dagreGraph.setDefaultEdgeLabel(() => ({}));
  dagreGraph.setGraph({ rankdir: direction, nodesep: 60, ranksep: 120, marginx: 40, marginy: 40 });

  nodes.forEach((n) => {
    dagreGraph.setNode(n.id, { width: 180, height: 60 });
  });
  edges.forEach((e) => {
    dagreGraph.setEdge(e.source, e.target);
  });

  dagre.layout(dagreGraph);

  const layoutedNodes = nodes.map((n) => {
    const nodeWithPosition = dagreGraph.node(n.id);
    return {
      ...n,
      position: { x: nodeWithPosition.x - 90, y: nodeWithPosition.y - 30 },
    };
  });

  return { nodes: layoutedNodes, edges };
}

// ─── Validation ───────────────────────────────────────────────────

function validateGraph(nodes: Node<WorkflowNodeData>[], edges: Edge[]): string[] {
  const errors: string[] = [];
  const names = nodes.map((n) => n.data.label).filter(Boolean);
  const nameSet = new Set<string>();

  // Duplicate names
  for (const name of names) {
    if (nameSet.has(name)) {
      errors.push(`Duplicate node name: "${name}"`);
    }
    nameSet.add(name);
  }

  // Start/End check
  const hasStart = nodes.some((n) => n.data.kind === "start");
  const hasEnd = nodes.some((n) => n.data.kind === "end");
  if (!hasStart) errors.push("Workflow should have a Start node");
  if (!hasEnd) errors.push("Workflow should have an End node");

  // Edge references
  const nodeIds = new Set(nodes.map((n) => n.id));
  for (const e of edges) {
    if (!nodeIds.has(e.source)) errors.push(`Edge references missing source: "${e.source}"`);
    if (!nodeIds.has(e.target)) errors.push(`Edge references missing target: "${e.target}"`);
  }

  // Empty labels
  const unlabeled = nodes.filter((n) => !n.data.label && n.data.kind !== "start" && n.data.kind !== "end");
  if (unlabeled.length > 0) {
    errors.push(`${unlabeled.length} node(s) have no name`);
  }

  return errors;
}

// ─── Main Component ───────────────────────────────────────────────

export function WorkflowCanvas({ graph, onChange, onValidateRef }: WorkflowCanvasProps) {
  const { t } = useTranslation();
  const [nodes, setNodes, onNodesChange] = useNodesState<Node<WorkflowNodeData>>([]);
  const [edges, setEdges, onEdgesChange] = useEdgesState<Edge>([]);
  const [selectedNodeId, setSelectedNodeId] = useState<string | null>(null);
  const [selectedEdgeId, setSelectedEdgeId] = useState<string | null>(null);
  const [editingEdgeId, setEditingEdgeId] = useState<string | null>(null);
  const [edgeLabelDraft, setEdgeLabelDraft] = useState("");
  const [showHints, setShowHints] = useState(true);
  const [validationErrors, setValidationErrors] = useState<string[]>([]);
  const idCounter = useRef(0);
  const initialized = useRef(false);
  const reactFlowWrapper = useRef<HTMLDivElement>(null);
  const [reactFlowInstance, setReactFlowInstance] = useState<any>(null);

  // Initialize from parent graph once
  useEffect(() => {
    if (initialized.current) return;
    initialized.current = true;
    const { rfNodes, rfEdges } = toFlow(graph);
    setNodes(rfNodes);
    setEdges(rfEdges);
    idCounter.current = rfNodes.length;
  }, [graph, setNodes, setEdges]);

  // Expose validation to parent
  useEffect(() => {
    if (onValidateRef) {
      onValidateRef.current = () => {
        const errors = validateGraph(nodes, edges);
        setValidationErrors(errors);
        return errors;
      };
    }
  }, [nodes, edges, onValidateRef]);

  // Sync local state → parent
  const syncRef = useRef<number>(0);
  const syncToParent = useCallback(
    (nds: Node<WorkflowNodeData>[], eds: Edge[]) => {
      cancelAnimationFrame(syncRef.current);
      syncRef.current = requestAnimationFrame(() => {
        onChange(toGraph(nds, eds));
      });
    },
    [onChange]
  );

  // Wrap onNodesChange
  const handleNodesChange: OnNodesChange<Node<WorkflowNodeData>> = useCallback(
    (changes) => {
      onNodesChange(changes);
      const hasPositionChange = changes.some(
        (c) => c.type === "position" || c.type === "remove" || c.type === "dimensions"
      );
      if (hasPositionChange) {
        setTimeout(() => {
          setNodes((nds) => {
            setEdges((eds) => { syncToParent(nds, eds); return eds; });
            return nds;
          });
        }, 0);
      }
    },
    [onNodesChange, syncToParent, setNodes, setEdges]
  );

  // Wrap onEdgesChange
  const handleEdgesChange: OnEdgesChange<Edge> = useCallback(
    (changes) => {
      onEdgesChange(changes);
      const hasRemove = changes.some((c) => c.type === "remove");
      if (hasRemove) {
        setTimeout(() => {
          setNodes((nds) => {
            setEdges((eds) => { syncToParent(nds, eds); return eds; });
            return nds;
          });
        }, 0);
      }
    },
    [onEdgesChange, syncToParent, setNodes, setEdges]
  );

  const onConnect = useCallback(
    (params: Connection) => {
      setEdges((eds) => {
        const updated = addEdge(
          { ...params, style: { stroke: "#94a3b8", strokeWidth: 2 }, markerEnd: { type: MarkerType.ArrowClosed, color: "#94a3b8" } },
          eds
        );
        setNodes((nds) => { syncToParent(nds, updated); return nds; });
        return updated;
      });
    },
    [setEdges, setNodes, syncToParent]
  );

  const onNodeClick = useCallback((_: React.MouseEvent, node: Node) => {
    setSelectedNodeId(node.id);
    setSelectedEdgeId(null);
    setEditingEdgeId(null);
  }, []);

  const onEdgeClick = useCallback((_: React.MouseEvent, edge: Edge) => {
    setSelectedEdgeId(edge.id);
    setSelectedNodeId(null);
  }, []);

  const onEdgeDoubleClick = useCallback((_: React.MouseEvent, edge: Edge) => {
    setEditingEdgeId(edge.id);
    setEdgeLabelDraft(typeof edge.label === "string" ? edge.label : "");
  }, []);

  const onPaneClick = useCallback(() => {
    setSelectedNodeId(null);
    setSelectedEdgeId(null);
    setEditingEdgeId(null);
  }, []);

  // ─── Delete node/edge (direct function, not dispatchEvent hack) ───

  const deleteNode = useCallback((nodeId: string) => {
    setNodes((nds) => {
      const updated = nds.filter((n) => n.id !== nodeId);
      setEdges((eds) => {
        const filtered = eds.filter((ed) => ed.source !== nodeId && ed.target !== nodeId);
        syncToParent(updated, filtered);
        return filtered;
      });
      return updated;
    });
    setSelectedNodeId(null);
  }, [setNodes, setEdges, syncToParent]);

  const deleteEdge = useCallback((edgeId: string) => {
    setEdges((eds) => {
      const updated = eds.filter((e) => e.id !== edgeId);
      setNodes((nds) => { syncToParent(nds, updated); return nds; });
      return updated;
    });
    setSelectedEdgeId(null);
  }, [setEdges, setNodes, syncToParent]);

  // Keyboard deletion
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Delete" || e.key === "Backspace") {
        if ((e.target as HTMLElement).tagName === "INPUT" || (e.target as HTMLElement).tagName === "TEXTAREA") return;
        if (selectedNodeId) deleteNode(selectedNodeId);
        else if (selectedEdgeId) deleteEdge(selectedEdgeId);
      }
      if (e.key === "Escape") {
        setSelectedNodeId(null);
        setSelectedEdgeId(null);
        setEditingEdgeId(null);
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, [selectedNodeId, selectedEdgeId, deleteNode, deleteEdge]);

  // ─── Add node ────────────────────────────────────────────────────

  const handleAddNode = useCallback(
    (kind: string, position?: { x: number; y: number }) => {
      idCounter.current += 1;
      const id = `${kind}_${idCounter.current}`;
      const isTerminal = kind === "start" || kind === "end";
      const newNode: Node<WorkflowNodeData> = {
        id,
        type: "workflow",
        position: position ?? { x: 300 + Math.random() * 200, y: 150 + Math.random() * 200 },
        data: {
          label: isTerminal ? (kind === "start" ? t("studio.workflow.kind.start") : t("studio.workflow.kind.end")) : "",
          kind: kind as WorkflowNodeData["kind"],
        },
      };
      setNodes((nds) => {
        const updated = [...nds, newNode];
        setEdges((eds) => { syncToParent(updated, eds); return eds; });
        return updated;
      });
    },
    [setNodes, setEdges, syncToParent]
  );

  // Drag-and-drop
  const onDragOver = useCallback((event: React.DragEvent) => {
    event.preventDefault();
    event.dataTransfer.dropEffect = "move";
  }, []);

  const onDrop = useCallback(
    (event: React.DragEvent) => {
      event.preventDefault();
      const kind = event.dataTransfer.getData("application/korus-node-kind");
      if (!kind || !reactFlowInstance) return;
      const position = reactFlowInstance.screenToFlowPosition({ x: event.clientX, y: event.clientY });
      handleAddNode(kind, position);
    },
    [reactFlowInstance, handleAddNode]
  );

  // ─── Node update with ID rename ──────────────────────────────────

  const handleNodeUpdate = useCallback(
    (updates: Partial<WorkflowNodeData>) => {
      if (!selectedNodeId) return;
      setNodes((nds) => {
        const updated = nds.map((n) => {
          if (n.id !== selectedNodeId) return n;
          const newData = { ...n.data, ...updates };
          return { ...n, data: newData };
        });
        // If label changed, update node ID and edge references
        if (updates.label !== undefined) {
          const oldNode = nds.find((n) => n.id === selectedNodeId);
          if (oldNode && updates.label !== oldNode.data.label) {
            const newId = updates.label || selectedNodeId;
            if (newId !== selectedNodeId) {
              const withNewId = updated.map((n) =>
                n.id === selectedNodeId ? { ...n, id: newId } : n
              );
              setEdges((eds) => {
                const reindexed = eds.map((e) => ({
                  ...e,
                  source: e.source === selectedNodeId ? newId : e.source,
                  target: e.target === selectedNodeId ? newId : e.target,
                }));
                syncToParent(withNewId, reindexed);
                setSelectedNodeId(newId);
                return reindexed;
              });
              return withNewId;
            }
          }
        }
        setEdges((eds) => { syncToParent(updated, eds); return eds; });
        return updated;
      });
    },
    [selectedNodeId, setNodes, setEdges, syncToParent]
  );

  // ─── Edge label editing ──────────────────────────────────────────

  const handleEdgeLabelSave = useCallback(() => {
    if (!editingEdgeId) return;
    setEdges((eds) => {
      const updated = eds.map((e) =>
        e.id === editingEdgeId ? { ...e, label: edgeLabelDraft || undefined, animated: !!edgeLabelDraft } : e
      );
      setNodes((nds) => { syncToParent(nds, updated); return nds; });
      return updated;
    });
    setEditingEdgeId(null);
    setEdgeLabelDraft("");
  }, [editingEdgeId, edgeLabelDraft, setEdges, setNodes, syncToParent]);

  // ─── Auto-layout ─────────────────────────────────────────────────

  const handleAutoLayout = useCallback(() => {
    const { nodes: layouted, edges: layoutedEdges } = getLayoutedElements(nodes, edges, "LR");
    setNodes(layouted);
    setEdges(layoutedEdges);
    syncToParent(layouted, layoutedEdges);
    setTimeout(() => reactFlowInstance?.fitView({ padding: 0.2 }), 50);
  }, [nodes, edges, setNodes, setEdges, syncToParent, reactFlowInstance]);

  // ─── Clear canvas ────────────────────────────────────────────────

  const handleClear = useCallback(() => {
    setNodes([]);
    setEdges([]);
    syncToParent([], []);
    setSelectedNodeId(null);
    setSelectedEdgeId(null);
    idCounter.current = 0;
  }, [setNodes, setEdges, syncToParent]);

  const nodeTypes: NodeTypes = useMemo(() => ({ workflow: WorkflowNode }), []);
  const selectedNode = nodes.find((n) => n.id === selectedNodeId);
  const editingEdge = edges.find((e) => e.id === editingEdgeId);

  return (
    <div className="flex h-[calc(100vh-220px)] min-h-[500px] rounded-lg border border-zinc-200 bg-white overflow-hidden">
      {/* Left: Node Palette */}
      <NodePalette onAddNode={handleAddNode} />

      {/* Center: Canvas */}
      <div className="flex-1 relative" ref={reactFlowWrapper}>
        {/* Toolbar */}
        <div className="absolute top-3 right-3 z-20 flex items-center gap-1.5">
          <button
            type="button"
            onClick={handleAutoLayout}
            className="rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-xs font-medium text-zinc-600 shadow-sm hover:bg-zinc-50 transition-colors"
            title={t("studio.workflow.autoLayout")}
          >
            ⊞ {t("studio.workflow.autoLayout")}
          </button>
          <button
            type="button"
            onClick={() => reactFlowInstance?.fitView({ padding: 0.2 })}
            className="rounded-lg border border-zinc-200 bg-white px-2.5 py-1.5 text-xs font-medium text-zinc-600 shadow-sm hover:bg-zinc-50 transition-colors"
            title={t("studio.workflow.fitView")}
          >
            ⊡ {t("studio.workflow.fitView")}
          </button>
          <button
            type="button"
            onClick={handleClear}
            className="rounded-lg border border-red-200 bg-white px-2.5 py-1.5 text-xs font-medium text-red-600 shadow-sm hover:bg-red-50 transition-colors"
            title={t("studio.workflow.clearCanvas")}
          >
            ✕ {t("studio.workflow.clearCanvas")}
          </button>
        </div>

        <ReactFlow
          nodes={nodes}
          edges={edges}
          onNodesChange={handleNodesChange}
          onEdgesChange={handleEdgesChange}
          onConnect={onConnect}
          onNodeClick={onNodeClick}
          onEdgeClick={onEdgeClick}
          onEdgeDoubleClick={onEdgeDoubleClick}
          onPaneClick={onPaneClick}
          onInit={setReactFlowInstance}
          onDragOver={onDragOver}
          onDrop={onDrop}
          nodeTypes={nodeTypes}
          fitView
          proOptions={{ hideAttribution: true }}
          defaultEdgeOptions={{
            style: { stroke: "#94a3b8", strokeWidth: 2 },
            markerEnd: { type: MarkerType.ArrowClosed, color: "#94a3b8" },
          }}
          deleteKeyCode={null}
          selectionOnDrag
          multiSelectionKeyCode="Shift"
        >
          <Background variant={BackgroundVariant.Dots} gap={20} size={1} color="#e2e8f0" />
          <Controls className="!border-zinc-200 !shadow-sm" />
          <MiniMap
            nodeStrokeWidth={3}
            nodeColor={(n) => {
              switch (n.data?.kind) {
                case "start": return "#10b981";
                case "end": return "#f43f5e";
                case "model": return "#3b82f6";
                case "tool": return "#f59e0b";
                case "agent": return "#8b5cf6";
                case "knowledge": return "#f97316";
                default: return "#a1a1aa";
              }
            }}
            maskColor="rgba(255,255,255,0.7)"
            className="!border-zinc-200"
          />
        </ReactFlow>

        {/* Empty state */}
        {nodes.length === 0 && (
          <div className="absolute inset-0 flex items-center justify-center pointer-events-none">
            <div className="text-center">
              <div className="text-4xl mb-3 opacity-30">⊞</div>
              <p className="text-sm font-medium text-zinc-400">{t("studio.workflow.emptyTitle")}</p>
              <p className="text-xs text-zinc-400 mt-1">{t("studio.workflow.emptyHint")}</p>
            </div>
          </div>
        )}

        {/* Validation errors */}
        {validationErrors.length > 0 && (
          <div className="absolute top-3 left-3 z-20 max-w-xs rounded-lg border border-amber-200 bg-amber-50 p-3 shadow-sm">
            <p className="text-xs font-semibold text-amber-700 mb-1">⚠ {t("studio.workflow.validationErrors")}</p>
            <ul className="space-y-0.5">
              {validationErrors.map((err, i) => (
                <li key={i} className="text-[11px] text-amber-600">• {err}</li>
              ))}
            </ul>
            <button
              type="button"
              onClick={() => setValidationErrors([])}
              className="mt-2 text-[10px] text-amber-600 hover:text-amber-800 underline"
            >
              {t("common.dismiss")}
            </button>
          </div>
        )}

        {/* Bottom status bar */}
        <div className="absolute bottom-3 left-3 flex items-center gap-3">
          <div className="rounded-full bg-zinc-800/90 px-3 py-1 text-[11px] font-medium text-white shadow backdrop-blur-sm">
            {nodes.length} {t("studio.workflow.nodes")} · {edges.length} {t("studio.workflow.edges")}
          </div>
          {(selectedNodeId || selectedEdgeId) && (
            <div className="rounded-full bg-teal-600/90 px-3 py-1 text-[11px] font-medium text-white shadow backdrop-blur-sm">
              {selectedNodeId
                ? `${t("studio.workflow.selected")}: ${nodes.find((n) => n.id === selectedNodeId)?.data.label || selectedNodeId}`
                : `${t("studio.workflow.edgeSelected")} (⌫ ${t("studio.workflow.delete")})`}
            </div>
          )}
        </div>

        {/* Keyboard hints (dismissible) */}
        {showHints && (
          <div className="absolute bottom-3 right-3 rounded-lg bg-zinc-800/80 px-3 py-2 text-[10px] text-zinc-300 shadow backdrop-blur-sm group">
            <button
              type="button"
              onClick={() => setShowHints(false)}
              className="absolute -top-1.5 -right-1.5 flex h-4 w-4 items-center justify-center rounded-full bg-zinc-600 text-white text-[8px] opacity-0 group-hover:opacity-100 transition-opacity"
            >
              ✕
            </button>
            <div className="flex flex-col gap-0.5">
              <span><kbd className="rounded bg-zinc-700 px-1 py-0.5 text-zinc-200">⌫</kbd> {t("studio.workflow.hintDelete")}</span>
              <span><kbd className="rounded bg-zinc-700 px-1 py-0.5 text-zinc-200">Esc</kbd> {t("studio.workflow.hintDeselect")}</span>
              <span><kbd className="rounded bg-zinc-700 px-1 py-0.5 text-zinc-200">Shift</kbd>+{t("studio.workflow.hintMultiSelect")}</span>
              <span>{t("studio.workflow.hintDrag")}</span>
              <span>{t("studio.workflow.hintEdgeDoubleClick")}</span>
            </div>
          </div>
        )}

        {/* Edge label editor overlay */}
        {editingEdgeId && editingEdge && (
          <div className="absolute top-1/2 left-1/2 -translate-x-1/2 -translate-y-1/2 z-50">
            <div className="rounded-xl border border-zinc-200 bg-white p-4 shadow-xl">
              <h4 className="mb-3 text-sm font-semibold text-zinc-800">
                {t("studio.workflow.edgeCondition")}
              </h4>
              <input
                autoFocus
                type="text"
                value={edgeLabelDraft}
                onChange={(e) => setEdgeLabelDraft(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === "Enter") handleEdgeLabelSave();
                  if (e.key === "Escape") { setEditingEdgeId(null); setEdgeLabelDraft(""); }
                }}
                placeholder={t("studio.workflow.edgeConditionPlaceholder")}
                className="w-64 rounded-lg border border-zinc-300 px-3 py-2 text-sm focus:border-teal-500 focus:outline-none focus:ring-1 focus:ring-teal-500"
              />
              <p className="mt-2 text-[11px] text-zinc-500">
                {t("studio.workflow.edgeConditionHint")}
              </p>
              <div className="mt-3 flex justify-end gap-2">
                <button type="button" onClick={() => { setEditingEdgeId(null); setEdgeLabelDraft(""); }} className="rounded-lg px-3 py-1.5 text-xs font-medium text-zinc-600 hover:bg-zinc-100">{t("common.cancel")}</button>
                <button type="button" onClick={handleEdgeLabelSave} className="rounded-lg bg-teal-600 px-3 py-1.5 text-xs font-medium text-white hover:bg-teal-700">{t("common.save")}</button>
              </div>
            </div>
          </div>
        )}
      </div>

      {/* Right: Config Panel */}
      {selectedNode && (
        <NodeConfigPanel
          data={selectedNode.data as WorkflowNodeData}
          nodeId={selectedNode.id}
          onUpdate={handleNodeUpdate}
          onDelete={() => deleteNode(selectedNodeId!)}
          onClose={() => setSelectedNodeId(null)}
        />
      )}
    </div>
  );
}
