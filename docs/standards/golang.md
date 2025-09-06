# Go Standards for LogFlux Go SDK

## Code Organization

### Package Structure
- Single package `sdk` for the SDK functionality
- All source files in root directory (acceptable for single-purpose library)
- Test files use `_test.go` suffix
- Clear separation of concerns across files:
  - `types.go` - Core types and structures
  - `client.go` - Basic client implementation
  - `batch_client.go` - Batch processing client
  - `config.go` - Configuration management
  - `auth.go` - Authentication handling
  - `utils.go` - Utility functions

### File Organization
- **Sources**: All `.go` files in root
- **Tests**: All `*_test.go` files in root
- **Documentation**: All documentation in `/docs`
- **Build artifacts**: All build outputs in `/tmp`
- **Binaries**: All binaries in `/bin`

## Code Style

### Formatting
- **MUST** use `gofmt` for all Go files
- **MUST** use `go vet` for static analysis
- **SHOULD** use `golangci-lint` for comprehensive linting

### Naming Conventions
- **Types**: PascalCase for exported types (`LogEntry`, `Client`)
- **Functions**: PascalCase for exported functions (`NewClient`, `SendLogEntry`)
- **Constants**: PascalCase with descriptive prefix (`LevelInfo`, `TypeLog`)
- **Variables**: camelCase for local variables
- **Packages**: Short, descriptive, lowercase (`sdk`)

### Constants
- **MUST** define constants for magic numbers
- **MUST** group related constants together
- **SHOULD** include comments for non-obvious constants

Example:
```go
// Log levels following syslog standard
const (
    LevelEmergency = 1 // System is unusable
    LevelAlert     = 2 // Action must be taken immediately
    LevelCritical  = 3 // Critical conditions
    // ... etc
)
```

### Error Handling
- **MUST** wrap errors with context using `fmt.Errorf` with `%w` verb
- **MUST** handle all error returns
- **SHOULD** provide descriptive error messages
- **SHOULD** use error chains for debugging

Example:
```go
if err := client.Connect(ctx); err != nil {
    return fmt.Errorf("failed to connect to agent: %w", err)
}
```

### Documentation
- **MUST** document all exported functions and types
- **SHOULD** include usage examples in comments
- **MUST** use proper godoc format

Example:
```go
// NewLogEntry creates a new log entry with default values.
// The entry will have the current timestamp, Info level, and Log type.
func NewLogEntry(message string) LogEntry {
    // ...
}
```

## Testing Standards

### Test Structure
- Test files must use `_test.go` suffix
- Test functions must start with `Test`
- Benchmark functions must start with `Benchmark`
- Use table-driven tests where appropriate
- Include edge cases and error scenarios

### Test Coverage
- **MUST** have tests for all exported functions
- **SHOULD** achieve >80% test coverage
- **MUST** test error conditions
- **SHOULD** include integration tests

### Test Organization
```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        // test cases
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // test implementation
        })
    }
}
```

## Performance Standards

### Memory Management
- **MUST** close resources properly (connections, files)
- **SHOULD** reuse objects where possible
- **SHOULD** use sync.Pool for frequently allocated objects
- **MUST** handle goroutine lifecycles properly

### Concurrency
- **MUST** use context.Context for cancellation
- **MUST** protect shared state with mutexes
- **SHOULD** use channels for communication between goroutines
- **MUST** handle race conditions properly

## Build Standards

### Go Modules
- **MUST** use Go modules (`go.mod`)
- **SHOULD** use semantic versioning
- **MUST** specify minimum Go version
- **SHOULD** keep dependencies minimal

### Build Configuration
- **MUST** use Makefile for build tasks
- **SHOULD** support Docker builds
- **MUST** include linting and testing in build process
- **SHOULD** generate coverage reports

### Dependencies
- **MUST** justify all external dependencies
- **SHOULD** prefer standard library when possible
- **MUST** keep dependency tree shallow
- **SHOULD** regularly update dependencies for security

## Security Standards

### Authentication
- **MUST** use secure communication protocols
- **SHOULD** implement proper authentication mechanisms  
- **MUST** never log secrets or credentials
- **SHOULD** use environment variables for sensitive configuration

### Input Validation
- **MUST** validate all external inputs
- **SHOULD** sanitize log messages
- **MUST** handle malformed data gracefully
- **SHOULD** implement rate limiting where appropriate

## API Design Standards

### Function Signatures
- **MUST** use context.Context as first parameter for operations that can be cancelled
- **SHOULD** return errors as the last return value
- **SHOULD** use functional options for complex configuration
- **MUST** maintain backwards compatibility

### Builder Pattern
- **SHOULD** use builder pattern for complex object construction
- **MUST** return copies, not modify receivers (for value types)
- **SHOULD** chain method calls for fluent API

Example:
```go
entry := sdk.NewLogEntry("message").
    WithLevel(sdk.LevelError).
    WithSource("my-app").
    WithLabel("key", "value")
```

## Version Control Standards

### Commit Messages
- Use conventional commits format
- Include scope when appropriate
- Keep first line under 50 characters
- Include detailed description when needed

### Branching
- Use feature branches for development
- Keep commits atomic and focused
- Include tests with feature commits
- Squash commits before merge when appropriate

## References

- [Official LogFlux Documentation](https://docs.logflux.io)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)

## License

This project is licensed under the Apache License 2.0. See [../../LICENSE-APACHE-2.0](../../LICENSE-APACHE-2.0) for details.