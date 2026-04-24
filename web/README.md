# Web Console

This directory is reserved for the future Agent Control Plane web console.

The console is intended to be the primary product surface for enterprise users,
not a thin Kubernetes resource dashboard. It should support:

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

## Design References

- [Console information architecture](../docs/phase3/console-information-architecture.md)
- [Tenancy and workspace model](../docs/phase3/tenancy-workspace-model.md)

## Implementation Direction

The first implementation should prioritize the build, evaluate, and release
loop inside a workspace:

1. Tenant and workspace shell
2. Agent list and detail pages
3. Visual orchestration studio
4. Evaluation comparison and release gate views
5. Run inspection and debugging
6. Provider and credential-reference management
