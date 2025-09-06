# LogFlux Go SDK Makefile
#
# Simple native Go build targets for the LogFlux Go SDK

# Build directories
BUILD_DIR=tmp/build
BIN_DIR=bin
COVERAGE_DIR=tmp/coverage

.PHONY: all build test test-integration test-all clean fmt vet lint ci deps coverage bench help examples quality

# Default target
all: deps fmt vet test build

# Build the SDK packages
build:
	@echo "Building Go SDK packages..."
	go build -v ./pkg/...
	@echo "Build complete"

# Run unit tests
test:
	@echo "Running Go SDK unit tests..."
	go test -v -race ./pkg/...
	@echo "Unit tests complete"

# Run integration tests (requires running LogFlux agent)
test-integration:
	@echo "Running integration tests..."
	@if [ ! -S "$${LOGFLUX_SOCKET:-/tmp/logflux-agent.sock}" ]; then \
		echo "ERROR: LogFlux agent not found at $${LOGFLUX_SOCKET:-/tmp/logflux-agent.sock}"; \
		echo "   Start agent or set LOGFLUX_SOCKET environment variable"; \
		exit 1; \
	fi
	@echo "Found LogFlux agent at $${LOGFLUX_SOCKET:-/tmp/logflux-agent.sock}"
	go test -tags=integration -v -race ./test/integration/
	@echo "Integration tests complete"

# Run all tests (unit + integration)
test-all: test test-integration

# Format code
fmt:
	@echo "Formatting Go code..."
	go fmt ./...
	@echo "Formatting complete"

# Run go vet
vet:
	@echo "Running go vet..."
	go vet ./pkg/...
	@echo "Vet complete"

# Lint code (if golangci-lint is available)
lint:
	@echo "Running linter..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not installed, skipping..."; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi
	@echo "Linting complete"

# CI target - runs all quality checks
ci: deps fmt vet lint test build
	@echo "CI checks complete"

# Quality target - runs formatting, linting, and static analysis
quality: fmt vet lint
	@echo "Code quality checks complete"

# Download and tidy dependencies
deps:
	@echo "Managing Go dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies updated"

# Generate test coverage report
coverage:
	@echo "Generating coverage report..."
	@mkdir -p $(COVERAGE_DIR)
	go test -v -race -coverprofile=$(COVERAGE_DIR)/coverage.out ./pkg/...
	go tool cover -html=$(COVERAGE_DIR)/coverage.out -o $(COVERAGE_DIR)/coverage.html
	@echo "Coverage report: $(COVERAGE_DIR)/coverage.html"

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./pkg/...
	@echo "Benchmarks complete"

# Build examples
examples:
	@echo "Building examples..."
	go build -v ./examples/...
	@echo "Examples built successfully"

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR) $(BIN_DIR) $(COVERAGE_DIR)
	rm -rf tmp/
	go clean -testcache -modcache -cache
	@echo "Build artifacts cleaned"

# Show help
help:
	@echo "LogFlux Go SDK Makefile"
	@echo "======================"
	@echo ""
	@echo "Available targets:"
	@echo "  all             - Run deps, fmt, vet, test and build"
	@echo "  build           - Build SDK packages"
	@echo "  test            - Run unit tests only"
	@echo "  test-integration- Run integration tests (requires agent)"
	@echo "  test-all        - Run unit + integration tests"
	@echo "  fmt             - Format all Go code"
	@echo "  vet             - Run go vet static analysis"
	@echo "  lint            - Run golangci-lint (if installed)"
	@echo "  ci              - Run full CI pipeline (deps, fmt, vet, lint, test, build)"
	@echo "  quality         - Run code quality checks (fmt, vet, lint)"
	@echo "  deps            - Download and tidy Go dependencies"
	@echo "  coverage        - Generate test coverage report"
	@echo "  bench           - Run performance benchmarks"
	@echo "  examples        - Build example applications"
	@echo "  clean           - Remove all build artifacts"
	@echo "  help            - Show this help message"
	@echo ""
	@echo "Usage examples:"
	@echo "  make                    # Full build pipeline (unit tests only)"
	@echo "  make test               # Run unit tests only" 
	@echo "  make test-integration   # Run integration tests (requires agent)"
	@echo "  make test-all           # Run all tests"
	@echo "  make coverage           # Generate coverage report"
	@echo "  make examples           # Build examples"
	@echo "  make clean              # Clean everything"
	@echo ""
	@echo "Requirements:"
	@echo "  - Go 1.21 or later"
	@echo "  - golangci-lint (optional, for linting)"
	@echo "  - LogFlux agent (for integration tests)"
	@echo ""
	@echo "Integration test setup:"
	@echo "  # Default agent socket"
	@echo "  make test-integration"
	@echo ""
	@echo "  # Custom agent socket"
	@echo "  LOGFLUX_SOCKET=/custom/path make test-integration"
	@echo ""
	@echo "Documentation:"
	@echo "  Complete testing guide: docs/testing.md"