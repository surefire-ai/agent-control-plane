# Manager To Operator Sync

Status: draft  
Last updated: 2026-04-29

## Purpose

The manager stores enterprise product state. The operator reconciles
Kubernetes runtime resources.

This document defines how those two components should interact without making
either one own the other's responsibilities.

## Core Rule

The manager may create, update, or mirror Kubernetes resources, but the
operator must not depend on the manager to reconcile existing CRDs.

Kubernetes-native mode remains valid.

## Sync Direction

### Manager To Kubernetes

The manager writes runtime resources when users perform product actions in the
Web Console.

Examples:

- create or update a lightweight Kubernetes `Workspace`
- create or update an `Agent`
- create an `AgentEvaluation`
- create an `AgentRun`
- create or update provider policy references
- create Kubernetes Secret references or external-secret bindings, never secret
  values in CRDs

### Kubernetes To Manager

The manager observes runtime state for product UX and durable audit history.

Examples:

- `Agent.status.compiledRevision`
- `Agent.status.endpoint`
- `AgentRun.status.phase`
- `AgentRun.status.workspaceRef`
- `AgentRun.status.traceRef`
- `AgentEvaluation.status.summary`
- `AgentEvaluation.status.gates`
- Kubernetes UID and resource version

The manager may cache these fields, but the operator remains the source of
truth for live runtime status.

## Resource Ownership

### Manager-Owned Product Records

The manager database owns:

- product tenant
- product workspace
- user and team membership
- provider account metadata
- visual draft graph
- release workflow state
- durable audit history
- usage aggregation

### Operator-Owned Runtime Records

Kubernetes owns:

- `Agent`
- `AgentRun`
- `AgentEvaluation`
- `AgentPolicy`
- `ToolProvider`
- `KnowledgeBase`
- `PromptTemplate`
- `Dataset`
- `MCPServer`
- `Skill`
- runtime-scope `Tenant` and `Workspace` bridge resources

### Shared Identity

Every manager-created Kubernetes resource should carry product identity labels
or annotations.

Recommended labels:

```yaml
korus.surefire.ai/tenant-id: tenant_...
korus.surefire.ai/workspace-id: ws_...
korus.surefire.ai/managed-by: manager
```

Recommended annotations:

```yaml
korus.surefire.ai/manager-resource-id: ...
korus.surefire.ai/manager-sync-generation: "..."
```

The exact label domain can change before implementation, but the contract
should stay explicit.

## Sync Triggers

### User-Initiated Writes

When a user edits an agent draft and publishes it:

1. The manager saves the draft.
2. The manager validates product-level permissions.
3. The manager renders or updates Kubernetes resources.
4. The operator compiles and reconciles those resources.
5. The manager observes status and updates product views.

### Runtime Status Changes

When a run or evaluation changes status:

1. The operator writes Kubernetes status.
2. The manager watches or polls runtime resources.
3. The manager records durable audit or history entries.
4. The console updates product views.

### Drift Detection

When Kubernetes resources are edited outside the manager:

1. The manager detects resource version or spec drift.
2. The manager marks the product record as `drifted`.
3. The console exposes the drift to users.
4. The user can accept, overwrite, or ignore drift depending on policy.

The manager should not silently overwrite manual Kubernetes changes unless a
clear product policy allows it.

## Conflict Policy

Recommended initial policy:

- manager-owned resources use last-manager-write wins only for fields the
  manager explicitly owns
- operator status always wins for runtime status
- external GitOps changes should be surfaced as drift
- Secret values are never read back into the manager from Kubernetes
- failed sync attempts are stored with error details but without leaking secret
  material

## Workspace Sync

In managed enterprise mode:

1. The manager creates the product workspace in its database.
2. The manager maps it to a Kubernetes namespace and optional `Workspace` CRD.
3. The manager creates or updates the lightweight `Workspace` CRD only with
   runtime-scope fields.
4. The operator validates `Agent`, `AgentEvaluation`, and `AgentRun`
   workspace references against Kubernetes-visible runtime scope.

The Kubernetes `Workspace` CRD should not contain:

- members
- teams
- UI draft graphs
- durable audit logs
- billing state
- full approval workflows

## Agent Publish Sync

When a visual draft is published:

1. The manager stores an immutable source snapshot reference.
2. The manager renders an `Agent` spec and related runtime resources.
3. The manager writes resources to Kubernetes.
4. The operator compiles the `Agent`.
5. The manager records the resulting `compiledRevision`.
6. The manager associates the compiled revision with an `AgentRevision`
   product record.

## Evaluation Sync

When an evaluation is started from the console:

1. The manager creates an `EvaluationRun` product record.
2. The manager creates or updates the runtime `AgentEvaluation`.
3. The operator creates managed `AgentRun` resources.
4. The operator writes status, gates, and deltas.
5. The manager stores durable report metadata and product history.

## Audit Sync

The manager should record durable audit events for:

- product writes from the console
- Kubernetes resource creation or updates initiated by the manager
- publish requests
- evaluation starts and finishes
- release approvals
- run invocations initiated through manager APIs
- drift detection and resolution

The operator can keep lightweight status and trace references, but durable
enterprise audit history belongs in manager storage.

## Initial Implementation Recommendation

The repository now includes a minimal manager HTTP scaffold in `cmd/manager`
and `internal/manager`. It exposes `/healthz`, `/readyz`, and `/api/v1/info`
before any database-backed product APIs are added. It also includes optional
database configuration and embedded migration files for the first manager-owned
product tables. PostgreSQL support is wired through pgx, and built-in
migrations run at startup only when the operator explicitly enables
`--migrate-on-start` or `MANAGER_MIGRATE_ON_START=true`.

Start with a one-way manager-to-Kubernetes write path and a simple
Kubernetes-to-manager status observer:

1. define product workspace and agent project tables
2. create manager API endpoints for workspace and agent draft CRUD
3. render `Agent` and lightweight `Workspace` resources
4. write them to Kubernetes with manager identity labels
5. observe `Agent.status` and `AgentRun.status`
6. store audit events for manager-initiated writes

Add drift resolution, release channels, and rich evaluation history after the
first build/evaluate/release loop is usable.
