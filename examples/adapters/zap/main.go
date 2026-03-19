package main

import (
	"errors"
	"os"
	"time"

	"github.com/logflux-io/logflux-go-sdk/v3/pkg/adapters"
	"github.com/logflux-io/logflux-go-sdk/v3/pkg/client"
)

func main() {
	// Set up environment variables
	os.Setenv("LOGFLUX_API_KEY", "lf_your_api_key_here")
	// Server URL removed - using discovery service

	// Create a LogFlux client
	resilientClient, err := client.NewResilientClientFromEnvWithHandshake("zap-example")
	if err != nil {
		panic(err)
	}
	defer resilientClient.Close()

	// Create a zap logger adapter
	logger := adapters.NewZapLogger(resilientClient)
	defer logger.Sync()

	// Use structured logging with fields
	logger.Info("User logged in",
		adapters.String("user_id", "1234"),
		adapters.String("action", "login"),
		adapters.Int("attempts", 1),
	)

	// Use different log levels
	logger.Debug("Debug message", adapters.String("component", "auth"))
	logger.Info("Info message", adapters.String("status", "success"))
	logger.Warn("Warning message", adapters.Float64("cpu_usage", 85.5))
	logger.Error("Error message", adapters.Error(errors.New("connection failed")))

	// Named logger
	dbLogger := logger.Named("database")
	dbLogger.Info("Database connection established",
		adapters.String("host", "example-host"),
		adapters.Int("port", 5432),
		adapters.Duration("connect_time", 150*time.Millisecond),
	)

	// Logger with persistent fields
	userLogger := logger.With(
		adapters.String("user_id", "5678"),
		adapters.String("session_id", "abc123"),
	)

	userLogger.Info("User action",
		adapters.String("action", "view_profile"),
		adapters.Bool("success", true),
	)

	// Use sugar logger for printf-style logging
	sugar := logger.Sugar()
	sugar.Infof("Processing %d items", 42)
	sugar.Warnf("Memory usage: %.2f%%", 85.5)
	sugar.Errorw("Failed to process request",
		"error", "timeout",
		"duration", "30s",
		"retry_count", 3,
	)

	// Complex structured logging
	logger.Info("Request processed",
		adapters.String("method", "POST"),
		adapters.String("endpoint", "/api/users"),
		adapters.Int("status_code", 200),
		adapters.Duration("response_time", 45*time.Millisecond),
		adapters.Int64("bytes_sent", 1024),
		adapters.Any("headers", map[string]string{
			"Content-Type":  "application/json",
			"Authorization": "Bearer token",
		}),
	)

	// Error with context
	err = errors.New("database connection failed")
	logger.Error("Operation failed",
		adapters.Error(err),
		adapters.String("operation", "user_lookup"),
		adapters.String("table", "users"),
		adapters.Int("retry_count", 3),
	)

	// Wait a bit for async processing
	time.Sleep(2 * time.Second)

	// Check statistics
	stats := resilientClient.GetStats()
	logger.Info("LogFlux statistics",
		adapters.Int64("sent", stats.EntriesSent),
		adapters.Int64("dropped", stats.EntriesDropped),
		adapters.Int64("queued", stats.QueueSize),
	)
}
