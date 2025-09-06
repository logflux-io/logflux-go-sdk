# LogFlux Agent Go SDK

The LogFlux Agent Go SDK provides a lightweight client library for communicating with the LogFlux agent's local server via Unix socket or TCP protocols.

For complete platform documentation, visit [docs.logflux.io](https://docs.logflux.io).

## Features

- **Multiple transport protocols**: Unix socket (default), TCP
- **Async mode**: Non-blocking sends with configurable buffer (default enabled)
- **Circuit breaker**: Automatic failure detection and recovery
- **Exponential backoff**: Intelligent retry with jitter for resilience
- **Automatic batching**: Configurable batch sizes and flush intervals
- **Simple configuration**: Basic configuration types for connection settings  
- **Authentication support**: TCP shared secret authentication (Unix sockets use filesystem permissions)
- **Type safety**: Strongly typed log entries and configuration

## Quick Start

### Basic Usage

```go
package main

import (
    "context"
    "log"
    
    "github.com/logflux-io/logflux-go-sdk/pkg/client"
    "github.com/logflux-io/logflux-go-sdk/pkg/types"
)

func main() {
    // Create a simple client - async mode enabled by default for non-blocking sends
    c := client.NewUnixClient("/tmp/logflux-agent.sock")
    
    ctx := context.Background()
    if err := c.Connect(ctx); err != nil {
        log.Fatalf("Failed to connect: %v", err)
    }
    defer c.Close()
    
    // Send a log entry (non-blocking with circuit breaker protection)
    entry := types.NewLogEntry("Hello, LogFlux!", "my-app").
        WithLogLevel(types.LevelInfo)
    
    if err := c.SendLogEntry(entry); err != nil {
        log.Fatalf("Failed to send log: %v", err)
    }
}
```

### Batch Client

```go
// Create a batch client for high-throughput logging
batchConfig := config.DefaultBatchConfig()
batchConfig.MaxBatchSize = 100
batchConfig.FlushInterval = 5 * time.Second

batchClient := client.NewBatchUnixClient("/tmp/logflux-agent.sock", batchConfig)

ctx := context.Background()
if err := batchClient.Connect(ctx); err != nil {
    log.Fatalf("Failed to connect: %v", err)
}
defer batchClient.Close()

// Send multiple entries - they'll be automatically batched
for i := 0; i < 1000; i++ {
    entry := types.NewLogEntry(fmt.Sprintf("Log message %d", i), "batch-app")
    batchClient.SendLogEntry(entry)
}

// Flush any remaining entries
batchClient.Flush()
```

### Custom Configuration

The SDK provides basic configuration types. Users manage their own config files in any format:

```go
// Your application config (JSON, YAML, TOML, etc.)
type AppConfig struct {
    LogFlux LogFluxConfig `json:"logflux"`
    // ... other app settings
}

type LogFluxConfig struct {
    SocketPath   string `json:"socket_path"`
    Network      string `json:"network"`
    SharedSecret string `json:"shared_secret,omitempty"`
    MaxRetries   int    `json:"max_retries"`
    TimeoutMs    int    `json:"timeout_ms"`
}

// Convert to SDK config
func (c *LogFluxConfig) ToSDKConfig() *config.Config {
    cfg := config.DefaultConfig()
    if c.SocketPath != "" {
        cfg.Address = c.SocketPath
    }
    if c.Network != "" {
        cfg.Network = c.Network
    }
    // ... map other fields
    return cfg
}

// Use with your preferred config format
var appConfig AppConfig
json.Unmarshal(configData, &appConfig)  // or yaml.Unmarshal, etc.
sdkConfig := appConfig.LogFlux.ToSDKConfig()
c := client.NewClient(sdkConfig)
```

Example JSON configuration:

```json
{
  "logflux": {
    "socket_path": "/tmp/logflux-agent.sock",
    "network": "unix",
    "max_retries": 3,
    "timeout_ms": 10000,
    "batch_size": 10,
    "flush_interval_ms": 5000
  }
}
```

## Transport Protocols

### Unix Socket (Recommended)
```go
c := client.NewUnixClient("/tmp/logflux-agent.sock")
```

### TCP
```go
c := client.NewTCPClient("localhost", 9999)
```

## Log Levels

The SDK supports standard syslog levels:

- `types.LevelEmergency` (1) - Emergency
- `types.LevelAlert` (2) - Alert  
- `types.LevelCritical` (3) - Critical
- `types.LevelError` (4) - Error
- `types.LevelWarning` (5) - Warning
- `types.LevelNotice` (6) - Notice
- `types.LevelInfo` (7) - Info
- `types.LevelDebug` (8) - Debug

## Entry Types

Currently, the minimal SDK only supports:
- `types.TypeLog` (1) - Standard log messages

Additional entry types (metrics, traces, events, audit) are planned for future releases.

## Payload Types

The minimal SDK currently supports basic payload type hints:

- `types.PayloadTypeGeneric` - Generic text logs (default)
- `types.PayloadTypeGenericJSON` - Generic JSON data (auto-detected)

Additional payload types for systemd, syslog, metrics, applications, and containers are planned for future releases.

### Payload Type Examples

```go
// Automatic payload type detection
entry := types.NewLogEntry(`{"level": "info", "message": "JSON log"}`, "app") // Auto-detects JSON

// Manual payload type assignment
entry := types.NewLogEntry("Custom log message", "app").WithPayloadType(types.PayloadTypeGeneric)
jsonEntry := types.NewLogEntry(`{"key": "value"}`, "app").WithPayloadType(types.PayloadTypeGenericJSON)
```

### JSON Detection

The SDK automatically detects JSON content and sets the appropriate payload type:

```go
// Automatically detected as PayloadTypeGenericJSON
entry := types.NewLogEntry(`{"user": "admin", "action": "login", "success": true}`, "auth")

// Check if content is JSON
if types.IsValidJSON(content) {
    // Handle as JSON
}

// Auto-detect payload type
payloadType := types.AutoDetectPayloadType(message)
```

## SDK Configuration

The SDK focuses on connection and protocol configuration only:

```go
// Basic config
cfg := config.DefaultConfig()
cfg.Network = "tcp"
cfg.Address = "localhost:8080"
cfg.SharedSecret = "secret"
cfg.MaxRetries = 5
cfg.AsyncMode = true  // Non-blocking sends (default)
cfg.CircuitBreakerThreshold = 5  // Failures before opening circuit

// Batch config  
batchCfg := config.DefaultBatchConfig()
batchCfg.MaxBatchSize = 50
batchCfg.FlushInterval = 10 * time.Second
```

Users handle application-level configuration management using their preferred format and libraries.

## Security

### Unix Socket Security
Unix sockets provide security through filesystem permissions. Only processes with appropriate file system access can connect to the agent socket.

### TCP Authentication  
TCP connections require explicit shared secret authentication:

```go
cfg := config.DefaultConfig()
cfg.Network = "tcp"
cfg.Address = "localhost:8080" 
cfg.SharedSecret = "your-shared-secret"

client := client.NewClient(cfg)
if err := client.Connect(ctx); err != nil {
    log.Fatal(err)
}

// Authenticate for TCP connections
if _, err := client.Authenticate(); err != nil {
    log.Fatal("Authentication failed:", err)
}
```

## Best Practices

1. **Use Unix sockets** for local communication (fastest and most secure)
2. **Keep async mode enabled** for non-blocking operation (default)
3. **Monitor circuit breaker** status during high failure rates
4. **Enable batching** for high-throughput scenarios
5. **Manage your own configuration** using your preferred format (JSON, YAML, TOML, etc.)
6. **Set appropriate log levels** to avoid noise
7. **Handle connection errors** gracefully with exponential backoff
8. **Close clients** properly to avoid resource leaks

## Example Plugin

See the plugins in `/plugins/` directory for complete examples of how to build plugins using this SDK.

## License

This project is licensed under the Apache License 2.0. See [../LICENSE-APACHE-2.0](../LICENSE-APACHE-2.0) for details.

## Additional Resources

- [LogFlux Documentation](https://docs.logflux.io) - Complete platform documentation
- [API Reference](https://docs.logflux.io/api) - REST API documentation
- [SDK Guide](https://docs.logflux.io/sdks/go) - Official Go SDK guide
- [Agent Configuration](https://docs.logflux.io/agent) - Agent setup and configuration