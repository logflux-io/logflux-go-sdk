# LogFlux Go SDK Integrations

Minimal integrations with popular Go logging libraries.

## Supported Libraries

### log (Standard Library)
**Go's built-in log package**

```go
import log_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/log"

// Create writer and set as log output
writer := log_integration.NewWriter(batchClient, "my-app")
log.SetOutput(writer.MultiWriter(os.Stdout)) // Send to both LogFlux and stdout

// Use log normally
log.Println("Application started")
```

### slog (Go 1.21+)
**Standard library structured logging**

```go
import slog_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/slog"

// Create handler
handler := slog_integration.NewHandler(batchClient, "my-app")
logger := slog.New(handler)

// Use slog normally
logger.Info("User logged in", "user_id", 123)
```

### Logrus
**Popular structured logging library**

```go
import logrus_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/logrus"

// Create and add hook
hook := logrus_integration.NewHook(batchClient, "my-app")
logrus.AddHook(hook)

// Use logrus normally
logrus.WithField("user_id", 123).Info("User logged in")
```

### Zap
**High-performance structured logging**

```go
import zap_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/zap"

// Create core and logger
core := zap_integration.NewCore(batchClient, "my-app", zapcore.InfoLevel)
logger := zap.New(core)

// Use zap normally
logger.Info("User logged in", zap.Int("user_id", 123))
```

### Zerolog
**Fast zero-allocation JSON logging**

```go
import zerolog_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/zerolog"

// Create writer and logger
writer := zerolog_integration.NewWriter(batchClient, "my-app")
logger := zerolog.New(writer.MultiWriter(os.Stdout)).With().Timestamp().Logger()

// Use zerolog normally
logger.Info().Int("user_id", 123).Msg("User logged in")
```

## Usage Pattern

1. **Create LogFlux batch client** for better performance
2. **Create integration** (handler/hook) with your source name
3. **Configure your logging library** to use the integration
4. **Log normally** - entries automatically sent to LogFlux

## Examples

See `examples/integrations/` for complete working examples.

## Benefits

- **Non-intrusive** - Keep your existing logging code
- **Structured data** - Metadata automatically extracted
- **Batched delivery** - Efficient transport to LogFlux agent
- **Standard patterns** - Uses each library's native extension points