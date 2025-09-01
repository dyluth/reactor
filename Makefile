# Reactor Build System

# Build variables
BINARY_NAME := reactor
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE := $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.GitCommit=$(GIT_COMMIT) \
           -X main.BuildDate=$(BUILD_DATE)

# Build settings
GOOS := $(shell go env GOOS)
GOARCH := $(shell go env GOARCH)
BUILD_DIR := ./build
CMD_DIR := ./cmd/reactor

# Test isolation settings
TEST_PREFIX := test-$(shell date +%s)-$(shell echo $$RANDOM)

.PHONY: all build test test-unit test-integration test-isolated test-coverage test-coverage-isolated lint clean install help deps ci check docker-images docker-clean

# Default target - show help
all: help

## Build the reactor binary
build:
	@echo "Building reactor $(VERSION) for $(GOOS)/$(GOARCH)..."
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_DIR)
	@echo "Binary created at $(BUILD_DIR)/$(BINARY_NAME)"

## Run all tests (unit + integration)
test: test-unit test-integration

## Run unit tests only  
test-unit:
	go test -v ./pkg/config ./pkg/core ./pkg/docker ./pkg/testutil

## Run integration tests only
test-integration:
	@echo "Running integration tests..."
	go test -v ./pkg/integration

## Run tests with isolation (recommended for CI/development)
test-isolated: test-unit-isolated test-integration-isolated

## Run unit tests with isolation
test-unit-isolated:
	@echo "Running unit tests with isolation prefix: $(TEST_PREFIX)"
	REACTOR_ISOLATION_PREFIX=$(TEST_PREFIX) go test -v ./pkg/config ./pkg/core ./pkg/docker ./pkg/testutil

## Run integration tests with isolation
test-integration-isolated:
	@echo "Running integration tests with isolation prefix: $(TEST_PREFIX)"
	REACTOR_ISOLATION_PREFIX=$(TEST_PREFIX) go test -v ./pkg/integration

## Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./pkg/config ./pkg/core ./pkg/docker ./pkg/testutil ./pkg/integration
	go tool cover -html=coverage.out -o coverage.html

## Run tests with coverage and isolation (recommended for CI)
test-coverage-isolated:
	@echo "Running coverage tests with isolation prefix: $(TEST_PREFIX)"
	REACTOR_ISOLATION_PREFIX=$(TEST_PREFIX) go test -v -coverprofile=coverage.out ./pkg/config ./pkg/core ./pkg/docker ./pkg/testutil ./pkg/integration
	go tool cover -html=coverage.out -o coverage.html

## Comprehensive CI check - runs all validation needed for production confidence
ci: deps fmt lint test-coverage-isolated
	@echo "✅ All CI checks passed! Ready for production."

## Quick development check - faster validation during development  
check: fmt lint test-isolated
	@echo "✅ Development checks passed!"

## Run linting
lint:
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	fi
	golangci-lint run

## Format code
fmt:
	go fmt ./...
	go mod tidy

## Install dependencies
deps:
	go mod download
	go mod verify

## Install binary to local system
install: build
	@echo "Installing $(BINARY_NAME) to $(GOPATH)/bin/$(BINARY_NAME)"
	cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/$(BINARY_NAME)

## Clean build artifacts
clean:
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html

## Run the binary locally (for development)
run: build
	$(BUILD_DIR)/$(BINARY_NAME)

## Show build info without building
info:
	@echo "Version: $(VERSION)"
	@echo "Git Commit: $(GIT_COMMIT)"  
	@echo "Build Date: $(BUILD_DATE)"
	@echo "Target: $(GOOS)/$(GOARCH)"

## Cross-compile for multiple platforms
build-all:
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	
	# macOS ARM64
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(CMD_DIR)
	
	# macOS AMD64
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(CMD_DIR)
	
	# Linux ARM64
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 $(CMD_DIR)
	
	# Linux AMD64
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(CMD_DIR)
	
	@echo "Cross-compilation complete. Binaries in $(BUILD_DIR)/"

## Build Docker images for all language environments
docker-images:
	@echo "Building Docker images..."
	@if [ -f "./scripts/build-images.sh" ]; then \
		./scripts/build-images.sh; \
	else \
		echo "Warning: ./scripts/build-images.sh not found. Skipping Docker image build."; \
	fi

## Clean Docker images (removes locally built reactor images)
docker-clean:
	@echo "Cleaning up reactor Docker images..."
	@docker images --format "table {{.Repository}}\t{{.Tag}}\t{{.ID}}" | grep "reactor-" | awk '{print $$3}' | xargs -r docker rmi -f || true
	@echo "Docker cleanup complete."

## Show available make targets and usage examples
help:
	@echo "🚀 Reactor Build System"
	@echo ""
	@echo "USAGE:"
	@echo "  make <target>           Run a specific target"
	@echo "  make                    Show this help (default)"
	@echo ""
	@echo "KEY TARGETS:"
	@echo "  ci                      🎯 Full CI validation (deps + fmt + lint + test + coverage)"
	@echo "  check                   ⚡ Quick dev validation (fmt + lint + test)"
	@echo "  build                   🔨 Build reactor binary"
	@echo "  test-isolated           🧪 Run all tests with isolation (recommended)"
	@echo ""
	@echo "ALL TARGETS:"
	@grep -E '^##' $(MAKEFILE_LIST) | sed 's/^## /  /' | sort
	@echo ""
	@echo "EXAMPLES:"
	@echo "  make ci                 # Run full CI pipeline locally"  
	@echo "  make check              # Quick validation during development"
	@echo "  make build              # Build binary for current platform"
	@echo "  make build-all          # Cross-compile for all platforms"
	@echo "  make docker-images      # Build all Docker environment images"