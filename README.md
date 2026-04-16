# Agent Control Plane

Agent Control Plane is a Kubernetes-native control plane for declaring, publishing, running, governing, and evaluating AI agents.

This project is owned and maintained by [windosx](https://github.com/windosx). The planned source repository is `github.com/windosx/agent-control-plane`, and the Kubernetes API group uses `windosx.com/v1alpha1`.

The initial shape follows the sample EHS hazard-identification resources in `examples/ehs`:

- `Agent` defines runtime, models, prompt references, knowledge references, tools, MCP servers, policy, graph, interfaces, and observability.
- `AgentRun` records one immutable execution request and its status.
- `PromptTemplate`, `KnowledgeBase`, `ToolProvider`, `MCPServer`, `AgentPolicy`, and `AgentEvaluation` provide the supporting control-plane resources.

## Architecture Direction

- Go hosts the Kubernetes API types, controllers, compiler, admission checks, and gateway.
- Python hosts the LangGraph-compatible runtime worker.
- PostgreSQL, pgvector, S3-compatible storage, and a queue provide state, retrieval, artifacts, and async execution.
- TypeScript can host the future console and generated SDKs.

## Current Bootstrap

This repository currently contains:

- `api/v1alpha1`: initial Go API types for the core custom resources.
- `internal/compiler`: a first static compiler pass that validates Agent references and generates a deterministic revision.
- `cmd/controller-manager`: a controller-manager entrypoint with selectable AgentRun runtime backend.
- `internal/runtime`: runtime abstraction with the default mock backend and a worker backend placeholder.
- `examples/ehs` and `config/samples/ehs`: the EHS sample resources used to drive the first implementation slice.

## Local Runtime Backends

The controller manager accepts `--runtime-backend`. The default is `mock`, which completes `AgentRun` objects deterministically for control-plane validation.

The first `worker` backend uses a Kubernetes Job in the `AgentRun` namespace. It starts with a placeholder command and marks the run complete after the Job succeeds. Use `--worker-job-image` and `--worker-job-command` to point it at a real worker image as the runtime matures.

## Next Milestones

1. Package controller-manager and worker into deployable container images.
2. Replace the placeholder worker Job command with a Python LangGraph worker image.
3. Add policy enforcement before runtime dispatch.
4. Add output schema validation before marking `AgentRun` succeeded.
5. Add tracing, events, and evaluation controllers.
