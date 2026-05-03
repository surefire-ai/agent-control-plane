import { useTranslation } from "react-i18next";
import type { AgentSpecData } from "@/types/api";
import { Card } from "@/components/shared/Card";

interface GraphPreviewProps {
  spec: AgentSpecData;
}

interface NodePosition {
  x: number;
  y: number;
}

// Layout configurations for each pattern
const patternGraphs: Record<string, { nodes: { name: string; label: string }[]; edges: { from: string; to: string; loop?: boolean }[] }> = {
  react: {
    nodes: [
      { name: "START", label: "Start" },
      { name: "reason", label: "Reason" },
      { name: "act", label: "Act" },
      { name: "observe", label: "Observe" },
      { name: "END", label: "End" },
    ],
    edges: [
      { from: "START", to: "reason" },
      { from: "reason", to: "act" },
      { from: "act", to: "observe" },
      { from: "observe", to: "reason", loop: true },
      { from: "reason", to: "END" },
    ],
  },
  router: {
    nodes: [
      { name: "START", label: "Start" },
      { name: "classify", label: "Classify" },
      { name: "route1", label: "Route 1" },
      { name: "route2", label: "Route 2" },
      { name: "END", label: "End" },
    ],
    edges: [
      { from: "START", to: "classify" },
      { from: "classify", to: "route1" },
      { from: "classify", to: "route2" },
      { from: "route1", to: "END" },
      { from: "route2", to: "END" },
    ],
  },
  reflection: {
    nodes: [
      { name: "START", label: "Start" },
      { name: "generate", label: "Generate" },
      { name: "critique", label: "Critique" },
      { name: "revise", label: "Revise" },
      { name: "END", label: "End" },
    ],
    edges: [
      { from: "START", to: "generate" },
      { from: "generate", to: "critique" },
      { from: "critique", to: "revise" },
      { from: "revise", to: "generate", loop: true },
      { from: "critique", to: "END" },
    ],
  },
  tool_calling: {
    nodes: [
      { name: "START", label: "Start" },
      { name: "model", label: "Model" },
      { name: "tools", label: "Tools" },
      { name: "END", label: "End" },
    ],
    edges: [
      { from: "START", to: "model" },
      { from: "model", to: "tools" },
      { from: "tools", to: "model", loop: true },
      { from: "model", to: "END" },
    ],
  },
  plan_execute: {
    nodes: [
      { name: "START", label: "Start" },
      { name: "plan", label: "Plan" },
      { name: "execute", label: "Execute" },
      { name: "END", label: "End" },
    ],
    edges: [
      { from: "START", to: "plan" },
      { from: "plan", to: "execute" },
      { from: "execute", to: "END" },
    ],
  },
};

function getNodePositions(nodes: { name: string }[], width: number): Record<string, NodePosition> {
  const positions: Record<string, NodePosition> = {};
  const count = nodes.length;

  if (count === 0) return positions;

  // Simple horizontal layout
  const padding = 60;
  const usableWidth = width - padding * 2;
  const spacing = usableWidth / (count - 1 || 1);

  nodes.forEach((node, i) => {
    // For loop patterns, arrange in a more circular way
    if (count >= 4) {
      const angle = (i / count) * Math.PI * 2 - Math.PI / 2;
      const cx = width / 2;
      const cy = 150;
      const rx = usableWidth / 2.5;
      const ry = 80;
      positions[node.name] = {
        x: cx + Math.cos(angle) * rx,
        y: cy + Math.sin(angle) * ry,
      };
    } else {
      positions[node.name] = {
        x: padding + i * spacing,
        y: 150,
      };
    }
  });

  return positions;
}

function getEdgePath(from: NodePosition, to: NodePosition, isLoop: boolean): string {
  if (isLoop) {
    const midX = (from.x + to.x) / 2;
    const midY = Math.min(from.y, to.y) - 40;
    return `M ${from.x} ${from.y} Q ${midX} ${midY} ${to.x} ${to.y}`;
  }
  return `M ${from.x} ${from.y} L ${to.x} ${to.y}`;
}

const NODE_RADIUS = 28;
const NODE_COLORS: Record<string, string> = {
  START: "#10b981",
  END: "#f43f5e",
  reason: "#6366f1",
  act: "#f59e0b",
  observe: "#06b6d4",
  classify: "#8b5cf6",
  route1: "#3b82f6",
  route2: "#ec4899",
  generate: "#6366f1",
  critique: "#f59e0b",
  revise: "#06b6d4",
  model: "#6366f1",
  tools: "#f59e0b",
  plan: "#6366f1",
  execute: "#10b981",
};

export function GraphPreview({ spec }: GraphPreviewProps) {
  const { t } = useTranslation();
  const width = 700;
  const height = 300;

  const patternType = spec.pattern?.type ?? "";
  const graphDef = patternGraphs[patternType];

  // For workflow, use the spec.graph directly
  if (patternType === "workflow" && spec.graph?.nodes && spec.graph.nodes.length > 0) {
    return <WorkflowGraphPreview graph={spec.graph} width={width} height={height} />;
  }

  if (!graphDef) {
    return (
      <div>
        <h3 className="mb-4 text-lg font-semibold text-zinc-950">{t("studio.preview.title")}</h3>
        <Card className="p-8">
          <p className="text-center text-sm text-zinc-400">{t("studio.preview.noGraph")}</p>
        </Card>
      </div>
    );
  }

  const positions = getNodePositions(graphDef.nodes, width);

  return (
    <div>
      <h3 className="mb-4 text-lg font-semibold text-zinc-950">{t("studio.preview.title")}</h3>
      <Card className="overflow-hidden">
        <svg width={width} height={height} className="w-full" viewBox={`0 0 ${width} ${height}`}>
          <defs>
            <marker
              id="arrowhead"
              markerWidth="8"
              markerHeight="6"
              refX="8"
              refY="3"
              orient="auto"
            >
              <polygon points="0 0, 8 3, 0 6" fill="#94a3b8" />
            </marker>
          </defs>

          {/* Edges */}
          {graphDef.edges.map((edge, i) => {
            const from = positions[edge.from];
            const to = positions[edge.to];
            if (!from || !to) return null;

            // Offset edge endpoints to node boundaries
            const dx = to.x - from.x;
            const dy = to.y - from.y;
            const dist = Math.sqrt(dx * dx + dy * dy) || 1;
            const startX = from.x + (dx / dist) * NODE_RADIUS;
            const startY = from.y + (dy / dist) * NODE_RADIUS;
            const endX = to.x - (dx / dist) * NODE_RADIUS;
            const endY = to.y - (dy / dist) * NODE_RADIUS;

            return (
              <path
                key={`edge-${i}`}
                d={getEdgePath(
                  { x: startX, y: startY },
                  { x: endX, y: endY },
                  edge.loop ?? false
                )}
                fill="none"
                stroke={edge.loop ? "#94a3b8" : "#cbd5e1"}
                strokeWidth={2}
                strokeDasharray={edge.loop ? "6 4" : "none"}
                markerEnd="url(#arrowhead)"
              />
            );
          })}

          {/* Nodes */}
          {graphDef.nodes.map((node) => {
            const pos = positions[node.name];
            if (!pos) return null;
            const color = NODE_COLORS[node.name] ?? "#6366f1";
            const isTerminal = node.name === "START" || node.name === "END";

            return (
              <g key={node.name}>
                <circle
                  cx={pos.x}
                  cy={pos.y}
                  r={NODE_RADIUS}
                  fill={isTerminal ? color : "white"}
                  stroke={color}
                  strokeWidth={2}
                />
                <text
                  x={pos.x}
                  y={pos.y}
                  textAnchor="middle"
                  dominantBaseline="central"
                  className="text-[11px] font-medium"
                  fill={isTerminal ? "white" : color}
                >
                  {node.label}
                </text>
              </g>
            );
          })}
        </svg>
      </Card>
    </div>
  );
}

function WorkflowGraphPreview({
  graph,
  width,
  height,
}: {
  graph: { nodes?: { name: string; kind: string }[]; edges?: { from: string; to: string }[] };
  width: number;
  height: number;
}) {
  const { t } = useTranslation();
  const nodes = graph.nodes ?? [];
  const edges = graph.edges ?? [];

  const positions = getNodePositions(nodes, width);

  return (
    <div>
      <h3 className="mb-4 text-lg font-semibold text-zinc-950">{t("studio.preview.title")}</h3>
      <Card className="overflow-hidden">
        <svg width={width} height={height} className="w-full" viewBox={`0 0 ${width} ${height}`}>
          <defs>
            <marker
              id="arrowhead-wf"
              markerWidth="8"
              markerHeight="6"
              refX="8"
              refY="3"
              orient="auto"
            >
              <polygon points="0 0, 8 3, 0 6" fill="#94a3b8" />
            </marker>
          </defs>

          {edges.map((edge, i) => {
            const from = positions[edge.from];
            const to = positions[edge.to];
            if (!from || !to) return null;
            const dx = to.x - from.x;
            const dy = to.y - from.y;
            const dist = Math.sqrt(dx * dx + dy * dy) || 1;
            return (
              <line
                key={`edge-${i}`}
                x1={from.x + (dx / dist) * NODE_RADIUS}
                y1={from.y + (dy / dist) * NODE_RADIUS}
                x2={to.x - (dx / dist) * NODE_RADIUS}
                y2={to.y - (dy / dist) * NODE_RADIUS}
                stroke="#cbd5e1"
                strokeWidth={2}
                markerEnd="url(#arrowhead-wf)"
              />
            );
          })}

          {nodes.map((node) => {
            const pos = positions[node.name];
            if (!pos) return null;
            const color = NODE_COLORS[node.name] ?? "#6366f1";

            return (
              <g key={node.name}>
                <circle cx={pos.x} cy={pos.y} r={NODE_RADIUS} fill="white" stroke={color} strokeWidth={2} />
                <text
                  x={pos.x}
                  y={pos.y}
                  textAnchor="middle"
                  dominantBaseline="central"
                  className="text-[11px] font-medium"
                  fill={color}
                >
                  {node.name}
                </text>
                <text
                  x={pos.x}
                  y={pos.y + NODE_RADIUS + 14}
                  textAnchor="middle"
                  className="text-[9px]"
                  fill="#94a3b8"
                >
                  {node.kind}
                </text>
              </g>
            );
          })}
        </svg>
      </Card>
    </div>
  );
}
