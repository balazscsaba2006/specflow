.PHONY: build test test-race lint vet fmt quality fix install clean help

BINARY := specflow
PKG := ./...

## Build
build: ## Build the binary
	go build -o bin/$(BINARY) ./cmd/specflow

install: ## Install to $GOPATH/bin
	go install ./cmd/specflow

## Quality
quality: fmt vet lint test ## Run all quality checks (format + vet + lint + test)

lint: ## Run golangci-lint
	golangci-lint run

vet: ## Run go vet
	go vet $(PKG)

fmt: ## Check formatting (fails if files need formatting)
	@test -z "$$(gofmt -l .)" || (echo "Files need formatting:" && gofmt -l . && exit 1)

fix: ## Auto-fix formatting and lint issues
	gofmt -w .
	goimports -w .
	golangci-lint run --fix

## Test
test: ## Run tests
	go test $(PKG) -count=1

test-race: ## Run tests with race detector
	go test $(PKG) -count=1 -race

test-cover: ## Run tests with coverage
	go test $(PKG) -count=1 -race -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

## Utility
clean: ## Remove build artifacts
	rm -rf bin/ coverage.out coverage.html

## Help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
