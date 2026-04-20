GO ?= go
CONTROLLER_GEN ?= go run sigs.k8s.io/controller-tools/cmd/controller-gen@v0.16.4
IMAGE_REPOSITORY ?= ghcr.io/surefire-ai
IMAGE_TAG ?= latest
LOCAL_DOCKER_ARCH ?= arm64

.PHONY: test
test:
	$(GO) test ./...

.PHONY: generate
generate:
	$(CONTROLLER_GEN) object:headerFile="" paths=./api/...

.PHONY: manifests
manifests:
	$(CONTROLLER_GEN) rbac:roleName=agent-control-plane-manager-role crd:crdVersions=v1,allowDangerousTypes=true paths=./api/... paths=./internal/controller/... output:rbac:artifacts:config=config/rbac output:crd:artifacts:config=config/crd/bases

.PHONY: install
install: manifests
	kubectl apply -k config/crd

.PHONY: uninstall
uninstall:
	kubectl delete -k config/crd

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: build
build:
	$(GO) build ./cmd/controller-manager ./cmd/worker

.PHONY: docker-build
docker-build:
	docker build -f Dockerfile.controller-manager -t $(IMAGE_REPOSITORY)/agent-control-plane-controller-manager:$(IMAGE_TAG) .
	docker build -f Dockerfile.worker -t $(IMAGE_REPOSITORY)/agent-control-plane-worker:$(IMAGE_TAG) .

.PHONY: docker-build-worker-local
docker-build-worker-local:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=$(LOCAL_DOCKER_ARCH) $(GO) build -o bin/agent-control-plane-worker ./cmd/worker
	docker build -f Dockerfile.worker.local -t agent-control-plane-worker:dev .

.PHONY: run
run:
	$(GO) run ./cmd/controller-manager

.PHONY: deploy
deploy: manifests
	kubectl apply -k config/default

.PHONY: helm-lint
helm-lint:
	helm lint charts/agent-control-plane

.PHONY: helm-template
helm-template:
	helm template agent-control-plane charts/agent-control-plane --namespace agent-control-plane-system --include-crds

.PHONY: undeploy
undeploy:
	kubectl delete -k config/default
