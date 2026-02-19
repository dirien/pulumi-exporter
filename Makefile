BINARY       := pulumi-exporter
MODULE       := github.com/dirien/pulumi-exporter
GO           := go
GOFLAGS      ?=
LDFLAGS      := -s -w
GOLANGCI     := golangci-lint
GORELEASER   := goreleaser

# OpenAPI code generation
OAPI_CODEGEN := go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
OAPI_SPEC    := https://api.pulumi.com/api/openapi/pulumi-spec.json
OAPI_CONFIG  := oapi-codegen.yaml
OAPI_OUTPUT  := internal/pulumiapi/client.gen.go

# Version info (injected via ldflags during release)
VERSION      ?= dev
REVISION     ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
BRANCH       ?= $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo unknown)
BUILD_DATE   ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

VERSION_PKG  := github.com/prometheus/common/version
LDFLAGS      += -X $(VERSION_PKG).Version=$(VERSION)
LDFLAGS      += -X $(VERSION_PKG).Revision=$(REVISION)
LDFLAGS      += -X $(VERSION_PKG).Branch=$(BRANCH)
LDFLAGS      += -X $(VERSION_PKG).BuildUser=$(USER)
LDFLAGS      += -X $(VERSION_PKG).BuildDate=$(BUILD_DATE)

.PHONY: all build test test-race lint fmt vet clean generate download-spec help
.PHONY: release-snapshot docker run

all: lint test build ## Run lint, test, and build

##@ Build

build: ## Build the binary
	CGO_ENABLED=0 $(GO) build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINARY) .

run: build ## Build and run locally (requires PULUMI_ACCESS_TOKEN)
	./$(BINARY) $(ARGS)

clean: ## Remove build artifacts
	rm -f $(BINARY)
	rm -rf dist/

##@ Test

test: ## Run tests
	$(GO) test ./...

test-race: ## Run tests with race detector
	$(GO) test -race ./...

test-cover: ## Run tests with coverage report
	$(GO) test -coverprofile=coverage.out ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

##@ Code Quality

lint: ## Run golangci-lint
	$(GOLANGCI) run

fmt: ## Format code with gofumpt
	$(GOLANGCI) fmt

vet: ## Run go vet
	$(GO) vet ./...

##@ Code Generation

generate: download-spec ## Generate Go client from Pulumi Cloud OpenAPI spec
	$(OAPI_CODEGEN) --config $(OAPI_CONFIG) pulumi-spec.json

download-spec: ## Download the latest Pulumi Cloud OpenAPI spec
	curl -sSfL $(OAPI_SPEC) -o pulumi-spec.json
	@echo "Downloaded OpenAPI spec to pulumi-spec.json"

##@ Release

release-snapshot: ## GoReleaser dry run (no publish)
	$(GORELEASER) release --snapshot --clean

docker: build ## Build Docker image locally
	docker build -t $(BINARY):local .

##@ Dependencies

deps: ## Download and tidy Go module dependencies
	$(GO) mod download
	$(GO) mod tidy

tools: ## Install development tools
	go install github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen@latest
	@echo "Installed oapi-codegen"

##@ Help

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-18s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(MAKEFILE_LIST)
