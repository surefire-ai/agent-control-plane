# Tenancy And Workspace Model

Status: draft  
Last updated: 2026-04-23

## Purpose

This document explains the initial tenancy and workspace model for the product.

The goal of the first iteration is not to finish multi-tenant isolation. The
goal is to establish a clear control-plane surface that future runtime,
governance, and console features can build on.

## Current Direction

The repository now defines two first-class control-plane resources:

- `Tenant`
- `Workspace`

These resources are the starting point for enterprise scoping in the platform.

## Why This Layer Exists

The project is no longer only a Kubernetes operator for technical users. It is
becoming an enterprise platform where the console must support:

- tenant switching
- workspace-scoped agent development
- evaluation and release workflows
- provider and policy scoping
- collaboration and governance

Without explicit tenant and workspace resources, those product workflows would
either become console-only state or be scattered across ad hoc labels and
namespace conventions.

## Resource Roles

### Tenant

`Tenant` is the top-level product boundary.

It is meant to represent an enterprise customer, business unit, or isolated
organizational domain.

The first API skeleton is intentionally small and currently carries:

- display metadata
- freeform profile metadata
- freeform governance metadata
- freeform provider metadata

Future iterations can expand this into:

- quota policy
- compliance profile
- provider policy
- billing or cost allocation metadata
- domain and identity mapping

### Workspace

`Workspace` is the primary working context inside a tenant.

It is meant to gather the agents, evaluations, datasets, provider bindings, and
team collaboration state for a single team, product, or business domain.

The first API skeleton currently carries:

- `tenantRef`
- display metadata
- optional namespace mapping
- freeform provider metadata
- freeform governance metadata

Future iterations can expand this into:

- member and role bindings
- release channels
- default providers
- workspace-scoped policies
- dataset and evaluation ownership rules
- audit and approval routing

## Initial Modeling Assumptions

### 1. Tenant and Workspace are control-plane resources first

They exist to shape product semantics and future policy boundaries before full
runtime isolation is implemented.

### 2. Workspace is the main user-facing scope

Most day-to-day product actions should happen inside a workspace, not directly
at the tenant layer.

### 3. Namespace mapping is explicit but not final

`Workspace.spec.namespace` exists as an early bridge to Kubernetes runtime
scoping, but it should not be mistaken for the finished isolation model.

Later iterations may support:

- one workspace to one namespace
- one workspace to multiple namespaces
- virtual workspaces above several namespaces

### 4. Freeform fields are temporary expansion joints

`provider` and `governance` are currently freeform on purpose. They let the
project begin modeling enterprise semantics without prematurely freezing the
shape of every policy and provider contract.

These fields should be tightened gradually as real workflows stabilize.

## What This Iteration Does Not Do Yet

This first skeleton does not yet:

- reconcile `Tenant` or `Workspace`
- enforce RBAC or quota boundaries
- inject workspace identity into runtime execution
- require agents or evaluations to reference a workspace
- define workspace membership or approval models
- define billing or chargeback semantics

That is deliberate. The current milestone is about introducing the product
surface before wiring enforcement and lifecycle behavior around it.

## Expected Next Steps

1. Add `Tenant` and `Workspace` status semantics through controllers
2. Decide how agents, evaluations, and providers attach to workspaces
3. Introduce workspace-aware provider and policy resolution
4. Map workspace semantics into console navigation and permissions
5. Tighten governance and provider substructures as real workflows settle
