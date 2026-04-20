# Project Governance

Agent Control Plane is developed under the `surefire-ai` GitHub organization.

## Project Identity

- GitHub organization: `surefire-ai`
- Repository: `github.com/surefire-ai/agent-control-plane`
- Kubernetes API group: `windosx.com/v1alpha1`
- Global project domain: `windosx.com`

## Repository Governance

- Changes to API types, CRD schemas, controller behavior, security policy, and runtime execution should receive project review.
- Public API group and resource names should remain stable once a version graduates beyond `v1alpha1`.
- Security-sensitive changes should include validation, tests, and a short risk note in the pull request.
