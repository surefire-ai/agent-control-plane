GO ?= go
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4

.PHONY: test
test:
	$(GO) test ./...

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="" paths=./api/...

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) crd:crdVersions=v1,allowDangerousTypes=true paths=./api/... output:crd:artifacts:config=config/crd/bases

.PHONY: install
install: manifests
	kubectl apply -k config/crd

.PHONY: uninstall
uninstall:
	kubectl delete -k config/crd

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: run
run:
	$(GO) run ./cmd/controller-manager
