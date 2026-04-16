GO ?= go

.PHONY: test
test:
	$(GO) test ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

.PHONY: run
run:
	$(GO) run ./cmd/controller-manager
