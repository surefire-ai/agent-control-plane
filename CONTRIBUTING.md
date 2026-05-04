# Contributing to Korus

Thanks for taking the time to improve Korus. The project is still early, so the
most valuable contributions are small, contract-aware changes that make the
control plane easier to trust, test, and extend.

## Project Direction

Korus is an enterprise, multi-tenant Agent Control Plane built on Kubernetes.
Keep the four component boundaries clear:

- `operator`: Kubernetes CRDs, controllers, compiler, status, and runtime
  dispatch.
- `manager`: optional database-backed product backend for tenants, workspaces,
  users, releases, durable audits, and UI drafts.
- `worker`: execution-side process that consumes compiled artifacts and run
  input.
- `runner`: pluggable agent execution engine boundary, starting with Eino.

Evaluation, provider governance, tenancy, and the Web Console are core
product concerns, not secondary add-ons.

## Good First Contribution Areas

- CRD validation and schema clarity
- compiler reference validation and deterministic artifact behavior
- focused worker/runtime tests
- provider catalog and capability modeling
- evaluation metrics, gates, and result reporting
- local Kubernetes smoke coverage
- Web Console product flows that preserve the manager/operator boundary
- documentation that reflects current implementation truth

## Development Setup

Install the usual Kubernetes operator toolchain:

- Go
- Docker or a compatible container runtime
- Kubernetes or local Kubernetes, such as OrbStack
- `kubectl`
- `make`

Optional:

- Helm
- Node.js for the Web Console

## Common Commands

```bash
make test
make build
make generate manifests
make helm-lint
make helm-template
make k8s-smoke-ehs
git diff --check
```

For Web Console work:

```bash
cd web
npm install
npm run build
npm run test:e2e
```

## Pull Request Expectations

Before opening a pull request:

1. Keep the change focused.
2. Update tests for behavior changes.
3. Update documentation or samples when behavior, APIs, or workflows change.
4. Run `make test` and `git diff --check`.
5. If API types changed, run `make generate manifests`.
6. If worker execution, runtime dispatch, or samples changed, prefer running
   `make k8s-smoke-ehs`.
7. If Web Console behavior changed, run the relevant Web build or E2E checks.

## API and Runtime Changes

When adding runtime behavior, prefer this order:

1. Define the typed API surface in `api/v1alpha1/`.
2. Validate references and ambiguity in `internal/compiler/`.
3. Encode behavior in compiled artifacts.
4. Interpret it in `internal/worker/` or a runner implementation.
5. Update docs and samples.

Do not hide major runtime behavior only inside worker code. The CRD, compiler,
and artifact contract should explain what will happen.

## Secrets and Credentials

Never commit secret values. Model credentials should flow through Kubernetes
`Secret` references or future external secret manager integrations.

Secret values must not be written into:

- CR specs
- status
- compiled artifacts
- worker result payloads
- logs
- documentation examples

Use placeholders such as `REPLACE_WITH_REAL_API_KEY` in examples.

## Documentation

Keep top-level docs concise and product-facing. Put detailed design notes in the
existing docs tree:

- `docs/architecture/`
- `docs/phase2/`
- `docs/phase3/`
- `docs/releases/`
- `web/README.md`

Do not create new top-level notes for temporary summaries or review scratch
work.

## Commit Style

Use conventional commit prefixes:

```text
feat: add provider catalog validation
fix: preserve worker failure reason
docs: clarify manager operator boundary
test: cover evaluation gate deltas
chore: update generated manifests
```

## AI Collaborators

AI coding agents should read `AGENTS.md` before making changes. That file
contains the repository-specific operating rules and architectural guardrails.
