GO ?= go
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4
IMAGE_REPOSITORY ?= ghcr.io/surefire-ai
IMAGE_TAG ?= latest
LOCAL_DOCKER_ARCH ?= arm64
KUBECTL ?= kubectl

.PHONY: test
test:
	$(GO) test -race ./...

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="" paths=./api/...

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) rbac:roleName=korus-manager-role crd:crdVersions=v1,allowDangerousTypes=true paths=./api/... paths=./internal/controller/... output:rbac:artifacts:config=config/rbac output:crd:artifacts:config=config/crd/bases

.PHONY: install
install: manifests
	$(KUBECTL) apply -k config/crd

.PHONY: uninstall
uninstall:
	$(KUBECTL) delete -k config/crd

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: build
build:
	$(GO) build ./cmd/controller-manager ./cmd/manager ./cmd/worker

.PHONY: docker-build
docker-build:
	docker build -f Dockerfile.controller-manager -t $(IMAGE_REPOSITORY)/korus-controller-manager:$(IMAGE_TAG) .
	docker build -f Dockerfile.worker -t $(IMAGE_REPOSITORY)/korus-worker:$(IMAGE_TAG) .

.PHONY: docker-build-worker-local
docker-build-worker-local:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=$(LOCAL_DOCKER_ARCH) $(GO) build -o bin/korus-worker ./cmd/worker
	docker build -f Dockerfile.worker.local -t korus-worker:dev .

.PHONY: docker-build-controller-local
docker-build-controller-local:
	docker build --build-arg TARGETARCH=$(LOCAL_DOCKER_ARCH) -f Dockerfile.controller-manager -t korus-controller-manager:dev .

.PHONY: k8s-smoke-ehs-setup
k8s-smoke-ehs-setup:
	@$(KUBECTL) version >/dev/null 2>&1 || (echo "Kubernetes API is unavailable; start OrbStack or point KUBECTL to a live cluster."; exit 1)
	$(KUBECTL) create namespace ehs --dry-run=client -o yaml | $(KUBECTL) apply -f -
	$(KUBECTL) apply -k config/samples/ehs-orbstack-smoke
	$(KUBECTL) -n ehs rollout status deployment/mock-openai --timeout=60s
	@for i in $$(seq 1 60); do \
		observed=$$($(KUBECTL) -n ehs get agent ehs-hazard-identification-agent -o jsonpath='{.status.observedGeneration}' 2>/dev/null); \
		generation=$$($(KUBECTL) -n ehs get agent ehs-hazard-identification-agent -o jsonpath='{.metadata.generation}' 2>/dev/null); \
		reason=$$($(KUBECTL) -n ehs get agent ehs-hazard-identification-agent -o jsonpath='{.status.conditions[?(@.type=="Ready")].reason}' 2>/dev/null); \
		if [ -n "$$generation" ] && [ "$$observed" = "$$generation" ] && [ "$$reason" = "CompilationSucceeded" ]; then \
			echo "Agent compilation is ready: generation=$$generation"; \
			break; \
		fi; \
		sleep 2; \
	done

.PHONY: k8s-smoke-ehs-run
k8s-smoke-ehs-run:
	$(KUBECTL) -n ehs delete agentrun ehs-hazard-run-20260416-0001 --ignore-not-found
	$(KUBECTL) apply -f config/samples/ehs/ehs-hazard-run-20260416-0001.yaml

.PHONY: k8s-smoke-ehs-status
k8s-smoke-ehs-status:
	@for i in $$(seq 1 60); do \
		phase=$$($(KUBECTL) -n ehs get agentrun ehs-hazard-run-20260416-0001 -o jsonpath='{.status.phase}' 2>/dev/null); \
		if [ "$$phase" = "Succeeded" ] || [ "$$phase" = "Failed" ]; then \
			echo "AgentRun phase: $$phase"; \
			break; \
		fi; \
		sleep 2; \
	done
	$(KUBECTL) -n ehs get agentrun ehs-hazard-run-20260416-0001 -o jsonpath='{.status.output}'; echo

.PHONY: k8s-smoke-ehs
k8s-smoke-ehs: k8s-smoke-ehs-setup k8s-smoke-ehs-run k8s-smoke-ehs-status

.PHONY: run
run:
	$(GO) run ./cmd/controller-manager

.PHONY: deploy
deploy: manifests
	$(KUBECTL) apply -k config/default

.PHONY: helm-lint
helm-lint:
	helm lint charts/korus

.PHONY: helm-template
helm-template:
	helm template korus charts/korus --namespace korus-system --include-crds

.PHONY: undeploy
undeploy:
	$(KUBECTL) delete -k config/default
