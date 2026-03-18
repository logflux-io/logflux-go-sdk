# LogFlux Go SDK Makefile

.PHONY: test build clean fmt lint examples help

# Default target
all: test build

# Run tests
test:
	@echo "Running tests..."
	go test -v -race ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -html=coverage.out -o coverage.html

# Run benchmarks
bench:
	@echo "Running benchmarks..."
	go test -bench=. ./...

# Build examples
build:
	@echo "Building examples..."
	go build -o bin/basic-example ./examples/basic
	go build -o bin/web-server-example ./examples/web-server
	go build -o bin/discovery-example ./examples/discovery

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...

# Lint code
lint:
	@echo "Linting code..."
	golangci-lint run

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	rm -rf bin/
	rm -f coverage*.out coverage*.html

# Install dependencies
deps:
	@echo "Installing dependencies..."
	go mod tidy
	go mod download

# Run examples
run-basic: build
	@echo "Running basic example..."
	./bin/basic-example

run-web-server: build
	@echo "Running web server example..."
	./bin/web-server-example

# Publish to public GitHub repo (dry run)
publish-dry:
	@echo "Dry-run publish to public repo..."
	./scripts/publish.sh --dry-run

# Publish to public GitHub repo
publish:
	@echo "Publishing to public repo..."
	./scripts/publish.sh

# Publish with tag
publish-tag:
	@test -n "$(TAG)" || (echo "Usage: make publish-tag TAG=v1.2.3" && exit 1)
	@echo "Publishing to public repo with tag $(TAG)..."
	./scripts/publish.sh --tag $(TAG)

# Setup development environment
dev-setup: deps
	@echo "Setting up development environment..."
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Show help
help:
	@echo "LogFlux Go SDK Makefile"
	@echo ""
	@echo "Available targets:"
	@echo "  test            - Run all tests"
	@echo "  test-coverage   - Run tests with coverage"
	@echo "  bench           - Run benchmarks"
	@echo "  build           - Build examples"
	@echo "  fmt             - Format code"
	@echo "  lint            - Lint code"
	@echo "  clean           - Clean build artifacts"
	@echo "  deps            - Install dependencies"
	@echo "  run-basic       - Run basic example"
	@echo "  run-web-server  - Run web server example"
	@echo "  publish-dry     - Dry-run publish to public GitHub repo"
	@echo "  publish         - Publish to public GitHub repo (force-push)"
	@echo "  publish-tag     - Publish with tag (TAG=v1.2.3)"
	@echo "  dev-setup       - Setup development environment"
	@echo "  help            - Show this help"