# Console Information Architecture

Status: draft  
Last updated: 2026-04-23

## Purpose

This document defines the intended information architecture for the future web
console.

The implementation root for the future console is [web/README.md](../../web/README.md).

The console is not a thin Kubernetes dashboard. It is the primary product
surface for:

- visual agent orchestration
- evaluation and revision comparison
- publishing and release management
- multi-tenant collaboration and governance

The console should let users work at the product level while still mapping
cleanly onto deterministic control-plane resources.

## Product Stance

The product should be understood as an enterprise agent platform with a
Kubernetes-native control plane underneath.

- Kubernetes remains the source of truth for control-plane objects.
- The console is the default user-facing workspace for building and operating
  agents.
- Users should not need to hand-author graph YAML for common orchestration
  workflows.
- Visual editing must still compile into explicit CRD fields and deterministic
  runner artifacts.

## Primary Users

### Platform Administrator

Owns tenant boundaries, workspace provisioning, provider setup, runtime policy,
RBAC, quotas, and global governance settings.

### Agent Builder

Designs agents, assembles prompts, tools, knowledge, skills, patterns, and
graph flows, then publishes revisions for evaluation and release.

### Evaluator

Owns datasets, thresholds, score interpretation, regression tracking, and
release gates.

### Application Operator

Inspects runs, traces, artifacts, policy outcomes, and release status in daily
operations.

### Reviewer or Approver

Participates in approval, exception, release, and governance workflows.

## Core Navigation Model

The console should use this top-level navigation:

1. Tenant switcher
2. Workspace navigation
3. Agents
4. Evaluations
5. Runs
6. Providers
7. Assets
8. Governance
9. Administration

### Tenant switcher

Top-level product boundary. Everything in the console should feel scoped to a
tenant before it is scoped to a workspace.

### Workspace navigation

Primary working context for application teams. A workspace should gather the
agents, runs, datasets, provider bindings, policies, and members needed for a
single team, product, or business domain.

## Primary Product Areas

### 1. Agents

This is the home for visual agent orchestration and release history.

Expected subviews:

- Agent list
- Agent overview
- Visual orchestration studio
- Prompt and interface configuration
- Tool and knowledge binding
- Skill and pattern configuration
- Revision history
- Publish and release status

The orchestration studio should be the default editing surface for:

- selecting a pattern such as `react`
- wiring tools and knowledge into the orchestration
- configuring graph nodes and edges
- previewing input/output contracts
- understanding what will compile into runner artifacts

### 2. Evaluations

This is a first-class product area, not a detail page attached to an agent.

Expected subviews:

- Dataset library
- Evaluation definitions
- Evaluation runs
- Current versus baseline comparison
- Metric breakdown
- Release gate status
- Regression history

Evaluation should answer these product questions quickly:

- Is the new revision better than the current baseline
- Which metrics improved or regressed
- Did any blocking threshold fail
- Is this revision releasable
- Which provider or model performed best on this dataset

### 3. Runs

This area supports debugging and live operations.

Expected subviews:

- Run list
- Run details
- Structured output
- Artifacts and trace references
- Policy outcomes
- Runtime logs and failure reasons

### 4. Providers

This area manages the provider capability matrix as a product concern.

Expected subviews:

- Provider catalog
- Model catalog
- Credential references
- Workspace-scoped provider bindings
- Capability differences
- Cost, latency, and support metadata

Provider management must treat Chinese domestic providers as first-class
options, not as side notes to OpenAI-compatible support.

### 5. Assets

This area collects reusable building blocks.

Expected subviews:

- Prompt templates
- Knowledge bases
- Tool providers
- Skills
- Datasets
- MCP servers
- Policies

### 6. Governance

This area exposes the enterprise operating model.

Expected subviews:

- Approval queues
- Policy violations
- Release gates
- Audit history
- Exceptions and waivers

### 7. Administration

This area is for platform admins.

Expected subviews:

- Tenants
- Workspaces
- Members and roles
- Quotas
- Runtime backends
- Global provider settings
- Storage and integration settings

## Core User Flows

### Flow A: Build an agent visually

1. Enter a workspace
2. Create or open an agent
3. Use the orchestration studio to select a pattern or graph skeleton
4. Bind prompts, models, tools, knowledge, and skills
5. Preview interfaces and compiled semantics
6. Save a draft revision

### Flow B: Evaluate a candidate revision

1. Select a candidate agent revision
2. Choose or create a dataset
3. Configure thresholds and evaluators
4. Pick a baseline revision or baseline agent
5. Run evaluation
6. Review metrics, regressions, and gate outcomes

### Flow C: Publish and release

1. Review evaluation readiness
2. Review policy and approval requirements
3. Publish the revision
4. Promote to a release channel or environment
5. Observe run health and regressions after rollout

### Flow D: Debug a failed run

1. Open a run from the workspace or agent page
2. Inspect structured output, trace references, and artifacts
3. Inspect model, tool, and retrieval bindings
4. Compare against the revision and evaluation history
5. Patch the agent and create a new revision

## Console Object Mapping

The console should present product-language objects, but every major action
must map to typed control-plane resources.

| Console concept | Primary backing resource | Notes |
| --- | --- | --- |
| Agent | `Agent` | Main orchestration and release unit |
| Agent draft or revision | `Agent` + compiled revision status | UI may show richer revision UX than raw CRD fields |
| Run | `AgentRun` | Immutable execution record |
| Evaluation definition | `AgentEvaluation` | Includes dataset, evaluators, thresholds, gate, reporting |
| Evaluation sample set | `Dataset` | Reusable benchmark and regression surface |
| Prompt asset | `PromptTemplate` | Reusable prompt building block |
| Tool binding | `ToolProvider` | Runtime tool contract and auth source |
| Knowledge source | `KnowledgeBase` | Retrieval and source configuration |
| Skill package | `Skill` | Reusable capability bundle |
| External runtime integration | `MCPServer` | Runtime-facing MCP integration |
| Policy | `AgentPolicy` | Governance and guardrail contract |

## UI-to-Control-Plane Rules

The console should follow these rules:

1. Visual edits must compile into typed CRD fields, not hidden console-only
   state.
2. Graph editing should prefer generating `Pattern` and `Skill` aware agent
   specs rather than inventing a second orchestration model.
3. Console concepts may aggregate multiple CRDs, but should not create a
   second source of truth for runtime behavior.
4. Evaluation, release, and governance decisions should remain inspectable from
   control-plane status and artifacts.
5. The UI can simplify authoring, but not at the cost of deterministic
   compiled artifacts.

## Phase 3 Scope Recommendations

The first usable console should prioritize:

1. Tenant and workspace shell
2. Agent list and detail pages
3. Visual orchestration studio for common agent assembly
4. Evaluation list, detail, and baseline comparison
5. Release readiness and publish workflow
6. Run inspection and debugging

The first console should not try to deliver every platform area on day one.
Marketplace, full governance workflows, and advanced administration can follow
after the build/evaluate/release loop is solid.

## Open Design Questions

- How should workspace boundaries map onto Kubernetes namespaces
- Which release concepts become first-class API fields versus console-only UX
- How much freeform graph editing should the first studio allow
- How should provider credentials and workspace bindings appear without leaking
  secret details
- Which evaluation visualizations are needed for daily decision-making versus
  audit and reporting
