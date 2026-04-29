# Tenancy And Workspace Model

Status: draft  
Last updated: 2026-04-29

## Purpose

This document explains the tenancy and workspace model after the project split
between operator, manager, worker, and runner responsibilities.

The manager data model is documented in
`../architecture/manager-data-model.md`, and the manager-to-operator sync
contract is documented in `../architecture/manager-operator-sync.md`.

The goal is to avoid making Kubernetes CRDs the canonical database for
enterprise product concepts such as workspaces, membership, collaboration,
release workflows, billing, and durable audits.

## Current Direction

The repository currently defines two Kubernetes resources:

- `Tenant`
- `Workspace`

These resources should now be treated as lightweight runtime-scope bridge
resources. They are useful for Kubernetes-native mode and for synchronizing
manager-owned product state into the operator, but they should not become the
canonical product data model.

The canonical product tenant and workspace records belong in the optional
manager database.

## Why This Layer Exists

The project is no longer only a Kubernetes operator for technical users. It is
becoming an enterprise platform where the console and manager must support:

- tenant switching
- workspace-scoped agent development
- evaluation and release workflows
- provider and policy scoping
- collaboration and governance

Without explicit tenant and workspace concepts, those workflows would be
scattered across ad hoc labels and namespace conventions. But making CRDs the
primary store for all of that product state would also be the wrong center of
gravity: the manager database is a better home for user-facing and
collaboration-heavy records.

## Resource Roles

### Tenant

In the manager, `Tenant` is the top-level product boundary.

It represents an enterprise customer, business unit, or isolated
organizational domain.

In Kubernetes, the current `Tenant` CRD is a lightweight runtime-scope bridge.
It currently carries:

- display metadata
- freeform profile metadata
- freeform governance metadata
- freeform provider metadata

Manager-owned tenant data can later expand into:

- quota policy
- compliance profile
- provider policy
- billing or cost allocation metadata
- domain and identity mapping

### Workspace

In the manager, `Workspace` is the primary working context inside a tenant.

It gathers agents, evaluations, datasets, provider bindings, and team
collaboration state for a single team, product, or business domain.

In Kubernetes, the current `Workspace` CRD is a runtime-scope bridge. It
currently carries:

- `tenantRef`
- display metadata
- optional namespace mapping
- optional default policy reference
- optional provider policy
- freeform provider metadata
- freeform governance metadata

Manager-owned workspace data can later expand into:

- member and role bindings
- release channels
- default providers
- workspace-scoped policies
- dataset and evaluation ownership rules
- audit and approval routing

These product fields should not be added wholesale to the `Workspace` CRD.

## Initial Modeling Assumptions

### 1. Tenant and Workspace product state is manager-owned

The manager database is the canonical source for enterprise tenant and
workspace records.

### 2. Kubernetes Workspace is a runtime-scope bridge

The Kubernetes `Workspace` CRD can help validate and carry runtime scope, but
it must stay small enough to remain GitOps-friendly and optional.

### 3. Workspace is the main user-facing scope

Most day-to-day product actions should happen inside a workspace, not directly
at the tenant layer.

### 4. Namespace mapping is explicit but not final

`Workspace.spec.namespace` exists as an early bridge to Kubernetes runtime
scoping, but it should not be mistaken for the finished isolation model.

Later iterations may support:

- one workspace to one namespace
- one workspace to multiple namespaces
- virtual workspaces above several namespaces

### 5. Freeform fields are temporary expansion joints

`provider` and `governance` are currently freeform on purpose. They let the
project begin modeling enterprise semantics without prematurely freezing the
shape of every policy and provider contract.

These fields should be tightened gradually as real workflows stabilize.

### 6. Manager must remain optional for core execution

The operator should be able to reconcile CRDs and execute runs without calling
the manager. Managed enterprise mode can add database-backed product workflows
on top, but Kubernetes-native mode must remain viable.

## Current Workspace Enforcement

The first workspace-aware enforcement path is attached to `Agent` compilation
and `AgentRun` status.

- `Agent.spec.workspaceRef` must point at a Ready `Workspace` when set.
- `AgentEvaluation.spec.workspaceRef` must point at a Ready `Workspace` when
  set.
- `Workspace.spec.policyRef` is used as the effective `AgentPolicy` when an
  agent does not set `Agent.spec.policyRef`.
- `Workspace.spec.providerPolicy.allowedProviders` restricts which model
  providers an agent can use before compilation starts.
- `Workspace.spec.providerPolicy.defaultProvider` can fill missing model
  provider fields.
- `Workspace.spec.providerPolicy.bindings` can provide provider-specific
  `baseURL` and Secret-backed credential references for agent models that do
  not override them.
- `AgentRun.spec.workspaceRef` can explicitly declare the run workspace, and
  `AgentRun.status.workspaceRef` records the effective workspace identity.
- Gateway-created runs inherit workspace identity from the target `Agent`.
- Evaluation-managed runs inherit workspace identity from the owning
  `AgentEvaluation`.
- A run fails fast with `WorkspaceMismatch` when its explicit workspace does
  not match the referenced agent's workspace.

This gives the console an early but real workspace-level governance model for
the build and evaluation flow while the product workspace model moves toward
the manager database.

## What This Iteration Does Not Do Yet

This foundation does not yet:

- enforce RBAC or quota boundaries
- inject full tenant/workspace security context into runtime execution
- require every agent, evaluation, or run to reference a workspace
- implement the manager database
- sync manager-owned workspace records into Kubernetes runtime scope resources
- define workspace membership or approval models
- define billing or chargeback semantics

That is deliberate. The current milestone is about introducing the product
surface and the first lifecycle hooks before wiring full enterprise enforcement
around them.

## Expected Next Steps

1. Design the manager database model for tenant, workspace, user, team,
   membership, provider account, release, and durable audit records
2. Define the manager-to-operator synchronization contract
3. Keep the Kubernetes `Workspace` CRD focused on runtime scope and policy
   references
4. Connect workspace identity to runtime audit trails and traces
5. Map workspace semantics into console navigation and permissions
6. Add release-channel semantics for workspace-scoped publish flows
