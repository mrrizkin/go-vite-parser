# Go Vite Parser Makefile

# Variables
BINARY_NAME=go-vite-parser
MAIN_PACKAGE=.
GO_FILES=$(shell find . -name "*.go" -type f -not -path "./vendor/*")
TEST_PACKAGES=$(shell go list ./... | grep -v /vendor/)

# Default target
.DEFAULT_GOAL := help

# Build the project
.PHONY: build
build: ## Build the project
	@echo "Building $(BINARY_NAME)..."
	go build -o bin/$(BINARY_NAME) $(MAIN_PACKAGE)

# Clean build artifacts
.PHONY: clean
clean: ## Clean build artifacts
	@echo "Cleaning..."
	rm -rf bin/
	go clean
	go clean -testcache

# Run tests
.PHONY: test
test: ## Run all tests
	@echo "Running tests..."
	go test -v ./...

# Run tests with coverage
.PHONY: test-coverage
test-coverage: ## Run tests with coverage report
	@echo "Running tests with coverage..."
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Run tests with race detection
.PHONY: test-race
test-race: ## Run tests with race detection
	@echo "Running tests with race detection..."
	go test -race -v ./...

# Run benchmarks
.PHONY: bench
bench: ## Run benchmarks
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Run benchmarks and save to file
.PHONY: bench-save
bench-save: ## Run benchmarks and save results to file
	@echo "Running benchmarks and saving to bench.txt..."
	go test -bench=. -benchmem ./... > bench.txt

# Format code
.PHONY: fmt
fmt: ## Format Go code
	@echo "Formatting code..."
	go fmt ./...

# Vet code
.PHONY: vet
vet: ## Vet Go code
	@echo "Vetting code..."
	go vet ./...

# Run golint (requires golint to be installed)
.PHONY: lint
lint: ## Run golint
	@echo "Running golint..."
	@if command -v golint >/dev/null 2>&1; then \
		golint ./...; \
	else \
		echo "golint not installed. Install with: go install golang.org/x/lint/golint@latest"; \
	fi

# Run staticcheck (requires staticcheck to be installed)
.PHONY: staticcheck
staticcheck: ## Run staticcheck
	@echo "Running staticcheck..."
	@if command -v staticcheck >/dev/null 2>&1; then \
		staticcheck ./...; \
	else \
		echo "staticcheck not installed. Install with: go install honnef.co/go/tools/cmd/staticcheck@latest"; \
	fi

# Run all quality checks
.PHONY: check
check: fmt vet lint staticcheck ## Run all code quality checks

# Install dependencies
.PHONY: deps
deps: ## Download and tidy dependencies
	@echo "Installing dependencies..."
	go mod download
	go mod tidy

# Update dependencies
.PHONY: deps-update
deps-update: ## Update dependencies
	@echo "Updating dependencies..."
	go get -u ./...
	go mod tidy

# Run tests in watch mode (requires entr to be installed)
.PHONY: test-watch
test-watch: ## Run tests in watch mode
	@echo "Running tests in watch mode (requires 'entr' to be installed)..."
	@if command -v entr >/dev/null 2>&1; then \
		find . -name "*.go" | entr -c go test -v ./...; \
	else \
		echo "entr not installed. Install with your package manager (e.g., apt install entr, brew install entr)"; \
	fi

# Generate documentation
.PHONY: docs
docs: ## Generate documentation
	@echo "Generating documentation..."
	godoc -http=:6060 &
	@echo "Documentation server started at http://localhost:6060"

# Run all tests and checks
.PHONY: ci
ci: deps check test test-race ## Run CI pipeline (deps, check, test, test-race)

# Show project statistics
.PHONY: stats
stats: ## Show project statistics
	@echo "Project Statistics:"
	@echo "=================="
	@echo "Go files: $(shell find . -name "*.go" -type f -not -path "./vendor/*" | wc -l)"
	@echo "Lines of code: $(shell find . -name "*.go" -type f -not -path "./vendor/*" -exec cat {} \; | wc -l)"
	@echo "Test files: $(shell find . -name "*_test.go" -type f | wc -l)"
	@echo "Packages: $(shell go list ./... | wc -l)"

# Install development tools
.PHONY: install-tools
install-tools: ## Install development tools
	@echo "Installing development tools..."
	go install golang.org/x/lint/golint@latest
	go install honnef.co/go/tools/cmd/staticcheck@latest
	go install golang.org/x/tools/cmd/godoc@latest

# Help target
.PHONY: help
help: ## Show this help message
	@echo "Go Vite Parser - Available targets:"
	@echo "=================================="
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)
	@echo ""
	@echo "Examples:"
	@echo "  make test          # Run all tests"
	@echo "  make test-coverage # Run tests with coverage"
	@echo "  make bench         # Run benchmarks"
	@echo "  make ci            # Run full CI pipeline"
