# Third-Party Notices

Korus is licensed under Apache-2.0. It also depends on
third-party Go modules that remain under their own licenses.

This file records the current dependency notice policy for source, binary, and
container-image distributions. It is not a replacement for the individual
licenses shipped by upstream projects.

## Current Direct Dependencies

The current direct Go dependencies in `go.mod` are:

| Dependency | Version | License family | Notes |
| --- | --- | --- | --- |
| `k8s.io/api` | `v0.31.0` | Apache-2.0 | Kubernetes API types. |
| `k8s.io/apiextensions-apiserver` | `v0.31.0` | Apache-2.0 | CRD extension APIs. |
| `k8s.io/apimachinery` | `v0.31.0` | Apache-2.0 | Kubernetes API machinery. |
| `k8s.io/client-go` | `v0.31.0` | Apache-2.0 | Kubernetes client libraries. |
| `sigs.k8s.io/controller-runtime` | `v0.19.0` | Apache-2.0 | Controller manager and reconciliation framework. |

The transitive module graph is locked by `go.sum` and can be inspected with:

```bash
go list -m all
```

## Distribution Requirements

When distributing this project as source:

- Include `LICENSE`.
- Include `NOTICE`.
- Keep upstream copyright, license, patent, trademark, and attribution notices
  that appear in vendored or copied third-party source files.

When distributing compiled binaries or container images:

- Include this project's `LICENSE` and `NOTICE` in the release artifact or image
  documentation.
- Include or make available third-party license texts for the module graph used
  to build the artifact.
- Preserve upstream NOTICE files for Apache-2.0 dependencies when they are
  present and applicable.
- Re-run the dependency license review after any `go.mod` or `go.sum` change.

## Current License Posture

The current direct dependencies are Apache-2.0. The current transitive Go module
graph is expected to be composed of permissive open source licenses commonly
used in the Go and Kubernetes ecosystems, including Apache-2.0, BSD-style,
MIT-style, and ISC-style licenses.

Known notice-bearing transitive dependencies in the current module cache:

| Dependency path | Notice |
| --- | --- |
| `sigs.k8s.io/yaml/goyaml.v2` | Copyright 2011-2016 Canonical Ltd.; Apache-2.0 notice retained in `NOTICE`. |
| `sigs.k8s.io/yaml/goyaml.v3` | Copyright 2011-2016 Canonical Ltd.; Apache-2.0 notice retained in `NOTICE`. |

Before a public release, generate an artifact-specific third-party license
bundle from the exact module graph and container base images used for that
release. A typical release checklist is:

1. Run `go list -m all` and save the exact module graph.
2. Collect license files from each module in the resolved module cache or from
   vendored dependencies.
3. Collect licenses and notices for container base images.
4. Review the resulting bundle for copyleft, source-availability, attribution,
   or NOTICE obligations.
5. Ship the bundle with release artifacts and container image documentation.
