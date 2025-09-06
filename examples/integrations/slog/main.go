package main

import (
	"context"
	"log/slog"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	slog_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/slog"
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

	// Create LogFlux slog handler
	handler := slog_integration.NewHandler(batchClient, "slog-example")

	// Create slog logger with LogFlux handler
	logger := slog.New(handler)

	// Use slog as normal - logs will be sent to LogFlux
	logger.Info("Application started", "version", "1.0.0")
	logger.Warn("This is a warning", "component", "auth")
	logger.Error("An error occurred", "error", "connection timeout", "retry_count", 3)

	// Structured logging with attributes
	logger.With("user_id", 123, "session", "abc-def").
		Info("User logged in")

	// Give time for batch to flush
	time.Sleep(3 * time.Second)
}
