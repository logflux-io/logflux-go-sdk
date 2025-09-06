# LogFlux Go SDK (BETA)

[![CI](https://github.com/logflux-io/logflux-go-sdk/actions/workflows/ci.yml/badge.svg)](https://github.com/logflux-io/logflux-go-sdk/actions/workflows/ci.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/logflux-io/logflux-go-sdk)](https://goreportcard.com/report/github.com/logflux-io/logflux-go-sdk)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE-APACHE-2.0)
[![Go Version](https://img.shields.io/github/go-mod/go-version/logflux-io/logflux-go-sdk)](go.mod)
[![GitHub issues](https://img.shields.io/github/issues/logflux-io/logflux-go-sdk)](https://github.com/logflux-io/logflux-go-sdk/issues)

> **BETA SOFTWARE**: This SDK is feature-complete for basic logging use cases but is marked as BETA while we gather community feedback and add additional features. The API is stable but may evolve based on user needs.

A lightweight Go SDK for communicating with the LogFlux agent.

## Current Status

- **Stable API** for core logging functionality
- **Production quality** code and testing  
- **Ready for evaluation** and non-critical use cases
- **Additional features** (metrics, traces, events) coming soon
- **Gathering feedback** for API refinements

## Documentation

- [**Official Documentation**](https://docs.logflux.io) - Complete LogFlux platform documentation
- [**API Documentation**](docs/README.md) - Complete SDK usage guide
- [**Go Standards**](docs/standards/golang.md) - Go coding standards and conventions
- [**Project Standards**](docs/standards/project.md) - Project organization and build standards

## Package Structure

```
pkg/
├── types/          # Core types (LogEntry, LogBatch, etc.)
├── client/         # Client implementations (Client, BatchClient)  
├── config/         # Configuration management
└── integrations/   # Logger integrations (logrus, slog, zap, etc.)
internal/
└── utils/          # Internal utilities
examples/           # Usage examples
├── basic/          # Basic client usage
├── batch/          # Batch client usage
├── config/         # Configuration examples
└── integrations/   # Integration examples
    ├── logrus/     # Logrus integration
    ├── slog/       # Structured logging (slog)
    ├── zap/        # Zap logger integration
    ├── zerolog/    # Zerolog integration
    └── log/        # Standard log package
```

## Quick Start

```go
import (
    "context"
    "github.com/logflux-io/logflux-go-sdk/pkg/client"
    "github.com/logflux-io/logflux-go-sdk/pkg/types"
)

// Create client - async mode enabled by default for non-blocking sends
c := client.NewUnixClient("/tmp/logflux-agent.sock")
ctx := context.Background()
c.Connect(ctx)
defer c.Close()

// Send log entry (non-blocking with circuit breaker protection)
entry := types.NewLogEntry("Hello, LogFlux!", "my-app").WithLogLevel(types.LevelInfo)
c.SendLogEntry(entry)
```

## BETA Limitations

While the core functionality is stable and production-ready, this BETA release has some limitations:

### Current Features
- **Core logging** - Send log entries via Unix socket or TCP
- **Async mode** - Non-blocking sends with configurable buffer (default enabled)
- **Circuit breaker** - Automatic failure detection and recovery
- **Exponential backoff** - Intelligent retry with jitter for resilience
- **Batch processing** - High-performance batching for throughput
- **Library integrations** - logrus, zap, zerolog, slog, standard log
- **Authentication** - TCP shared secret authentication

### Planned Features
- **Metrics support** - Structured metrics collection and sending
- **Trace support** - Distributed tracing integration  
- **Event support** - Application and system events
- **Audit support** - Audit log structured data
- **Advanced payload types** - Syslog, systemd, container formats

### Known Limitations
- Integration tests require manual LogFlux agent setup
- Performance thresholds may need environment-specific tuning
- API may evolve based on community feedback

## Feedback Welcome

This is BETA software - we'd love your feedback! Please [open an issue](https://github.com/logflux-io/logflux-go-sdk/issues) for:
- Bug reports
- Feature requests  
- Documentation improvements
- API design suggestions

## Build

```bash
make all    # Build and test everything
make test   # Run tests
make fmt    # Format code
```

See the [Makefile](Makefile) for all available targets.

## Testing

### Unit Tests
```bash
make test                    # Run unit tests only
go test ./pkg/...           # Direct go test
```

### Integration Tests
Integration tests validate SDK functionality against a real LogFlux agent:

```bash
# Start LogFlux agent first
make test-integration        # Run integration tests
make test-all               # Run unit + integration tests

# Direct go test
go test -tags=integration -v ./test/integration/

# Custom agent socket
LOGFLUX_SOCKET=/custom/path make test-integration
```

**Complete testing guide**: [docs/testing.md](docs/testing.md)

### Integration Test Requirements

Before running integration tests, ensure you have:

1. **LogFlux Agent Running** - Integration tests require a live agent
2. **Socket Access** - Default: `/tmp/logflux-agent.sock` (set `LOGFLUX_SOCKET` for custom paths)  
3. **Proper Permissions** - Ensure the test process can access the agent socket

```bash
# Start LogFlux agent first
docker run -d --name logflux-agent -v /tmp:/tmp logflux/agent:latest

# Then run integration tests
make test-integration

# Or run specific tests
go test -tags=integration -v ./test/integration/ -run TestConnectivity
```

See [docs/testing.md](docs/testing.md) for complete integration test setup and troubleshooting.

## Requirements

- Go 1.21 or later
- golangci-lint (optional, for enhanced linting)
- LogFlux agent (for integration tests)

## License

This project is licensed under the Apache License 2.0. See [LICENSE-APACHE-2.0](LICENSE-APACHE-2.0) for details.

## Support

- [Documentation](https://docs.logflux.io)
- [API Reference](https://docs.logflux.io/api)
- [Issue Tracker](https://github.com/logflux-io/logflux-go-sdk/issues)
