# Contributing to LogFlux Go SDK

Thank you for your interest in contributing to the LogFlux Go SDK! This document provides guidelines and information for contributors.

## Development Setup

### Prerequisites

- **Go 1.21 or later** - Required for building and testing
- **golangci-lint** - Optional but recommended for enhanced linting
- **LogFlux Agent** - Required for integration tests
- **Docker** - Optional, for containerized development

### Getting Started

1. **Clone the repository**:
   ```bash
   git clone <repository-url>
   cd logflux-go-sdk
   ```

2. **Install dependencies**:
   ```bash
   make deps
   ```

3. **Run tests**:
   ```bash
   make test        # Unit tests only
   make test-all    # Unit + integration tests (requires agent)
   ```

4. **Build the project**:
   ```bash
   make build
   ```

## Development Workflow

### Making Changes

1. **Create a feature branch** from main:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the coding standards:
   - See [docs/standards/golang.md](docs/standards/golang.md) for Go-specific standards
   - See [docs/standards/project.md](docs/standards/project.md) for project organization

3. **Write tests** for your changes:
   - Unit tests for all new functions
   - Integration tests for agent interaction features
   - Ensure >80% test coverage

4. **Format and lint your code**:
   ```bash
   make fmt vet lint
   ```

5. **Run all tests**:
   ```bash
   make test-all    # Requires running LogFlux agent
   ```

### Testing

#### Unit Tests
Run unit tests with:
```bash
make test
go test -v -race ./pkg/...
```

#### Integration Tests
Integration tests require a running LogFlux agent:

```bash
# Start LogFlux agent first
docker run -d --name logflux-agent -v /tmp:/tmp logflux/agent:latest

# Run integration tests
make test-integration

# Or with custom socket path
LOGFLUX_SOCKET=/custom/path make test-integration
```

See [docs/testing.md](docs/testing.md) for complete testing documentation.

#### Coverage Reports
Generate coverage reports with:
```bash
make coverage
# Opens: tmp/coverage/coverage.html
```

### Code Quality Standards

#### Go Code Standards
- **MUST** pass `gofmt`, `go vet`, and `golangci-lint`
- **MUST** document all exported functions and types
- **MUST** handle errors properly with context
- **MUST** use `context.Context` for cancellable operations
- **SHOULD** achieve >80% test coverage

Example error handling:
```go
if err := client.Connect(ctx); err != nil {
    return fmt.Errorf("failed to connect to agent: %w", err)
}
```

#### Documentation
- **MUST** use proper godoc format for all exported items
- **SHOULD** include usage examples in comments
- **MUST** update documentation when changing APIs

#### Testing Standards
- **MUST** test all exported functions
- **MUST** test error conditions and edge cases
- **SHOULD** use table-driven tests where appropriate
- **MUST** include integration tests for agent interactions

## Project Structure

### Package Organization
```
pkg/
├── types/          # Core types (LogEntry, LogBatch, etc.)
├── client/         # Client implementations (Client, BatchClient)  
├── config/         # Configuration management
└── integrations/   # Logger integrations (logrus, slog, zap, etc.)
internal/
└── utils/          # Internal utilities
examples/           # Usage examples
test/
└── integration/    # Integration tests
docs/               # ALL documentation (markdown only)
├── standards/      # Coding and project standards
└── schemas/        # Database schemas (if applicable)
tmp/                # Temporary files, builds, logs
bin/                # Binary outputs only
```

### File Naming
- Source files: `*.go` in appropriate packages
- Test files: `*_test.go` alongside source files
- Documentation: `*.md` in `/docs` directory
- Build outputs: `/tmp` for temporary, `/bin` for binaries

## Build System

### Makefile Targets
The project uses a central Makefile with these key targets:

```bash
make all            # Full build pipeline (deps, fmt, vet, test, build)
make build          # Build SDK packages
make test           # Run unit tests only
make test-integration # Run integration tests (requires agent)
make test-all       # Run unit + integration tests
make fmt            # Format all Go code
make vet            # Run go vet static analysis
make lint           # Run golangci-lint
make deps           # Download and tidy dependencies
make coverage       # Generate test coverage report
make bench          # Run performance benchmarks
make examples       # Build example applications
make clean          # Remove build artifacts
make help           # Show help with all targets
```

### Before Committing
Always run the full build pipeline:
```bash
make all
```

This ensures your code:
- Is properly formatted (`gofmt`)
- Passes static analysis (`go vet`)
- Passes all unit tests
- Builds successfully

## Submitting Changes

### Pull Request Process

1. **Ensure your branch is up to date**:
   ```bash
   git checkout main
   git pull origin main
   git checkout your-feature-branch
   git rebase main
   ```

2. **Run the full build pipeline**:
   ```bash
   make all
   ```

3. **Run integration tests** (if applicable):
   ```bash
   make test-integration
   ```

4. **Push your branch** and create a pull request

5. **Write a clear PR description** including:
   - What changes were made
   - Why the changes were necessary
   - Any breaking changes
   - Testing performed

### Commit Message Format
Use conventional commits format:
```
type(scope): short description

Longer description if needed

Breaking changes and additional details
```

Examples:
- `feat(client): add retry mechanism for failed connections`
- `fix(batch): resolve race condition in batch processing`
- `docs: update integration test setup instructions`

## Integration Guidelines

### Logger Integrations
When adding new logger integrations:

1. **Create integration in** `pkg/integrations/<logger>/`
2. **Follow existing patterns** from other integrations
3. **Include comprehensive tests** with real logger instances
4. **Add usage example** in `examples/integrations/<logger>/`
5. **Update documentation** in relevant files

### New Features
For significant new features:

1. **Discuss the design** first (create an issue)
2. **Update relevant documentation** in `/docs`
3. **Include comprehensive tests**
4. **Add usage examples**
5. **Ensure backwards compatibility**

## Security Guidelines

- **NEVER** commit secrets or credentials
- **ALWAYS** validate external inputs
- **MUST** use secure communication protocols
- **SHOULD** implement proper authentication
- **NEVER** log sensitive information

## Getting Help

- **Documentation**: [docs/README.md](docs/README.md)
- **Testing Guide**: [docs/testing.md](docs/testing.md)
- **Standards**: [docs/standards/](docs/standards/)
- **Examples**: [examples/](examples/)
- **Issues**: Use the project's issue tracker

## License

By contributing to this project, you agree that your contributions will be licensed under the Apache License 2.0. See [LICENSE-APACHE-2.0](LICENSE-APACHE-2.0) for details.

## Code of Conduct

This project follows standard open source community guidelines. Be respectful, constructive, and professional in all interactions.