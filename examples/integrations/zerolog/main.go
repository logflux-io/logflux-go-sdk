package main

import (
	"context"
	"os"
	"time"

	"github.com/logflux-io/logflux-go-sdk/pkg/client"
	"github.com/logflux-io/logflux-go-sdk/pkg/config"
	zerolog_integration "github.com/logflux-io/logflux-go-sdk/pkg/integrations/zerolog"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
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

	// Create LogFlux zerolog writer
	logWriter := zerolog_integration.NewWriter(batchClient, "zerolog-example")

	// Option 1: Replace zerolog output entirely
	logger := zerolog.New(logWriter).With().Timestamp().Logger()

	// Option 2: Send to both LogFlux and stdout (more common)
	multiWriter := logWriter.MultiWriter(os.Stdout)
	logger = zerolog.New(multiWriter).With().Timestamp().Logger()

	// Option 3: Update global logger
	log.Logger = zerolog.New(multiWriter).With().Timestamp().Logger()

	// Use zerolog normally - logs will be sent to LogFlux
	logger.Info().Msg("Application started")

	logger.Info().
		Str("version", "1.0.0").
		Int("port", 8080).
		Msg("Server configuration")

	logger.Warn().
		Float64("usage_percent", 85.5).
		Str("component", "memory").
		Msg("High resource usage detected")

	logger.Error().
		Str("error", "connection timeout").
		Int("retry_count", 3).
		Dur("timeout", 30*time.Second).
		Msg("Database connection failed")

	// Structured logging with context
	contextLogger := logger.With().
		Str("user_id", "123").
		Str("session", "abc-def-ghi").
		Logger()

	contextLogger.Info().
		Str("action", "file_upload").
		Int64("file_size", 2048576).
		Msg("User action performed")

	contextLogger.Debug().
		Str("filename", "document.pdf").
		Msg("Processing file")

	// Using global logger
	log.Info().Str("component", "main").Msg("Using global logger")

	// Complex nested data
	logger.Info().
		Interface("metadata", map[string]interface{}{
			"tags":    []string{"production", "api", "v1"},
			"metrics": map[string]float64{"cpu": 45.2, "memory": 78.9},
		}).
		Msg("Complex data logging")

	// Give time for batch to flush
	time.Sleep(3 * time.Second)
}
