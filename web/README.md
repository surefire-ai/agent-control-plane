# Korus Web Console

This directory contains the Korus Web Console.

Relevant product backend design docs:

- [Component boundaries](../docs/architecture/component-boundaries.md)
- [Manager data model](../docs/architecture/manager-data-model.md)
- [Manager to operator sync](../docs/architecture/manager-operator-sync.md)

The console is intended to be the primary product surface for enterprise users,
not a thin Kubernetes resource dashboard. It is backed by the optional manager
service API and should eventually use the manager database for product state,
then sync runtime resources to the Kubernetes-native operator. It should
support:

- tenant and workspace navigation
- visual agent orchestration
- agent build, evaluation, publish, and release workflows
- run debugging and artifact inspection
- provider management and model selection
- knowledge binding retrieval controls (`topK`, `scoreThreshold`)
- governance, approval, and collaboration workflows

## Current Status

The Web Console is in place:

- Vite + React + TypeScript application shell
- Tailwind CSS styling entrypoint
- React Router navigation for tenants and workspaces
- TanStack Query API hooks for manager-backed data fetching
- i18next English and Simplified Chinese locales
- Playwright e2e coverage for tenant and workspace flows
- `lucide-react` for open-source, componentized icons

The current UI is a product console rather than a landing page. It covers tenant navigation, workspace lists,
workspace detail, workspace creation, workspace editing, and workspace deletion
confirmation. Agents, Evaluations, Runs, and Providers have tenant-scoped list
and detail pages backed by manager API contracts. The Visual Orchestration
Studio supports six agent patterns, a React Flow workflow canvas, model cards
with `baseURL` and structured Secret `credentialRef.name/key`, and
knowledge-binding retrieval controls (`topK`, `scoreThreshold`). Runs expose a
tenant-scoped execution history list and detail view with status, runtime, and
trace references. The sidebar also reserves Settings so the console information
architecture stays aligned with the enterprise roadmap while that backend is
implemented.

Generated frontend artifacts are ignored by git:

- `node_modules/`
- `dist/`
- `playwright-report/`
- `test-results/`
- `tsconfig.tsbuildinfo`

Keep this directory focused on the console implementation. Control-plane
API types, controllers, compiler behavior, and worker runtime code should stay
in the existing Go packages.

Product tenants, workspaces, users, teams, membership, UI drafts, release
workflows, and durable audits should be treated as manager-owned product state,
not as direct CRD-only state.

## Implementation Direction

The implementation prioritizes the build, evaluate, and release
loop inside a workspace:

1. Tenant and workspace shell
2. Manager-backed workspace membership and provider binding basics
3. Agent list and detail pages
4. Visual orchestration studio
5. Evaluation comparison and release gate views
6. Run inspection and debugging
7. Provider and credential-reference management

## Development

Install dependencies:

```bash
npm install
```

Run the console against the manager fake API:

```bash
cd ..
go run ./cmd/manager --mode=fake
```

```bash
npm run dev
```

Build the console:

```bash
npm run build
```

Run e2e tests:

```bash
npm run test:e2e
```

## UI Conventions

- Use `lucide-react` for UI icons instead of inline SVG.
- Keep the first screen as the usable console experience, not a marketing
  landing page.
- Use the existing shell components for global layout, language switching,
  tenant switching, page headers, cards, tables, empty states, and modals.
- Preserve the enterprise console direction: tenancy, workspaces, provider
  management, evaluation, release, and collaboration should remain visible in
  the product information architecture as the UI expands.
