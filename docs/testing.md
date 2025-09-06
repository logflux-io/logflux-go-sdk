# LogFlux Go SDK Testing Guide

This guide covers all aspects of testing the LogFlux Go SDK, from unit tests to integration tests and performance benchmarks.

## Overview

The SDK includes comprehensive testing at multiple levels:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test against a real LogFlux agent
- **Performance Tests**: Measure throughput and latency
- **Example Tests**: Verify example code works correctly

## Unit Testing

### Running Unit Tests

```bash
# Run all unit tests
make test
go test ./pkg/...

# Run tests with coverage
make test-coverage
go test -coverprofile=coverage.out ./pkg/...

# Run specific package tests
go test ./pkg/client/
go test ./pkg/types/
go test ./pkg/config/

# Run with verbose output
go test -v ./pkg/...

# Run specific test
go test ./pkg/client/ -run TestClientConnect
```

### Unit Test Coverage

Current unit test coverage includes:

**pkg/client/**
- Client creation and configuration
- Connection establishment and management
- Log entry and batch sending
- Retry logic and error handling
- Authentication for TCP connections

**pkg/types/**
- LogEntry creation and manipulation
- LogBatch operations
- JSON detection and payload type assignment
- Authentication request/response types
- Type marshaling/unmarshaling

**pkg/config/**
- Default configuration creation
- Configuration validation
- Batch configuration management

### Writing Unit Tests

Follow these patterns when adding unit tests:

```go
func TestNewClient(t *testing.T) {
    cfg := config.DefaultConfig()
    client := client.NewClient(cfg)
    
    if client == nil {
        t.Fatal("Expected non-nil client")
    }
    
    // Test configuration is set
    if client.config != cfg {
        t.Error("Client config not set correctly")
    }
}
```

## Integration Testing

Integration tests validate SDK functionality against a real LogFlux agent.

### Prerequisites

1. **LogFlux Agent Running**
   ```bash
   # Using Docker (recommended)
   docker run -d --name logflux-agent \
     -v /tmp:/tmp \
     logflux/agent:latest
   
   # Using local binary
   ./logflux-agent --socket=/tmp/logflux-agent.sock
   
   # Using systemd
   sudo systemctl start logflux-agent
   ```

2. **Verify Agent is Running**
   ```bash
   ls -la /tmp/logflux-agent.sock
   # Should show: srw-rw-rw- 1 user group 0 Jan 1 12:00 /tmp/logflux-agent.sock
   ```

### Running Integration Tests

```bash
# Run all integration tests
make test-integration

# Run with custom socket path
LOGFLUX_SOCKET=/custom/path/agent.sock make test-integration

# Direct go test command
go test -tags=integration -v ./test/integration/

# Run specific integration test
go test -tags=integration -v ./test/integration/ -run TestConnectivity

# Run performance tests only
go test -tags=integration -v ./test/integration/ -run TestPerformanceBaseline
```

### Integration Test Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LOGFLUX_SOCKET` | `/tmp/logflux-agent.sock` | Path to agent Unix socket |
| `LOGFLUX_TEST_TIMEOUT` | `5s` | Timeout for individual operations |
| `LOGFLUX_TEST_BATCH_SIZE` | `3` | Batch size for batch testing |

### Integration Test Coverage

**TestConnectivity**
- Basic connection establishment
- Ping/pong health check
- Connection cleanup

**TestLogTransmission**  
- Single log entry sending
- Different payload types (text, JSON)
- Metadata handling
- Error conditions

**TestBatchOperations**
- Batch client functionality
- Auto-batching behavior
- Manual flush operations
- Batch size limits

**TestAllLogLevels**
- All log levels (Emergency through Debug)
- Log level validation
- Level-specific handling

**TestPerformanceBaseline**
- Throughput measurement
- Latency analysis
- Performance regression detection

### Example Integration Test

```go
//go:build integration
// +build integration

func TestConnectivity(t *testing.T) {
    socket := getAgentSocket()
    
    if _, err := os.Stat(socket); os.IsNotExist(err) {
        t.Skipf("LogFlux agent socket not found at %s", socket)
    }
    
    client := client.NewUnixClient(socket)
    
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
    defer cancel()
    
    err := client.Connect(ctx)
    if err != nil {
        t.Fatalf("Failed to connect to agent: %v", err)
    }
    defer client.Close()
    
    // Test ping
    resp, err := client.Ping()
    if err != nil {
        t.Fatalf("Ping failed: %v", err)
    }
    
    if resp.Status != "pong" {
        t.Errorf("Expected pong response, got: %s", resp.Status)
    }
}
```

## Troubleshooting Integration Tests

### Common Issues

**Test Skipped - Socket Not Found**
```
LogFlux agent socket not found at /tmp/logflux-agent.sock - skipping integration test
```

**Solution:**
- Ensure LogFlux agent is running: `ps aux | grep logflux-agent`
- Check socket exists: `ls -la /tmp/logflux-agent.sock`
- Verify agent configuration and restart if needed

**Connection Refused**
```
Failed to connect to agent: dial unix /tmp/logflux-agent.sock: connect: connection refused
```

**Solution:**
- Agent may not be listening on the socket
- Check agent logs: `docker logs logflux-agent`
- Verify socket permissions
- Restart agent if necessary

**Permission Denied**
```
Failed to connect to agent: dial unix /tmp/logflux-agent.sock: connect: permission denied
```

**Solution:**
- Check socket permissions: `ls -la /tmp/logflux-agent.sock`
- Ensure test process has access to socket
- Fix permissions: `chmod 666 /tmp/logflux-agent.sock`

**Performance Tests Failing**
```
Performance below expectations: 5.2 msg/sec (expected >10)
Average latency too high: 150ms per message (expected <100ms)
```

**Solution:**
- System may be under heavy load
- Check agent resource usage: `docker stats logflux-agent`
- Adjust performance thresholds for your environment
- Consider running tests on dedicated hardware

### Agent Configuration for Testing

Minimal `logflux-agent.yml` for integration testing:

```yaml
server:
  socket_path: /tmp/logflux-agent.sock
  tcp_port: 9999  # Optional for TCP tests
  
logging:
  level: info
  format: json
  
output:
  type: console  # Outputs to stdout for testing
  
performance:
  batch_size: 100
  flush_interval: 1s
```

## Performance Testing

### Running Performance Tests

```bash
# Run performance baseline test
go test -tags=integration -v ./test/integration/ -run TestPerformanceBaseline

# Run with custom message count
LOGFLUX_PERF_MESSAGES=1000 go test -tags=integration ./test/integration/ -run TestPerformanceBaseline

# Benchmark tests (if available)
go test -bench=. ./pkg/...
```

### Performance Expectations

**Baseline Performance (local Unix socket):**
- Throughput: >10 messages/second (individual sends)
- Latency: <100ms average per message  
- Batch throughput: >100 messages/second
- Memory usage: <10MB for typical workloads

**Factors Affecting Performance:**
- System load and available resources
- Agent configuration and output destinations
- Message size and complexity
- Batch size and flush intervals

### Performance Tuning

**For High Throughput:**
```go
// Use batch client
batchConfig := config.DefaultBatchConfig()
batchConfig.MaxBatchSize = 100
batchConfig.FlushInterval = 100 * time.Millisecond
batchConfig.AutoFlush = true

batchClient := client.NewBatchClient(baseClient, batchConfig)
```

**For Low Latency:**
```go
// Use individual sends with small timeouts
cfg := config.DefaultConfig()
cfg.Timeout = 50 * time.Millisecond
cfg.MaxRetries = 1

client := client.NewClient(cfg)
```

## Test Data Management

### Test Socket Cleanup

```bash
# Clean up test sockets after testing
sudo rm -f /tmp/logflux-agent.sock
sudo rm -f /tmp/test-*.sock
```

### Docker Test Environment

```bash
# Start clean test environment
docker run -d --name logflux-test \
  -v /tmp:/tmp \
  -e LOGFLUX_SOCKET=/tmp/logflux-agent.sock \
  logflux/agent:latest

# Run tests
make test-integration

# Clean up
docker stop logflux-test
docker rm logflux-test
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Tests
on: [push, pull_request]

jobs:
  unit-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Run unit tests
      run: make test
    
    - name: Upload coverage
      uses: codecov/codecov-action@v3
      
  integration-tests:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - uses: actions/setup-go@v4
      with:
        go-version: '1.21'
    
    - name: Start LogFlux agent
      run: |
        docker run -d --name logflux-agent \
          -v /tmp:/tmp \
          logflux/agent:latest
        
        # Wait for agent to start
        timeout 30s bash -c 'until [[ -S /tmp/logflux-agent.sock ]]; do sleep 1; done'
    
    - name: Run integration tests
      run: make test-integration
      
    - name: Cleanup
      if: always()
      run: docker stop logflux-agent || true
```

## Test Organization

```
test/
├── integration/
│   ├── integration_test.go    # Main integration tests
│   ├── performance_test.go    # Performance benchmarks
│   └── README.md             # Integration test documentation
└── fixtures/
    ├── configs/              # Test configurations
    └── data/                # Test data files

pkg/*/
├── *_test.go                # Unit tests alongside source
└── testdata/                # Package-specific test data
```

## Best Practices

### Unit Testing
- Test public interfaces, not implementation details
- Use table-driven tests for multiple scenarios  
- Mock external dependencies
- Test error conditions thoroughly
- Maintain >80% code coverage

### Integration Testing  
- Always check if agent is available before running
- Use build tags to separate integration tests
- Clean up resources in defer statements
- Make tests independent and repeatable
- Include performance regression tests

### Test Data
- Use realistic but minimal test data
- Avoid hardcoded values where possible
- Generate test data programmatically
- Clean up test artifacts

### CI/CD
- Run unit tests on every commit
- Run integration tests on pull requests
- Monitor test performance over time
- Fail fast on test failures
- Generate and publish coverage reports