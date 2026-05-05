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

### Design Tokens

The visual system is defined in `src/index.css` using CSS custom properties and
Tailwind `@theme` variables. Key token groups:

| Token group | Example variables | Usage |
|---|---|---|
| Accent | `--color-accent`, `--color-accent-light`, `--color-accent-glow` | Teal brand, focus rings, active indicators |
| Surface | `--color-surface`, `--surface-elevated`, `--surface-muted`, `--surface-panel` | Card and panel backgrounds |
| Border | `--color-border`, `--color-border-strong`, `--color-border-accent` | Separators, card outlines |
| Status | `--color-status-success/warning/danger/info/muted` | StatusBadge, status dots |
| Shadow | `--shadow-surface`, `--shadow-elevated`, `--shadow-glow` | Depth hierarchy |
| Motion | `--duration-fast/normal/slow`, `--ease-out` | Transitions and micro-interactions |

Component-layer classes (`surface`, `surface-elevated`, `surface-muted`,
`surface-panel`, `control-input`, `control-button`, `data-card`, `status-dot`,
`section-divider`) provide reusable visual primitives. Prefer these over
ad-hoc Tailwind combinations when building new UI.

All motion respects `prefers-reduced-motion: reduce` via a base-layer media
query that collapses animation and transition durations.

### Micro-interactions

CSS animation classes for perceived quality (all defined in `src/index.css`):

| Class | Effect | Usage |
|---|---|---|
| `tab-content-enter` | Fade-in + slide-up | Tab content switching |
| `save-pulse` | Gentle opacity pulse | Save status indicator, running status dots |
| `toast-enter` / `toast-exit` | Slide-up appear / slide-up disappear | Toast notifications |
| `panel-slide-in` | Slide from right | Side panels (NodeConfigPanel, NodePalette) |
| `modal-panel-reveal` | Scale + fade-in | Modal dialog entrance |
| `overlay-fade-in` | Fade overlay | Modal backdrop |
| `alert-slide-in` | Slide-down appear | Error alerts |
| `skeleton-shimmer` | Gradient shimmer | Loading skeleton placeholders |
| `focus-ring-visible` | Accent outline on `:focus-visible` | Keyboard focus indicators |
| `status-badge-animate` | Color/shadow transition | StatusBadge state changes |

### Shared Components

| Component | File | Description |
|---|---|---|
| `Button` | `shared/Button.tsx` | primary/secondary/ghost/danger, sm/md sizes |
| `Card` | `shared/Card.tsx` | default/elevated/muted/interactive variants |
| `Input` / `Select` / `Textarea` | `shared/` | Unified focus ring and border styling |
| `StatusBadge` | `shared/StatusBadge.tsx` | Status dot + label, animated transitions |
| `PageHeader` | `shared/PageHeader.tsx` | Eyebrow, title, subtitle, actions |
| `EmptyState` | `shared/EmptyState.tsx` | Product-semantic empty placeholders |
| `LoadingSkeleton` | `shared/LoadingSkeleton.tsx` | list / detail / table variants with shimmer |
| `Modal` | `shared/Modal.tsx` | Focus trap, ESC close, overlay dismiss |
| `ConfirmDialog` | `shared/ConfirmDialog.tsx` | Destructive confirm with spinner on pending |
| `ErrorAlert` | `shared/ErrorAlert.tsx` | Rose-themed alert with optional retry |
| `Toast` | `shared/Toast.tsx` | success/error/warning/info, auto-dismiss, useToast hook |
