# LogFlux Go SDK

The official Go SDK for [LogFlux.io](https://logflux.io) -- secure, zero-knowledge log ingestion with end-to-end encryption.

## Features

- **End-to-end encryption** -- AES-256-GCM with RSA key exchange. Server never sees plaintext.
- **7 entry types** -- Log, Metric, Trace, Event, Audit, Telemetry, TelemetryManaged
- **Async by default** -- Non-blocking queue with background workers
- **Automatic breadcrumbs** -- Trail of recent events attached to error captures
- **Distributed tracing** -- Span helpers with context propagation
- **Framework middleware** -- Gin, Echo, Fiber, Chi integrations
- **Logger adapters** -- Logrus, Zap, Zerolog, stdlib drop-in hooks
- **Failsafe** -- SDK errors never crash your application

## Installation

```bash
go get github.com/logflux-io/logflux-go-sdk/v3
```

Framework middleware (separate modules):
```bash
go get github.com/logflux-io/logflux-go-sdk/v3/gin
go get github.com/logflux-io/logflux-go-sdk/v3/echo
go get github.com/logflux-io/logflux-go-sdk/v3/fiber
go get github.com/logflux-io/logflux-go-sdk/v3/chi
```

## Quick Start

```go
package main

import (
    "log"
    "time"

    logflux "github.com/logflux-io/logflux-go-sdk/v3"
)

func main() {
    err := logflux.Init(logflux.Options{
        APIKey:      "eu-lf_your_api_key",
        Source:      "my-service",
        Environment: "production",
        Release:     "v1.2.3",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer logflux.Close()
    defer logflux.Flush(2 * time.Second)

    logflux.Info("server started")
    logflux.Warnf("high latency: %dms", 500)
}
```

## Entry Types

### Log (Type 1)

Standard application logs with 8 severity levels.

```go
logflux.Debug("cache miss for key users:123")
logflux.Info("request processed")
logflux.Warn("deprecated API called")
logflux.Error("database connection failed")
logflux.Critical("out of memory")
logflux.Fatal("unrecoverable state")

// With attributes
logflux.Log(logflux.LogLevelError, "query timeout", logflux.Fields{
    "db.host":     "primary.db.internal",
    "duration_ms": "5023",
})

// Printf-style
logflux.Infof("user %s logged in from %s", userID, ipAddr)
logflux.Errorf("failed after %d retries: %v", retries, err)
```

### Metric (Type 2)

Counters, gauges, and distributions.

```go
logflux.Counter("http.requests.total", 1, logflux.Fields{
    "method": "GET",
    "status": "200",
})

logflux.Gauge("system.cpu.usage", 85.2, logflux.Fields{
    "host": "web-01",
})

logflux.Metric("http.request.duration", 142.5, "distribution", logflux.Fields{
    "route": "/api/users",
})
```

### Event (Type 4)

Discrete application events.

```go
logflux.Event("user.signup", logflux.Fields{
    "user_id": "usr_987",
    "plan":    "starter",
    "method":  "oauth_google",
})

logflux.Event("deploy.completed", logflux.Fields{
    "version":     "v2.1.0",
    "duration_ms": "45000",
})
```

### Audit (Type 5)

Immutable audit trail with Object Lock storage (365-day retention).

```go
logflux.Audit("record.deleted", "usr_456", "invoice", "inv_789", logflux.Fields{
    "reason":     "customer_request",
    "ip":         "10.0.1.42",
    "user_agent": "Mozilla/5.0...",
})
```

### Telemetry (Types 6 and 7)

Device/sensor data. Use the `payload` package for structured telemetry:

```go
import "github.com/logflux-io/logflux-go-sdk/v3/pkg/payload"

p := payload.NewTelemetry("edge-gateway", "dev_001", []payload.Reading{
    {Name: "cpu_temp", Value: 72.5, Unit: "celsius"},
    {Name: "memory_used", Value: 85.2, Unit: "percent"},
})
```

## Error Capture

Capture Go errors with automatic stack traces and breadcrumbs.

```go
if err := db.Query(ctx, sql); err != nil {
    // Auto stack trace + error type + breadcrumb trail
    logflux.CaptureError(err)

    // With extra context
    logflux.CaptureErrorWithAttrs(err, logflux.Fields{
        "sql":     sql,
        "db.host": "primary",
    })

    // Custom message (original error in attributes)
    logflux.CaptureErrorWithMessage(err, "payment processing failed", logflux.Fields{
        "order_id": orderID,
    })
}
```

Error chain unwrapping: if the error was wrapped with `fmt.Errorf("...: %w", err)`, the full chain is captured up to 10 levels deep.

## Breadcrumbs

Breadcrumbs record a trail of events leading up to an error. They are automatically added for log and event calls, and attached to `CaptureError`.

```go
// Automatic: log calls (level <= Info) and events add breadcrumbs
logflux.Info("loading config")         // auto breadcrumb
logflux.Event("user.login", nil)       // auto breadcrumb

// Manual
logflux.AddBreadcrumb("http", "GET /api/users", logflux.Fields{
    "status": "200",
})

// When an error occurs, all breadcrumbs are attached
logflux.CaptureError(err) // includes breadcrumb trail

// Clear if needed
logflux.ClearBreadcrumbs()
```

Configure the buffer size:
```go
logflux.Init(logflux.Options{
    MaxBreadcrumbs: 200, // default: 100
})
```

## Scopes

Scopes provide per-request context isolation. Attributes and breadcrumbs set on a scope are merged into every entry sent through it.

```go
logflux.WithScope(func(scope *logflux.Scope) {
    scope.SetUser("usr_456")
    scope.SetRequest("GET", "/api/users", "req_abc123")
    scope.SetAttribute("tenant", "acme-corp")

    scope.Info("processing request")       // includes all scope attrs
    scope.AddBreadcrumb("db", "SELECT * FROM users", nil)

    if err != nil {
        scope.CaptureError(err)            // scope attrs + scope breadcrumbs
    }
})
```

## Distributed Tracing

### Spans

```go
// Root span (generates trace ID)
span := logflux.StartSpan("http.server", "GET /api/users")
defer span.End() // auto-computes duration, sends trace entry

span.SetAttribute("http.method", "GET")
span.SetAttribute("http.url", "/api/users")

// Child span (inherits trace ID)
dbSpan := span.StartChild("db.query", "SELECT * FROM users")
// ... query ...
dbSpan.End()

// Mark error
if err != nil {
    span.SetError(err) // sets status=error + error.message attribute
}
```

### Trace Context Propagation

Propagate trace context across services via HTTP headers.

```go
// Client side: inject trace header into outgoing request
logflux.InjectTraceContext(outgoingReq, span)

// Server side: continue trace from incoming request
span := logflux.ContinueFromRequest(req, "http.server", "GET /api")
defer span.End()
```

### Framework Middleware

Auto-creates spans for every HTTP request.

**Gin:**
```go
import logfluxgin "github.com/logflux-io/logflux-go-sdk/v3/gin"

r := gin.Default()
r.Use(logfluxgin.Middleware())
```

**Echo:**
```go
import logfluxecho "github.com/logflux-io/logflux-go-sdk/v3/echo"

e := echo.New()
e.Use(logfluxecho.Middleware())
```

**Fiber:**
```go
import logfluxfiber "github.com/logflux-io/logflux-go-sdk/v3/fiber"

app := fiber.New()
app.Use(logfluxfiber.Middleware())
```

**Chi:**
```go
import logfluxchi "github.com/logflux-io/logflux-go-sdk/v3/chi"

r := chi.NewRouter()
r.Use(logfluxchi.Middleware)
```

**stdlib net/http:**
```go
mux := http.NewServeMux()
handler := logflux.TracingMiddleware(mux)
http.ListenAndServe(":8080", handler)
```

All middleware: auto span creation, trace context propagation, panic recovery, HTTP attribute recording.

## Logger Adapters

Drop-in hooks for popular Go logging frameworks.

**Logrus:**
```go
import "github.com/logflux-io/logflux-go-sdk/v3/pkg/adapters"

logger := adapters.NewLogrusLogger(client)
logger.WithField("user_id", "123").Info("request processed")
```

**Zap:**
```go
logger := adapters.NewZapLogger(client)
logger.Info("request processed", adapters.String("user_id", "123"))
```

**Zerolog:**
```go
logger := adapters.NewZerologLogger(client)
logger.Info().Str("user_id", "123").Msg("request processed")
```

## Configuration

### Init Options

| Option | Type | Default | Description |
|--------|------|---------|-------------|
| `APIKey` | string | (required) | API key (`<region>-lf_<key>`) |
| `Source` | string | Node value | Service name, auto-attached to all payloads |
| `Environment` | string | | Attached to `meta.environment` |
| `Release` | string | | Attached to `meta.release` |
| `Node` | string | hostname | Host identifier |
| `QueueSize` | int | 1000 | In-memory buffer capacity |
| `FlushInterval` | Duration | 5s | Auto-flush interval |
| `BatchSize` | int | 100 | Entries per HTTP request |
| `WorkerCount` | int | 2 | Background goroutines |
| `MaxRetries` | int | 3 | Max retry attempts |
| `InitialDelay` | Duration | 1s | First retry delay |
| `MaxDelay` | Duration | 30s | Max retry delay |
| `BackoffFactor` | float64 | 2.0 | Exponential multiplier |
| `HTTPTimeout` | Duration | 30s | HTTP request timeout |
| `Failsafe` | bool | true | Never crash host app |
| `EnableCompression` | bool | true | Gzip before encryption |
| `SampleRate` | float64 | 1.0 | 0.0-1.0, send probability |
| `MaxBreadcrumbs` | int | 100 | Ring buffer size |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `LOGFLUX_API_KEY` | API key (required) |
| `LOGFLUX_ENVIRONMENT` | Deployment environment |
| `LOGFLUX_NODE` | Host identifier |
| `LOGFLUX_LOG_GROUP` | Default log group |
| `LOGFLUX_QUEUE_SIZE` | Queue capacity |
| `LOGFLUX_FLUSH_INTERVAL` | Flush interval (seconds) |
| `LOGFLUX_BATCH_SIZE` | Entries per request |
| `LOGFLUX_MAX_RETRIES` | Max retry attempts |
| `LOGFLUX_HTTP_TIMEOUT` | HTTP timeout (seconds) |
| `LOGFLUX_FAILSAFE_MODE` | Silent error handling |
| `LOGFLUX_WORKER_COUNT` | Background workers |
| `LOGFLUX_ENABLE_COMPRESSION` | Gzip compression |
| `LOGFLUX_DEBUG` | SDK diagnostic logging |

```go
logflux.InitFromEnv("my-node")
```

## BeforeSend Hooks

Filter or modify entries before they are sent. Return `nil` to drop.

```go
logflux.Init(logflux.Options{
    // Drop debug logs
    BeforeSendLog: func(p *payload.Log) *payload.Log {
        if p.Level == logflux.LogLevelDebug {
            return nil
        }
        return p
    },

    // Scrub PII from audit entries
    BeforeSendAudit: func(p *payload.Audit) *payload.Audit {
        delete(p.Attributes, "ip")
        return p
    },
})
```

Available per-type hooks: `BeforeSendLog`, `BeforeSendError`, `BeforeSendMetric`, `BeforeSendEvent`, `BeforeSendAudit`, `BeforeSendTrace`, `BeforeSendTelemetry`.

## Sampling

Drop a percentage of entries to reduce volume:

```go
logflux.Init(logflux.Options{
    SampleRate: 0.1, // send 10% of entries
})
```

Audit entries (type 5) are never sampled -- compliance requirement.

## Client Statistics

```go
stats := logflux.Stats()
fmt.Printf("Sent: %d, Dropped: %d, Queued: %d\n",
    stats.EntriesSent, stats.EntriesDropped, stats.EntriesQueued)
fmt.Printf("Drop reasons: %v\n", stats.DropReasons)
```

Drop reasons: `queue_overflow`, `network_error`, `send_error`, `ratelimit_backoff`, `quota_exceeded`, `before_send`, `validation_error`.

## Security

- **Zero-knowledge encryption**: All payloads encrypted client-side with AES-256-GCM. Server stores encrypted data without decryption capability.
- **RSA key exchange**: AES keys negotiated via RSA-2048 OAEP handshake. Server never sees plaintext keys.
- **Key safety**: AES keys copied on creation, zeroed from memory on `Close()`, mutex-protected for thread-safe access.
- **No secret leakage**: API keys masked in all public getters. Error messages never contain secrets.
- **Bounded responses**: All HTTP response reads are size-limited to prevent OOM.
- **Failsafe mode**: SDK errors never crash the host application.

## Serverless (Lambda)

```go
func handler(ctx context.Context, event events.APIGatewayProxyRequest) (Response, error) {
    defer logflux.Flush(2 * time.Second)

    logflux.Info("processing request")
    // ...
}
```

## Requirements

- Go 1.23 or later
- LogFlux.io account with API key (get one at https://dashboard.logflux.io)

## License

This SDK is licensed under the [Elastic License 2.0 (ELv2)](LICENSE).

You may use this SDK freely in your applications. The only restriction is that you may not offer it as a hosted or managed service to third parties.
