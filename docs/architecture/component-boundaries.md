# Component Boundaries

Status: draft  
Last updated: 2026-04-29

## Purpose

Korus is moving toward four explicit components:

- `operator`
- `manager`
- `worker`
- `runner`

The goal is to keep Kubernetes-native execution and enterprise product
management from collapsing into one process or one storage model.

## Component Model

```text
Web Console
  |
Manager API + Database
  |
Operator / Kubernetes API
  |
Worker
  |
Runner
```

## Operator

The operator is the Kubernetes-native control plane.

It owns:

- CRD reconciliation
- compiled artifact status
- `AgentRun` lifecycle
- Kubernetes Job dispatch
- runtime status updates
- Kubernetes Secret references
- lightweight runtime scope validation

It should not own:

- user accounts
- team membership
- workspace collaboration state
- UI drafts
- billing state
- full audit log storage
- product-facing RBAC matrices

The operator must keep working without the manager so the project remains
useful as a Kubernetes-native open source control plane.

## Manager

The manager is an optional product backend used by the Web Console.

It owns product state in a database.

Expected manager-owned data includes:

- organizations and tenants
- workspaces as product entities
- users, teams, memberships, and roles
- provider accounts and credential binding metadata
- evaluation projects and result history
- release channels and approval workflow state
- UI draft graphs and collaboration state
- durable audit logs
- quota, usage, and billing metadata

The manager should create, update, or mirror Kubernetes resources through the
operator-facing CRD API, but it should not make the operator depend on the
manager for basic reconciliation.

## Worker

The worker is the execution-side process.

It consumes:

- compiled agent artifacts
- run input
- runtime metadata
- Secret-backed credentials injected by Kubernetes

It produces:

- structured run output
- trace references
- failure details
- runner-specific execution metadata

The worker should not implement product management concerns such as workspace
membership, UI state, or billing.

## Runner

The runner is the pluggable agent execution engine boundary used by the worker.

Examples:

- Eino runner
- future LangGraph compatibility adapter
- remote runner
- custom enterprise runner

The runner should focus on agent execution semantics: model calls, tool calls,
retrieval, graph steps, and structured output.

## Workspace Ownership

Workspace product data belongs in the manager database.

The Kubernetes `Workspace` CRD, if present, is only a lightweight runtime scope
bridge for Kubernetes-native use and manager-to-operator synchronization. It
must not become the canonical product database for enterprise workspace
semantics.

Use the manager database for:

- workspace display and lifecycle metadata
- membership
- role bindings
- collaboration state
- UI drafts
- release workflows
- durable audit history
- billing and quota state

Use Kubernetes resources for:

- runtime scope references
- namespace mapping hints
- policy references needed before dispatch
- provider restrictions needed before compilation
- Secret references, never Secret values
- run status and runtime audit identifiers

## Operating Modes

### Kubernetes-Native Mode

Users can apply CRDs directly without running the manager.

In this mode:

- `Workspace` may exist as a lightweight CRD for runtime scoping.
- `Agent`, `AgentEvaluation`, and `AgentRun` can reference a `Workspace`.
- The operator validates only the Kubernetes-side scope it can observe.

This mode is important for local development, open source adoption, and
platform teams that prefer GitOps.

### Managed Enterprise Mode

The Web Console talks to the manager, and the manager stores product state in a
database.

In this mode:

- the manager is the source of truth for product workspaces
- the manager maps product IDs to Kubernetes namespaces and CRD references
- the manager creates or syncs CRDs for runtime execution
- the operator treats workspace IDs and references as runtime metadata and
  validation inputs, not as full product records

## Design Rules

1. Do not add membership, UI drafts, billing, or collaboration fields to the
   `Workspace` CRD.
2. Keep CRDs deterministic and suitable for GitOps.
3. Keep manager database state optional for core operator execution.
4. Use Kubernetes Secrets or external secret managers for credentials.
5. Let `AgentRun` carry runtime identity and audit handles, but keep durable
   product audit history in manager-owned storage.
6. Keep runner implementations behind the worker boundary.

## Migration Guidance

The current `Tenant` and `Workspace` CRDs should be treated as early runtime
scope resources.

Near-term work should:

1. update docs to stop describing `Workspace` as the canonical product
   workspace
2. keep existing controller behavior for compatibility
3. avoid adding more product-only fields to `Workspace`
4. introduce manager design docs before adding database-backed implementation
5. eventually decide whether the Kubernetes-side resource should remain named
   `Workspace` or become a clearer runtime-scope resource
