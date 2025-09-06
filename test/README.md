# Test Directory

This directory contains specialized test suites that require external dependencies.

## Structure

```
test/
├── integration/          # Integration tests (require running LogFlux agent)
│   └── integration_test.go
└── README.md            # This file
```

## Quick Start

```bash
# Run integration tests (requires LogFlux agent)
make test-integration

# Custom agent socket
LOGFLUX_SOCKET=/custom/path make test-integration
```

## Full Documentation

For complete testing documentation, see [docs/testing.md](../docs/testing.md):
- Unit test guidelines
- Integration test setup
- CI/CD configuration
- Performance benchmarking
- Troubleshooting guide