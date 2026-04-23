# AGENTS.md

This file guides AI collaborators working in this repository.

## Project Identity

Agent Control Plane is a Kubernetes-native control plane for AI agents.

Treat this project as an **operator plus runtime system**, not as a generic app
or a prompt playground.

- The **operator/control plane** lives in `api/`, `internal/controller/`,
  `internal/compiler/`, `internal/gateway/`, and `internal/runtime/`.
- The **execution plane** lives in `cmd/worker/` and `internal/worker/`.
- The declarative API surface is defined by Kubernetes CRDs under
  `api/v1alpha1/` and generated manifests under `config/crd/bases/`.

Current repository direction:

- Default runtime direction is `runtime.engine=eino` with
  `runtime.runnerClass=adk`.
- `Skill` is a reusable capability bundle.
- `Pattern` is a compile-time convenience layer.
- `react` is the first supported pattern preset.
- Do **not** introduce a separate `rag` preset. ReAct should consume the
  agent's normal `knowledgeRefs` and `toolRefs`.
- Product direction is explicitly **enterprise, multi-tenant, evaluation-led,
  and UX-aware**.
- The future console is a **first-class visual orchestration, evaluation, and
  release surface**, not a thin dashboard layered on top of CRDs.

## What To Optimize For

When making changes, optimize for these goals in order:

1. Preserve a clean boundary between control plane and execution plane.
2. Keep CRDs and compiled artifacts deterministic and auditable.
3. Make the product shape enterprise-ready: tenancy, isolation, governance,
   provider breadth, and evaluation are not optional extras.
4. Treat the web console as a product surface for visual orchestration,
   evaluation, release, and collaboration, not as a passive admin view.
5. Prefer incremental extension of the existing model over broad redesign.
6. Keep the worker runtime simple enough to validate locally in Kubernetes.
7. Make changes that support the current roadmap:
   `Skill -> Pattern -> Runtime semantics -> SubAgent/A2A`.

## Build, Buy, Integrate Policy

Use this rule of thumb when deciding whether to implement something here.

### Build in this repository

These are the core differentiators and should stay first-class:

- CRD API shape
- compiler rules and validation
- deterministic compiled artifacts
- `AgentRun` lifecycle and status contract
- evaluation contract, revision comparison, and release-gate semantics
- provider abstraction and capability modeling
- tenant and workspace boundaries in the control-plane model
- Kubernetes runtime dispatch and secret-handling boundaries
- opinionated `Skill` and `Pattern` behavior

### Borrow ideas from other projects

These areas are worth studying and adapting, but not necessarily copying:

- tenancy and namespace strategy
- package and marketplace models
- SubAgent composition
- A2A-compatible resource boundaries
- enterprise evaluation UX and workflow patterns
- provider catalog and model-switching UX
- product-facing console and platform workflows

### Integrate instead of rewrite

Do not rebuild these unless the user explicitly asks for it and there is a
clear project-specific reason:

- model provider integrations below the control-plane contract
- graph execution engines
- vector databases and retrieval infrastructure
- object storage and queue infrastructure
- tracing, metrics, and logging foundations
- generic UI infrastructure

The project should own the **API, compiler, and runtime contract**, not every
implementation detail beneath them.

## Architectural Guardrails

### 1. Do not collapse controller and worker responsibilities

Keep these responsibilities separate:

- `controller-manager` reconciles resources, compiles artifacts, manages status,
  and dispatches runs.
- `worker` consumes compiled artifacts and run input, then executes models,
  tools, retrieval, and future graph semantics.

Do not move execution logic into controllers just because it seems convenient.

### 2. CRDs are the product surface

If you add new behavior, prefer expressing it through:

- typed API fields in `api/v1alpha1/`
- compiled artifact data in `internal/compiler/`
- runtime interpretation in `internal/worker/`

Avoid hidden behavior that only exists in worker code without an API or
artifact representation.

### 3. Compiler first, runtime second

For new orchestration features:

1. add or refine API shape
2. validate references and ambiguity in the compiler
3. encode the behavior into compiled artifacts
4. only then interpret it in the worker/runtime

This is especially important for:

- `Skill`
- `Pattern`
- future `SubAgent`
- future A2A interoperability

### 4. Prefer one obvious path

Do not add parallel abstractions unless there is a strong reason.

Examples:

- Do not add a separate `rag` preset; make `react` consume `knowledgeRefs`.
- Do not create a second skill system outside `Skill` CRDs.
- Do not create a second runtime contract outside compiled artifacts and
  `AgentRun`.

## Current Implementation Truths

These are current facts of the repo and should be preserved unless the user
explicitly asks for a directional change:

- `Agent`, `AgentRun`, `PromptTemplate`, `ToolProvider`, `KnowledgeBase`,
  `Dataset`, `MCPServer`, `AgentPolicy`, `AgentEvaluation`, and `Skill` are
  CRD-backed resources.
- The product target is an enterprise multi-tenant platform, not a single-team
  sandbox.
- The product target is also a user-facing enterprise platform where the web
  console is expected to support visual agent orchestration, evaluation,
  publishing, and release management.
- Evaluation should grow into a flagship capability, not remain an auxiliary
  CRD.
- `AgentEvaluation` is moving toward a first-class enterprise contract with
  typed dataset, baseline, evaluator, threshold gate, and reporting fields.
- `AgentEvaluation` can now evaluate both a current agent and an optional
  baseline agent, then publish score deltas and gate deltas into status;
  extend that comparison path rather than creating a second revision-compare
  mechanism.
- `Dataset` is the reusable evaluation sample surface; prefer referencing it
  from `AgentEvaluation.datasetRef` over embedding large sample sets directly
  into runtime config.
- `Dataset.spec.samples[].expected` is the first-class rule-eval surface for
  simple metrics such as exact field matches and count checks; extend that
  before adding a parallel evaluation DSL.
- Structured evaluators should layer on top of the same `Dataset.expected`
  surface. Current examples are `risk_level_match` and `hazard_coverage`.
- `AgentEvaluation` can already create a managed `AgentRun` from
  `spec.runtime.sampleInput` or `spec.runtime.samples` and fold aggregated
  run/gate status back into its own status; extend that path instead of
  inventing a parallel evaluation engine.
- Model provider support should evolve into a capability matrix that treats
  Chinese domestic providers as first-class targets.
- `ModelSpec.provider` is no longer just a free-form string in practice: the
  compiler validates it against the provider catalog, emits provider family
  metadata into artifacts, and the worker currently routes the
  `openai-compatible` family through the existing chat-model path.
- `Skill` can currently contribute prompts, tools, knowledge, functions, and
  graph fragments.
- `react` can expand into a runner graph when `spec.graph` is empty.
- ReAct expansion should consume normal agent-selected knowledge and tools.
- Worker execution currently supports model, tool, retrieval, function, and
  step-based graph execution in a staged form.
- OrbStack local Kubernetes smoke validation is part of the intended developer
  workflow.

## File Ownership Guide

- `api/v1alpha1/`
  - API types and declarative surface
  - if edited, regenerate deepcopy and CRDs
- `internal/compiler/`
  - reference validation, pattern expansion, artifact construction
- `internal/contract/`
  - typed artifact and worker result contracts
- `internal/controller/`
  - reconciliation and status lifecycle
- `internal/runtime/`
  - worker Job construction and runtime dispatch
- `internal/worker/`
  - execution semantics
- `config/crd/bases/`
  - generated CRD output, never hand-author as source of truth
- `config/samples/`
  - canonical samples and smoke overlays
- `docs/phase2/`
  - roadmap and design docs for the runtime direction

## Required Workflow For Code Changes

If you change API types:

1. update `api/v1alpha1/...`
2. run `make generate manifests`
3. verify generated files changed as expected

If you change compiler, runtime, or worker behavior:

1. update focused tests first or alongside the change
2. run at least the affected package tests
3. run `make test` before finishing
4. consider whether the change affects tenancy, evaluation, provider support,
   visual orchestration semantics, or future UI semantics; if so, update the
   relevant docs

If you change samples or local validation behavior:

1. keep `config/samples/ehs` as the canonical sample source
2. keep `config/samples/ehs-orbstack-smoke` aligned with the local smoke path
3. prefer `kustomize`-based sample application

## Commands You Should Actually Use

Use these commands by default:

```bash
make test
make generate manifests
make build
make k8s-smoke-ehs
```

Useful targeted commands:

```bash
go test ./internal/compiler/...
go test ./internal/worker/...
go test ./internal/runtime/...
```

## Local Validation Expectations

Before concluding substantial runtime or compiler work:

- run `make test`
- run `git diff --check`

For changes affecting worker execution, runtime dispatch, or EHS samples, also
prefer validating with local Kubernetes when available:

```bash
make k8s-smoke-ehs
```

## Anti-Patterns To Avoid

Do not:

- add secret values to status, artifacts, logs, or compiled artifacts
- bypass `Secret` references by inlining credentials into specs
- hand-edit generated deepcopy or CRD files without regenerating from source
- introduce major product concepts without a typed API and compiler story
- add speculative frameworks that are not on the current roadmap
- split samples across duplicate directories

## Documentation Expectations

When behavior changes materially, update the relevant docs:

- `README.md`
- `README.zh-CN.md`
- `docs/phase2/eino-runtime-design.md`
- `docs/phase2/agent-patterns-and-a2a-todo.md`

When the change affects enterprise product direction, also keep these topics
current in docs and code comments where appropriate:

- multi-tenant and workspace boundaries
- evaluation-first product semantics
- provider capability matrix and domestic provider support
- UX implications for the future web console

Keep docs aligned with current truth. Do not leave roadmap tables claiming
"Not started" when code already exists.

## When In Doubt

If a change could go in multiple directions, prefer the option that:

- extends the existing CRD/compiler/runtime pipeline
- preserves deterministic compiled artifacts
- keeps runtime semantics explicit
- helps future `SubAgent` and A2A work instead of creating a parallel model
