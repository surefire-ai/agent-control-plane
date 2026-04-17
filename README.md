# Agent Control Plane

<p align="center">
  English | <a href="./README.zh-CN.md">中文</a>
</p>

Agent Control Plane is a Kubernetes-native control plane for declaring,
publishing, running, governing, and evaluating AI agents.

This project is owned and maintained by [windosx](https://github.com/windosx).
The source repository is `github.com/windosx/agent-control-plane`, and the
Kubernetes API group uses `windosx.com/v1alpha1`.

The current implementation is driven by the EHS hazard-identification examples
in `examples/ehs` and `config/samples/ehs`.

## What It Provides

- `Agent` declares runtime, models, prompts, knowledge, tools, MCP servers,
  policies, graph shape, interfaces, memory, and observability settings.
- `AgentRun` records an immutable execution request and its execution status.
- `PromptTemplate`, `KnowledgeBase`, `ToolProvider`, `MCPServer`,
  `AgentPolicy`, and `AgentEvaluation` provide the supporting control-plane
  resources.
- A controller-manager compiles `Agent` resources, publishes deterministic
  status, and dispatches `AgentRun` resources to a runtime backend.
- A worker runtime backend can dispatch each run as a Kubernetes Job. The
  current worker is intentionally a placeholder while the LangGraph runtime is
  being built.

## Architecture Direction

- Go hosts the Kubernetes API types, CRD controllers, compiler, admission
  checks, runtime dispatch, and future gateway.
- Python is expected to host the LangGraph-compatible runtime worker.
- PostgreSQL, pgvector, S3-compatible storage, and a queue are expected to
  provide state, retrieval, artifacts, and async execution as the system matures.
- TypeScript can host the future console, marketplace UI, and generated SDKs.

## Current Progress

Status date: 2026-04-17.

| Area | Status | Evidence |
| --- | --- | --- |
| YAML Agent Spec | In progress | Go API types and CRDs exist under `api/v1alpha1` and `config/crd/bases`; EHS YAML examples exist under `examples/ehs` and `config/samples/ehs`. |
| Compile to LangGraph | Partial | `internal/compiler` validates cross-resource references and produces a deterministic revision. It does not yet emit an executable LangGraph graph. |
| Publish endpoint | Partial | `Agent.status.endpoint.invoke` is published by the Agent controller as a stable Kubernetes-style invoke path. A real gateway/handler is not implemented yet. |
| Trace | Partial | `AgentRun.status.traceRef` exists, and mock/worker backends populate it. Full distributed tracing and trace storage are not implemented yet. |
| Version | Partial | `Agent.status.compiledRevision` and `AgentRun.status.agentRevision` exist. Semantic versioning, release channels, and revision history are still pending. |
| Runtime execution | Bootstrap | `mock` runtime completes runs deterministically. `worker` runtime creates Kubernetes Jobs and reports placeholder output. |
| Policy | Spec only | `AgentPolicy` CRD and `Agent.spec.policyRef` exist. Enforcement before runtime dispatch is pending. |
| Evaluation | Spec only | `AgentEvaluation` CRD exists. Evaluation reconciliation and result reporting are pending. |

## Milestones

### Phase 1: Core Agent Control Plane

Goal: make one Kubernetes-declared agent compile, publish, run, trace, and carry
version identity end to end.

| Milestone | Current state | Next work |
| --- | --- | --- |
| YAML Agent Spec | Initial CRDs and sample YAML are present. | Harden schema validation, defaults, required fields, and admission checks. |
| Compile to LangGraph | Static reference compiler exists. | Emit a LangGraph-compatible intermediate representation and persist or pass it to the runtime worker. |
| Publish endpoint | Status publishes the planned `:invoke` path. | Add the gateway/API handler that accepts invoke requests and creates `AgentRun` resources. |
| Trace | `TraceRef` is carried through `AgentRun` status. | Integrate OpenTelemetry or runtime-native tracing and store trace IDs consistently. |
| Version | Revision hash exists for compiled agents. | Add revision history, compatibility rules, release labels, and rollback semantics. |

Phase 1 exit criteria:

- Applying the EHS sample resources produces a Ready `Agent`.
- Invoking the published endpoint creates an `AgentRun`.
- The run executes through a real LangGraph worker, not the mock backend.
- The run records output, trace reference, and the exact agent revision.
- The controller-manager and worker images are buildable and deployable.

### Phase 2: Product Surface and Governance

Goal: make the platform usable by teams, not only by cluster operators.

| Milestone | Current state | Next work |
| --- | --- | --- |
| UI | Not started in this repository. | Build a console for agents, runs, traces, evaluations, and publishing workflows. |
| Marketplace | Not started. | Define package metadata, publishing workflow, trust signals, and install flow for reusable agents/tools. |
| Policy | CRD shape exists. | Enforce model/tool budgets, guardrails, approval gates, security boundaries, and runtime constraints. |
| Tenant | Not started. | Add tenancy model, namespace mapping, RBAC boundaries, quotas, and audit trails. |

Phase 2 exit criteria:

- Users can publish, inspect, invoke, and debug agents from the UI.
- Marketplace packages can be listed, installed, versioned, and reviewed.
- Policy decisions block or require approval before unsafe runs start.
- Tenant isolation is explicit across API, runtime, storage, and observability.

### Phase 3: Distributed Agent Runtime

Goal: scale from single-agent execution to a multi-runtime, multi-agent fabric.

| Milestone | Current state | Next work |
| --- | --- | --- |
| Multi-runtime | Runtime interface supports backend selection between `mock` and `worker`. | Add real adapters for LangGraph, remote runtimes, and future non-Python runtimes. |
| Agent Autoscaling | Not started. | Add queue-depth, latency, and cost-aware scaling signals for runtime workers. |
| Agent Mesh | Not started. | Define agent-to-agent discovery, invocation, policy propagation, identity, and trace correlation. |

Phase 3 exit criteria:

- Multiple runtime backends can run compatible agent revisions.
- Agents scale automatically based on demand and policy limits.
- Agent-to-agent calls preserve identity, policy, version, and trace context.

## Local Development

Run the Go test suite:

```bash
go test ./...
```

Generate deepcopy code:

```bash
make generate
```

Generate CRD manifests:

```bash
make manifests
```

Run the controller manager locally:

```bash
make run
```

Build controller-manager and worker binaries:

```bash
make build
```

Build container images:

```bash
make docker-build
```

Deploy the CRDs, RBAC, and controller-manager to the current Kubernetes
context:

```bash
make deploy
```

Remove the deployed control plane:

```bash
make undeploy
```

For local OrbStack validation, build the local worker image with:

```bash
make docker-build-worker-local
```

## Runtime Backends

The controller manager accepts `--runtime-backend`.

- `mock`: default backend. It completes `AgentRun` objects deterministically for
  control-plane validation.
- `worker`: creates a Kubernetes Job in the `AgentRun` namespace. It uses
  `--worker-job-image` and `--worker-job-command` to point at a worker image and
  command.

The repository includes two image entrypoints:

- `cmd/controller-manager`: reconciles control-plane resources.
- `cmd/worker`: validates injected run environment and emits a structured
  placeholder result.

## Repository Layout

```text
api/v1alpha1/                 Kubernetes API types
cmd/controller-manager/        controller-manager entrypoint
cmd/worker/                    worker entrypoint
config/crd/                    generated CRD manifests
config/samples/ehs/            sample custom resources
examples/ehs/                  source sample resources
internal/compiler/             Agent compiler and reference validation
internal/controller/           Agent and AgentRun reconcilers
internal/runtime/              runtime backend abstraction and implementations
internal/worker/               placeholder worker implementation
```

## License

Agent Control Plane is licensed under the Apache License, Version 2.0. See
`LICENSE`.

This project depends on third-party Go modules under their own open source
licenses. The current direct runtime dependencies are Kubernetes and
controller-runtime modules licensed under Apache-2.0. Transitive dependencies
include permissive licenses such as Apache-2.0, BSD-style, MIT-style, and ISC.

Before distributing source archives, binaries, or container images, preserve the
project `LICENSE`, preserve `NOTICE`, and include third-party license notices as
described in `THIRD_PARTY_NOTICES.md`.
