## Summary

<!-- What changed and why? -->

## Test Plan

<!-- List the commands or manual checks you ran. -->

## Risk Notes

- [ ] API/resource schema impact reviewed
- [ ] Security-sensitive behavior reviewed
- [ ] Secret handling reviewed
- [ ] Tenant/workspace boundary reviewed
- [ ] Evaluation behavior reviewed
- [ ] Provider compatibility reviewed
- [ ] Documentation or examples updated when behavior changed

## Component Scope

- [ ] CRD/API
- [ ] Compiler
- [ ] Controller/operator
- [ ] Manager
- [ ] Worker/runtime
- [ ] Runner/Eino
- [ ] Evaluation
- [ ] Web Console
- [ ] Helm/deployment
- [ ] Documentation only

## Required Checks

- [ ] `make test`
- [ ] `git diff --check`
- [ ] `make generate manifests` if API types changed
- [ ] `make k8s-smoke-ehs` if runtime, worker, or sample behavior changed
- [ ] Web build/E2E checks if Web Console behavior changed
