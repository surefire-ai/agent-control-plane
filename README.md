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
| Compile to LangGraph | Partial | `internal/compiler` validates cross-resource references, emits a runtime-oriented compiled artifact, and produces a deterministic revision. It does not yet emit an executable LangGraph graph. |
| Publish endpoint | Bootstrap | `Agent.status.endpoint.invoke` is published by the Agent controller, and the invoke gateway can create `AgentRun` resources from POST requests. |
| Trace | Partial | `AgentRun.status.traceRef` exists, and mock/worker backends populate it. Full distributed tracing and trace storage are not implemented yet. |
| Version | Partial | `Agent.status.compiledRevision` and `AgentRun.status.agentRevision` exist. Semantic versioning, release channels, and revision history are still pending. |
| Runtime execution | Bootstrap | `mock` runtime completes runs deterministically. `worker` runtime creates Kubernetes Jobs and reports placeholder output. |
| Policy | Spec only | `AgentPolicy` CRD and `Agent.spec.policyRef` exist. Enforcement before runtime dispatch is pending. |
| Evaluation | Spec only | `AgentEvaluation` CRD exists. Evaluation reconciliation and result reporting are pending. |

## Milestones

### Phase 1: Kubernetes-Native MVP

Goal: make one Kubernetes-declared agent compile, publish status, run through a
Kubernetes Job, and report output, trace reference, and revision identity end to
end.

| Milestone | Current state | Next work |
| --- | --- | --- |
| YAML Agent Spec | Initial CRDs and EHS sample YAML are present. | Harden schema validation, defaults, required fields, and admission checks. |
| Agent compiler | Static reference compiler exists, writes `Agent.status.compiledArtifact`, and produces artifact-based revisions. | Pass the compiled artifact to workers and evolve it toward a LangGraph-compatible IR. |
| AgentRun lifecycle | `Pending`, `Running`, `Succeeded`, and `Failed` transitions are implemented. | Add cancellation, timeout, retry, and idempotency semantics. |
| Kubernetes Job runtime | `worker` backend creates Jobs and updates `AgentRun` status after completion. | Persist richer worker output and surface Job/Pod failure details. |
| Invoke gateway | `Agent.status.endpoint.invoke` publishes the planned path. | Add the gateway/API handler that accepts invoke requests and creates `AgentRun` resources. |
| Packaging and deployment | Dockerfiles, RBAC, and `config/default` deployment manifests exist. | Add CI, image publishing, release tags, and installable release artifacts. |

Phase 1 exit criteria:

- Applying the EHS sample resources produces a Ready `Agent`.
- Invoking an Agent through the gateway creates an `AgentRun`.
- The run executes through the Kubernetes Job runtime backend.
- The run records output, trace reference, and the exact agent revision.
- The controller-manager and worker images are buildable, deployable, and
  releasable.

### Phase 2: Real Agent Runtime

Goal: replace the placeholder worker with a real LangGraph-compatible runtime
while preserving the Kubernetes-native control-plane contract.

| Milestone | Current state | Next work |
| --- | --- | --- |
| LangGraph compile IR | Static reference compiler exists. | Emit a LangGraph-compatible intermediate representation. |
| Python runtime worker | Go placeholder worker validates injected run context. | Execute compiled graphs with LangGraph and return structured results. |
| Runtime contract | `AgentRun` carries input, output, trace reference, and revision. | Define artifacts, logs, errors, cancellation, and retry behavior. |
| Policy checks | `AgentPolicy` CRD and `Agent.spec.policyRef` exist. | Enforce pre-dispatch model/tool budgets, guardrails, and approval gates. |
| Durable run records | Status is stored on `AgentRun`. | Add durable trace, artifact, and result storage. |
| Evaluation | `AgentEvaluation` CRD exists. | Add an evaluation reconciler and result reporting. |

Phase 2 exit criteria:

- An EHS AgentRun executes through a real LangGraph worker.
- Policy can block or require approval before unsafe runs start.
- Run artifacts and traces can be inspected after worker Pods are gone.
- Evaluation resources can execute against an agent revision and publish results.

### Phase 3: Product Surface and Governance

Goal: make the platform usable by teams, not only by cluster operators.

| Milestone | Current state | Next work |
| --- | --- | --- |
| UI | Not started in this repository. | Build a console for agents, runs, traces, evaluations, and publishing workflows. |
| Marketplace | Not started. | Define package metadata, publishing workflow, trust signals, and install flow for reusable agents/tools. |
| Tenant | Not started. | Add tenancy model, namespace mapping, RBAC boundaries, quotas, and audit trails. |
| Governance workflows | Policy CRD exists. | Add review, approval, human-in-the-loop, and exception workflows. |

Phase 3 exit criteria:

- Users can publish, inspect, invoke, and debug agents from the UI.
- Marketplace packages can be listed, installed, versioned, and reviewed.
- Tenant isolation is explicit across API, runtime, storage, and observability.
- Governance workflows are auditable and enforceable.

### Phase 4: Distributed Agent Fabric

Goal: scale from single-agent execution to a multi-runtime, multi-agent fabric.

| Milestone | Current state | Next work |
| --- | --- | --- |
| Multi-runtime | Runtime interface supports backend selection between `mock` and `worker`. | Add adapters for LangGraph, remote runtimes, and future non-Python runtimes. |
| Agent autoscaling | Not started. | Add queue-depth, latency, and cost-aware scaling signals for runtime workers. |
| Agent mesh | Not started. | Define agent-to-agent discovery, invocation, policy propagation, identity, and trace correlation. |

Phase 4 exit criteria:

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

## Invoke Gateway

The controller-manager starts an invoke gateway on `--gateway-bind-address`
(`:8082` by default). It accepts:

```text
POST /apis/windosx.com/v1alpha1/namespaces/{namespace}/agents/{agent}:invoke
```

Request body:

```json
{
  "input": {
    "task": "identify_hazard",
    "payload": {
      "text": "inspection text"
    }
  },
  "execution": {
    "mode": "sync"
  }
}
```

For a deployed local control plane, port-forward the gateway service and invoke
the EHS sample agent:

```bash
kubectl -n agent-control-plane-system port-forward svc/agent-control-plane-gateway 8082:8082
curl -sS -X POST http://127.0.0.1:8082/apis/windosx.com/v1alpha1/namespaces/ehs/agents/ehs-hazard-identification-agent:invoke \
  -H 'Content-Type: application/json' \
  -d '{"input":{"task":"identify_hazard","payload":{"text":"巡检发现配电箱门打开，现场地面有积水。"}},"execution":{"mode":"sync"}}'
```

The gateway returns the accepted `AgentRun` name. The `AgentRun` controller then
dispatches it through the configured runtime backend.

## Repository Layout

```text
api/v1alpha1/                 Kubernetes API types
cmd/controller-manager/        controller-manager entrypoint
cmd/worker/                    worker entrypoint
config/crd/                    generated CRD manifests
config/default/                installable Kustomize entrypoint
config/manager/                controller-manager and gateway service manifests
config/samples/ehs/            sample custom resources
examples/ehs/                  source sample resources
internal/compiler/             Agent compiler and reference validation
internal/controller/           Agent and AgentRun reconcilers
internal/gateway/              invoke gateway
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
