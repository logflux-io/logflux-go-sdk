package main

import (
	"context"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	zap_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/zap"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func main() {
	// Create LogFlux batch client
	batchConfig := config.DefaultBatchConfig()
	batchConfig.MaxBatchSize = 10
	batchConfig.FlushInterval = 2 * time.Second

	batchClient := client.NewBatchUnixClient("/tmp/logflux-agent.sock", batchConfig)

	// Connect to LogFlux agent
	ctx := context.Background()
	if err := batchClient.Connect(ctx); err != nil {
		panic("Failed to connect to LogFlux agent: " + err.Error())
	}
	defer batchClient.Close()

	// Create LogFlux Zap core
	logfluxCore := zap_integration.NewCore(batchClient, "zap-example", zapcore.DebugLevel)

	// Option 1: Use LogFlux core only
	logger := zap.New(logfluxCore)

	// Option 2: Use multiple cores (LogFlux + console)
	consoleCore := zapcore.NewCore(
		zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
		zapcore.AddSync(zapcore.Lock(zapcore.NewMultiWriteSyncer(zapcore.AddSync(zapcore.Lock(zapcore.Lock(zapcore.NewMultiWriteSyncer())))))),
		zapcore.DebugLevel,
	)
	combinedCore := zapcore.NewTee(logfluxCore, consoleCore)
	logger = zap.New(combinedCore)

	// Use Zap normally - logs will be sent to LogFlux
	logger.Info("Application started",
		zap.String("version", "1.0.0"),
		zap.Int("port", 8080),
	)

	logger.Warn("High memory usage detected",
		zap.Float64("usage_percent", 85.5),
		zap.String("component", "cache"),
	)

	logger.Error("Database connection failed",
		zap.String("error", "connection timeout"),
		zap.Int("retry_count", 3),
		zap.Duration("timeout", 30*time.Second),
	)

	// Structured logging with nested fields - context fields are preserved
	contextLogger := logger.With(
		zap.String("user_id", "123"),
		zap.String("session", "abc-def-ghi"),
	)

	// Both context fields and log-specific fields will be included
	contextLogger.Info("User action performed",
		zap.String("action", "file_upload"),
		zap.Int64("file_size", 2048576),
	)

	// Context fields persist for additional log entries
	contextLogger.Debug("Processing file", zap.String("filename", "document.pdf"))
	contextLogger.Info("Upload completed", zap.Duration("duration", 1500*time.Millisecond))

	// Sync to ensure all logs are flushed
	logger.Sync()

	// Give time for batch to flush
	time.Sleep(3 * time.Second)
}
