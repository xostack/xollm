# xollm - XOStack LLM Abstractions for Go
# Makefile following XOStack standards

# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt
GOVET=$(GOCMD) vet

# Project parameters
BINARY_NAME=xollm
BINARY_PATH=./cmd/$(BINARY_NAME)
COVERAGE_FILE=coverage.out
COVERAGE_HTML=coverage.html

# Build tags and flags
BUILD_FLAGS=-v
TEST_FLAGS=-v -race
COVERAGE_FLAGS=-coverprofile=$(COVERAGE_FILE) -covermode=atomic

.PHONY: all build clean test deps lint vet fmt coverage help install installuser run

# Default target
all: deps fmt vet lint test build

# Build the library (currently no main package, so this validates compilation)
build:
	@echo "Building xollm library..."
	$(GOBUILD) $(BUILD_FLAGS) ./...

# Run the application (placeholder - library doesn't have main)
run: build
	@echo "xollm is a library - no main package to run"
	@echo "Use 'make test' to verify functionality"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	$(GOCLEAN)
	rm -f $(COVERAGE_FILE) $(COVERAGE_HTML)

# Remove everything and return to pristine state
distclean: clean
	@echo "Removing all generated files..."
	$(GOMOD) clean -cache

# Run tests
test:
	@echo "Running tests..."
	$(GOTEST) $(TEST_FLAGS) ./...

# Run tests with coverage
coverage:
	@echo "Running tests with coverage..."
	$(GOTEST) $(TEST_FLAGS) $(COVERAGE_FLAGS) ./...
	$(GOCMD) tool cover -html=$(COVERAGE_FILE) -o $(COVERAGE_HTML)
	@echo "Coverage report generated: $(COVERAGE_HTML)"

# Check test coverage percentage
check-coverage: coverage
	@echo "Coverage summary:"
	$(GOCMD) tool cover -func=$(COVERAGE_FILE) | grep total | awk '{print "Total coverage: " $$3}'

# Manage dependencies
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Lint code
lint:
	@echo "Running golangci-lint..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed. Install with: curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b \$$(go env GOPATH)/bin"; \
	fi

# Run go vet
vet:
	@echo "Running go vet..."
	$(GOVET) ./...

# Format code
fmt:
	@echo "Formatting code..."
	$(GOFMT) ./...

# Install for system use (library doesn't install binaries)
install:
	@echo "xollm is a library - use 'go get github.com/xostack/xollm' to install"

# Install for current user (library doesn't install binaries)
installuser: install

# Development helpers
dev-setup:
	@echo "Setting up development environment..."
	@if ! command -v golangci-lint >/dev/null 2>&1; then \
		echo "Installing golangci-lint..."; \
		curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin; \
	fi
	@echo "Development environment ready!"

# Validate all code without running tests
validate: deps fmt vet lint
	@echo "Code validation complete"

# Generate mocks for testing
generate:
	@echo "Generating mocks and code..."
	$(GOCMD) generate ./...

# Security scan
security:
	@echo "Running security scan..."
	@if command -v gosec >/dev/null 2>&1; then \
		gosec ./...; \
	else \
		echo "gosec not installed. Install with: go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest"; \
	fi

# Display help
help:
	@echo "Available targets:"
	@echo "  all         - Run deps, fmt, vet, lint, test, and build"
	@echo "  build       - Compile the library (validation)"
	@echo "  test        - Run all tests"
	@echo "  coverage    - Run tests with coverage report"
	@echo "  check-coverage - Show coverage percentage"
	@echo "  deps        - Download and tidy dependencies"
	@echo "  lint        - Run golangci-lint"
	@echo "  vet         - Run go vet"
	@echo "  fmt         - Format code with go fmt"
	@echo "  clean       - Remove build artifacts"
	@echo "  distclean   - Remove all generated files"
	@echo "  dev-setup   - Set up development environment"
	@echo "  validate    - Run all validation without tests"
	@echo "  generate    - Run go generate"
	@echo "  security    - Run security scan with gosec"
	@echo "  help        - Show this help message"
	@echo ""
	@echo "For library usage, import: github.com/xostack/xollm"
