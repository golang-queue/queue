GO ?= go
TOOLS_MOD := -modfile=go.tools.mod

## test: run tests
test:
	$(GO) test -race -v -cover -covermode=atomic -coverprofile=coverage.out ./...

## upgrade: upgrade dependencies
upgrade:
	$(GO) get -u ./...

## install-tools: download tool dependencies (golangci-lint) via tools modfile
install-tools:
	$(GO) mod download $(TOOLS_MOD)

## fmt: format go files using golangci-lint
fmt:
	$(GO) tool $(TOOLS_MOD) golangci-lint fmt

## lint: run golangci-lint to check for issues
lint:
	$(GO) tool $(TOOLS_MOD) golangci-lint run

.PHONY: test upgrade install-tools fmt lint
