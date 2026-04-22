# Agent Patterns, SubAgents, and A2A TODO

This TODO tracks capabilities that should shape the Agent API before the real
Eino runtime becomes too concrete.

## Why This Matters

Users should still declare Agents as Kubernetes CRDs. The goal is narrower:
users should not have to hand-write a full `spec.graph` for common agent
designs. `AgentSpec` should support recognized patterns such as ReAct while
still letting users choose models, tools, knowledge, MCP servers, policies,
interfaces, memory, and observability as normal CRD fields.

## Agent Pattern Presets

Status: in progress.

Add a first-class pattern field so users can declare the orchestration pattern
without writing the full graph by hand. Other Agent inputs remain explicit and
selectable.

Proposed shape:

```yaml
spec:
  pattern:
    type: react
    version: v1
    modelRef: planner
    toolRefs:
      - rectify-ticket-api
    maxIterations: 6
    stopWhen: final_answer
  models:
    planner:
      provider: openai
      model: gpt-4.1
      credentialRef:
        name: openai-credentials
        key: apiKey
  knowledgeRefs:
    - name: regulations
      ref: ehs-regulations
  toolRefs:
    - rectify-ticket-api
  policyRef: ehs-default-safety-policy
```

Initial presets to support:

| Pattern | Purpose |
| --- | --- |
| `react` | Reasoning plus tool use loop. |
| `plan_execute` | Planner creates steps, executor completes steps. |
| `router` | Classify task and route to specialized branch or SubAgent. |
| `reflection` | Generate, critique, and revise. |
| `tool_calling` | Model-driven structured tool calls without full graph authoring. |
| `rag` | Retrieval-augmented generation with knowledge refs. |
| `workflow` | Deterministic graph/workflow compiled from explicit nodes. |

## Skill Support

Status: not started.

Add first-class reusable skills so common capability bundles can be attached to
an Agent without forcing users to inline every tool, knowledge binding, prompt,
or graph fragment manually.

Proposed shape:

```yaml
spec:
  skillRefs:
    - name: hazard_triage
      ref: ehs-hazard-triage-skill
    - name: ticketing
      ref: ehs-ticketing-skill
```

API TODO:

- Added `Skill` CRD and `AgentSpec.skillRefs`.
- Added compiler support for skill-provided tools, knowledge, prompts, and
  graph fragments.
- Keep tightening the precedence and ambiguity rules as patterns arrive.

Compiler TODO:

- Resolve skill references before pattern expansion.
- Merge skill-provided tool, knowledge, prompt, and graph metadata into the
  compiled artifact with deterministic precedence rules.
- Preserve the selected skill revisions in `Agent.status.compiledArtifact`.

Runtime TODO:

- Surface resolved skills in worker runtime metadata and traces.
- Allow pattern presets such as `react`, `router`, and `rag` to consume skill
  bundles as first-class inputs.
- Keep skill expansion compatible with future SubAgent and A2A boundaries.

Compiler TODO:

- Expand `spec.pattern` into `runner.graph` when `spec.graph` is empty.
- Preserve user-selected models, tools, knowledge, MCP servers, policies, and
  interfaces as explicit inputs to the pattern expansion.
- Preserve model credential references during pattern expansion without
  resolving or copying secret values into the compiled artifact payload.
- Reject ambiguous configurations where both `pattern` and incompatible
  explicit graph nodes are present.
- Preserve the original pattern declaration in the compiled artifact.
- Include pattern expansion metadata in `Agent.status.compiledArtifact`.

Runtime TODO:

- Map `react` to an Eino ADK/Graph loop.
- Enforce iteration limits and tool allowlists.
- Report pattern metadata in worker output and trace references.

## SubAgent Support

Status: not started.

Add SubAgent references as a first-class part of `AgentSpec`.

Proposed shape:

```yaml
spec:
  subAgentRefs:
    - name: risk_scorer
      ref: ehs-risk-scoring-agent
    - name: ticket_creator
      ref: ehs-ticket-agent

  graph:
    nodes:
      - name: score_risk
        kind: agent
        agentRef: risk_scorer
```

API TODO:

- Add `AgentBindingSpec` with `name`, `ref`, optional namespace, and optional
  policy propagation settings.
- Add `AgentSpec.subAgentRefs`.
- Add `AgentGraphNode.agentRef`.

Compiler TODO:

- Validate SubAgent references.
- Capture SubAgent endpoint and revision in the compiled artifact.
- Detect cycles where possible.
- Preserve policy and trace propagation requirements.

Runtime TODO:

- Invoke SubAgents through the internal gateway first.
- Carry parent run identity and trace context.
- Preserve SubAgent result summaries under the parent `AgentRun.status.output`.

## A2A Protocol Support

Status: future Agent Mesh work; design should not block it.

Full A2A support belongs to the distributed Agent Fabric roadmap, but Phase 2
should avoid schema choices that make it hard later.

Needed surfaces:

- Agent Card endpoint for public capability discovery.
- A2A task/message/artifact mapping to `AgentRun`.
- A2A auth metadata mapping to policy and gateway auth.
- Streaming/SSE mapping to run events or trace storage.
- Trace correlation across A2A task boundaries.

Possible API shape:

```yaml
spec:
  interfaces:
    a2a:
      enabled: true
      skills:
        - id: identify_hazard
          name: Hazard Identification
          description: Identify EHS hazards from inspection input.
      capabilities:
        streaming: false
        pushNotifications: false
```

Gateway TODO:

- Serve an Agent Card for published Agents that opt into A2A.
- Accept A2A task creation and map it to `AgentRun`.
- Translate A2A task state to `AgentRun.status.phase`.
- Translate A2A artifacts to `AgentRun.status.output` and durable artifacts.

## Roadmap Placement

- Phase 2: document API direction, support pattern expansion for the first real
  Eino runner, and keep compiled artifacts future-compatible.
- Phase 3: expose pattern selection and SubAgent composition in product
  surfaces.
- Phase 4: implement full Agent Mesh and A2A interoperability.
