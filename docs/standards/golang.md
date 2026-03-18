# Go Standards and Conventions

## Language Version
- **Go Version**: 1.22+
- All code must compile with Go 1.22 or later

## Module Structure
- Primary module: `github.com/logflux-io/logflux-go-sdk/v3`
- Shared functionality for multiple modules should be placed in `/shared`
- Model/struct definitions shared across modules go in `/shared`
- Database migrations shared across modules go in `/shared`

## Directory Structure
- Binaries: `/bin` (root level only, no subdirectories)
- Temporary files: `/tmp` (root level)
- Logs: `/tmp/logs` for testing and debugging
- Documentation: `/docs` (all markdown documentation)
- Standards: `/docs/standards` (project conventions)
- Schemas: `/docs/schemas` (database schema documentation)

## Code Organization
- Package structure follows standard Go conventions
- Core components in `/pkg` directory
- Examples in `/examples` directory
- Tests in `tests/` directory for integration tests
- Unit tests alongside source files (`*_test.go`)

## Build System
- Central `Makefile` in root directory only
- No Makefiles in subdirectories
- Standard targets: `test`, `build`, `clean`, `fmt`, `lint`, `deps`

## Dependencies
- Use `go mod` for dependency management
- Run `go mod tidy` to clean up dependencies
- Prefer standard library where possible
- External dependencies must be justified and documented

## Testing
- Unit tests for all packages
- Integration tests in `/tests` directory
- Benchmark tests for performance validation
- Coverage reporting with HTML output
- Test command: `make test`

## Code Quality
- Use `gofmt` for formatting
- Use `golangci-lint` for linting
- All code must pass lint checks before commit
- Run `make fmt` and `make lint` before submitting changes

## Deployment
- Target platform: Linux ARM64
- Development platform: macOS ARM64 (darwin-arm64)
- Cross-compilation: `GOOS=linux GOARCH=arm64 go build`
- No Docker usage unless absolutely necessary

## Database
- Use `golang-migrate` for all database migrations
- Schema definitions in `/docs/schemas` as markdown files
- Migration files follow golang-migrate naming conventions