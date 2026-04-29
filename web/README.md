# Web Console

This directory is reserved for the future Agent Control Plane web console.

The console is intended to be the primary product surface for enterprise users,
not a thin Kubernetes resource dashboard. It should be backed by the optional
manager service and its database, then sync runtime resources to the
Kubernetes-native operator. It should support:

- tenant and workspace navigation
- visual agent orchestration
- agent build, evaluation, publish, and release workflows
- run debugging and artifact inspection
- provider management and model selection
- governance, approval, and collaboration workflows

## Current Status

No web application has been scaffolded yet.

Keep this directory focused on the future console implementation. Control-plane
API types, controllers, compiler behavior, and worker runtime code should stay
in the existing Go packages.

Product tenants, workspaces, users, teams, membership, UI drafts, release
workflows, and durable audits should be treated as manager-owned product state,
not as direct CRD-only state.

## Implementation Direction

The first implementation should prioritize the build, evaluate, and release
loop inside a workspace:

1. Tenant and workspace shell
2. Manager-backed workspace membership and provider binding basics
3. Agent list and detail pages
4. Visual orchestration studio
5. Evaluation comparison and release gate views
6. Run inspection and debugging
7. Provider and credential-reference management
