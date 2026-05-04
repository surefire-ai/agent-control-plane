# Manager Data Model

Status: living design / implemented foundation<br>
Last updated: 2026-05-04

## Purpose

The manager is the optional database-backed product backend for the Web Console.
It owns enterprise product state that does not belong in Kubernetes CRDs.

This document started as the first data model boundary before implementation.
The foundation is now implemented in the manager backend, but the document remains
a living design note rather than a final SQL schema.

## Storage Direction

The expected default database is PostgreSQL.

The manager may later use:

- PostgreSQL for product state
- pgvector for evaluation examples, embeddings, and retrieval-adjacent metadata
- S3-compatible storage for large artifacts, reports, and exported datasets
- a queue for async sync, evaluation, and reporting jobs

The operator must remain usable without the manager database.

## Entity Groups

### Identity And Tenancy

#### Organization

Represents a customer, company, business unit, or isolated enterprise account.

Suggested fields:

- `id`
- `slug`
- `display_name`
- `status`
- `created_at`
- `updated_at`

#### Tenant

Represents the top-level product boundary inside an organization. In simple
installations, organization and tenant may map one-to-one.

Suggested fields:

- `id`
- `organization_id`
- `slug`
- `display_name`
- `status`
- `default_region`
- `metadata`
- `created_at`
- `updated_at`

#### Workspace

Represents the main user-facing working context.

Suggested fields:

- `id`
- `tenant_id`
- `slug`
- `display_name`
- `description`
- `status`
- `kubernetes_namespace`
- `kubernetes_workspace_name`
- `default_policy_id`
- `metadata`
- `created_at`
- `updated_at`

The Kubernetes `Workspace` CRD may mirror `kubernetes_workspace_name`, but the
manager database is the product source of truth.

### Users And Collaboration

#### User

Represents a human or service account known to the manager.

Suggested fields:

- `id`
- `external_subject`
- `email`
- `display_name`
- `status`
- `last_seen_at`
- `created_at`
- `updated_at`

#### Team

Groups users for product permissions and collaboration.

Suggested fields:

- `id`
- `tenant_id`
- `slug`
- `display_name`
- `description`
- `created_at`
- `updated_at`

#### Membership

Connects users or teams to tenants and workspaces.

Suggested fields:

- `id`
- `scope_type`
- `scope_id`
- `principal_type`
- `principal_id`
- `role`
- `created_at`
- `updated_at`

Roles should start coarse-grained and become finer as real workflows settle.

### Provider Management

#### Provider Account

Represents a tenant- or workspace-scoped model provider account.

Suggested fields:

- `id`
- `tenant_id`
- `workspace_id`
- `provider`
- `display_name`
- `base_url`
- `credential_ref`
- `status`
- `capabilities`
- `created_at`
- `updated_at`

`credential_ref` must point to a Kubernetes Secret, external secret manager
entry, or future credential broker reference. It must never contain secret
values. During Manager-to-CRD sync, provider account fields are preserved on
`ToolProvider`: `provider` and `display_name` map to `spec.type` and
`spec.description`; `base_url` and `credential_ref` map to `spec.http.baseURL`
and `spec.http.credentialRef`; `family`, `domestic`, and capability booleans
map to `spec.runtime.family`, `spec.runtime.domestic`, and
`spec.runtime.capabilities`.

#### Provider Policy

Defines provider restrictions and defaults at tenant or workspace scope.

Suggested fields:

- `id`
- `scope_type`
- `scope_id`
- `allowed_providers`
- `default_provider_account_id`
- `budget_policy`
- `created_at`
- `updated_at`

### Agent Product Records

#### Agent Project

Represents the product-facing agent workspace entry.

Suggested fields:

- `id`
- `workspace_id`
- `slug`
- `display_name`
- `description`
- `status`
- `kubernetes_agent_name`
- `created_by`
- `created_at`
- `updated_at`

The Kubernetes `Agent` remains the runtime declaration and compiled artifact
surface.

#### Agent Revision

Tracks publishable agent revisions in product workflows.

Suggested fields:

- `id`
- `agent_project_id`
- `revision`
- `kubernetes_agent_name`
- `compiled_revision`
- `source_snapshot_ref`
- `status`
- `created_by`
- `created_at`

#### Agent Run

Stores tenant-scoped execution history for run inspection, release debugging,
and trace handoff. The manager records durable product metadata and runtime
references; Kubernetes `AgentRun` remains the operator/runtime lifecycle
surface.

Current manager run record fields:

- `id`
- `tenant_id`
- `workspace_id`
- `agent_id`
- `evaluation_id`
- `agent_revision`
- `status`
- `runtime_engine`
- `runner_class`
- `started_at`
- `completed_at`
- `summary`
- `trace_ref`
- `metadata`
- `created_at`
- `updated_at`

### Visual Orchestration

#### Agent Draft

Stores the UI draft graph and authoring state before it is compiled to CRDs.

Suggested fields:

- `id`
- `agent_project_id`
- `workspace_id`
- `version`
- `draft_graph`
- `draft_config`
- `status`
- `updated_by`
- `updated_at`

Drafts are product state. They should not be stored in Kubernetes CRDs.

### Evaluation

#### Evaluation Project

Groups datasets, runs, thresholds, and comparison reports.

Suggested fields:

- `id`
- `workspace_id`
- `agent_project_id`
- `slug`
- `display_name`
- `description`
- `created_at`
- `updated_at`

#### Dataset Catalog Entry

Tracks product metadata for evaluation datasets.

Suggested fields:

- `id`
- `workspace_id`
- `name`
- `revision`
- `storage_ref`
- `kubernetes_dataset_name`
- `created_by`
- `created_at`
- `updated_at`

The Kubernetes `Dataset` CRD can still provide runtime-visible samples for
Kubernetes-native mode and simple managed sync.

#### Evaluation Run

Stores durable evaluation history and comparison metadata.

Suggested fields:

- `id`
- `evaluation_project_id`
- `agent_revision_id`
- `baseline_agent_revision_id`
- `kubernetes_evaluation_name`
- `status`
- `summary`
- `report_ref`
- `created_by`
- `created_at`
- `finished_at`

### Release And Governance

#### Release Channel

Represents environments or product release tracks.

Suggested fields:

- `id`
- `workspace_id`
- `slug`
- `display_name`
- `current_agent_revision_id`
- `policy`
- `created_at`
- `updated_at`

#### Approval Request

Tracks human review and exception workflows.

Suggested fields:

- `id`
- `scope_type`
- `scope_id`
- `subject_type`
- `subject_id`
- `status`
- `requested_by`
- `reviewed_by`
- `decision`
- `created_at`
- `updated_at`

### Audit And Usage

#### Audit Event

Durable product audit log entry.

Suggested fields:

- `id`
- `tenant_id`
- `workspace_id`
- `actor_id`
- `action`
- `resource_type`
- `resource_id`
- `kubernetes_ref`
- `request_id`
- `metadata`
- `created_at`

#### Usage Record

Aggregated usage and quota accounting.

Suggested fields:

- `id`
- `tenant_id`
- `workspace_id`
- `agent_project_id`
- `provider`
- `model`
- `metric`
- `quantity`
- `period_start`
- `period_end`
- `created_at`

## Kubernetes Mapping Fields

Manager entities that create or mirror Kubernetes resources should store:

- Kubernetes namespace
- Kubernetes resource kind
- Kubernetes resource name
- Kubernetes UID when available
- last synced resource version
- sync status
- last sync error

This lets the manager show product state and runtime state without becoming
the runtime controller itself.

## Initial Implementation Recommendation

The repository now includes the manager process in `cmd/manager` and
`internal/manager`. It exposes health/readiness endpoints, embedded Web Console
assets, Manager CRUD APIs for the Phase 3 product resources, and best-effort
Manager-to-CRD synchronization. It intentionally does not open a database
connection unless a database URL is supplied. The embedded schema lives under
`internal/manager/migrations/`, and `internal/manager` includes a migration
runner that can apply those migrations through `database/sql`. PostgreSQL
support is wired through the pgx stdlib driver using the `pgx` driver name.

Start with the smallest useful manager schema:

1. organizations
2. tenants
3. workspaces
4. users
5. memberships
6. provider_accounts
7. agent_projects
8. agent_drafts
9. audit_events

Then add evaluation, release, and usage tables once the console workflow needs
them.
